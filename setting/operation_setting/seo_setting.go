package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type SEOSetting struct {
	SeoAIEnabled bool   `json:"seo_ai_enabled"`  // 是否启用 AI 自动生成 SEO
	SeoAIModel   string `json:"seo_ai_model"`    // AI 模型，如 gpt-4o-mini
	SeoAIBaseURL string `json:"seo_ai_base_url"` // API 基础地址
	SeoAIApiKey  string `json:"seo_ai_api_key"`  // API Key
}

var seoSetting = SEOSetting{
	SeoAIEnabled: false,
	SeoAIModel:   "gpt-4o-mini",
}

func init() {
	config.GlobalConfig.Register("seo_setting", &seoSetting)
}

func GetSEOSetting() *SEOSetting {
	return &seoSetting
}
