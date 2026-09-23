package sud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linuxea/sud-server-sdk/api"
)

// ErrURLKeyUnknown urlKey 不在 SDK 已知集合内（SDK 未适配或拼写错误）。
var ErrURLKeyUnknown = errors.New("sud: 未知的 API 地址 key")

// ErrURLKeyNotConfigured 配置服务未返回该 key 的地址（Sud 侧未开通该服务）。
var ErrURLKeyNotConfigured = errors.New("sud: API 地址未配置")

// Client Sud 服务端 SDK 出站客户端。
// 持有签名器、HTTP 执行器与 API 地址缓存，并实现 api.Poster 接口；
// 各出站 API 服务域以组合字段形式暴露。
//
// 用法：
//
//	client := sud.New(appID, appSecret)
//	defer client.Close()
//	list, err := client.GameList.List(ctx, sudapi.GameListReq{Platform: sudapi.PlatformIOS})
type Client struct {
	cfg    Config
	doer   Doer
	signer *Signer
	cache  *apiCache

	// GameList 游戏列表/信息
	GameList *api.GameListService
	// Report 游戏上报查询（单个/分页）
	Report *api.ReportService
	// Order 游戏内付费订单
	Order *api.OrderService
	// EntryScore 带分入场查询（德州扑克/TeenPatti）
	EntryScore *api.EntryScoreService
	// LLM 大模型音色与 AI 角色
	LLM *api.LLMService
	// PushEvent 推送事件到游戏服务
	PushEvent *api.PushEventService
}

// New 创建客户端。appID/appSecret 为 Sud 平台分配的凭证，
// 可选项见 WithXxx 系列。非法配置值（如非正的超时）回退为默认值。
func New(appID, appSecret string, opts ...Option) *Client {
	cfg := DefaultConfig()
	cfg.AppID, cfg.AppSecret = appID, appSecret
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = DefaultConfig().HTTPTimeout
	}
	if cfg.RetryInterval < 0 {
		cfg.RetryInterval = 0
	}
	if cfg.APIConfigBase == "" {
		cfg.APIConfigBase = DefaultConfig().APIConfigBase
	}
	c := &Client{cfg: cfg, signer: NewSigner(appID, appSecret)}
	if cfg.Doer != nil {
		c.doer = cfg.Doer
	} else {
		c.doer = &http.Client{Timeout: cfg.HTTPTimeout}
	}
	c.cache = newAPICache(c)

	// 组合各出站服务域（Client 实现 api.Poster）
	c.GameList = api.NewGameListService(c)
	c.Report = api.NewReportService(c)
	c.Order = api.NewOrderService(c)
	c.EntryScore = api.NewEntryScoreService(c)
	c.LLM = api.NewLLMService(c)
	c.PushEvent = api.NewPushEventService(c)
	return c
}

// AppID 返回应用 id。
func (c *Client) AppID() string { return c.cfg.AppID }

// AppSecret 返回应用密钥。
func (c *Client) AppSecret() string { return c.cfg.AppSecret }

// Signer 返回签名器（供回调服务验签复用：callback.WithSigner(client.Signer())）。
func (c *Client) Signer() *Signer { return c.signer }

// Close 释放客户端资源（停止 API 地址缓存的后台刷新协程）。
// Close 之后继续 Post 会返回 ErrClosed。幂等，可重复调用。
func (c *Client) Close() { c.cache.close() }

// APIConfig 手动强制刷新并返回 API 地址配置（正常使用无需调用，
// Post 内部会按需拉取与刷新）。刷新失败时若有旧缓存则回退返回旧值。
func (c *Client) APIConfig(ctx context.Context) (*APIConfig, error) {
	cfg, err := c.cache.refresh(ctx)
	if err != nil {
		if old, ok := c.cache.current(); ok {
			return old, nil
		}
		return nil, err
	}
	return cfg, nil
}

// apiRespShell 出站 API 的统一响应壳。
type apiRespShell struct {
	RetCode int             `json:"ret_code"`
	RetMsg  string          `json:"ret_msg"`
	Data    json.RawMessage `json:"data"`
}

// Post 按配置 key 解析真实 URL，发送带 Sud-Auth 签名的 POST 请求，
// 校验响应壳并将 data 解析到 resp。该方法实现 api.Poster 接口。
//
// 重试策略：仅对瞬态失败重试（网络错误、HTTP 5xx、响应壳不可解析），
// 重试前按文档建议刷新一次 API 地址缓存（失败保留旧缓存）。
// 以下确定性失败不重试，直接返回：
//   - 业务错误（ret_code != 0，如参数错误——重试结果必然相同）
//   - HTTP 4xx
//   - 未知 key（ErrURLKeyUnknown）/ 地址未配置（ErrURLKeyNotConfigured）
func (c *Client) Post(ctx context.Context, urlKey string, req any, resp any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("sud: 序列化请求失败: %w", err)
	}

	var lastErr error
	for attempt := 0; ; attempt++ {
		data, retriable, err := c.postOnce(ctx, urlKey, body)
		if err == nil {
			if resp == nil {
				return nil
			}
			if err := json.Unmarshal(data, resp); err != nil {
				return fmt.Errorf("sud: 解析响应 data 失败(key=%s): %w", urlKey, err)
			}
			return nil
		}
		lastErr = err
		if !retriable || attempt >= c.cfg.RetryCount {
			return lastErr
		}
		if err := c.retryWait(ctx); err != nil {
			return err
		}
	}
}

