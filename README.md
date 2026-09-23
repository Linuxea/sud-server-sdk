# sud-server-sdk

Sud MGP（互动小游戏平台）服务端 Go SDK，封装 [Sud GitBook 文档](https://docs.sud.tech/zh-CN/app/Server/StartUp.html) 中的服务端接口：业务方调 Sud 的出站 API + 业务方提供给 Sud 调用的 HTTPS 回调。

## 特性

- **高覆盖**：出站 16 个 API 方法、push_event 20 种事件、入站 7 个回调 + 15 种异步 notify（文档中另有 2 个接口 get_player_results / report_game_round_bill 暂未封装，地址 key 已预留）
- **零第三方依赖**：主 module 仅用标准库（gin 适配器为独立 module）
- **Go 惯用设计**：泛型统一请求封装、functional options、struct 组合（服务域平铺在 Client 上，NotifyCallbacks 可嵌入）、标准 middleware 验签、`errors.As` 友好错误
- **全链路 context**、API 地址自动发现与缓存（含失败刷新回退）

## 安装

```
go get github.com/linuxea/sud-server-sdk
```

## 快速开始

### 出站（调 Sud）

```go
client := sud.New(appID, appSecret, sud.WithRetry(3, 100*time.Millisecond))
defer client.Close()

list, err := client.GameList.List(ctx, api.GameListReq{Platform: api.PlatformAndroid})
err = client.PushEvent.QuickStart(ctx, mgID, events.QuickStartReqData{RoomID: roomID, UserInfos: ...})
report, err := client.Report.QueryGameReport(ctx, api.QueryGameReportReq{ReportGameInfoKey: key})
```

### 入站（Sud 调我方）

```go
cbs := callback.Callbacks{
    GetSSToken:     func(ctx, req) (*callback.GetSSTokenResp, error) { ... },
    UpdateSSToken:  ...,
    GetUserInfo:    ...,
    ReportGameInfo: func(ctx, req) error { ... },  // req.ParseGameStart() / req.ParseGameSettle()
    OnNotify:       func(ctx, event, payload) { ... }, // notify 兜底
    NotifyCallbacks: callback.NotifyCallbacks{      // 或按事件精确注册
        OnOrderChanged: func(ctx, data) { ... },
    },
}
srv := callback.NewServer(cbs,
    callback.WithSigner(client.Signer()), // 可选验签
    callback.WithLogger(log.Default()),
)
http.Handle("/sud/", srv.Handler())
```

gin 框架接入（独立 module，不污染主依赖）：

```go
import ginsud "github.com/linuxea/sud-server-sdk/adapter/gin"
ginsud.Mount(router, "/sud", srv)
```

默认路由（前缀可用 `callback.WithPathPrefix` 修改，需在 Sud 平台配置对应 URL）：

| 路径 | 回调 |
|---|---|
| `POST {prefix}/get_sstoken` | Code 换 SSToken（返回可带 user_info 省一次调用） |
| `POST {prefix}/update_sstoken` | 刷新 SSToken |
| `POST {prefix}/get_user_info` | SSToken 换用户信息 |
| `POST {prefix}/report_game_info` | 对局上报（game_start / game_settle） |
| `POST {prefix}/get_account` | Betting 类游戏获取账户 |
| `POST {prefix}/get_score` | （已废弃）获取积分 |
| `POST {prefix}/update_score` | （已废弃）更新积分 |
| `POST {prefix}/notify` | 15 种异步通知统一入口 |

## 全接口清单

### 出站 API（Client 组合字段）

| 字段 | 接口 | 文档 |
|---|---|---|
| `client.GameList` | `List` 获取游戏列表V2 / `Info` 获取游戏信息V2 | ObtainTheGameListV2 / ObtainGameInformationV2 |
| `client.Report` | `QueryGameReport` 按局/key 查上报 / `QueryByPage` 分页查房间上报（自动注入凭证） | QueryGameReportInformation / ObtainTheReportInformationOfGameByPage |
| `client.Order` | `Create` / `BatchCreate` / `Query` | CreateOrder / BatchCreateOrder / QueryOrder |
| `client.EntryScore` | `QueryMatchBase` / `QueryMatchRoundIds` / `QueryUserSettle` | 带分入场三件套 |
| `client.LLM` | `CreateVoice` / `TrainVoice` / `GetVoice` / `CreateAICharacter` / `GetAICharacter` | 大模型五件套 |
| `client.PushEvent` | 20 种事件（见下）+ `Push` 自定义事件 | PushEventToMgServer |

### push_event 事件（`client.PushEvent.*`）

`QuickStart` `GameStart` `GameEnd` `RoomClear` `RoomInfo`(有响应) `UserIn` `UserInBatch` `UserOut` `UserReady` `UserKick` `CaptainChange` `GameSetting` `AiAdd`(有响应) `LLMAiAdd` `LLMAiExit` `RefreshUserItem`(有响应) `ModeExChange` `DrawImageClear` `UserToPlayer` `PlayerAsset`(有响应)

事件 data 结构体在 `events` 包：`events.QuickStartReqData` 等。

### 异步通知（`callback.NotifyCallbacks`）

`sud.mg.merchant.room.users.changed` `order.changed` `order.batch.changed` `bid.result` `user.settle` `match.start` `match.settle` `room.game.rule` `game.process` `player.status` `game.player.connect.status` `room.seat.changed` `player.draw.image` `llm.ai.create.result` `llm.game.situation.summary`

对应 `OnRoomUsersChanged` 等 15 个 func 字段，未注册的事件走 `Callbacks.OnNotify` 兜底。各事件需联系 Sud 开启。

## 签名与验签

- 出站：`Authorization: Sud-Auth app_id="..",timestamp="..",nonce="..",signature=".."`，签名为 `HmacSHA1(appSecret, "appID\nts\nnonce\nbody\n")`（注意结尾 `\n`），SDK 自动处理
- 入站：`callback.WithSigner(client.Signer())` 开启，校验 `Sud-AppId/Sud-Timestamp/Sud-Nonce/Sud-Signature` 四个头

## 错误处理

```go
var apiErr *sud.APIError
if errors.As(err, &apiErr) {
    log.Printf("ret_code=%d http=%d url=%s", apiErr.RetCode, apiErr.HTTPStatus, apiErr.URL)
}
```

回调侧返回错误：`&callback.CallbackError{RetCode: 1, RetMsg: "...", SDKErrorCode: 1005}`；update_score 专用 `callback.ErrInsufficientBalance` / `callback.ErrDuplicateOrderID`。

## 约定与坑（来自文档）

- API 实际地址需先调「获取服务端API配置」（`GET https://asc.sudden.ltd/{HmacMD5(app_id)}`），SDK 内部自动拉取与缓存（默认 24h 刷新、失败回退旧缓存重试）
- 上报查询仅支持一个月内数据；列表/信息接口限频 10 次/秒，建议业务侧缓存
- `quick_start` 不要把机器人放 user_infos 第一位
- `report_game_info_key`（≤64 字节）是贯穿 push_event → 回调 → 对账查询的透传主键
- 回调 body 解析失败时 SDK 记日志并返回成功壳（避免 Sud 无意义重试），与 gama 现网行为一致

## 模块

| 路径 | 说明 |
|---|---|
| 根包 `sud` | Client、Signer（签名/验签）、Config、API 地址缓存 |
| `api/` | 出站服务域（GameList/Report/Order/EntryScore/LLM/PushEvent） |
| `events/` | push_event 20 种事件 data 结构 |
| `callback/` | HTTPS 回调服务（net/http，标准 middleware） |
| `adapter/gin/` | gin 适配器（**独立 go.mod**，require gin v1.7.2） |
| `example/` | 最小接入示例（`go run ./example`） |

## 开发

```
go build ./... && go vet ./... && go test ./...
```

adapter 模块单独验证：`cd adapter/gin && go build ./... && go test ./...`

### 本地 workspace（克隆后执行一次）

仓库为双 module 结构（根 + `adapter/gin`），本地开发需生成 go.work（已 gitignore）：

```
go work init . ./adapter/gin
go work edit -replace github.com/linuxea/sud-server-sdk@v0.0.0=./
```

生成的 go.work 形如：

```
go 1.27.1

use (
	.
	./adapter/gin
)

replace github.com/linuxea/sud-server-sdk v0.0.0 => ./
```

> 为什么需要 replace：仅靠 `use` 对"require 一个从未发布的 v0.0.0 占位版本"在
> module graph 加载时覆盖不彻底（golang/go#50750 一族已知问题），必须配合
> **带版本限定**的 replace；不带版本会与 use 冲突（报 "replaced at all versions"）。
> 根 module 打 tag 后把 adapter 的 require 改为真实版本，此 replace 自动失效。

### 发布

1. 根 module 打 tag（如 `git tag v1.0.0 && git push --tags`）
2. `adapter/gin/go.mod` 中 `sud-server-sdk v0.0.0` 改为 `v1.0.0`
3. 删除本地 `go.work` 后在 `adapter/gin` 下验证 `go build ./...` 可解析
