package callback

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Server Sud HTTPS 回调服务。
type Server struct {
	cbs    Callbacks
	signer *sud.Signer // 非 nil 时开启验签
	prefix string      // 路由前缀，默认 /sud/
	logger Logger
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

// NewServer 创建回调服务。Callbacks 中未实现的同步回调会返回错误壳。
func NewServer(cbs Callbacks, opts ...Option) *Server {
	s := &Server{cbs: cbs, prefix: "/sud/", logger: nopLogger{}}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// Handler 返回组装好的 http.Handler（含路由与可选验签中间件），
// 直接挂到业务 mux：http.Handle("/sud/", srv.Handler())。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.register(mux)
	var h http.Handler = mux
	if s.signer != nil {
		h = s.verifyMiddleware(h)
	}
	return h
}

// verifyMiddleware 标准中间件签名，读取原始 body 验签后回填。
// 验签失败返回错误壳（ret_code=1）。
func (s *Server) verifyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
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

// decodeBody 解析请求 body。解析失败按 gama 现网行为：记日志并返回成功空壳，
// 避免 Sud 侧无意义重试；返回 false 表示响应已写完。
func (s *Server) decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Printf("sud callback: 读 body 失败 %s: %v", r.URL.Path, err)
		writeJSON(w, resp{RetCode: 0})
		return false
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return true // 空 body，target 保持零值
	}
	if err := json.Unmarshal(body, target); err != nil {
		s.logger.Printf("sud callback: 解析 body 失败 %s: %v", r.URL.Path, err)
		writeJSON(w, resp{RetCode: 0})
		return false
	}
	return true
}

// writeResult 把回调结果转成统一响应壳。
func (s *Server) writeResult(w http.ResponseWriter, data any, err error) {
	if err == nil {
		writeJSON(w, resp{RetCode: 0, Data: data})
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

func (s *Server) handleGetSSToken(w http.ResponseWriter, r *http.Request) {
	var req GetSSTokenReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.GetSSToken == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetSSToken(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleUpdateSSToken(w http.ResponseWriter, r *http.Request) {
	var req UpdateSSTokenReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.UpdateSSToken == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.UpdateSSToken(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	var req GetUserInfoReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.GetUserInfo == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetUserInfo(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleReportGameInfo(w http.ResponseWriter, r *http.Request) {
	var req ReportGameInfoReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.ReportGameInfo == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	err := s.cbs.ReportGameInfo(r.Context(), req)
	s.writeResult(w, nil, err)
}

func (s *Server) handleGetAccount(w http.ResponseWriter, r *http.Request) {
	var req GetAccountReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.GetAccount == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetAccount(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleGetScore(w http.ResponseWriter, r *http.Request) {
	var req GetScoreReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.GetScore == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.GetScore(r.Context(), req)
	s.writeResult(w, respData, err)
}

func (s *Server) handleUpdateScore(w http.ResponseWriter, r *http.Request) {
	var req UpdateScoreReq
	if !s.decodeBody(w, r, &req) {
		return
	}
	if s.cbs.UpdateScore == nil {
		s.writeResult(w, nil, notImplementedErr)
		return
	}
	respData, err := s.cbs.UpdateScore(r.Context(), req)
	s.writeResult(w, respData, err)
}

// handleNotify 异步通知入口（stage 7 实现）。
func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	s.dispatchNotify(w, r)
}
