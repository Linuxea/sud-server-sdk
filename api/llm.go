package api

import "context"

// LLMService 大模型服务域（音色与 AI 角色）。
type LLMService struct{ p Poster }

// NewLLMService 创建服务域。
func NewLLMService(p Poster) *LLMService { return &LLMService{p: p} }

// 音频格式常量。
const (
	AudioFormatWAV = "wav"
	AudioFormatMP3 = "mp3"
)

// 音色状态常量。
const (
	VoiceStatusCreated  int32 = 0
	VoiceStatusTraining int32 = 1
	VoiceStatusSuccess  int32 = 2
	VoiceStatusFailed   int32 = 3
)

// CreateVoiceReq 创建音色请求。AudioData 与 AudioFormat 同时有值时触发训练。
type CreateVoiceReq struct {
	OutVoiceID  string `json:"out_voice_id"`           // 客户自定义音色 id（64 字符以内）
	AudioData   string `json:"audio_data,omitempty"`   // base64
	AudioFormat string `json:"audio_format,omitempty"` // wav / mp3
}

// CreateVoiceResp 创建音色响应。
type CreateVoiceResp struct {
	VoiceID string `json:"voice_id"`
}

// TrainVoiceReq 训练音色请求。
type TrainVoiceReq struct {
	VoiceID     string `json:"voice_id"` // SUD 音色 id
	AudioData   string `json:"audio_data"`
	AudioFormat string `json:"audio_format"`
}

// GetVoiceReq 查询音色请求。VoiceID 与 OutVoiceID 不能同时为空，VoiceID 优先。
type GetVoiceReq struct {
	VoiceID    string `json:"voice_id,omitempty"`
	OutVoiceID string `json:"out_voice_id,omitempty"`
}

// GetVoiceResp 查询音色响应。
type GetVoiceResp struct {
	VoiceID       string `json:"voice_id"`
	OutVoiceID    string `json:"out_voice_id"`
	Status        int32  `json:"status"`                    // 见 VoiceStatus 常量
	DemoAudioText string `json:"demo_audio_text,omitempty"` // success 状态返回
	DemoAudioData string `json:"demo_audio_data,omitempty"` // base64，success 状态返回
}

// CreateAICharacterReq 创建 AI 角色请求（角色存在则更新）。
type CreateAICharacterReq struct {
	OutAIID             string `json:"out_ai_id"` // 客户自定义 AI 角色 id（64 字符以内）
	VoiceID             string `json:"voice_id"`
	Gender              string `json:"gender"`   // male / female
	Birthday            string `json:"birthday"` // yyyy-MM-dd
	BloodType           string `json:"blood_type"`
	MBTI                string `json:"mbti"`
	Personality         string `json:"personality"`
	LanguageStyle       string `json:"language_style"`
	LanguageDetailStyle string `json:"language_detail_style"`
}

// CreateAICharacterResp 创建 AI 角色响应。
type CreateAICharacterResp struct {
	AIID string `json:"ai_id"`
}

// GetAICharacterReq 查询 AI 角色请求。AIID 与 OutAIID 不能同时为空，AIID 优先。
type GetAICharacterReq struct {
	AIID    string `json:"ai_id,omitempty"`
	OutAIID string `json:"out_ai_id,omitempty"`
}

// AICharacter AI 角色信息。
type AICharacter struct {
	AIID                string `json:"ai_id"`
	OutAIID             string `json:"out_ai_id"`
	VoiceID             string `json:"voice_id"`
	Gender              string `json:"gender"`
	Birthday            string `json:"birthday"`
	BloodType           string `json:"blood_type"`
	MBTI                string `json:"mbti"`
	Personality         string `json:"personality"`
	LanguageStyle       string `json:"language_style"`
	LanguageDetailStyle string `json:"language_detail_style"`
}

// CreateVoice 创建音色。
func (s *LLMService) CreateVoice(ctx context.Context, req CreateVoiceReq) (*CreateVoiceResp, error) {
	var resp CreateVoiceResp
	if err := s.p.Post(ctx, string(KeyLLMCreateVoice), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// TrainVoice 训练音色。
func (s *LLMService) TrainVoice(ctx context.Context, req TrainVoiceReq) error {
	return s.p.Post(ctx, string(KeyLLMTrainVoice), req, nil)
}

// GetVoice 查询音色。
func (s *LLMService) GetVoice(ctx context.Context, req GetVoiceReq) (*GetVoiceResp, error) {
	var resp GetVoiceResp
	if err := s.p.Post(ctx, string(KeyLLMGetVoice), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateAICharacter 创建 AI 角色（存在则更新）。
func (s *LLMService) CreateAICharacter(ctx context.Context, req CreateAICharacterReq) (*CreateAICharacterResp, error) {
	var resp CreateAICharacterResp
	if err := s.p.Post(ctx, string(KeyLLMCreateAICharacter), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAICharacter 查询 AI 角色。
func (s *LLMService) GetAICharacter(ctx context.Context, req GetAICharacterReq) (*AICharacter, error) {
	var resp AICharacter
	if err := s.p.Post(ctx, string(KeyLLMGetAICharacter), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
