package api

import "context"

// EntryScoreService 带分入场查询服务域（德州扑克、TeenPatti）。
type EntryScoreService struct{ p Poster }

// NewEntryScoreService 创建服务域。
func NewEntryScoreService(p Poster) *EntryScoreService { return &EntryScoreService{p: p} }

// 游戏场次状态常量。
const (
	MatchStatusPlaying = "PLAYING"
	MatchStatusClosed  = "CLOSED"
)

// QueryMatchBaseReq 查询单场游戏基础信息请求。
// MatchID 与 ReportGameInfoKey 不能同时为空；同时存在时 MatchID 优先。
type QueryMatchBaseReq struct {
	MatchID           string `json:"match_id,omitempty"`
	ReportGameInfoKey string `json:"report_game_info_key,omitempty"`
}

// PlayerSettleModel 玩家结算数据。
type PlayerSettleModel struct {
	UID             string `json:"uid"` // 机器人为空字符
	IsAI            int32  `json:"is_ai"`
	Score           int32  `json:"score"`        // 该场游戏总得分
	RemainScore     int32  `json:"remain_score"` // 剩余积分
	CommissionScore int32  `json:"commission_score,omitempty"`
}

// QueryMatchBaseResp 查询单场游戏基础信息响应。
type QueryMatchBaseResp struct {
	MGID                 string              `json:"mg_id"`
	RoomID               string              `json:"room_id"`
	GameMode             int32               `json:"game_mode"`
	MatchID              string              `json:"match_id"`
	BattleStartAt        string              `json:"battle_start_at"` // 秒（字符串时间戳）
	BattleEndAt          string              `json:"battle_end_at,omitempty"`
	BattleDuration       string              `json:"battle_duration,omitempty"`
	Results              []PlayerSettleModel `json:"results,omitempty"`
	ReportGameInfoKey    string              `json:"report_game_info_key,omitempty"`
	ReportGameInfoExtras string              `json:"report_game_info_extras,omitempty"`
	Status               string              `json:"status"` // 见 MatchStatus 常量
}

// QueryMatchRoundIdsReq 查询单场游戏内所有局 id 请求。
type QueryMatchRoundIdsReq struct {
	MatchID           string `json:"match_id,omitempty"`
	ReportGameInfoKey string `json:"report_game_info_key,omitempty"`
}

// QueryMatchRoundIdsResp 查询局 id 响应。
type QueryMatchRoundIdsResp struct {
	MatchID           string   `json:"match_id"`
	Total             int32    `json:"total"`
	RoundIDs          []string `json:"round_ids"`
	ReportGameInfoKey string   `json:"report_game_info_key,omitempty"`
}

// QueryUserSettleReq 查询用户结算信息请求。
// OutOrderID 与 OrderID 不能同时为空；同时存在时 OrderID 优先。
type QueryUserSettleReq struct {
	OutOrderID string `json:"out_order_id,omitempty"`
	OrderID    string `json:"order_id,omitempty"`
}

// QueryUserSettleResp 查询用户结算信息响应。
type QueryUserSettleResp struct {
	OrderID         string `json:"order_id"`
	OutOrderID      string `json:"out_order_id"`
	MGID            string `json:"mg_id"`
	RoomID          string `json:"room_id"`
	MatchID         string `json:"match_id"`
	UID             string `json:"uid"`
	IsAI            int32  `json:"is_ai"`
	Score           int32  `json:"score"`
	RemainScore     int32  `json:"remain_score"`
	CommissionScore int32  `json:"commission_score,omitempty"`
	SettleAt        string `json:"settle_at"`
}

// QueryMatchBase 查询单场游戏基础信息。
func (s *EntryScoreService) QueryMatchBase(ctx context.Context, req QueryMatchBaseReq) (*QueryMatchBaseResp, error) {
	var resp QueryMatchBaseResp
	if err := s.p.Post(ctx, string(KeyQueryMatchBase), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryMatchRoundIds 查询单场游戏内的所有局 id（按局索引正序）。
func (s *EntryScoreService) QueryMatchRoundIds(ctx context.Context, req QueryMatchRoundIdsReq) (*QueryMatchRoundIdsResp, error) {
	var resp QueryMatchRoundIdsResp
	if err := s.p.Post(ctx, string(KeyQueryMatchRoundIds), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryUserSettle 按订单查询用户结算信息。
func (s *EntryScoreService) QueryUserSettle(ctx context.Context, req QueryUserSettleReq) (*QueryUserSettleResp, error) {
	var resp QueryUserSettleResp
	if err := s.p.Post(ctx, string(KeyQueryUserSettle), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
