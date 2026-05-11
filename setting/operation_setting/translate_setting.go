package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type TranslateSetting struct {
	TranslateAIEnabled bool   `json:"translate_ai_enabled"`  // 是否启用 AI 翻译
	TranslateAIModel   string `json:"translate_ai_model"`    // AI 模型，如 gpt-4o-mini
	TranslateAIBaseURL string `json:"translate_ai_base_url"` // API 基础地址
	TranslateAIApiKey  string `json:"translate_ai_api_key"`  // API Key
}

var translateSetting = TranslateSetting{
	TranslateAIEnabled: false,
	TranslateAIModel:   "gpt-4o-mini",
}

func init() {
	config.GlobalConfig.Register("translate_setting", &translateSetting)
}

func GetTranslateSetting() *TranslateSetting {
	return &translateSetting
}
