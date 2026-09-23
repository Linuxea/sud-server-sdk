package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// fakePoster 记录请求 key/body 并返回预设 data。
type fakePoster struct {
	key      string
	reqBody  []byte
	respData string
}

func (f *fakePoster) Post(_ context.Context, key string, req any, resp any) error {
	f.key = key
	if req != nil {
		b, err := json.Marshal(req)
		if err != nil {
			return err
		}
		f.reqBody = b
	} else {
		f.reqBody = nil
	}
	if resp != nil {
		if err := json.Unmarshal([]byte(f.respData), resp); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakePoster) AppID() string     { return "app123" }
func (f *fakePoster) AppSecret() string { return "secret123" }

func TestGameListService(t *testing.T) {
	p := &fakePoster{respData: `{"mg_info_list":[{"mg_id":"m1","name":{"default":"Bumper","zh-CN":"碰碰"},"game_mode_list":[{"mode":1,"count":[2,9],"team_count":[2,9],"team_member_count":[1,1],"rule":"{}"}]}]}`}
	s := NewGameListService(p)
	resp, err := s.List(context.Background(), GameListReq{Platform: PlatformAndroid})
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyMGList) {
		t.Fatalf("key = %s", p.key)
	}
	if !strings.Contains(string(p.reqBody), `"platform":2`) {
		t.Fatalf("请求体缺少 platform: %s", p.reqBody)
	}
	if len(resp.MGInfoList) != 1 || resp.MGInfoList[0].Name.Get("zh-CN") != "碰碰" {
		t.Fatalf("响应解析错误: %+v", resp)
	}
	if resp.MGInfoList[0].Name.Get("ja-JP") != "Bumper" {
		t.Fatal("I18nText.Get 应回退 default")
	}
}

func TestReportServiceQueryByPageInjectsCredentials(t *testing.T) {
	p := &fakePoster{respData: `[{"game_round_id":"r1","report_info":{"game_settle":{"mg_id":123,"mg_id_str":"123","room_id":"9009","game_mode":1,"game_round_id":"r1","battle_start_at":1,"battle_end_at":2,"battle_duration":1,"results":[{"uid":"u1","rank":1,"is_escaped":0,"is_ai":0,"is_win":2}]}}}]`}
	s := NewReportService(p)
	list, err := s.QueryByPage(context.Background(), "9009", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyGameReportInfoPage) {
		t.Fatalf("key = %s", p.key)
	}
	if !strings.Contains(string(p.reqBody), `"app_id":"app123"`) || !strings.Contains(string(p.reqBody), `"app_secret":"secret123"`) {
		t.Fatalf("分页查询应自动注入凭证: %s", p.reqBody)
	}
	if len(list) != 1 || list[0].ReportInfo.GameSettle.Results[0].IsWin != 2 {
		t.Fatalf("响应解析错误: %+v", list)
	}
}

func TestOrderServiceCreate(t *testing.T) {
	p := &fakePoster{respData: `{"out_order_id":"o1","order_id":"s1"}`}
	s := NewOrderService(p)
	resp, err := s.Create(context.Background(), CreateOrderReq{OutOrderID: "o1", MGID: "m1", RoomID: "r1", Cmd: "add_score", FromUID: "u1", ToUID: "u2", Value: 100})
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyCreateOrder) || resp.OrderID != "s1" {
		t.Fatalf("key=%s resp=%+v", p.key, resp)
	}
}

func TestEntryScoreService(t *testing.T) {
	p := &fakePoster{respData: `{"mg_id":"m1","room_id":"r1","game_mode":1,"match_id":"mm1","battle_start_at":"1663991010","status":"CLOSED","results":[{"uid":"u1","is_ai":0,"score":-100,"remain_score":0}]}`}
	s := NewEntryScoreService(p)
	resp, err := s.QueryMatchBase(context.Background(), QueryMatchBaseReq{MatchID: "mm1"})
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyQueryMatchBase) || resp.Status != MatchStatusClosed {
		t.Fatalf("key=%s resp=%+v", p.key, resp)
	}
}

func TestLLMService(t *testing.T) {
	p := &fakePoster{respData: `{"ai_id":"a1"}`}
	s := NewLLMService(p)
	resp, err := s.CreateAICharacter(context.Background(), CreateAICharacterReq{OutAIID: "oa1", VoiceID: "v1", Gender: "male", Birthday: "2000-01-01", BloodType: "A", MBTI: "ISTJ", Personality: "p", LanguageStyle: "s", LanguageDetailStyle: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if p.key != string(KeyLLMCreateAICharacter) || resp.AIID != "a1" {
		t.Fatalf("key=%s resp=%+v", p.key, resp)
	}

	p2 := &fakePoster{respData: ``}
	s2 := NewLLMService(p2)
	if err := s2.TrainVoice(context.Background(), TrainVoiceReq{VoiceID: "v1", AudioData: "x", AudioFormat: AudioFormatMP3}); err != nil {
		t.Fatal(err)
	}
	if p2.key != string(KeyLLMTrainVoice) {
		t.Fatalf("key=%s", p2.key)
	}
}
