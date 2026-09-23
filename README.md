# sud-server-sdk

Sud MGP（互动小游戏平台）服务端 Go SDK，封装 GitBook 文档中全部服务端接口。

## 特性

- 出站 API：API 地址配置、游戏列表/信息、上报查询（单个/分页）、push_event 推送（19 种事件）、游戏内付费订单、带分入场（德州/TeenPatti）、LLM AI 角色/语音
- 入站回调：get_sstoken / update_sstoken / get_user_info / report_game_info / get_account / game 系列 / 16 种异步 notify
- 出站 Sud-Auth 签名、入站回调验签（HmacSHA1）
- 零第三方依赖（gin 适配器为独立 module）
- 全链路 context、泛型统一请求封装

## 快速开始

（接入指南与全接口清单在阶段 9 补全）

## 模块

| 目录 | 说明 |
|---|---|
| 根包 `sud` | Client、签名、配置、API 地址缓存 |
| `events/` | push_event 19 种事件数据结构 |
| `api/` | 出站 API 服务域 |
| `callback/` | HTTPS 回调服务（net/http） |
| `adapter/gin/` | gin 框架适配器（独立 module） |
