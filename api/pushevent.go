package api

import (
	"context"
	"strconv"
	"time"

	"github.com/linuxea/sud-server-sdk/events"
)

// PushEventService 推送事件到游戏服务。
type PushEventService struct {
	p Poster
}

// NewPushEventService 创建服务域。
func NewPushEventService(p Poster) *PushEventService { return &PushEventService{p: p} }

// 游戏事件名常量。
const (
	EventUserIn          = "user_in"           // 用户加入
	EventUserOut         = "user_out"          // 用户退出
	EventUserReady       = "user_ready"        // 用户准备/取消准备
	EventGameStart       = "game_start"        // 游戏开始
	EventCaptainChange   = "captain_change"    // 队长更换
	EventUserKick        = "user_kick"         // 用户踢人
	EventGameEnd         = "game_end"          // 游戏结束
	EventGameSetting     = "game_setting"      // 游戏玩法设置
	EventAiAdd           = "ai_add"            // 加入 AI
	EventRoomInfo        = "room_info"         // 获取房间座位信息
	EventQuickStart      = "quick_start"       // 一键开始游戏
	EventRoomClear       = "room_clear"        // 房间清理
	EventRefreshUserItem = "refresh_user_item" // 刷新玩家道具
	EventModeExChange    = "mode_ex_change"    // 子模式更换
	EventUserInBatch     = "user_in_batch"     // 批量用户加入
	EventLLMAiAdd        = "llm_ai_add"        // 加入大模型 AI
	EventLLMAiExit       = "llm_ai_exit"       // 大模型 AI 退出
	EventDrawImageClear  = "draw_image_clear"  // 画图清除
	EventUserToPlayer    = "user_to_player"    // 指定默认玩家
	EventPlayerAsset     = "player_asset"      // 批量获取玩家资产
)

// pushEventReq push_event 的请求壳。
type pushEventReq struct {
	Event     string `json:"event"`
	MGID      string `json:"mg_id"`
	Data      any    `json:"data"`
	Timestamp string `json:"timestamp"` // 毫秒
}

// Push 推送自定义事件（data 为任意结构）。Sud 未定义的事件请谨慎使用。
func (s *PushEventService) Push(ctx context.Context, event, mgID string, data any) error {
	return s.push(ctx, event, mgID, data, nil)
}

// PushWithResp 推送事件并解析响应 data（ai_add/room_info 等有返回值的事件）。
func (s *PushEventService) PushWithResp(ctx context.Context, event, mgID string, data any, resp any) error {
	return s.push(ctx, event, mgID, data, resp)
}