// retryWait 重试前的等待：先感知调用方 ctx 取消，
// 再按文档建议刷新一次 API 地址配置（与请求 ctx 解耦，失败保留旧缓存）。
func (c *Client) retryWait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.cfg.RetryInterval):
	}
	rctx, cancel := context.WithTimeout(context.Background(), c.cfg.HTTPTimeout)
	_, _ = c.cache.refresh(rctx)
	cancel()
	return nil
}

// postOnce 执行一次完整请求：解析 URL → 签名 → 发送 → 校验壳。
// retriable 表示该失败是否值得重试。
func (c *Client) postOnce(ctx context.Context, urlKey string, body []byte) (data json.RawMessage, retriable bool, err error) {
	url, err := c.urlFor(ctx, urlKey)
	if err != nil {
		return nil, false, err // 未配置/未知 key 是确定性失败
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set(HeaderAuthorization, c.signer.Authorization(body))

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, true, err // 网络错误可重试
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err // 读取中断可重试
	}
	if resp.StatusCode != http.StatusOK {
		// 5xx 瞬态可重试；4xx 客户端错误重试无意义
		return nil, resp.StatusCode >= 500, &APIError{URL: url, HTTPStatus: resp.StatusCode, Body: truncateBody(respBody)}
	}

	shell := apiRespShell{}
	if err := json.Unmarshal(respBody, &shell); err != nil {
		// 网关异常响应可能瞬态，可重试
		return nil, true, fmt.Errorf("sud: 解析响应壳失败(key=%s): %s", urlKey, truncateBody(respBody))
	}
	if shell.RetCode != 0 {
		// 业务错误（参数/状态类）是确定性失败，不重试
		return nil, false, &APIError{URL: url, RetCode: shell.RetCode, RetMsg: shell.RetMsg, Body: truncateBody(respBody)}
	}
	if len(shell.Data) == 0 {
		return json.RawMessage("null"), false, nil
	}
	return shell.Data, false, nil
}

// urlResolvers urlKey → 从 APIConfig 取地址。集中一处维护，避免长 switch。
var urlResolvers = map[string]func(*APIConfig) string{
	"mg_list":                   func(c *APIConfig) string { return c.API.MGList },
	"mg_info":                   func(c *APIConfig) string { return c.API.MGInfo },
	"get_game_report_info":      func(c *APIConfig) string { return c.API.GetGameReportInfo },
	"get_game_report_info_page": func(c *APIConfig) string { return c.API.GetGameReportInfoPage },
	"query_game_report_info":    func(c *APIConfig) string { return c.API.QueryGameReportInfo },
	"get_player_results":        func(c *APIConfig) string { return c.API.GetPlayerResults },
	"report_game_round_bill":    func(c *APIConfig) string { return c.API.ReportGameRoundBill },
	"push_event":                func(c *APIConfig) string { return c.API.PushEvent },
	"create_order":              func(c *APIConfig) string { return c.API.CreateOrder },
	"batch_create_order":        func(c *APIConfig) string { return c.API.BatchCreateOrder },
	"query_order":               func(c *APIConfig) string { return c.API.QueryOrder },
	"query_match_base":          func(c *APIConfig) string { return c.API.QueryMatchBase },
	"query_match_round_ids":     func(c *APIConfig) string { return c.API.QueryMatchRoundIds },
	"query_user_settle":         func(c *APIConfig) string { return c.API.QueryUserSettle },
	"llm_create_voice":          func(c *APIConfig) string { return c.LLMAPI.CreateVoice },
	"llm_train_voice":           func(c *APIConfig) string { return c.LLMAPI.TrainVoice },
	"llm_get_voice":             func(c *APIConfig) string { return c.LLMAPI.GetVoice },
	"llm_create_ai_character":   func(c *APIConfig) string { return c.LLMAPI.CreateAICharacter },
	"llm_get_ai_character":      func(c *APIConfig) string { return c.LLMAPI.GetAICharacter },
}

// urlFor 从缓存取 key 对应的真实 API 地址。
// key 未知返回 ErrURLKeyUnknown；Sud 未返回地址返回 ErrURLKeyNotConfigured。
func (c *Client) urlFor(ctx context.Context, urlKey string) (string, error) {
	resolve, ok := urlResolvers[urlKey]
	if !ok {
		return "", fmt.Errorf("%w(key=%s)", ErrURLKeyUnknown, urlKey)
	}
	cfg, err := c.cache.config(ctx)
	if err != nil {
		return "", fmt.Errorf("sud: 获取API配置失败: %w", err)
	}
	url := resolve(cfg)
	if url == "" {
		return "", fmt.Errorf("%w(key=%s)", ErrURLKeyNotConfigured, urlKey)
	}
	return url, nil
}
