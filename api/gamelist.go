package api

import "context"

// GameListService 游戏列表/游戏信息服务域。
type GameListService struct{ p Poster }

// NewGameListService 创建服务域。
func NewGameListService(p Poster) *GameListService { return &GameListService{p: p} }

// 客户端平台常量。
const (
	PlatformIOS     = 1
	PlatformAndroid = 2
	PlatformWeb     = 3
)

// GameListReq 获取游戏列表请求。
type GameListReq struct {
	// Platform 客户端平台，默认 1
	Platform int `json:"platform,omitempty"`
	// UnityEditorVersion unity 引擎版本，默认 2020.3.25f1c1
	UnityEditorVersion string `json:"unity_engine_version,omitempty"`
}

// GameListResp 获取游戏列表响应。
type GameListResp struct {
	MGInfoList []MGInfo `json:"mg_info_list"`
}

// GameInfoReq 获取单个游戏信息请求。
type GameInfoReq struct {
	MGID               string `json:"mg_id"`
	Platform           int    `json:"platform,omitempty"`
	UnityEditorVersion string `json:"unity_engine_version,omitempty"`
}

// GameInfoResp 获取单个游戏信息响应。
type GameInfoResp struct {
	MGInfo MGInfo `json:"mg_info"`
}

// MGInfo 小游戏元信息。
type MGInfo struct {
	MGID             string     `json:"mg_id"`
	Name             I18nText   `json:"name"`
	Desc             I18nText   `json:"desc"`
	Thumbnail332x332 I18nText   `json:"thumbnail332x332"`
	Thumbnail192x192 I18nText   `json:"thumbnail192x192"`
	Thumbnail128x128 I18nText   `json:"thumbnail128x128"`
	Thumbnail80x80   I18nText   `json:"thumbnail80x80"`
	BigLoadingPic    I18nText   `json:"big_loading_pic"`
	GameModeList     []GameMode `json:"game_mode_list"`
}

// I18nText 多语言文本，key 为语言码（default/en-US/zh-CN 等）。
type I18nText map[string]string

// Get 取指定语言文本，缺失时回退 default，再回退空串。
func (t I18nText) Get(lang string) string {
	if v, ok := t[lang]; ok && v != "" {
		return v
	}
	return t["default"]
}

// GameMode 游戏模式。
type GameMode struct {
	Mode            int    `json:"mode"`
	Count           []int  `json:"count"`             // [0]:最小人数 [1]:最大人数
	TeamCount       []int  `json:"team_count"`        // [0]:最小队伍数 [1]:最大队伍数
	TeamMemberCount []int  `json:"team_member_count"` // [0]:每队最小人数 [1]:每队最大人数
	Rule            string `json:"rule"`              // 游戏规则（JSON 字符串）
}

// List 获取当前 app 关联的小游戏列表（建议业务侧缓存，限频 10次/秒）。
func (s *GameListService) List(ctx context.Context, req GameListReq) (*GameListResp, error) {
	var resp GameListResp
	if err := s.p.Post(ctx, string(KeyMGList), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Info 按 mg_id 获取小游戏信息（建议业务侧缓存，限频 10次/秒）。
func (s *GameListService) Info(ctx context.Context, req GameInfoReq) (*MGInfo, error) {
	var resp GameInfoResp
	if err := s.p.Post(ctx, string(KeyMGInfo), req, &resp); err != nil {
		return nil, err
	}
	return &resp.MGInfo, nil
}
