package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxea/sud-server-sdk"
)

const (
	testAppID     = "1461564080052506636"
	testAppSecret = "test-app-secret"
)

// signedRequest 构造带验签头的请求。
func signedRequest(t *testing.T, method, url, body string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		signer := sud.NewSigner(testAppID, testAppSecret)
		req.Header.Set(sud.HeaderSudAppID, testAppID)
		req.Header.Set(sud.HeaderSudTimestamp, "1654079242000")
		req.Header.Set(sud.HeaderSudNonce, "keVJLJTItd1VBtGT")
		req.Header.Set(sud.HeaderSudSignature, signer.Signature("1654079242000", "keVJLJTItd1VBtGT", []byte(body)))
	}
	return req
}

func newTestServer(cbs Callbacks, verify bool) (*Server, *httptest.Server) {
	s := NewServer(cbs)
	if verify {
		s.signer = sud.NewSigner(testAppID, testAppSecret)
	}
	ts := httptest.NewServer(s.Handler())
	return s, ts
}

// post 发送 JSON POST 并检查错误。
func post(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestGetSSToken(t *testing.T) {
	_, ts := newTestServer(Callbacks{
		GetSSToken: func(ctx context.Context, req GetSSTokenReq) (*GetSSTokenResp, error) {
			if req.Code != "c1" {
				t.Fatalf("code = %s", req.Code)
			}
			return &GetSSTokenResp{
				SSToken:    "token1",
				ExpireDate: 1630417861359,
				UserInfo:   &UserInfo{UID: "u1", NickName: "萌萌", AvatarURL: "a", Gender: "female"},
			}, nil
		},
	}, false)
	defer ts.Close()

	resp, err := http.DefaultClient.Do(signedRequest(t, "POST", ts.URL+"/sud/get_sstoken", `{"code":"c1"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["ret_code"].(float64) != 0 {
		t.Fatalf("ret_code != 0: %v", got)
	}
	data := got["data"].(map[string]any)
	if data["ss_token"] != "token1" {
		t.Fatalf("data: %v", data)
	}
	if data["user_info"].(map[string]any)["uid"] != "u1" {
		t.Fatalf("user_info: %v", data)
	}
}

func TestNotImplemented(t *testing.T) {
	_, ts := newTestServer(Callbacks{}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/update_sstoken", `{"ss_token":"t"}`)
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["ret_code"].(float64) != 1 || !strings.Contains(got["ret_msg"].(string), "not implemented") {
		t.Fatalf("未实现回调应返回错误壳: %v", got)
	}
}

func TestCallbackError(t *testing.T) {
	_, ts := newTestServer(Callbacks{
		UpdateScore: func(ctx context.Context, req UpdateScoreReq) (*UpdateScoreResp, error) {
			return nil, ErrDuplicateOrderID
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/update_score", `{"order_id":"o1","mg_id":"m","round_id":"r","uid":"u","score":1,"type":1}`)
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["ret_code"].(float64) != 1 || got["sdk_error_code"].(float64) != 9001 {
		t.Fatalf("业务错误码应透传: %v", got)
	}
}

func TestReportGameInfo(t *testing.T) {
	var gotType string
	_, ts := newTestServer(Callbacks{
		ReportGameInfo: func(ctx context.Context, req ReportGameInfoReq) error {
			gotType = req.ReportType
			settle, err := req.ParseGameSettle()
			if err != nil {
				t.Fatal(err)
			}
			if settle.RoomID != "9009" || len(settle.Results) != 1 || settle.Results[0].IsWin != 2 {
				t.Fatalf("settle: %+v", settle)
			}
			return nil
		},
	}, false)
	defer ts.Close()

	body := `{"report_type":"game_settle","report_msg":{"mg_id":123,"mg_id_str":"123","room_id":"9009","game_mode":1,"game_round_id":"rr1","battle_start_at":1,"battle_end_at":2,"battle_duration":1,"results":[{"uid":"u1","rank":1,"is_escaped":0,"is_ai":0,"is_win":2}]},"uid":"u1","ss_token":"t"}`
	resp := post(t, ts.URL+"/sud/report_game_info", body)
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["ret_code"].(float64) != 0 || gotType != "game_settle" {
		t.Fatalf("report_game_info: %v %s", got, gotType)
	}
}

func TestVerifySignature(t *testing.T) {
	_, ts := newTestServer(Callbacks{}, true)
	defer ts.Close()

	// 无签名头 → 拒绝
	resp := post(t, ts.URL+"/sud/get_sstoken", `{}`)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got["ret_code"].(float64) != 1 || !strings.Contains(got["ret_msg"].(string), "signature") {
		t.Fatalf("缺签名应拒绝: %v", got)
	}

	// 带合法签名 → 通过（回调未实现返回 not implemented 而非验签错误）
	resp2, err := http.DefaultClient.Do(signedRequest(t, "POST", ts.URL+"/sud/get_sstoken", `{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var got2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&got2)
	if !strings.Contains(got2["ret_msg"].(string), "not implemented") {
		t.Fatalf("合法签名应放行: %v", got2)
	}
}

func TestNotifyDispatch(t *testing.T) {
	var gotOrder *OrderChangedModel
	var fallbackEvent string
	_, ts := newTestServer(Callbacks{
		NotifyCallbacks: NotifyCallbacks{
			OnOrderChanged: func(ctx context.Context, data OrderChangedModel) {
				gotOrder = &data
			},
		},
		OnNotify: func(ctx context.Context, event string, payload json.RawMessage) {
			fallbackEvent = event
		},
	}, false)
	defer ts.Close()

	// 已注册事件 → 定向分发
	body := `{"notify_id":"n1","notify_time":"1","app_id":"a","notify_event":"sud.mg.merchant.order.changed","data":{"order_id":"o1","out_order_id":"oo1","mg_id":"m","room_id":"r","cmd":"gift","from_uid":"u1","to_uid":"u2","status":"EXECUTE_SUCCESS"}}`
	resp := post(t, ts.URL+"/sud/notify", body)
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if got["ret_code"].(float64) != 0 || got["ret_msg"] != "SUCCESS" {
		t.Fatalf("notify 应答: %v", got)
	}
	if gotOrder == nil || gotOrder.OrderID != "o1" || gotOrder.Status != "EXECUTE_SUCCESS" {
		t.Fatalf("OnOrderChanged: %+v", gotOrder)
	}

	// 未注册事件 → OnNotify 兜底
	body2 := `{"notify_id":"n2","notify_time":"1","app_id":"a","notify_event":"sud.mg.merchant.player.draw.image","data":{"uid":"u1"}}`
	resp2 := post(t, ts.URL+"/sud/notify", body2)
	resp2.Body.Close()
	if fallbackEvent != NotifyPlayerDrawImage {
		t.Fatalf("OnNotify fallback: %s", fallbackEvent)
	}
}

func TestPathPrefix(t *testing.T) {
	_, ts := newTestServer(Callbacks{}, false)
	defer ts.Close()
	s2 := NewServer(Callbacks{}, WithPathPrefix("/custom"))
	ts2 := httptest.NewServer(s2.Handler())
	defer ts2.Close()
	resp := post(t, ts2.URL+"/custom/get_sstoken", `{}`)
	defer resp.Body.Close()
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if _, ok := got["ret_msg"]; !ok {
		t.Fatal("自定义前缀路由失败")
	}
}
