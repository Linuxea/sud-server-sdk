package callback

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"github.com/linuxea/sud-server-sdk"
)

// CallbackError 业务回调返回错误时的自定义响应内容。
type CallbackError struct {
	RetCode      int32
	RetMsg       string
	SDKErrorCode int32
}

func (e *CallbackError) Error() string {
	return fmt.Sprintf("callback error ret_code=%d sdk_error_code=%d: %s", e.RetCode, e.SDKErrorCode, e.RetMsg)
}

// NewCallbackError 构造自定义错误响应（handler 返回它可控制 ret_code/sdk_error_code/ret_msg）。
func NewCallbackError(retCode int32, retMsg string, sdkErrorCode int32) *CallbackError {
	return &CallbackError{RetCode: retCode, RetMsg: retMsg, SDKErrorCode: sdkErrorCode}
}

// resp 统一响应壳。
type resp struct {
	RetCode      int32  `json:"ret_code"`
	RetMsg       string `json:"ret_msg"`
	SDKErrorCode int32  `json:"sdk_error_code,omitempty"`
	Data         any    `json:"data,omitempty"`
}

// Logger 简单日志接口，业务方可注入自定义实现。
type Logger interface {
	Printf(format string, v ...any)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}

// DefaultMaxBodySize 回调 body 默认上限（draw.image 等通知可携带 MB 级 base64 图片）。
const DefaultMaxBodySize = 10 << 20 // 10MB

// Server Sud HTTPS 回调服务。
type Server struct {
	cbs          Callbacks
	signer       *sud.Signer // 非 nil 时开启验签
	prefix       string      // 路由前缀，默认 /sud/
	logger       Logger
	maxBodySize  int64
	strictDecode bool // body 解析失败时返回错误壳触发 Sud 重试
}

// Option 服务可选项。
type Option func(*Server)

// WithSigner 开启回调验签（通常传 client.Signer()）。
func WithSigner(s *sud.Signer) Option {
	return func(srv *Server) { srv.signer = s }
}

// WithPathPrefix 自定义路由前缀（默认 "/sud/"）。
// 需以 / 开头；结尾的 / 可有可无。
func WithPathPrefix(p string) Option {
	return func(srv *Server) {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		srv.prefix = p
	}
}

// WithLogger 注入日志实现（默认丢弃日志）。
func WithLogger(l Logger) Option {
	return func(srv *Server) { srv.logger = l }
}

// WithMaxBodySize 设置回调 body 读取上限（默认 10MB），超限按解析失败处理。
func WithMaxBodySize(n int64) Option {
	return func(srv *Server) {
		if n > 0 {
			srv.maxBodySize = n
		}
	}
}

// WithStrictDecode 严格模式：body 非法/为空/超限时返回 ret_code=1 触发 Sud 重试，
// 而非默认的"记日志返回成功壳"（默认行为与 gama 现网一致，避免无意义重试；
// 代价是非法回调数据被静默丢弃，只留日志）。
func WithStrictDecode() Option {
	return func(srv *Server) { srv.strictDecode = true }
}

// NewServer 创建回调服务。Callbacks 中未实现的同步回调会返回错误壳。
func NewServer(cbs Callbacks, opts ...Option) *Server {
	s := &Server{cbs: cbs, prefix: "/sud/", logger: nopLogger{}, maxBodySize: DefaultMaxBodySize}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// Handler 返回组装好的 http.Handler：
// 方法守卫（非 POST 405）→ 可选验签 → panic recover → 路由。
// 直接挂到业务 mux：http.Handle("/sud/", srv.Handler())。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.register(mux)
	var h http.Handler = mux
	h = s.recoverMiddleware(h)
	if s.signer != nil {
		h = s.verifyMiddleware(h)
	}
	return methodGuard(h)
}

