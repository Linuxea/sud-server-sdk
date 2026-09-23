package ginsud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/linuxea/sud-server-sdk/callback"
)

func TestMount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	srv := callback.NewServer(callback.Callbacks{
		GetSSToken: func(ctx context.Context, req callback.GetSSTokenReq) (*callback.GetSSTokenResp, error) {
			return &callback.GetSSTokenResp{SSToken: "tk"}, nil
		},
	})
	Mount(router, "/sud", srv)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sud/get_sstoken", strings.NewReader(`{"code":"c1"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"ss_token":"tk"`) {
		t.Fatalf("body: %s", resp.Body.String())
	}
}
