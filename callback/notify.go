package callback

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/linuxea/sud-server-sdk/api"
)

// 异步通知事件值。
const (
	NotifyRoomUsersChanged        = "sud.mg.merchant.room.users.changed"         // 房间用户人数变更
	NotifyOrderChanged            = "sud.mg.merchant.order.changed"              // 订单状态变更
	NotifyOrderBatchChanged       = "sud.mg.merchant.order.batch.changed"        // 订单状态批量变更
	NotifyBidResult               = "sud.mg.merchant.bid.result"                 // 竞价结果
	NotifyUserSettle              = "sud.mg.merchant.user.settle"                // 用户结算（德州扑克/TeenPatti）
	NotifyMatchStart              = "sud.mg.merchant.match.start"                // 单场游戏开始（德州扑克/TeenPatti）
	NotifyMatchSettle             = "sud.mg.merchant.match.settle"               // 单场游戏结算（德州扑克/TeenPatti）
	NotifyRoomGameRule            = "sud.mg.merchant.room.game.rule"             // 房间玩法规则设置（德州扑克/TeenPatti）
	NotifyGameProcess             = "sud.mg.merchant.game.process"               // 游戏过程事件
	NotifyPlayerStatus            = "sud.mg.merchant.player.status"              // 玩家状态事件
	NotifyGamePlayerConnectStatus = "sud.mg.merchant.game.player.connect.status" // 游戏中玩家连接状态
	NotifyRoomSeatChanged         = "sud.mg.merchant.room.seat.changed"          // 座位状态变更
	NotifyPlayerDrawImage         = "sud.mg.merchant.player.draw.image"          // 玩家画图数据
	NotifyLLMAICreateResult       = "sud.mg.merchant.llm.ai.create.result"       // 大模型 AI 创建结果
	NotifyLLMGameSituationSummary = "sud.mg.merchant.llm.game.situation.summary" // 大模型游戏局势总结
)

// NotifyCallbacks 15 种异步通知的逐事件处理。
// 按 Need 实现，未实现的事件走 Callbacks.OnNotify 兜底或仅记日志。
// 各事件需在 Sud 后台开启（notify_url 指向 {prefix}/notify）。
type NotifyCallbacks struct {
	OnRoomUsersChanged        func(ctx context.Context, data RoomUsersChangedModel)
	OnOrderChanged            func(ctx context.Context, data OrderChangedModel)
	OnOrderBatchChanged       func(ctx context.Context, data OrderBatchChangedModel)
	OnBidResult               func(ctx context.Context, data BidResultModel)
	OnUserSettle              func(ctx context.Context, data UserSettleModel)
	OnMatchStart              func(ctx context.Context, data MatchStartModel)
	OnMatchSettle             func(ctx context.Context, data MatchSettleModel)
	OnRoomGameRule            func(ctx context.Context, data RoomGameRuleModel)
	OnGameProcess             func(ctx context.Context, data GameProcessModel)
	OnPlayerStatus            func(ctx context.Context, data PlayerStatusModel)
	OnGamePlayerConnectStatus func(ctx context.Context, data GamePlayerConnectStatusModel)
	OnRoomSeatChanged         func(ctx context.Context, data SeatChangedModel)
	OnPlayerDrawImage         func(ctx context.Context, data DrawImageModel)
	OnLLMAICreateResult       func(ctx context.Context, data AICreateResultModel)
	OnLLMGameSituationSummary func(ctx context.Context, data GameSituationSummaryModel)
}

// notifyReq notify 请求信封。
type notifyReq struct {
	NotifyID    string          `json:"notify_id"`
	NotifyTime  string          `json:"notify_time"` // 毫秒时间戳
	AppID       string          `json:"app_id"`
	NotifyEvent string          `json:"notify_event"`
	Data        json.RawMessage `json:"data"`
}

// RoomUsersChangedModel 房间用户人数变更。
type RoomUsersChangedModel struct {
	RoomID      string `json:"room_id"`
	MGID        string `json:"mg_id"`
	PlayerTotal int32  `json:"player_total"`
	OBTotal     int32  `json:"ob_total"`
	ChangedTime string `json:"changed_time"` // 毫秒时间戳
}