// methodGuard 仅放行 POST。
func methodGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// verifyMiddleware 标准中间件签名，读取原始 body 验签后回填。
// 验签失败返回错误壳（ret_code=1）。
func (s *Server) verifyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxBodySize))
		_ = r.Body.Close()
		if err != nil {
			writeJSON(w, resp{RetCode: 1, RetMsg: "read body failed"})
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		if !s.signer.VerifyCallback(r.Header, body) {
			s.logger.Printf("sud callback: 验签失败 %s %s", r.Method, r.URL.Path)
			writeJSON(w, resp{RetCode: 1, RetMsg: "invalid signature"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoverMiddleware 业务 handler panic 时记日志并返回错误壳。
// notify 侧 Sud 会重试，业务需保证 notify 处理幂等。
func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Printf("sud callback: panic %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				writeJSON(w, resp{RetCode: 1, RetMsg: "internal error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) register(mux *http.ServeMux) {
	reg := func(path string, h http.HandlerFunc) {
		mux.HandleFunc(s.prefix+path, h)
	}
	reg("get_sstoken", s.handleGetSSToken)
	reg("update_sstoken", s.handleUpdateSSToken)
	reg("get_user_info", s.handleGetUserInfo)
	reg("report_game_info", s.handleReportGameInfo)
	reg("get_account", s.handleGetAccount)
	reg("get_score", s.handleGetScore)
	reg("update_score", s.handleUpdateScore)
	reg("notify", s.handleNotify)
}

// syncFailResp 同步回调 body 失败时的应答。
func (s *Server) syncFailResp() resp {
	if s.strictDecode {
		return resp{RetCode: 1, RetMsg: "invalid body"}
	}
	return resp{RetCode: 0}
}

// notifyFailResp notify body 失败时的应答（文档要求 ret_msg 为 SUCCESS）。
func (s *Server) notifyFailResp() resp {
	if s.strictDecode {
		return resp{RetCode: 1, RetMsg: "invalid body"}
	}
	return resp{RetCode: 0, RetMsg: "SUCCESS"}
}

// readJSON 读取（受 maxBodySize 限制）并解析 body 到 target。
// 空 body 与非法 body 一致处理：记日志并写 failResp，不进业务 handler。
// 返回 false 表示响应已写完。
func (s *Server) readJSON(w http.ResponseWriter, r *http.Request, target any, failResp resp) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxBodySize))
	if err != nil {
		s.logger.Printf("sud callback: 读 body 失败/超限 %s: %v", r.URL.Path, err)
		writeJSON(w, failResp)
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		s.logger.Printf("sud callback: body 为空 %s", r.URL.Path)
		writeJSON(w, failResp)
		return false
	}
	if err := json.Unmarshal(body, target); err != nil {
		s.logger.Printf("sud callback: 解析 body 失败 %s: %v body=%s", r.URL.Path, err, truncateBodyStr(body))
		writeJSON(w, failResp)
		return false
	}
	return true
}

// nilSafe 将 typed-nil（如业务返回 (*GetSSTokenResp)(nil), nil）归一为 untyped nil，
// 使 resp.Data 的 omitempty 生效，避免输出 "data":null。
func nilSafe(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
	}
	return v
}

// writeResult 把回调结果转成统一响应壳。
func (s *Server) writeResult(w http.ResponseWriter, data any, err error) {
	if err == nil {
		writeJSON(w, resp{RetCode: 0, Data: nilSafe(data)})
		return
	}
	var cbErr *CallbackError
	if errors.As(err, &cbErr) {
		writeJSON(w, resp{RetCode: cbErr.RetCode, RetMsg: cbErr.RetMsg, SDKErrorCode: cbErr.SDKErrorCode})
		return
	}
	writeJSON(w, resp{RetCode: 1, RetMsg: err.Error()})
}

// notImplementedErr 未实现回调的错误。
var notImplementedErr = &CallbackError{RetCode: 1, RetMsg: "callback not implemented"}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// truncateBodyStr body 过长时截断，供日志使用。
func truncateBodyStr(b []byte) string {
	const max = 256
	if len(b) > max {
		return string(b[:max]) + "...(truncated)"
	}
	return string(b)
}
