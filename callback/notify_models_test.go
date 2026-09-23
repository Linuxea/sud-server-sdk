package callback

import (
	"context"
	"encoding/json"
	"testing"
)

// notifyEnv 构造 notify 请求体。
func notifyEnv(event, data string) string {
	return `{"notify_id":"n1","notify_time":"1","app_id":"a","notify_event":"` + event + `","data":` + data + `}`
}

// TestNotifyModelsTypedDecode 逐事件注册对应 handler 并断言解码字段。
func TestNotifyModelsTypedDecode(t *testing.T) {
	type record struct {
		event string
		room  string
		note  string
	}
	var rec record

	cbs := Callbacks{
		NotifyCallbacks: NotifyCallbacks{
			OnRoomUsersChanged: func(ctx context.Context, d RoomUsersChangedModel) { rec = record{"users", d.RoomID, ""} },
			OnOrderChanged:     func(ctx context.Context, d OrderChangedModel) { rec = record{"order", d.RoomID, d.Status} },
			OnOrderBatchChanged: func(ctx context.Context, d OrderBatchChangedModel) {
				rec = record{"order.batch", d.RoomID, d.Orders[0].Status}
			},
			OnBidResult: func(ctx context.Context, d BidResultModel) { rec = record{"bid", d.RoomID, d.PlayerBids[0].OrderID} },
			OnUserSettle: func(ctx context.Context, d UserSettleModel) {
				rec = record{"user.settle", d.RoomID, d.Results[0].OrderID}
			},
			OnMatchStart: func(ctx context.Context, d MatchStartModel) { rec = record{"match.start", d.RoomID, d.BattleStartAt} },
			OnMatchSettle: func(ctx context.Context, d MatchSettleModel) {
				rec = record{"match.settle", d.RoomID, d.BattleDuration}
			},
			OnRoomGameRule: func(ctx context.Context, d RoomGameRuleModel) { rec = record{"rule", d.RoomID, d.Rule} },
			OnGameProcess:  func(ctx context.Context, d GameProcessModel) { rec = record{"process", d.RoomID, d.Event} },
			OnPlayerStatus: func(ctx context.Context, d PlayerStatusModel) {
				rec = record{"status", d.RoomID, d.Players[0].Timestamp}
			},
			OnGamePlayerConnectStatus: func(ctx context.Context, d GamePlayerConnectStatusModel) {
				rec = record{"connect", d.RoomID, d.Player.OfflineTimestamp}
			},
			OnRoomSeatChanged: func(ctx context.Context, d SeatChangedModel) {
				rec = record{"seat", d.RoomID, d.SeatPlayers[0].Status}
			},
			OnPlayerDrawImage: func(ctx context.Context, d DrawImageModel) { rec = record{"draw", d.RoomID, d.ImageData} },
			OnLLMAICreateResult: func(ctx context.Context, d AICreateResultModel) {
				rec = record{"llm.ai", d.RoomID, d.BizErrorMsg}
			},
			OnLLMGameSituationSummary: func(ctx context.Context, d GameSituationSummaryModel) {
				rec = record{"llm.summary", d.RoomID, d.SituationSummary}
			},
		},
	}
	_, ts := newTestServer(cbs, false)
	defer ts.Close()

	cases := []struct {
		event string
		data  string
		want  record
	}{
		{NotifyRoomUsersChanged, `{"room_id":"r1","mg_id":"m1","player_total":2,"ob_total":3,"changed_time":"1"}`, record{"users", "r1", ""}},
		{NotifyOrderChanged, `{"order_id":"o1","out_order_id":"oo1","mg_id":"m1","room_id":"r1","cmd":"gift","from_uid":"u1","to_uid":"u2","status":"EXECUTE_SUCCESS"}`, record{"order", "r1", "EXECUTE_SUCCESS"}},
		{NotifyOrderBatchChanged, `{"mg_id":"m1","room_id":"r1","orders":[{"order_id":"o1","out_order_id":"oo1","cmd":"gift","from_uid":"u","to_uid":"v","value":1,"status":"EXECUTE_FAIL"}]}`, record{"order.batch", "r1", "EXECUTE_FAIL"}},
		{NotifyBidResult, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","player_bids":[{"order_id":"bo1","out_order_id":"oo1","uid":"u1","bid_rs":0,"bid_value":100}]}`, record{"bid", "r1", "bo1"}},
		{NotifyUserSettle, `{"mg_id":"m1","room_id":"r1","match_id":"mm1","results":[{"uid":"u1","order_id":"so1","out_order_id":"oo1","is_ai":0,"score":10,"remain_score":90}],"settle_at":"1"}`, record{"user.settle", "r1", "so1"}},
		{NotifyMatchStart, `{"mg_id":"m1","room_id":"r1","game_mode":1,"match_id":"mm1","battle_start_at":"1663991010"}`, record{"match.start", "r1", "1663991010"}},
		{NotifyMatchSettle, `{"mg_id":"m1","room_id":"r1","game_mode":1,"match_id":"mm1","battle_start_at":"1","battle_end_at":"2","battle_duration":"10","results":[{"uid":"u1","is_ai":0,"score":-100,"remain_score":0}]}`, record{"match.settle", "r1", "10"}},
		{NotifyRoomGameRule, `{"mg_id":"m1","room_id":"r1","rule":"{\"small_blind\":1}"}`, record{"rule", "r1", `{"small_blind":1}`}},
		{NotifyGameProcess, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","event":"self_die","players":[{"uid":"u1"}]}`, record{"process", "r1", "self_die"}},
		{NotifyPlayerStatus, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","event":"manage_status","players":[{"uid":"u1","timestamp":"1716814942000"}]}`, record{"status", "r1", "1716814942000"}},
		{NotifyGamePlayerConnectStatus, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","player":{"uid":"u1","offline_timestamp":"1716814942000","status":0,"offline_seconds":30}}`, record{"connect", "r1", "1716814942000"}},
		{NotifyRoomSeatChanged, `{"mg_id":"m1","room_id":"r1","changed_time":"1","seat_changed":[{"uid":"u1","changed_type":3}],"seat_players":[{"uid":"u1","seat_index":0,"status":"READY"}]}`, record{"seat", "r1", "READY"}},
		{NotifyPlayerDrawImage, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","uid":"u1","image_data":"data:image/png;base64,xx","timestamp":"1"}`, record{"draw", "r1", "data:image/png;base64,xx"}},
		{NotifyLLMAICreateResult, `{"mg_id":"m1","room_id":"r1","uid":"u1","biz_error_code":0,"biz_error_msg":"ok"}`, record{"llm.ai", "r1", "ok"}},
		{NotifyLLMGameSituationSummary, `{"mg_id":"m1","room_id":"r1","game_round_id":"g1","situation_summary":"局面胶着"}`, record{"llm.summary", "r1", "局面胶着"}},
	}
	for _, tc := range cases {
		resp := post(t, ts.URL+"/sud/notify", notifyEnv(tc.event, tc.data))
		var ack map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&ack)
		resp.Body.Close()
		if ack["ret_code"].(float64) != 0 || ack["ret_msg"] != "SUCCESS" {
			t.Fatalf("%s 应答错误: %v", tc.event, ack)
		}
		if rec != tc.want {
			t.Fatalf("%s 解码错误: got %+v want %+v", tc.event, rec, tc.want)
		}
	}
}
