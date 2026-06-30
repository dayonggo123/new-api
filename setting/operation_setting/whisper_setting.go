package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type WhisperSetting struct {
	WhisperEnabled bool   `json:"whisper_enabled"` // 是否启用 Whisper 接口代理
	WhisperBaseURL string `json:"whisper_base_url"` // Whisper API 基础地址，如 https://api.apimart.ai
	WhisperApiKey  string `json:"whisper_api_key"`  // Whisper API Key
}

var whisperSetting = WhisperSetting{
	WhisperEnabled: false,
	WhisperBaseURL: "https://api.apimart.ai",
}

func init() {
	config.GlobalConfig.Register("whisper_setting", &whisperSetting)
}

func GetWhisperSetting() *WhisperSetting {
	return &whisperSetting
}
