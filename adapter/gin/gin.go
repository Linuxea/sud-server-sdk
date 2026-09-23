// Package ginsud 提供 Sud 回调服务到 gin 框架的适配。
//
// 使用方式：
//
//	srv := callback.NewServer(cbs, callback.WithSigner(client.Signer()))
//	ginsud.Mount(router, "/sud", srv)
//
// prefix 需与 callback.WithPathPrefix 配置一致（默认 "/sud"）。
// 适配器为独立 module（引入 gin 依赖），主 SDK 保持零第三方依赖。
package ginsud

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/linuxea/sud-server-sdk/callback"
)

// Mount 将 Sud 回调服务挂载到 gin 路由树的 prefix 下。
// prefix 以 / 开头、不以 / 结尾（如 "/sud"），需与 callback.WithPathPrefix 配置一致
// （不一致时内部 mux 会 404，SDK 无法在编译期拦截）。
//
// 注意 gin/httprouter 限制：同一前缀下已注册静态路由（如 /sud/other）后再 Mount
// 会在注册期 panic（catch-all 冲突），请先 Mount 或使用互不相干的前缀。
func Mount(r gin.IRouter, prefix string, srv *callback.Server) {
	prefix = "/" + strings.Trim(prefix, "/")
	r.Any(prefix+"/*path", gin.WrapH(srv.Handler()))
}
