package api

import "context"

// ReportService 游戏上报查询服务域。
type ReportService struct{ p Poster }

// NewReportService 创建服务域。
func NewReportService(p Poster) *ReportService { return &ReportService{p: p} }

// 上报类型常量（filter_types 取值）。
const (
	ReportTypeGameStart  = "game_start"
	ReportTypeGameSettle = "game_settle"
)

// QueryGameReportReq 查询游戏上报信息请求。
// ReportGameInfoKey 与 GameRoundID 不能同时为空；同时存在时优先 GameRoundID。
type QueryGameReportReq struct {
	ReportGameInfoKey string   `json:"report_game_info_key,omitempty"`
	GameRoundID       string   `json:"game_round_id,omitempty"`
	FilterTypes       []string `json:"filter_types,omitempty"`
}

// QueryGameReportResp 查询游戏上报信息响应。
// data 为 <上报类型, 上报数据> 映射，未过滤时 game_start/game_settle 均可能存在。
type QueryGameReportResp struct {
	GameStart  *GameStartObject  `json:"game_start,omitempty"`
	GameSettle *GameSettleObject `json:"game_settle,omitempty"`
}

// GameStartObject 战斗开始上报。
type GameStartObject struct {
	MGID                 int64          `json:"mg_id"`
	MGIDStr              string         `json:"mg_id_str"`
	RoomID               string         `json:"room_id"`
	GameMode             int32          `json:"game_mode"`
	GameModeEx           int32          `json:"game_mode_ex,omitempty"`
	GameRoundID          string         `json:"game_round_id"`
	BattleStartAt        int32          `json:"battle_start_at"` // 秒
	Players              []ReportPlayer `json:"players"`
	ReportGameInfoKey    string         `json:"report_game_info_key,omitempty"`
	ReportGameInfoExtras string         `json:"report_game_info_extras,omitempty"`
}

// GameSettleObject 战斗结算上报。
type GameSettleObject struct {
	MGID                 int64                `json:"mg_id"`
	MGIDStr              string               `json:"mg_id_str"`
	RoomID               string               `json:"room_id"`
	GameMode             int32                `json:"game_mode"`
	GameModeEx           int32                `json:"game_mode_ex,omitempty"`
	GameRoundID          string               `json:"game_round_id"`
	BattleStartAt        int32                `json:"battle_start_at"` // 秒
	BattleEndAt          int32                `json:"battle_end_at"`   // 秒
	BattleDuration       int32                `json:"battle_duration"` // 秒
	Results              []ReportPlayerResult `json:"results,omitempty"`
	ResultsURL           string               `json:"results_url,omitempty"` // 玩家结果数据下载地址，有效期 1 小时
	ReportGameInfoKey    string               `json:"report_game_info_key,omitempty"`
	ReportGameInfoExtras string               `json:"report_game_info_extras,omitempty"`
}

// ReportPlayer 上报中的玩家。
type ReportPlayer struct {
	UID     string `json:"uid"`
	IsAI    int32  `json:"is_ai"`              // 0:普通用户 1:机器人
	AILevel int32  `json:"ai_level,omitempty"` // 0/1:简单 2:中级 3:高级
}

// ReportPlayerResult 上报中的玩家结算结果。
type ReportPlayerResult struct {
	UID             string `json:"uid"`
	Rank            int32  `json:"rank"`       // 从 1 开始，平局排名相同
	IsEscaped       int32  `json:"is_escaped"` // 0:正常 1:逃跑
	IsAI            int32  `json:"is_ai"`
	Role            int32  `json:"role,omitempty"`
	Score           int32  `json:"score,omitempty"`
	CommissionScore int32  `json:"commission_score,omitempty"`
	IsWin           int32  `json:"is_win,omitempty"` // 0:无信息 1:输 2:赢 3:平局
	Award           int32  `json:"award,omitempty"`
	Extras          string `json:"extras,omitempty"`
	IsManaged       int32  `json:"is_managed,omitempty"` // 0:未托管 1:托管
}

// GameReportInfo 分页查询返回的单局上报。
type GameReportInfo struct {
	GameRoundID string              `json:"game_round_id"`
	ReportInfo  QueryGameReportResp `json:"report_info"`
}

// reportPageReq 分页查询上报请求（app 凭证由服务域注入，不对外暴露）。
type reportPageReq struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	RoomID    string `json:"room_id"`
	PageNo    int32  `json:"page_no,omitempty"`
	PageSize  int32  `json:"page_size,omitempty"`
}

// QueryGameReport 按自定义游戏局 key 或 game_round_id 查询上报信息。
// 仅支持查询一个月以内的数据，限频 10次/秒。
func (s *ReportService) QueryGameReport(ctx context.Context, req QueryGameReportReq) (*QueryGameReportResp, error) {
	var resp QueryGameReportResp
	if err := s.p.Post(ctx, string(KeyQueryGameReportInfo), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryByPage 分页获取房间内游戏上报信息（app 凭证自动注入）。
// pageNo 默认 0；pageSize 默认 5、最大 10。仅支持一个月以内的数据。
func (s *ReportService) QueryByPage(ctx context.Context, roomID string, pageNo, pageSize int32) ([]GameReportInfo, error) {
	req := reportPageReq{
		AppID:     s.p.AppID(),
		AppSecret: s.p.AppSecret(),
		RoomID:    roomID,
		PageNo:    pageNo,
		PageSize:  pageSize,
	}
	var resp []GameReportInfo
	if err := s.p.Post(ctx, string(KeyGameReportInfoPage), req, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}
