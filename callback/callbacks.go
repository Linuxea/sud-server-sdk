// Package callback 实现 Sud 调用业务方的 HTTPS 回调服务。
//
// Sud 侧各回调 URL（生产/测试）需在 Sud 平台配置并指向本服务暴露的路径。
// 业务方按需实现 Callbacks 中的 func 字段，未实现的同步回调返回错误壳、
// 未实现的 notify 仅记日志。
//
//	srv := callback.NewServer(callback.Callbacks{...}, callback.WithSigner(client.Signer()))
//	http.Handle("/", srv.Handler())
package callback

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/linuxea/sud-server-sdk/api"
)

// Callbacks 业务方接入点：每个回调一个 func 字段，按需实现。
type Callbacks struct {
	// GetSSToken 必实现。Code 换 SSToken（鉴权链入口）。
	// 建议在返回中携带 UserInfo，Sud 有数据时不再调用 GetUserInfo。
	GetSSToken func(ctx context.Context, req GetSSTokenReq) (*GetSSTokenResp, error)

	// UpdateSSToken 必实现。验旧 SSToken 换新。
	UpdateSSToken func(ctx context.Context, req UpdateSSTokenReq) (*UpdateSSTokenResp, error)

	// GetUserInfo 必实现。SSToken 换用户信息。
	GetUserInfo func(ctx context.Context, req GetUserInfoReq) (*UserInfo, error)

	// ReportGameInfo 必实现。对局上报（game_start / game_settle）。
	// 用 req.ParseGameStart() / req.ParseGameSettle() 解析 ReportMsg。
	ReportGameInfo func(ctx context.Context, req ReportGameInfoReq) error

	// GetAccount Betting 类游戏（德州扑克/TeenPatti 等）进入游戏时获取用户账户。
	GetAccount func(ctx context.Context, req GetAccountReq) (*GetAccountResp, error)

	// GetScore 已废弃，仅为兼容保留。
	//
	// Deprecated: 文档标注已废弃，由 GetAccount 取代。
	GetScore func(ctx context.Context, req GetScoreReq) (*GetScoreResp, error)

	// UpdateScore 更新用户积分（下注/获胜通知）。已废弃，仅为兼容保留。
	// 注意幂等（order_id）与积分加锁；余额不足/订单重复用 ErrInsufficientBalance / ErrDuplicateOrderID。
	//
	// Deprecated: 文档标注已废弃。
	UpdateScore func(ctx context.Context, req UpdateScoreReq) (*UpdateScoreResp, error)

	// OnNotify 异步通知兜底：notify 事件未注册对应处理时调用（仅记日志亦可）。
	// event 为 sud.mg.merchant.* 值，payload 为原始 JSON。
	OnNotify func(ctx context.Context, event string, payload json.RawMessage)

	// NotifyCallbacks 15 种异步通知的逐事件处理（嵌入组合，字段直接平铺访问）。
	NotifyCallbacks
}

// GetSSTokenReq get_sstoken 请求。
type GetSSTokenReq struct {
	Code string `json:"code"`
}

// UserInfo 用户信息（get_sstoken.user_info 与 get_user_info.data 共用）。
type UserInfo struct {
	UID       string `json:"uid"`
	NickName  string `json:"nick_name"`
	AvatarURL string `json:"avatar_url"`
	Gender    string `json:"gender"` // female / male / ""
	IsAI      int32  `json:"is_ai,omitempty"`
	AILevel   int32  `json:"ai_level,omitempty"`
}

// GetSSTokenResp get_sstoken 响应。
type GetSSTokenResp struct {
	SSToken       string    `json:"ss_token"`
	ExpireDate    int64     `json:"expire_date"` // 毫秒
	ExpireDateStr string    `json:"expire_date_str,omitempty"`
	UserInfo      *UserInfo `json:"user_info,omitempty"`
}

// UpdateSSTokenReq update_sstoken 请求。
type UpdateSSTokenReq struct {
	SSToken string `json:"ss_token"`
}

