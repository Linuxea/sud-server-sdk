package sud

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APIConfig 「获取服务端API配置」返回的全部地址分组。
type APIConfig struct {
	// API 通用服务端接口（查询/推送/订单/带分入场）
	API APIConfigAPI `json:"api"`
	// MatchAPI 匹配相关（文档示例中出现，当前 SDK 未使用）
	MatchAPI json.RawMessage `json:"match_api,omitempty"`
	// CrossAppAPI 跨应用（文档示例中出现，当前 SDK 未使用）
	CrossAppAPI json.RawMessage `json:"cross_app_api,omitempty"`
	// BulletAPI 弹幕相关（文档示例中出现，当前 SDK 未使用）
	BulletAPI json.RawMessage `json:"bullet_api,omitempty"`
	// LLMAPI 大模型服务
	LLMAPI APIConfigLLM `json:"llm_api"`
}

// APIConfigAPI 通用服务端接口地址。
type APIConfigAPI struct {
	MGList                string `json:"mg_list"`
	MGInfo                string `json:"mg_info"`
	GetGameReportInfo     string `json:"get_game_report_info"`
	GetGameReportInfoPage string `json:"get_game_report_info_page"`
	QueryGameReportInfo   string `json:"query_game_report_info"`
	GetPlayerResults      string `json:"get_player_results"`
	ReportGameRoundBill   string `json:"report_game_round_bill"`
	PushEvent             string `json:"push_event"`
	CreateOrder           string `json:"create_order"`
	BatchCreateOrder      string `json:"batch_create_order"`
	QueryOrder            string `json:"query_order"`
	QueryMatchBase        string `json:"query_match_base"`
	QueryMatchRoundIds    string `json:"query_match_round_ids"`
	QueryUserSettle       string `json:"query_user_settle"`
	// 以下出现在返回示例但未列入文档表格
	AuthAppList  string `json:"auth_app_list"`
	AuthRoomList string `json:"auth_room_list"`
}

// APIConfigLLM 大模型服务接口地址。
type APIConfigLLM struct {
	CreateVoice       string `json:"create_voice"`
	TrainVoice        string `json:"train_voice"`
	GetVoice          string `json:"get_voice"`
	CreateAICharacter string `json:"create_ai_character"`
	GetAICharacter    string `json:"get_ai_character"`
}

// appServerSign 生成 API 配置服务的路径签名：HmacMD5(app_id)，密钥为 app_secret。
func appServerSign(appID, appSecret string) string {
	mac := hmac.New(md5.New, []byte(appSecret))
	mac.Write([]byte(appID))
	return hex.EncodeToString(mac.Sum(nil))
}

// ErrClosed Client.Close 之后继续使用时返回。
var ErrClosed = fmt.Errorf("sud: 客户端已关闭")

// apiCache API 地址缓存：惰性拉取 + 后台定时刷新 + 失败回退上次结果。
//
// 并发约定：
//   - stop/stopped 在构造时预建，永不替换（消除字段写竞争）
//   - closed/started/cfg/inflight 均由 mu 保护
//   - Close 之后再调用 config 返回 ErrClosed，且不会启动后台协程（防泄漏）
type apiCache struct {
	client *Client

	mu  sync.RWMutex
	cfg *APIConfig

	stop    chan struct{}
	stopped chan struct{}

	closed   bool
	started  bool
	inflight chan struct{} // 首次并发拉取去重（singleflight 简版）
}

func newAPICache(c *Client) *apiCache {
	return &apiCache{
		client:  c,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

// config 返回缓存的 API 配置；未拉取过则同步拉取一次（并发去重）。
// 拉取失败且无历史缓存时返回错误。
func (a *apiCache) config(ctx context.Context) (*APIConfig, error) {
	a.mu.RLock()
	cfg, closed := a.cfg, a.closed
	a.mu.RUnlock()
	if cfg != nil {
		return cfg, nil
	}
	if closed {
		return nil, ErrClosed
	}

	// singleflight 简版：并发首次调用只发起一次拉取，其余等待结果
	a.mu.Lock()
	if a.cfg != nil { // double check
		cfg := a.cfg
		a.mu.Unlock()
		return cfg, nil
	}
	if a.closed {
		a.mu.Unlock()
		return nil, ErrClosed
	}
	call := a.inflight
	first := call == nil
	if first {
		call = make(chan struct{})
		a.inflight = call
	}
	a.mu.Unlock()

	if !first {
		select {
		case <-call:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return a.config(ctx)
	}

	cfg, err := a.fetch(ctx)
	a.mu.Lock()
	if err == nil {
		a.cfg = cfg
	}
	close(call)
	a.inflight = nil
	a.mu.Unlock()
	if err != nil {
		return nil, err
	}
	a.startBackgroundRefresh()
	return cfg, nil
}

// refresh 强制重新拉取；失败时保留旧缓存。成功返回新配置。
func (a *apiCache) refresh(ctx context.Context) (*APIConfig, error) {
	cfg, err := a.fetch(ctx)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return cfg, nil
}

// fetch 实际请求配置服务。GET {APIConfigBase}/{HmacMD5(app_id)}，无认证头。
func (a *apiCache) fetch(ctx context.Context) (*APIConfig, error) {
	base := strings.TrimSuffix(a.client.cfg.APIConfigBase, "/")
	url := base + "/" + appServerSign(a.client.cfg.AppID, a.client.cfg.AppSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := a.client.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{URL: url, HTTPStatus: resp.StatusCode, Body: truncateBody(body)}
	}
	cfg := &APIConfig{}
	if err := json.Unmarshal(body, cfg); err != nil {
		return nil, fmt.Errorf("sud: 解析API配置失败: %w", err)
	}
	return cfg, nil
}

// startBackgroundRefresh 按配置间隔启动后台刷新协程（仅启动一次）。
// interval 非正值（含 0）不启动自动刷新。
func (a *apiCache) startBackgroundRefresh() {
	interval := a.client.cfg.APIConfigRefreshInterval
	if interval <= 0 {
		return
	}
	a.mu.Lock()
	if a.started || a.closed {
		a.mu.Unlock()
		return
	}
	a.started = true
	a.mu.Unlock()

	go func() {
		defer close(a.stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), a.client.cfg.HTTPTimeout)
				_, _ = a.refresh(ctx) // 失败保留旧缓存
				cancel()
			case <-a.stop:
				return
			}
		}
	}()
}

// close 停止后台刷新。幂等：重复调用安全。
// 在首次拉取之前 close 时后台协程从未启动，也不会再启动。
func (a *apiCache) close() {
	a.mu.Lock()
	if a.closed {
		started := a.started
		a.mu.Unlock()
		if started {
			<-a.stopped // 等待已在途的停止完成
		}
		return
	}
	a.closed = true
	started := a.started
	a.mu.Unlock()

	if !started {
		return
	}
	close(a.stop)
	<-a.stopped
}

// truncateBody 响应体过长时截断，供错误信息携带。
func truncateBody(b []byte) string {
	const max = 512
	if len(b) > max {
		return string(b[:max]) + "...(truncated)"
	}
	return string(b)
}