func (s *PushEventService) push(ctx context.Context, event, mgID string, data any, resp any) error {
	req := pushEventReq{
		Event:     event,
		MGID:      mgID,
		Data:      data,
		Timestamp: strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	return s.p.Post(ctx, string(KeyPushEvent), req, resp)
}

// QuickStart 一键开始游戏（携带玩家列表，可含 AI 补位）。
func (s *PushEventService) QuickStart(ctx context.Context, mgID string, data events.QuickStartReqData) error {
	return s.Push(ctx, EventQuickStart, mgID, data)
}

// GameStart 游戏开始。
func (s *PushEventService) GameStart(ctx context.Context, mgID string, data events.GameStartReqData) error {
	return s.Push(ctx, EventGameStart, mgID, data)
}

// GameEnd 游戏结束（强制终局）。uid 为空表示游戏提前结束。
func (s *PushEventService) GameEnd(ctx context.Context, mgID, roomID, uid string) error {
	return s.Push(ctx, EventGameEnd, mgID, events.GameEndReqData{RoomID: roomID, UID: uid})
}

// RoomClear 房间清理（进行中的游戏会结算并踢出所有玩家）。
func (s *PushEventService) RoomClear(ctx context.Context, mgID, roomID string) error {
	return s.Push(ctx, EventRoomClear, mgID, events.RoomClearReqData{RoomID: roomID})
}

// RoomInfo 获取房间座位信息。
func (s *PushEventService) RoomInfo(ctx context.Context, mgID, roomID string) (*events.RoomInfoRespData, error) {
	var resp events.RoomInfoRespData
	err := s.PushWithResp(ctx, EventRoomInfo, mgID, events.RoomInfoReqData{RoomID: roomID}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UserIn 用户加入。
func (s *PushEventService) UserIn(ctx context.Context, mgID string, data events.UserInReqData) error {
	return s.Push(ctx, EventUserIn, mgID, data)
}

// UserInBatch 批量用户加入。
func (s *PushEventService) UserInBatch(ctx context.Context, mgID string, data events.UserInBatchReqData) error {
	return s.Push(ctx, EventUserInBatch, mgID, data)
}

// UserOut 用户退出。
func (s *PushEventService) UserOut(ctx context.Context, mgID string, data events.UserOutReqData) error {
	return s.Push(ctx, EventUserOut, mgID, data)
}

// UserReady 用户准备/取消准备。
func (s *PushEventService) UserReady(ctx context.Context, mgID, roomID, uid string, isReady bool) error {
	return s.Push(ctx, EventUserReady, mgID, events.UserReadyReqData{RoomID: roomID, UID: uid, IsReady: isReady})
}

// UserKick 用户踢人（不能踢队长）。
func (s *PushEventService) UserKick(ctx context.Context, mgID, roomID, kickedUID string) error {
	return s.Push(ctx, EventUserKick, mgID, events.UserKickReqData{RoomID: roomID, KickedUID: kickedUID})
}

// CaptainChange 队长更换。
func (s *PushEventService) CaptainChange(ctx context.Context, mgID, roomID, captainUID string) error {
	return s.Push(ctx, EventCaptainChange, mgID, events.CaptainChangeReqData{RoomID: roomID, CaptainUID: captainUID})
}

// GameSetting 游戏玩法设置。
func (s *PushEventService) GameSetting(ctx context.Context, mgID, roomID string, rule any) error {
	return s.Push(ctx, EventGameSetting, mgID, events.GameSettingReqData{RoomID: roomID, Rule: rule})
}

// AiAdd 加入 AI，返回加入成功的 uid 列表。
func (s *PushEventService) AiAdd(ctx context.Context, mgID string, data events.AiAddReqData) (*events.AiAddRespData, error) {
	var resp events.AiAddRespData
	err := s.PushWithResp(ctx, EventAiAdd, mgID, data, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// LLMAiAdd 加入大模型 AI。
func (s *PushEventService) LLMAiAdd(ctx context.Context, mgID string, data events.LLMAiAddReqData) error {
	return s.Push(ctx, EventLLMAiAdd, mgID, data)
}

// LLMAiExit 大模型 AI 退出。
func (s *PushEventService) LLMAiExit(ctx context.Context, mgID string, data events.LLMAiExitReqData) error {
	return s.Push(ctx, EventLLMAiExit, mgID, data)
}

// RefreshUserItem 刷新玩家道具，返回刷新成功的道具列表。
func (s *PushEventService) RefreshUserItem(ctx context.Context, mgID string, data events.RefreshUserItemReqData) (*events.RefreshUserItemRespData, error) {
	var resp events.RefreshUserItemRespData
	err := s.PushWithResp(ctx, EventRefreshUserItem, mgID, data, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ModeExChange 子模式更换。
func (s *PushEventService) ModeExChange(ctx context.Context, mgID, roomID string, modeEx int32) error {
	return s.Push(ctx, EventModeExChange, mgID, events.ModeExChangeReqData{RoomID: roomID, ModeEx: modeEx})
}

// DrawImageClear 画图清除。
func (s *PushEventService) DrawImageClear(ctx context.Context, mgID, roomID, uid string) error {
	return s.Push(ctx, EventDrawImageClear, mgID, events.DrawImageClearReqData{RoomID: roomID, UID: uid})
}

// UserToPlayer 指定用户成为默认玩家（小丑牌等游戏）。
func (s *PushEventService) UserToPlayer(ctx context.Context, mgID string, data events.UserToPlayerReqData) error {
	return s.Push(ctx, EventUserToPlayer, mgID, data)
}

// PlayerAsset 批量获取玩家资产（uids 最多 10 个）。
func (s *PushEventService) PlayerAsset(ctx context.Context, mgID string, uids []string) (*events.PlayerAssetRespData, error) {
	var resp events.PlayerAssetRespData
	err := s.PushWithResp(ctx, EventPlayerAsset, mgID, events.PlayerAssetReqData{UIDs: uids}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