// UpdateSSTokenResp update_sstoken 响应。
type UpdateSSTokenResp struct {
	SSToken       string `json:"ss_token"`
	ExpireDate    int64  `json:"expire_date"`
	ExpireDateStr string `json:"expire_date_str,omitempty"`
}

// GetUserInfoReq get_user_info 请求。
type GetUserInfoReq struct {
	SSToken string `json:"ss_token"`
}

// 上报类型常量（与 api 包对齐，值来自文档）。
const (
	ReportTypeGameStart  = api.ReportTypeGameStart
	ReportTypeGameSettle = api.ReportTypeGameSettle
)

// ReportGameInfoReq report_game_info 请求。
type ReportGameInfoReq struct {
	ReportType string          `json:"report_type"` // game_start / game_settle
	ReportMsg  json.RawMessage `json:"report_msg"`
	// UID / SSToken 文档标注后续不再维护，建议不关注
	UID     string `json:"uid"`
	SSToken string `json:"ss_token"`
}

// ParseGameStart 解析 game_start 上报体。
func (r *ReportGameInfoReq) ParseGameStart() (*api.GameStartObject, error) {
	var msg api.GameStartObject
	if err := json.Unmarshal(r.ReportMsg, &msg); err != nil {
		return nil, fmt.Errorf("解析 game_start_object: %w", err)
	}
	return &msg, nil
}

// ParseGameSettle 解析 game_settle 上报体。
func (r *ReportGameInfoReq) ParseGameSettle() (*api.GameSettleObject, error) {
	var msg api.GameSettleObject
	if err := json.Unmarshal(r.ReportMsg, &msg); err != nil {
		return nil, fmt.Errorf("解析 game_settle_object: %w", err)
	}
	return &msg, nil
}

// GetAccountReq get_account 请求。
type GetAccountReq struct {
	UID string `json:"uid"`
}

// GetAccountResp get_account 响应。
type GetAccountResp struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Score     int64  `json:"score"`
	VIPLevel  int64  `json:"vip_level"` // 0-3
}

// GetScoreReq get_score 请求。
type GetScoreReq struct {
	UID string `json:"uid"`
}

// GetScoreResp get_score 响应。
//
// Deprecated: 接口已废弃。
type GetScoreResp struct {
	Score int64 `json:"score"`
}

// UpdateScore 操作类型。
const (
	UpdateScoreTypeConsume int32 = 1 // 消耗
	UpdateScoreTypeGain    int32 = 2 // 获得
)

// update_score 专用业务错误码。
const (
	errCodeInsufficientBalance int32 = 9000
	errCodeDuplicateOrderID    int32 = 9001
)

// 哨兵错误：请勿修改其字段值，直接返回即可（需要自定义时用 NewCallbackError）。
var (
	// ErrInsufficientBalance 余额不足（update_score 返回，错误码 9000）。
	ErrInsufficientBalance = &CallbackError{RetCode: 1, RetMsg: "insufficient balance", SDKErrorCode: errCodeInsufficientBalance}
	// ErrDuplicateOrderID 订单 id 重复（update_score 返回，错误码 9001）。
	ErrDuplicateOrderID = &CallbackError{RetCode: 1, RetMsg: "duplicate order id", SDKErrorCode: errCodeDuplicateOrderID}
)

// UpdateScoreReq update_score 请求。
type UpdateScoreReq struct {
	OrderID string `json:"order_id"` // 需幂等
	MGID    string `json:"mg_id"`
	RoundID string `json:"round_id"`
	UID     string `json:"uid"`
	Score   int64  `json:"score"` // 消耗或获得的积分数
	Type    int32  `json:"type"`  // 1:消耗 2:获得
}

// UpdateScoreResp update_score 响应。
//
// Deprecated: 接口已废弃。
type UpdateScoreResp struct {
	Score int64 `json:"score"` // 操作后的用户积分
}
