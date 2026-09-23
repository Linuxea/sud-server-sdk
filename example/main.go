// sud-server-sdk 最小接入示例。
//
// 运行：go run ./example
// 环境变量：SUD_APP_ID / SUD_APP_SECRET
//
// 本示例同时演示出站（Client）与入站（callback.Server）两端：
//  1. 启动回调服务（http://127.0.0.1:8080/sud/...），把各回调 URL 配置到 Sud 平台
//  2. 用 Client 拉取游戏列表、推 quick_start 事件
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sud "github.com/linuxea/sud-server-sdk"
	"github.com/linuxea/sud-server-sdk/api"
	"github.com/linuxea/sud-server-sdk/callback"
	"github.com/linuxea/sud-server-sdk/events"
)

func main() {
	appID := os.Getenv("SUD_APP_ID")
	appSecret := os.Getenv("SUD_APP_SECRET")
	if appID == "" || appSecret == "" {
		log.Fatal("请设置 SUD_APP_ID / SUD_APP_SECRET")
	}

	ctx := context.Background()

	// ---------- 入站：Sud 调用我方 ----------
	srv := callback.NewServer(callback.Callbacks{
		// 鉴权链（必实现）
		GetSSToken: func(ctx context.Context, req callback.GetSSTokenReq) (*callback.GetSSTokenResp, error) {
			uid, err := resolveByCode(req.Code) // 业务方实现：code -> uid
			if err != nil {
				return nil, &callback.CallbackError{RetCode: 1, RetMsg: "invalid code", SDKErrorCode: 1005}
			}
			return &callback.GetSSTokenResp{
				SSToken:    issueToken(uid),
				ExpireDate: time.Now().Add(6 * time.Hour).UnixMilli(),
				UserInfo:   &callback.UserInfo{UID: uid, NickName: "示例用户", AvatarURL: "https://icon.png", Gender: ""},
			}, nil
		},
		UpdateSSToken: func(ctx context.Context, req callback.UpdateSSTokenReq) (*callback.UpdateSSTokenResp, error) {
			uid, err := verifyToken(req.SSToken) // 验旧换新
			if err != nil {
				return nil, err
			}
			return &callback.UpdateSSTokenResp{SSToken: issueToken(uid), ExpireDate: time.Now().Add(6 * time.Hour).UnixMilli()}, nil
		},
		GetUserInfo: func(ctx context.Context, req callback.GetUserInfoReq) (*callback.UserInfo, error) {
			uid, err := verifyToken(req.SSToken)
			if err != nil {
				return nil, err
			}
			return &callback.UserInfo{UID: uid, NickName: "示例用户", AvatarURL: "https://icon.png", Gender: ""}, nil
		},

		// 对局上报（结算发奖等核心业务在这里）
		ReportGameInfo: func(ctx context.Context, req callback.ReportGameInfoReq) error {
			switch req.ReportType {
			case callback.ReportTypeGameStart:
				start, err := req.ParseGameStart()
				if err != nil {
					return err
				}
				log.Printf("对局开始 mg=%s room=%s round=%s key=%s", start.MGIDStr, start.RoomID, start.GameRoundID, start.ReportGameInfoKey)
			case callback.ReportTypeGameSettle:
				settle, err := req.ParseGameSettle()
				if err != nil {
					return err
				}
				log.Printf("对局结算 mg=%s round=%s 结果数=%d", settle.MGIDStr, settle.GameRoundID, len(settle.Results))
			}
			return nil
		},

		// 异步通知：按事件注册，或用 OnNotify 兜底
		NotifyCallbacks: callback.NotifyCallbacks{
			OnOrderChanged: func(ctx context.Context, data callback.OrderChangedModel) {
				log.Printf("订单变更 out=%s status=%s", data.OutOrderID, data.Status)
			},
		},
		OnNotify: func(ctx context.Context, event string, payload json.RawMessage) {
			log.Printf("收到通知 %s: %s", event, payload)
		},
	}, callback.WithLogger(log.Default()))
	// 生产环境建议开启验签：callback.WithSigner(client.Signer())

	http.Handle("/sud/", srv.Handler())
	go func() { log.Fatal(http.ListenAndServe(":8080", nil)) }()

	// ---------- 出站：我方调用 Sud ----------
	client := sud.New(appID, appSecret, sud.WithRetry(3, 100*time.Millisecond))
	defer client.Close()

	// 游戏列表（建议缓存）
	list, err := client.GameList.List(ctx, api.GameListReq{Platform: api.PlatformAndroid})
	if err != nil {
		log.Printf("获取游戏列表失败: %v", err)
	} else {
		for _, g := range list.MGInfoList {
			fmt.Printf("游戏 %s: %s\n", g.MGID, g.Name.Get("zh-CN"))
		}
	}

	// 一键开局（携带玩家列表）
	err = client.PushEvent.QuickStart(ctx, "mg_id_xxx", events.QuickStartReqData{
		RoomID: "room_1",
		UserInfos: []events.QuickStartUserInfo{
			{UserInfo: events.UserInfo{UID: "u1", NickName: "玩家1", AvatarURL: "https://icon.png", Gender: "male"}},
			{UserInfo: events.UserInfo{UID: "ai_1", NickName: "机器人", AvatarURL: "https://icon.png", Gender: "female", IsAI: 1}},
		},
		ReportGameInfoKey: "my-game-key-001", // 对账用透传 key
	})
	if err != nil {
		log.Printf("quick_start 失败: %v", err)
	}

	// 房间清理
	if err := client.PushEvent.RoomClear(ctx, "mg_id_xxx", "room_1"); err != nil {
		log.Printf("room_clear 失败: %v", err)
	}

	// 对账：按 key 查上报
	report, err := client.Report.QueryGameReport(ctx, api.QueryGameReportReq{ReportGameInfoKey: "my-game-key-001"})
	if err != nil {
		log.Printf("查询上报失败: %v", err)
	} else if report.GameSettle != nil {
		log.Printf("查到结算: round=%s", report.GameSettle.GameRoundID)
	}

	select {} // 保持回调服务运行
}

// ---- 以下为业务方自行实现的占位 ----

func resolveByCode(code string) (string, error) { return "u1", nil }

func issueToken(uid string) string { return "token-" + uid }

func verifyToken(token string) (string, error) { return "u1", nil }
