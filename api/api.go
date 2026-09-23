// Package api 封装 Sud 服务端出站 API 的各服务域。
//
// 各 Service 通过 Poster 接口与根包 Client 解耦（Client 实现该接口），
// 业务方通常不直接构造本包服务，而是使用 sud.Client 上的组合字段：
//
//	client.GameList.List(ctx, ...)
//	client.Report.QueryGameReport(ctx, ...)
package api

import "context"

// Poster 由根包 sud.Client 实现，服务域通过它发送已签名请求。
type Poster interface {
	// Post 按配置 key 解析真实 URL，发送 POST 并解出响应 data 到 resp。
	Post(ctx context.Context, urlKey string, req any, resp any) error
	// AppID / AppSecret 部分接口（如分页查询上报）需在 body 中携带凭证。
	AppID() string
	AppSecret() string
}

// URLKey API 地址 key，对应「获取服务端API配置」返回的字段名。
type URLKey string

// 全部出站 API 的地址 key。
const (
	KeyMGList              URLKey = "mg_list"
	KeyMGInfo              URLKey = "mg_info"
	KeyGetGameReportInfo   URLKey = "get_game_report_info"
	KeyGameReportInfoPage  URLKey = "get_game_report_info_page"
	KeyQueryGameReportInfo URLKey = "query_game_report_info"
	KeyGetPlayerResults    URLKey = "get_player_results"
	KeyReportGameRoundBill URLKey = "report_game_round_bill"
	KeyPushEvent           URLKey = "push_event"
	KeyCreateOrder         URLKey = "create_order"
	KeyBatchCreateOrder    URLKey = "batch_create_order"
	KeyQueryOrder          URLKey = "query_order"
	KeyQueryMatchBase      URLKey = "query_match_base"
	KeyQueryMatchRoundIds  URLKey = "query_match_round_ids"
	KeyQueryUserSettle     URLKey = "query_user_settle"

	KeyLLMCreateVoice       URLKey = "llm_create_voice"
	KeyLLMTrainVoice        URLKey = "llm_train_voice"
	KeyLLMGetVoice          URLKey = "llm_get_voice"
	KeyLLMCreateAICharacter URLKey = "llm_create_ai_character"
	KeyLLMGetAICharacter    URLKey = "llm_get_ai_character"
)