// OrderChangedModel 订单状态变更。
type OrderChangedModel struct {
	OrderID     string `json:"order_id"`
	OutOrderID  string `json:"out_order_id"`
	OutGroupID  string `json:"out_group_id,omitempty"`
	MGID        string `json:"mg_id"`
	RoomID      string `json:"room_id"`
	GameRoundID string `json:"game_round_id,omitempty"`
	Cmd         string `json:"cmd"`
	FromUID     string `json:"from_uid"`
	ToUID       string `json:"to_uid"`
	Value       int32  `json:"value,omitempty"`
	Seq         int32  `json:"seq,omitempty"` // 时序敏感场景使用
	Payload     any    `json:"payload,omitempty"`
	Status      string `json:"status"` // EXECUTE_FAIL / EXECUTE_SUCCESS
}

// OrderBatchChangedEntry 批量订单变更单条。
type OrderBatchChangedEntry struct {
	OrderID     string `json:"order_id"`
	OutOrderID  string `json:"out_order_id"`
	GameRoundID string `json:"game_round_id,omitempty"`
	Cmd         string `json:"cmd"`
	FromUID     string `json:"from_uid"`
	ToUID       string `json:"to_uid"`
	Value       int32  `json:"value"`
	Payload     any    `json:"payload,omitempty"`
	Status      string `json:"status"`
}

// OrderBatchChangedModel 订单状态批量变更。
type OrderBatchChangedModel struct {
	MGID   string                   `json:"mg_id"`
	RoomID string                   `json:"room_id"`
	Orders []OrderBatchChangedEntry `json:"orders"`
}

// PlayerBidResultModel 玩家竞价结果。
type PlayerBidResultModel struct {
	OrderID    string `json:"order_id"`
	OutOrderID string `json:"out_order_id"`
	UID        string `json:"uid"`
	BidRS      int32  `json:"bid_rs"` // 0:成功 1:失败
	BidValue   int32  `json:"bid_value"`
	Payload    any    `json:"payload,omitempty"` // 如狼人杀 {"role_id":1}
}

// BidResultModel 竞价结果。
type BidResultModel struct {
	MGID        string                 `json:"mg_id"`
	RoomID      string                 `json:"room_id"`
	GameRoundID string                 `json:"game_round_id"`
	PlayerBids  []PlayerBidResultModel `json:"player_bids"`
}

// UserSettlePlayerModel 用户结算单条。
type UserSettlePlayerModel struct {
	UID             string `json:"uid"` // 机器人为空字符
	OrderID         string `json:"order_id"`
	OutOrderID      string `json:"out_order_id"`
	IsAI            int32  `json:"is_ai"`
	Score           int32  `json:"score"`
	RemainScore     int32  `json:"remain_score"`
	CommissionScore int32  `json:"commission_score,omitempty"`
}

// UserSettleModel 用户结算（德州扑克/TeenPatti）。
type UserSettleModel struct {
	MGID     string                  `json:"mg_id"`
	RoomID   string                  `json:"room_id"`
	MatchID  string                  `json:"match_id"`
	Results  []UserSettlePlayerModel `json:"results"`
	SettleAt string                  `json:"settle_at"` // 秒时间戳
}

// MatchStartModel 单场游戏开始（德州扑克/TeenPatti）。
type MatchStartModel struct {
	MGID                 string `json:"mg_id"`
	RoomID               string `json:"room_id"`
	GameMode             int32  `json:"game_mode"`
	MatchID              string `json:"match_id"`
	BattleStartAt        string `json:"battle_start_at"` // 秒时间戳
	ReportGameInfoKey    string `json:"report_game_info_key,omitempty"`
	ReportGameInfoExtras string `json:"report_game_info_extras,omitempty"`
}

// MatchSettleModel 单场游戏结算（德州扑克/TeenPatti）。
type MatchSettleModel struct {
	MGID                 string                  `json:"mg_id"`
	RoomID               string                  `json:"room_id"`
	GameMode             int32                   `json:"game_mode"`
	MatchID              string                  `json:"match_id"`
	BattleStartAt        string                  `json:"battle_start_at"`
	BattleEndAt          string                  `json:"battle_end_at"`
	BattleDuration       string                  `json:"battle_duration"` // 秒
	Results              []api.PlayerSettleModel `json:"results"`
	ReportGameInfoKey    string                  `json:"report_game_info_key,omitempty"`
	ReportGameInfoExtras string                  `json:"report_game_info_extras,omitempty"`
}

