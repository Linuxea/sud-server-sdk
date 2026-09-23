# AGENTS.md

Sud MGP（小游戏平台）服务端 Go SDK：封装官方 GitBook 文档（docs.sud.tech，`Server` 章节下）全部服务端接口——出站 API（业务方调 Sud）+ HTTPS 回调（Sud 调业务方）。**字段保真是本 SDK 的核心价值**，一切结构体以文档为准。独立 git 仓库，直接在 main 提交，commit message 用中文短描述。

## 验证命令

- 主 module（零第三方依赖）：仓库根 `go build ./... && go vet ./... && go test ./...`
- gin 适配器（独立 module）：`cd adapter/gin && go build ./... && go vet ./... && go test ./...`
- 两处都过才算全绿；example/ 在主 module 内随根构建验证
- go.mod 固定 `go 1.19`（与 gama 系仓库一致，保证将来 gama replace 引用兼容），工具链 1.27 可用但别升 go 指令；gin 锁 v1.7.2（与 gama-admin/cluster 一致）

## go.work（本地开发必需，不入库）

双 module 结构（根 + adapter/gin）。克隆后、动 adapter 前，必须先生成 go.work：

```
go work init . ./adapter/gin
go work edit -replace github.com/linuxea/sud-server-sdk@v0.0.0=./
```

三个实测踩过的坑：

- **仅 use 不够**：adapter require 的是未发布的 `v0.0.0` 占位，源码 import gin 触发完整 module graph 加载时，go 仍会去远程拉 v0.0.0 的 go.mod → 404（golang/go#50750 一族问题）。必须配合上面的带版本 replace。
- **replace 必须带版本限定**（`@v0.0.0`）：不带会与 use 冲突，报 "replaced at all versions"。
- 发布 tag、require 改真实版本后，该 replace 因版本不匹配自动失效，届时 go.work 只需 use。

## 发布流程

1. 根 module 打 tag（如 v1.0.0）
2. `adapter/gin/go.mod` 的 `sud-server-sdk v0.0.0` 改为真实版本
3. 删本地 go.work，在 adapter/gin 下验证 `go build ./...` 可解析

## 字段保真铁律

- 改任何 req/resp 结构体前先对照官方文档。离线镜像可用 wget 抓 `docs-gitbook-dcdn.sud.tech`（静态 GitBook，正文嵌在 HTML 的 `markdown-section` 区块）；文档参数表与返回示例矛盾时取**返回示例**（如 game_settle 的 `extras` vs `report_game_info_extras`）。
- 忠实保留文档原文，即使疑似笔误：`RoomInfoRespData` 的 json key 是单数 `player`、房间状态常量 `WATING`（文档如此，不是 WAITING）。
- **零值敏感字段禁止加 omitempty**：`UserReadyReqData.IsReady`（false=取消准备是有效请求）、`ModeExChangeReqData.ModeEx`。
- `UserInReqData.UserInfo` 必须是 `*UserInfo`：值类型空 struct 无法被 omitempty 省略，会导致文档约定的 code 回退路径失效。

## 新增出站 API 的三处同步

`api/api.go` 的 URLKey 常量 → `apiconfig.go` 的 APIConfig 字段（json tag 对齐配置接口返回）→ `sud.go` 的 urlResolvers map。漏任何一处运行时报"API 地址未配置/未知 key"，编译期无提示。现有测试 `TestURLResolversCoverAllKeys` 会兜住数量级漂移。

## 协议与行为约定

- 签名串固定四行、每行以 `\n` 结尾（含最后一行），HmacSHA1 hex 小写；Authorization 头值带双引号。**golden 测试向量必须用 python hmac 独立计算**，用实现自算自己比是循环论证。
- 出站重试只针对瞬态失败（网络错误/HTTP 5xx/响应壳不可解析）；业务 `ret_code!=0`、HTTP 4xx、地址未配置是确定性失败不重试（避免对下单类非幂等接口重复提交）。
- 回调 body 解析失败/为空默认记日志返回成功壳（与 gama 现网行为对齐，防 Sud 无意义重试；代价是静默丢数据）；`WithStrictDecode` 改为错误壳触发重试。
- notify 应答恒为 `{"ret_code":0,"ret_msg":"SUCCESS"}`（无 data 字段）；同步回调成功应答带 data。
- notify 强类型解析失败时回退调 `OnNotify` 传原始 payload（Sud 可能新增字段）。

## 参考

- gama 现网 Sud 集成（行为交叉验证用）：`/home/linuxea/project/qh/gama/Gamarepo` 的 `gama/game/servers/common/sud_opt.go`、`gama/callback/controllers/game/`
- 预留未封装：`KeyGetPlayerResults`、`KeyReportGameRoundBill`（文档接口存在，只有 key 常量无服务方法，README 特性一节有说明）
