package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/linuxea/sud-server-sdk/events"
)

// golden 断言 helper：marshal 后逐子串断言（零值敏感字段防回归）。
func assertJSON(t *testing.T, got []byte, wants []string, notWants []string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(string(got), w) {
			t.Fatalf("缺少 %q:\n%s", w, got)
		}
	}
	for _, nw := range notWants {
		if strings.Contains(string(got), nw) {
			t.Fatalf("不应出现 %q:\n%s", nw, got)
		}
	}
}

func TestGoldenUserReadyFalseKept(t *testing.T) {
	b, _ := json.Marshal(events.UserReadyReqData{RoomID: "r1", UID: "u1", IsReady: false})
	assertJSON(t, b, []string{`"is_ready":false`}, nil)
}

func TestGoldenModeExZeroKept(t *testing.T) {
	b, _ := json.Marshal(events.ModeExChangeReqData{RoomID: "r1", ModeEx: 0})
	assertJSON(t, b, []string{`"mode_ex":0`}, nil)
}

func TestGoldenUserInCodeOnly(t *testing.T) {
	b, _ := json.Marshal(events.UserInReqData{Code: "c1", RoomID: "r1"})
	assertJSON(t, b, []string{`"code":"c1"`, `"room_id":"r1"`},
		[]string{"user_info", `"mode"`}) // code 回退生效：无 user_info、无零值 mode
}

func TestGoldenUserInWithUserInfo(t *testing.T) {
	b, _ := json.Marshal(events.UserInReqData{
		RoomID:   "r1",
		UserInfo: &events.UserInfo{UID: "u1", NickName: "n", AvatarURL: "a", Gender: "male"},
		Mode:     2,
	})
	assertJSON(t, b, []string{`"user_info":{"uid":"u1"`, `"mode":2`}, []string{`"code"`})
}

func TestGoldenQuickStart(t *testing.T) {
	b, _ := json.Marshal(events.QuickStartReqData{
		RoomID: "r1",
		UserInfos: []events.QuickStartUserInfo{
			{UserInfo: events.UserInfo{UID: "u1", NickName: "n", AvatarURL: "a", Gender: "male"}},
			{UserInfo: events.UserInfo{UID: "ai1", IsAI: 1}, AIType: 1, AIID: "7"},
		},
		ReportGameInfoKey: "key001",
		CaptainUID:        "u1",
	})
	assertJSON(t, b, []string{
		`"room_id":"r1"`, `"uid":"u1"`, `"is_ai":1`, `"ai_type":1`, `"ai_id":"7"`,
		`"report_game_info_key":"key001"`, `"captain_uid":"u1"`,
	}, nil)
}

func TestPushEventGoldenBodies(t *testing.T) {
	// 经 fakePoster 校验便捷方法构造的完整 body
	p := &fakePoster{}
	s := NewPushEventService(p)

	_ = s.UserReady(context.Background(), "mg1", "r1", "u1", false)
	var raw map[string]any
	_ = json.Unmarshal(p.reqBody, &raw)
	data := raw["data"].(map[string]any)
	if data["is_ready"] != false {
		t.Fatalf("is_ready=false 丢失: %s", p.reqBody)
	}

	p2 := &fakePoster{}
	s2 := NewPushEventService(p2)
	_ = s2.ModeExChange(context.Background(), "mg1", "r1", 0)
	_ = json.Unmarshal(p2.reqBody, &raw)
	data = raw["data"].(map[string]any)
	if v, ok := data["mode_ex"]; !ok || v != float64(0) {
		t.Fatalf("mode_ex=0 丢失: %s", p2.reqBody)
	}

	p3 := &fakePoster{}
	s3 := NewPushEventService(p3)
	_ = s3.UserIn(context.Background(), "mg1", events.UserInReqData{Code: "c1", RoomID: "r1"})
	assertJSON(t, p3.reqBody, []string{`"code":"c1"`}, []string{"user_info"})
}

func TestRoomInfoRespPlayerSingular(t *testing.T) {
	p := &fakePoster{respData: `{"status":"WATING","captain_uid":"u1","player":[{"uid":"u1","seat_index":0,"status":"READY","is_ai":0}]}`}
	s := NewPushEventService(p)
	resp, err := s.RoomInfo(context.Background(), "mg1", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != events.RoomStatusWaiting || len(resp.Player) != 1 || resp.Player[0].Status != events.SeatPlayerStatusReady {
		t.Fatalf("RoomInfoRespData: %+v", resp)
	}
}

func TestURLKeyConstants(t *testing.T) {
	// 每个 Key 常量与 urlResolvers 的 key 一一对应（根包维护映射，此处防常量拼写漂移）
	for _, k := range []URLKey{KeyMGList, KeyMGInfo, KeyGameReportInfoPage, KeyQueryGameReportInfo,
		KeyPushEvent, KeyCreateOrder, KeyBatchCreateOrder, KeyQueryOrder, KeyQueryMatchBase,
		KeyQueryMatchRoundIds, KeyQueryUserSettle, KeyLLMCreateVoice, KeyLLMTrainVoice,
		KeyLLMGetVoice, KeyLLMCreateAICharacter, KeyLLMGetAICharacter} {
		if k == "" {
			t.Fatal("空 URLKey")
		}
	}
}