// RoomGameRuleModel 房间玩法规则设置（德州扑克/TeenPatti）。
type RoomGameRuleModel struct {
	MGID   string `json:"mg_id"`
	RoomID string `json:"room_id"`
	Rule   string `json:"rule"` // JSON 字符串，各游戏配置见文档
}

// ProcessPlayer 游戏过程/状态事件中的玩家。
type ProcessPlayer struct {
	UID     string `json:"uid"`
	Type    string `json:"type,omitempty"` // INTERNAL / EXTERNAL
	Payload any    `json:"payload,omitempty"`
}

// GameProcessModel 游戏过程事件。
type GameProcessModel struct {
	MGID        string          `json:"mg_id"`
	RoomID      string          `json:"room_id"`
	GameRoundID string          `json:"game_round_id"`
	Event       string          `json:"event"` // 各游戏事件见文档（self_die/chess_end 等）
	Players     []ProcessPlayer `json:"players"`
	Payload     any             `json:"payload,omitempty"`
}

// StatusPlayer 玩家状态事件中的玩家。
type StatusPlayer struct {
	UID       string `json:"uid"`
	Type      string `json:"type,omitempty"`
	Timestamp string `json:"timestamp"` // 事件发生毫秒时间戳
	Payload   any    `json:"payload,omitempty"`
}

// PlayerStatusModel 玩家状态事件（如托管 manage_status）。
type PlayerStatusModel struct {
	MGID        string         `json:"mg_id"`
	RoomID      string         `json:"room_id"`
	GameRoundID string         `json:"game_round_id,omitempty"` // 游戏中事件才有值
	Event       string         `json:"event"`
	Players     []StatusPlayer `json:"players"`
	Payload     any            `json:"payload,omitempty"`
}

// ConnectStatusPlayer 玩家连接状态。
type ConnectStatusPlayer struct {
	UID              string `json:"uid"`
	OfflineTimestamp string `json:"offline_timestamp"` // 毫秒
	Status           int32  `json:"status"`            // 0:下线 1:上线
	OfflineSeconds   int32  `json:"offline_seconds"`
	Payload          any    `json:"payload,omitempty"`
}

// GamePlayerConnectStatusModel 游戏中玩家连接状态。
type GamePlayerConnectStatusModel struct {
	MGID        string              `json:"mg_id"`
	RoomID      string              `json:"room_id"`
	GameRoundID string              `json:"game_round_id"`
	Player      ConnectStatusPlayer `json:"player"`
	Payload     any                 `json:"payload,omitempty"`
}

// 座位变更类型。
const (
	SeatChangedTypeSit     int32 = 1 // 上座位
	SeatChangedTypeStand   int32 = 2 // 下座位
	SeatChangedTypeReady   int32 = 3 // 准备
	SeatChangedTypeUnready int32 = 4 // 取消准备
)

// SeatChangedObject 座位变更单条。
type SeatChangedObject struct {
	UID         string `json:"uid"`
	ChangedType int32  `json:"changed_type"` // 见 SeatChangedType 常量
}

// SeatPlayerObject 变更后座位上的玩家。
type SeatPlayerObject struct {
	UID       string `json:"uid"`
	SeatIndex int32  `json:"seat_index"`
	Status    string `json:"status"` // IDLE / READY
}

// SeatChangedModel 座位状态变更。
type SeatChangedModel struct {
	MGID        string              `json:"mg_id"`
	RoomID      string              `json:"room_id"`
	ChangedTime string              `json:"changed_time"` // 毫秒
	SeatChanged []SeatChangedObject `json:"seat_changed"`
	SeatPlayers []SeatPlayerObject  `json:"seat_players,omitempty"`
}

// DrawImageModel 玩家画图数据。
type DrawImageModel struct {
	MGID        string `json:"mg_id"`
	RoomID      string `json:"room_id"`
	GameRoundID string `json:"game_round_id"`
	UID         string `json:"uid"`
	ImageData   string `json:"image_data"` // data:{mime_type};base64,{base64_data}
	Timestamp   string `json:"timestamp"`  // 毫秒
	Payload     any    `json:"payload,omitempty"`
}

