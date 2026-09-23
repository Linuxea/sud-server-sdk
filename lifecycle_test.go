package sud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestURLResolversCoverAllKeys(t *testing.T) {
	// 全 key 表驱动：每个 key 都能在完整配置中解析出非空地址
	full := &APIConfig{
		API: APIConfigAPI{
			MGList: "u1", MGInfo: "u2", GetGameReportInfo: "u3", GetGameReportInfoPage: "u4",
			QueryGameReportInfo: "u5", GetPlayerResults: "u6", ReportGameRoundBill: "u7",
			PushEvent: "u8", CreateOrder: "u9", BatchCreateOrder: "u10", QueryOrder: "u11",
			QueryMatchBase: "u12", QueryMatchRoundIds: "u13", QueryUserSettle: "u14",
			AuthAppList: "u15", AuthRoomList: "u16",
		},
		LLMAPI: APIConfigLLM{
			CreateVoice: "l1", TrainVoice: "l2", GetVoice: "l3", CreateAICharacter: "l4", GetAICharacter: "l5",
		},
	}
	for key, resolve := range urlResolvers {
		if resolve(full) == "" {
			t.Fatalf("key=%s 在完整配置下解析为空", key)
		}
	}
	if len(urlResolvers) != 19 {
		t.Fatalf("应有 19 个 key 映射，实际 %d", len(urlResolvers))
	}
}

func TestCloseLifecycle(t *testing.T) {
	// Close 先于首次拉取：不启动后台协程，之后 Post 返回 ErrClosed
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	c.Close()
	err := c.Post(context.Background(), "push_event", nil, nil)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Close 后应返回 ErrClosed: %v", err)
	}
	c.Close() // 幂等

	// 正常生命周期：拉取 → Close → Post 报 ErrClosed
	c2, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{}}`))
	})
	if err := c2.Post(context.Background(), "push_event", nil, nil); err != nil {
		t.Fatalf("Close 前 Post 应成功: %v", err)
	}
	c2.Close()
	c2.Close()
	err = c2.Post(context.Background(), "push_event", nil, nil)
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("Close 后应返回 ErrClosed: %v", err)
	}
}

func TestBackgroundRefresh(t *testing.T) {
	var cfgFetches int32
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{"ok":1}}`))
	}))
	defer apiSrv.Close()
	cfgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&cfgFetches, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{"api": map[string]string{"push_event": apiSrv.URL}})
	}))
	defer cfgSrv.Close()

	c := New(testAppID, testAppSecret,
		WithAPIConfigBase(cfgSrv.URL+"/"),
		WithRetry(0, 0),
		WithAPIConfigRefreshInterval(30*time.Millisecond))
	defer c.Close()

	if err := c.Post(context.Background(), "push_event", nil, nil); err != nil {
		t.Fatal(err)
	}
	first := atomic.LoadInt32(&cfgFetches)
	if first != 1 {
		t.Fatalf("首次拉取应 1 次，实际 %d", first)
	}
	// 等待后台 ticker 至少触发一次
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&cfgFetches) <= first && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&cfgFetches) <= first {
		t.Fatal("后台刷新未发生")
	}
}

func TestAPIConfigFallbackToStaleCache(t *testing.T) {
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{}}`))
	}))
	defer apiSrv.Close()
	var fail int32
	cfgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 第一次正常返回，之后 500
		if atomic.AddInt32(&fail, 1) > 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"api": map[string]string{"push_event": apiSrv.URL}})
	}))
	defer cfgSrv.Close()

	c := New(testAppID, testAppSecret, WithAPIConfigBase(cfgSrv.URL+"/"), WithRetry(0, 0))
	defer c.Close()

	// 首次 Post 成功（缓存建立）
	if err := c.Post(context.Background(), "push_event", nil, nil); err != nil {
		t.Fatal(err)
	}
	// 配置服务 500 后：手动 APIConfig 应回退旧缓存
	cfg, err := c.APIConfig(context.Background())
	if err != nil {
		t.Fatalf("刷新失败应回退旧缓存: %v", err)
	}
	if cfg.API.PushEvent != apiSrv.URL {
		t.Fatalf("旧缓存内容错误: %+v", cfg.API)
	}
	// Post 仍可使用旧缓存成功
	if err := c.Post(context.Background(), "push_event", nil, nil); err != nil {
		t.Fatalf("旧缓存下 Post 应成功: %v", err)
	}
}

func TestPostContextCancelPropagates(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{}}`))
	}, WithRetry(3, 50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.Post(ctx, "push_event", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ctx 取消应传播: %v", err)
	}
}

func TestUnknownURLKey(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	err := c.Post(context.Background(), "no_such_key", nil, nil)
	if !errors.Is(err, ErrURLKeyUnknown) {
		t.Fatalf("未知 key 应返回 ErrURLKeyUnknown: %v", err)
	}
}
