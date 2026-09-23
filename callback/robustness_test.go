package callback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMethodNotAllowed(t *testing.T) {
	_, ts := newTestServer(Callbacks{}, false)
	defer ts.Close()
	resp := post(t, ts.URL+"/sud/get_sstoken", `{}`)
	resp.Body.Close()
	// post helper 是 POST；改用 GET 验证 405
	getResp, err := http.Get(ts.URL + "/sud/get_sstoken")
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET 应 405，实际 %d", getResp.StatusCode)
	}
}

func TestEmptyBodySkipsHandler(t *testing.T) {
	called := false
	_, ts := newTestServer(Callbacks{
		GetSSToken: func(ctx context.Context, req GetSSTokenReq) (*GetSSTokenResp, error) {
			called = true
			return nil, nil
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/get_sstoken", ``)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if called {
		t.Fatal("空 body 不应进入 handler")
	}
	if got["ret_code"].(float64) != 0 {
		t.Fatalf("默认模式空 body 应成功壳: %v", got)
	}
}

func TestInvalidBodyDefaultVsStrict(t *testing.T) {
	// 默认：成功壳
	ts1 := startServer(t, NewServer(Callbacks{}))
	resp := post(t, ts1+"/sud/get_sstoken", `{invalid`)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got["ret_code"].(float64) != 0 {
		t.Fatalf("默认模式非法 body 应成功壳: %v", got)
	}

	// 严格：错误壳（触发 Sud 重试）
	ts2 := startServer(t, NewServer(Callbacks{}, WithStrictDecode()))
	resp2 := post(t, ts2+"/sud/get_sstoken", `{invalid`)
	var got2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&got2)
	resp2.Body.Close()
	if got2["ret_code"].(float64) != 1 {
		t.Fatalf("严格模式非法 body 应错误壳: %v", got2)
	}

	// notify 失败应答的 ret_msg 应为 SUCCESS
	resp3 := post(t, ts1+"/sud/notify", `{invalid`)
	var got3 map[string]any
	_ = json.NewDecoder(resp3.Body).Decode(&got3)
	resp3.Body.Close()
	if got3["ret_code"].(float64) != 0 || got3["ret_msg"] != "SUCCESS" {
		t.Fatalf("notify 失败应答应为 SUCCESS: %v", got3)
	}
}

// startServer 启动独立 Server 并返回 URL（自动清理）。
func startServer(t *testing.T, s *Server) string {
	t.Helper()
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestTypedNilRespOmitsData(t *testing.T) {
	_, ts := newTestServer(Callbacks{
		GetSSToken: func(ctx context.Context, req GetSSTokenReq) (*GetSSTokenResp, error) {
			return (*GetSSTokenResp)(nil), nil // typed-nil
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/get_sstoken", `{"code":"c"}`)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"data"`) {
		t.Fatalf("typed-nil 应省略 data: %s", body)
	}
}

func TestNotifyParseFailFallsBackToOnNotify(t *testing.T) {
	var rawPayload string
	_, ts := newTestServer(Callbacks{
		NotifyCallbacks: NotifyCallbacks{
			// RoomUsersChangedModel 的 player_total 是 int，传字符串使其解析失败
			OnRoomUsersChanged: func(ctx context.Context, d RoomUsersChangedModel) {
				t.Fatal("解析失败不应进入强类型 handler")
			},
		},
		OnNotify: func(ctx context.Context, event string, payload json.RawMessage) {
			rawPayload = string(payload)
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/notify", notifyEnv(NotifyRoomUsersChanged, `{"room_id":"r1","player_total":"bad"}`))
	var ack map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ack)
	resp.Body.Close()
	if ack["ret_code"].(float64) != 0 {
		t.Fatalf("应答: %v", ack)
	}
	if !strings.Contains(rawPayload, `"room_id":"r1"`) {
		t.Fatalf("OnNotify 兜底应收到原始 payload: %s", rawPayload)
	}
}

func TestPanicRecovered(t *testing.T) {
	_, ts := newTestServer(Callbacks{
		GetSSToken: func(ctx context.Context, req GetSSTokenReq) (*GetSSTokenResp, error) {
			panic("boom")
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/get_sstoken", `{"code":"c"}`)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	if got["ret_code"].(float64) != 1 {
		t.Fatalf("panic 应返回错误壳: %v", got)
	}
}

func TestMaxBodySizeExceeded(t *testing.T) {
	_, ts := newTestServer(Callbacks{}, false)
	defer ts.Close()

	big := strings.Repeat("a", 1025)
	resp := post(t, ts.URL+"/sud/get_sstoken", `{"code":"`+big+`"}`)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	// 默认上限 10MB 不会触发；这里只验证请求正常处理
	if got["ret_code"] == nil {
		t.Fatal("应有响应壳")
	}

	// 显式 1KB 上限 → 超限按解析失败处理（成功壳，handler 未实现也不影响）
	srv := startServer(t, NewServer(Callbacks{}, WithMaxBodySize(1024)))
	resp2 := post(t, srv+"/sud/get_sstoken", `{"code":"`+big+`"}`)
	var got2 map[string]any
	_ = json.NewDecoder(resp2.Body).Decode(&got2)
	resp2.Body.Close()
	if got2["ret_code"].(float64) != 0 {
		t.Fatalf("超限默认成功壳: %v", got2)
	}
}

func TestGetAccountPath(t *testing.T) {
	_, ts := newTestServer(Callbacks{
		GetAccount: func(ctx context.Context, req GetAccountReq) (*GetAccountResp, error) {
			if req.UID != "u1" {
				t.Fatalf("uid = %s", req.UID)
			}
			return &GetAccountResp{Nickname: "n", AvatarURL: "a", Score: 100, VIPLevel: 2}, nil
		},
	}, false)
	defer ts.Close()

	resp := post(t, ts.URL+"/sud/get_account", `{"uid":"u1"}`)
	var got map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&got)
	resp.Body.Close()
	data := got["data"].(map[string]any)
	if data["vip_level"].(float64) != 2 || data["score"].(float64) != 100 {
		t.Fatalf("get_account data: %v", data)
	}
}