// AICreateResultModel 大模型 AI 创建结果。
type AICreateResultModel struct {
	MGID         string `json:"mg_id"`
	RoomID       string `json:"room_id"`
	UID          string `json:"uid"`
	BizErrorCode int32  `json:"biz_error_code"`
	BizErrorMsg  string `json:"biz_error_msg,omitempty"`
}

// GameSituationSummaryModel 大模型游戏局势总结。
type GameSituationSummaryModel struct {
	MGID             string `json:"mg_id"`
	RoomID           string `json:"room_id"`
	GameRoundID      string `json:"game_round_id"`
	SituationSummary string `json:"situation_summary"`
}

// dispatchNotify notify 入口：按事件分发到已注册的处理，
// 未注册走 OnNotify 兜底，均无则记日志；应答恒为 SUCCESS。
func (s *Server) dispatchNotify(w http.ResponseWriter, r *http.Request) {
	var env notifyReq
	if !s.readJSON(w, r, &env, s.notifyFailResp()) {
		return
	}
	ctx := r.Context()
	if !s.notifyHandled(ctx, env.NotifyEvent, env.Data) {
		if s.cbs.OnNotify != nil {
			s.cbs.OnNotify(ctx, env.NotifyEvent, env.Data)
		} else {
			s.logger.Printf("sud notify: 未处理事件 %s notify_id=%s", env.NotifyEvent, env.NotifyID)
		}
	}
	writeJSON(w, resp{RetCode: 0, RetMsg: "SUCCESS"})
}

// notifyHandled 按事件值分发；返回是否已有注册的处理被调用。
func (s *Server) notifyHandled(ctx context.Context, event string, raw json.RawMessage) bool {
	cbs := &s.cbs.NotifyCallbacks
	switch event {
	case NotifyRoomUsersChanged:
		return callNotify(s, ctx, event, raw, cbs.OnRoomUsersChanged)
	case NotifyOrderChanged:
		return callNotify(s, ctx, event, raw, cbs.OnOrderChanged)
	case NotifyOrderBatchChanged:
		return callNotify(s, ctx, event, raw, cbs.OnOrderBatchChanged)
	case NotifyBidResult:
		return callNotify(s, ctx, event, raw, cbs.OnBidResult)
	case NotifyUserSettle:
		return callNotify(s, ctx, event, raw, cbs.OnUserSettle)
	case NotifyMatchStart:
		return callNotify(s, ctx, event, raw, cbs.OnMatchStart)
	case NotifyMatchSettle:
		return callNotify(s, ctx, event, raw, cbs.OnMatchSettle)
	case NotifyRoomGameRule:
		return callNotify(s, ctx, event, raw, cbs.OnRoomGameRule)
	case NotifyGameProcess:
		return callNotify(s, ctx, event, raw, cbs.OnGameProcess)
	case NotifyPlayerStatus:
		return callNotify(s, ctx, event, raw, cbs.OnPlayerStatus)
	case NotifyGamePlayerConnectStatus:
		return callNotify(s, ctx, event, raw, cbs.OnGamePlayerConnectStatus)
	case NotifyRoomSeatChanged:
		return callNotify(s, ctx, event, raw, cbs.OnRoomSeatChanged)
	case NotifyPlayerDrawImage:
		return callNotify(s, ctx, event, raw, cbs.OnPlayerDrawImage)
	case NotifyLLMAICreateResult:
		return callNotify(s, ctx, event, raw, cbs.OnLLMAICreateResult)
	case NotifyLLMGameSituationSummary:
		return callNotify(s, ctx, event, raw, cbs.OnLLMGameSituationSummary)
	}
	return false
}

// callNotify 泛型分发：handler 为 nil 返回未处理；
// data 解析失败记日志并把原始 payload 交给 OnNotify 兜底（Sud 可能新增字段导致
// 强类型解析失败，兜底可保证业务仍能拿到原始数据），均视为已处理。
func callNotify[T any](s *Server, ctx context.Context, event string, raw json.RawMessage, h func(context.Context, T)) bool {
	if h == nil {
		return false
	}
	var data T
	if err := json.Unmarshal(raw, &data); err != nil {
		s.logger.Printf("sud notify: 解析 data 失败 event=%s: %v", event, err)
		if s.cbs.OnNotify != nil {
			s.cbs.OnNotify(ctx, event, raw)
		}
		return true
	}
	h(ctx, data)
	return true
}
