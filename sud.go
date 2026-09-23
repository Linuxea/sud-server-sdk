package sud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linuxea/sud-server-sdk/api"
)

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
}

// New 创建客户端。appID/appSecret 为 Sud 平台分配的凭证，
// 可选项见 WithXxx 系列。
func New(appID, appSecret string, opts ...Option) *Client {
	cfg := DefaultConfig()
	cfg.AppID, cfg.AppSecret = appID, appSecret
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
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
	return c
}

// AppID 返回应用 id。
func (c *Client) AppID() string { return c.cfg.AppID }

// AppSecret 返回应用密钥。
func (c *Client) AppSecret() string { return c.cfg.AppSecret }

// Signer 返回签名器（供回调服务验签复用：callback.WithSigner(client.Signer())）。
func (c *Client) Signer() *Signer { return c.signer }

// Close 释放客户端资源（停止 API 地址缓存的后台刷新协程）。
func (c *Client) Close() { c.cache.close() }

// APIConfig 手动强制刷新并返回 API 地址配置（正常使用无需调用，
// Post 内部会按需拉取与刷新）。
func (c *Client) APIConfig(ctx context.Context) (*APIConfig, error) {
	if err := c.cache.refresh(ctx); err != nil {
		return nil, err
	}
	return c.cache.config(ctx)
}

// apiRespShell 出站 API 的统一响应壳。
type apiRespShell struct {
	RetCode int             `json:"ret_code"`
	RetMsg  string          `json:"ret_msg"`
	Data    json.RawMessage `json:"data"`
}

// Post 按配置 key 解析真实 URL，发送带 Sud-Auth 签名的 POST 请求，
// 校验响应壳并将 data 解析到 resp。失败按配置重试，重试前会刷新 API 地址缓存。
// 该方法实现 api.Poster 接口。
func (c *Client) Post(ctx context.Context, urlKey string, req any, resp any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("sud: 序列化请求失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			if err := c.retryWait(ctx, attempt); err != nil {
				return err
			}
		}
		data, err := c.postOnce(ctx, urlKey, body)
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
	}
	return lastErr
}

// retryWait 重试前的等待；同时按文档建议刷新一次 API 地址配置（失败保留旧缓存）。
func (c *Client) retryWait(ctx context.Context, attempt int) error {
	rctx, cancel := context.WithTimeout(context.Background(), c.cfg.HTTPTimeout)
	_ = c.cache.refresh(rctx)
	cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.cfg.RetryInterval):
		return nil
	}
}

// postOnce 执行一次完整请求：解析 URL → 签名 → 发送 → 校验壳。
func (c *Client) postOnce(ctx context.Context, urlKey string, body []byte) (json.RawMessage, error) {
	url, err := c.urlFor(ctx, urlKey)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set(HeaderAuthorization, c.signer.Authorization(body))

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{URL: url, HTTPStatus: resp.StatusCode, Body: truncateBody(respBody)}
	}

	shell := apiRespShell{}
	if err := json.Unmarshal(respBody, &shell); err != nil {
		return nil, fmt.Errorf("sud: 解析响应壳失败(key=%s): %s", urlKey, truncateBody(respBody))
	}
	if shell.RetCode != 0 {
		return nil, &APIError{URL: url, RetCode: shell.RetCode, RetMsg: shell.RetMsg, Body: truncateBody(respBody)}
	}
	if len(shell.Data) == 0 {
		return json.RawMessage("null"), nil
	}
	return shell.Data, nil
}

// urlFor 从缓存取 key 对应的真实 API 地址。
func (c *Client) urlFor(ctx context.Context, urlKey string) (string, error) {
	cfg, err := c.cache.config(ctx)
	if err != nil {
		return "", fmt.Errorf("sud: 获取API配置失败: %w", err)
	}
	var url string
	switch urlKey {
	case "mg_list":
		url = cfg.API.MGList
	case "mg_info":
		url = cfg.API.MGInfo
	case "get_game_report_info":
		url = cfg.API.GetGameReportInfo
	case "get_game_report_info_page":
		url = cfg.API.GetGameReportInfoPage
	case "query_game_report_info":
		url = cfg.API.QueryGameReportInfo
	case "get_player_results":
		url = cfg.API.GetPlayerResults
	case "report_game_round_bill":
		url = cfg.API.ReportGameRoundBill
	case "push_event":
		url = cfg.API.PushEvent
	case "create_order":
		url = cfg.API.CreateOrder
	case "batch_create_order":
		url = cfg.API.BatchCreateOrder
	case "query_order":
		url = cfg.API.QueryOrder
	case "query_match_base":
		url = cfg.API.QueryMatchBase
	case "query_match_round_ids":
		url = cfg.API.QueryMatchRoundIds
	case "query_user_settle":
		url = cfg.API.QueryUserSettle
	case "llm_create_voice":
		url = cfg.LLMAPI.CreateVoice
	case "llm_train_voice":
		url = cfg.LLMAPI.TrainVoice
	case "llm_get_voice":
		url = cfg.LLMAPI.GetVoice
	case "llm_create_ai_character":
		url = cfg.LLMAPI.CreateAICharacter
	case "llm_get_ai_character":
		url = cfg.LLMAPI.GetAICharacter
	}
	if url == "" {
		return "", fmt.Errorf("sud: API 地址未配置(key=%s)", urlKey)
	}
	return url, nil
}
