// Package events 定义 push_event 推送到游戏服务的全部事件数据结构。
// 事件常量见 api 包 EventXxx；本包仅承载各事件的 data 结构。
package events

// UserInfo 通用用户信息（user_in / quick_start / user_in_batch / user_to_player 共用基础）。
type UserInfo struct {
	UID       string `json:"uid"`
	NickName  string `json:"nick_name"`
	AvatarURL string `json:"avatar_url"`
	Gender    string `json:"gender"`             // female / male / ""
	IsAI      int32  `json:"is_ai,omitempty"`    // 0:普通用户 1:机器人
	AILevel   int32  `json:"ai_level,omitempty"` // 0/1:简单 2:中级 3:高级
}

// QuickStartUserInfo quick_start 的用户信息，在 UserInfo 基础上支持大模型机器人。
type QuickStartUserInfo struct {
	UserInfo
	AIType int32  `json:"ai_type,omitempty"` // 0:普通机器人 1:大模型机器人
	AIID   string `json:"ai_id,omitempty"`   // 大模型 AI 模板 id，空则随机
}

// BatchUserInInfo user_in_batch 的用户信息，携带座位信息。
type BatchUserInInfo struct {
	UserInfo
	SeatIndex    int32 `json:"seat_index,omitempty"`     // -1 随机入座
	IsSeatRandom bool  `json:"is_seat_random,omitempty"` // 座位被占时是否随机换空位
	TeamID       int32 `json:"team_id,omitempty"`        // 1 或 2
	IsReady      bool  `json:"is_ready,omitempty"`
}

// AIPlayer ai_add 的 AI 用户信息（注意字段名与 UserInfo 不同：avatar/name）。
type AIPlayer struct {
	UID     string `json:"uid"`
	Avatar  string `json:"avatar"`
	Name    string `json:"name"`
	Gender  string `json:"gender"`
	AILevel int    `json:"ai_level"`
}

// AiAddReqData 加入 AI。
type AiAddReqData struct {
	RoomID    string     `json:"room_id"`
	AIPlayers []AIPlayer `json:"ai_players"`
	IsReady   int        `json:"is_ready"` // 1:加入后自动准备 0:不自动准备
}

// AiAddRespData 加入 AI 响应。
type AiAddRespData struct {
	UIDs []string `json:"uids"` // 加入成功的 uid 列表
}

// CaptainChangeReqData 队长更换。
type CaptainChangeReqData struct {
	RoomID     string `json:"room_id"`
	CaptainUID string `json:"captain_uid"`
}

// DrawImageClearReqData 画图清除。
type DrawImageClearReqData struct {
	RoomID string `json:"room_id"`
	UID    string `json:"uid"`
}

// GameEndReqData 游戏结束（强制终局）。
type GameEndReqData struct {
	RoomID string `json:"room_id"`
	// UID 指定用户结束游戏时携带；空表示游戏提前结束
	UID string `json:"uid,omitempty"`
}

// GameSettingReqData 游戏玩法设置。
type GameSettingReqData struct {
	RoomID string `json:"room_id"`
	Rule   any    `json:"rule"` // 玩法配置，各游戏结构见文档
}

// GameStartReqData 游戏开始。
type GameStartReqData struct {
	RoomID               string `json:"room_id"`
	ReportGameInfoExtras string `json:"report_game_info_extras,omitempty"` // 最大 1024 字节
	ReportGameInfoKey    string `json:"report_game_info_key,omitempty"`    // 最大 64 字节
}

// LLMAIPlayer llm_ai_add 的 AI 用户信息。
type LLMAIPlayer struct {
	UID    string `json:"uid"`
	Avatar string `json:"avatar"`
	Name   string `json:"name"`
	AIID   string `json:"ai_id,omitempty"` // 空则随机并与 uid 绑定复用
}

// LLMAiAddReqData 加入大模型 AI。
type LLMAiAddReqData struct {
	RoomID    string        `json:"room_id"`
	AIPlayers []LLMAIPlayer `json:"ai_players"`
	IsEnter   int           `json:"is_enter"` // 1:自动上座位 0:不自动
	IsReady   int           `json:"is_ready"` // 1:自动准备 0:不自动
}

// LLMAiExitReqData 大模型 AI 退出。
type LLMAiExitReqData struct {
	RoomID string   `json:"room_id"`
	UIDs   []string `json:"uids"`
}

// ModeExChangeReqData 子模式更换。
type ModeExChangeReqData struct {
	RoomID string `json:"room_id"`
	ModeEx int32  `json:"mode_ex"`
}

// PlayerAssetData 玩家资产。
type PlayerAssetData struct {
	UID   string `json:"uid"`
	Asset any    `json:"asset"` // 资产数据，结构随游戏而定
}

// PlayerAssetReqData 批量获取玩家资产。
type PlayerAssetReqData struct {
	UIDs []string `json:"uids"` // 最多 10 个
}

