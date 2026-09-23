package api

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/linuxea/sud-server-sdk/events"
)

func TestPushEventBodyShape(t *testing.T) {
	p := &fakePoster{}
	s := NewPushEventService(p)
	err := s.RoomClear(context.Background(), "mg1", "room1")
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyPushEvent) {
		t.Fatalf("key = %s", p.key)
	}
	// 校验请求壳结构
	var raw map[string]any
	if err := json.Unmarshal(p.reqBody, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["event"] != EventRoomClear || raw["mg_id"] != "mg1" {
		t.Fatalf("event/mg_id 错误: %s", p.reqBody)
	}
	ts, ok := raw["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp 应为字符串: %s", p.reqBody)
	}
	if _, err := strconv.ParseInt(ts, 10, 64); err != nil {
		t.Fatalf("timestamp 非法: %v", err)
	}
	data, _ := raw["data"].(map[string]any)
	if data["room_id"] != "room1" {
		t.Fatalf("data.room_id 错误: %s", p.reqBody)
	}
}

func TestPushEventWithResp(t *testing.T) {
	p := &fakePoster{respData: `{"uids":["u1"]}`}
	s := NewPushEventService(p)
	resp, err := s.AiAdd(context.Background(), "mg1", events.AiAddReqData{
		RoomID:    "r1",
		AIPlayers: []events.AIPlayer{{UID: "u1", Avatar: "a", Name: "n", Gender: "male", AILevel: 1}},
		IsReady:   1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.UIDs) != 1 || resp.UIDs[0] != "u1" {
		t.Fatalf("AiAddRespData 解析错误: %+v", resp)
	}
	var raw map[string]any
	_ = json.Unmarshal(p.reqBody, &raw)
	if raw["event"] != EventAiAdd {
		t.Fatalf("event = %v", raw["event"])
	}
}

func TestPushEventConvenience(t *testing.T) {
	p := &fakePoster{}
	s := NewPushEventService(p)
	if err := s.GameEnd(context.Background(), "mg1", "r1", "u1"); err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	_ = json.Unmarshal(p.reqBody, &raw)
	if raw["event"] != EventGameEnd {
		t.Fatal("GameEnd event 错误")
	}
	if err := s.UserReady(context.Background(), "mg1", "r1", "u1", true); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(p.reqBody, &raw)
	if raw["event"] != EventUserReady {
		t.Fatal("UserReady event 错误")
	}
}
