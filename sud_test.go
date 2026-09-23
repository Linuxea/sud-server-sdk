package sud

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAppID     = "1461564080052506636"
	testAppSecret = "test-app-secret"
)

// golden：python hmac(md5) 独立计算
const goldenAppServerSign = "69b198d59bc9d492a515643914d9f82b"

func TestAppServerSign(t *testing.T) {
	if got := appServerSign(testAppID, testAppSecret); got != goldenAppServerSign {
		t.Fatalf("appServerSign got %s want %s", got, goldenAppServerSign)
	}
}

// newTestClient 搭建配置服务与目标 API 的 httptest 环境。
func newTestClient(t *testing.T, apiHandler http.HandlerFunc, opts ...Option) (*Client, *httptest.Server) {
	t.Helper()
	apiSrv := httptest.NewServer(apiHandler)
	cfgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("配置接口应为 GET，实际 %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, goldenAppServerSign) {
			t.Errorf("配置接口路径应带签名 %s，实际 %s", goldenAppServerSign, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"api":     map[string]string{"push_event": apiSrv.URL},
			"llm_api": map[string]string{},
		})
	}))
	t.Cleanup(func() { apiSrv.Close(); cfgSrv.Close() })

	all := append([]Option{WithAPIConfigBase(cfgSrv.URL + "/"), WithRetry(0, time.Millisecond)}, opts...)
	c := New(testAppID, testAppSecret, all...)
	t.Cleanup(c.Close)
	return c, apiSrv
}

func TestClientPost(t *testing.T) {
	var gotAuth, gotBody string
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotAuth = r.Header.Get(HeaderAuthorization)
		gotBody = string(body)
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{"ok":1}}`))
	})

	var resp struct {
		OK int `json:"ok"`
	}
	if err := c.Post(context.Background(), "push_event", map[string]string{"a": "b"}, &resp); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp.OK != 1 {
		t.Fatalf("data 解析失败: %+v", resp)
	}
	if !strings.HasPrefix(gotAuth, `Sud-Auth app_id="`+testAppID) {
		t.Fatalf("Authorization 头错误: %s", gotAuth)
	}
	if gotBody != `{"a":"b"}` {
		t.Fatalf("请求 body: %s", gotBody)
	}
}

func TestClientPostRetCodeError(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ret_code":1001,"ret_msg":"参数错误","data":null}`))
	})
	err := c.Post(context.Background(), "push_event", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("应为 APIError: %v", err)
	}
	if apiErr.RetCode != 1001 {
		t.Fatalf("RetCode: %d", apiErr.RetCode)
	}
}

func TestClientPostRetry(t *testing.T) {
	var calls int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) <= 2 { // 前两次失败
			_, _ = w.Write([]byte(`{"ret_code":500,"ret_msg":"server error","data":null}`))
			return
		}
		_, _ = w.Write([]byte(`{"ret_code":0,"ret_msg":"","data":{"ok":9}}`))
	}, WithRetry(3, time.Millisecond))

	var resp struct {
		OK int `json:"ok"`
	}
	if err := c.Post(context.Background(), "push_event", nil, &resp); err != nil {
		t.Fatalf("重试后应成功: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("应请求 3 次，实际 %d", got)
	}
}

func TestClientURLNotConfigured(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	err := c.Post(context.Background(), "llm_create_voice", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "API 地址未配置") {
		t.Fatalf("未配置 key 应报错: %v", err)
	}
}

func TestClientConfigFetchFailNoCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(testAppID, testAppSecret, WithAPIConfigBase(srv.URL+"/"), WithRetry(0, 0))
	defer c.Close()
	err := c.Post(context.Background(), "push_event", nil, nil)
	if err == nil {
		t.Fatal("配置拉取失败应报错")
	}
}
