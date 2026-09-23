// Package sud 封装 Sud MGP（小游戏平台）服务端全部接口：
// 出站（业务方调 Sud：API 查询、推送事件、下单等）与
// 入站（业务方提供 HTTPS 回调：鉴权链、对局上报、异步通知等）。
//
// 快速接入（请求/响应类型在 api 包，import sudapi "github.com/linuxea/sud-server-sdk/api"）：
//
//	client := sud.New(appID, appSecret)
//	list, err := client.GameList.List(ctx, sudapi.GameListReq{Platform: 1})
//
//	srv := callback.NewServer(callback.Callbacks{
//		GetSSToken: func(ctx context.Context, req callback.GetSSTokenReq) (*callback.GetSSTokenResp, error) { ... },
//	}, callback.WithSigner(client.Signer()))
//	http.Handle("/sud/", srv.Handler())
package sud

import (
	"net/http"
	"time"
)

// Doer 抽象 HTTP 执行器。默认使用 *http.Client，测试时可注入自定义实现。
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Config 客户端配置，通过 New 的 functional options 填充。
type Config struct {
	// AppID / AppSecret Sud 平台分配的应用凭证
	AppID     string
	AppSecret string

	// APIConfigBase 获取服务端 API 配置的基地址，最终请求
	// {APIConfigBase}/{HmacMD5(app_id)}。默认生产环境 https://asc.sudden.ltd/
	APIConfigBase string

	// HTTPTimeout 单次 HTTP 请求超时，默认 10s
	HTTPTimeout time.Duration

	// RetryCount 出站请求失败重试次数（不含首次），默认 3
	RetryCount int
	// RetryInterval 重试间隔，默认 100ms
	RetryInterval time.Duration

	// APIConfigRefreshInterval API 实际地址缓存的自动刷新间隔，
	// 默认 24h；设为负值表示不自动刷新（仅首次拉取）。
	APIConfigRefreshInterval time.Duration

	// Doer 自定义 HTTP 执行器，默认 &http.Client{Timeout: HTTPTimeout}
	Doer Doer
}

// DefaultConfig 各字段默认值（AppID/AppSecret 由 New 传入）。
func DefaultConfig() Config {
	return Config{
		APIConfigBase:            "https://asc.sudden.ltd/",
		HTTPTimeout:              10 * time.Second,
		RetryCount:               3,
		RetryInterval:            100 * time.Millisecond,
		APIConfigRefreshInterval: 24 * time.Hour,
	}
}

// Option config 的可选项，见 WithXxx 系列。
type Option func(*Config)

// WithHTTPTimeout 设置单次 HTTP 请求超时。
func WithHTTPTimeout(d time.Duration) Option {
	return func(c *Config) { c.HTTPTimeout = d }
}

// WithRetry 设置失败重试次数（不含首次）与重试间隔。
func WithRetry(count int, interval time.Duration) Option {
	return func(c *Config) { c.RetryCount, c.RetryInterval = count, interval }
}

// WithDoer 注入自定义 HTTP 执行器（测试用）。
func WithDoer(d Doer) Option {
	return func(c *Config) { c.Doer = d }
}

// WithAPIConfigBase 覆盖 API 配置服务基地址（默认 https://asc.sudden.ltd/）。
func WithAPIConfigBase(base string) Option {
	return func(c *Config) { c.APIConfigBase = base }
}

// WithAPIConfigRefreshInterval 设置 API 地址缓存自动刷新间隔；
// 传负值关闭自动刷新。
func WithAPIConfigRefreshInterval(d time.Duration) Option {
	return func(c *Config) { c.APIConfigRefreshInterval = d }
}