// PlayerAssetRespData 批量获取玩家资产响应。
type PlayerAssetRespData struct {
	PlayerAssets []PlayerAssetData `json:"player_assets"`
}

// QuickStartReqData 一键开始游戏。
// 注意：不要把机器人放在 user_infos 第一个位置。
type QuickStartReqData struct {
	UserInfos            []QuickStartUserInfo `json:"user_infos"`
	RoomID               string               `json:"room_id"`
	Mode                 int32                `json:"mode,omitempty"`
	Language             string               `json:"language,omitempty"`
	Rule                 any                  `json:"rule,omitempty"`
	ReportGameInfoExtras string               `json:"report_game_info_extras,omitempty"`
	ReportGameInfoKey    string               `json:"report_game_info_key,omitempty"`
	ModeEx               int32                `json:"mode_ex,omitempty"`
	CaptainUID           string               `json:"captain_uid,omitempty"` // 只能指定真实玩家
}

// ItemInfo 用户道具。
type ItemInfo struct {
	UID   string `json:"uid"`
	Cmd   string `json:"cmd"`
	Count int32  `json:"count"`
}

// RefreshUserItemReqData 刷新玩家道具。
type RefreshUserItemReqData struct {
	RoomID    string     `json:"room_id"`
	ItemInfos []ItemInfo `json:"item_infos,omitempty"`
}

// RefreshUserItemRespData 刷新玩家道具响应（刷新成功的道具）。
type RefreshUserItemRespData struct {
	ItemInfos []ItemInfo `json:"item_infos,omitempty"`
}

// RoomClearReqData 房间清理（进行中的游戏会结算并踢出所有玩家）。
type RoomClearReqData struct {
	RoomID string `json:"room_id"`
}

// 房间状态常量（RoomInfoRespData.Status）。
const (
	RoomStatusWaiting = "WATING"
	RoomStatusPlaying = "PLAYING"
	RoomStatusPending = "PENDING" // 服务器已收到开始指令处理中，稍后再查
)

// 座位玩家状态常量。
const (
	SeatPlayerStatusIdle  = "IDLE"
	SeatPlayerStatusReady = "READY"
)

// SeatPlayer 游戏位玩家。
type SeatPlayer struct {
	UID       string `json:"uid"`
	SeatIndex int32  `json:"seat_index"` // 从 0 开始
	Status    string `json:"status"`     // IDLE / READY
	IsAI      int32  `json:"is_ai"`
	AILevel   int32  `json:"ai_level,omitempty"`
}

// RoomInfoReqData 获取房间座位信息。
type RoomInfoReqData struct {
	RoomID string `json:"room_id"`
}

// RoomInfoRespData 房间座位信息响应。
type RoomInfoRespData struct {
	Status     string       `json:"status"` // WATING / PLAYING / PENDING
	CaptainUID string       `json:"captain_uid"`
	Mode       int32        `json:"mode,omitempty"`
	ModeEx     int32        `json:"mode_ex,omitempty"`
	Player     []SeatPlayer `json:"player"` // 注意：文档字段名为 player
}

// UserInBatchReqData 批量用户加入。
type UserInBatchReqData struct {
	UserInfos []BatchUserInInfo `json:"user_infos,omitempty"`
	RoomID    string            `json:"room_id"`
	Mode      int32             `json:"mode,omitempty"`
	Language  string            `json:"language,omitempty"`
}

// UserInReqData 用户加入。
// UserInfo 优先使用；为空时使用 Code。
type UserInReqData struct {
	Code         string   `json:"code,omitempty"`
	UserInfo     UserInfo `json:"user_info,omitempty"`
	RoomID       string   `json:"room_id"`
	Mode         int32    `json:"mode"`
	Language     string   `json:"language,omitempty"`
	SeatIndex    int32    `json:"seat_index,omitempty"`
	IsSeatRandom bool     `json:"is_seat_random,omitempty"`
	TeamID       int32    `json:"team_id,omitempty"`
	IsReady      bool     `json:"is_ready,omitempty"`
}

// UserKickReqData 用户踢人。
type UserKickReqData struct {
	RoomID    string `json:"room_id"`
	KickedUID string `json:"kicked_uid"` // 不能为队长
}

// UserOutReqData 用户退出。
type UserOutReqData struct {
	RoomID        string `json:"room_id"`
	UID           string `json:"uid"`
	IsCancelReady bool   `json:"is_cancel_ready,omitempty"` // 准备态下 true 才能退出
}

// UserReadyReqData 用户准备/取消准备。
type UserReadyReqData struct {
	RoomID  string `json:"room_id"`
	UID     string `json:"uid"`
	IsReady bool   `json:"is_ready"`
}

// UserToPlayerReqData 指定用户成为默认玩家（小丑牌等游戏）。
type UserToPlayerReqData struct {
	RoomID   string   `json:"room_id"`
	UserInfo UserInfo `json:"user_info"`
	Mode     int32    `json:"mode,omitempty"`
	Language string   `json:"language,omitempty"`
}
