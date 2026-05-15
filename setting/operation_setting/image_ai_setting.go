package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type ImageAISetting struct {
	ImageAIEnabled bool   `json:"image_ai_enabled"`  // 是否启用 AI 自动生成图片
	ImageAIModel   string `json:"image_ai_model"`    // AI 模型，如 dall-e-3
	ImageAIBaseURL string `json:"image_ai_base_url"` // API 基础地址
	ImageAIApiKey  string `json:"image_ai_api_key"`  // API Key
	ImageAISize    string `json:"image_ai_size"`     // 默认尺寸，如 1024x1024
	ImageAIN       int    `json:"image_ai_n"`        // 默认生成数量
}

var imageAISetting = ImageAISetting{
	ImageAIEnabled: false,
	ImageAIModel:   "dall-e-3",
	ImageAISize:    "1024x1024",
	ImageAIN:       1,
}

func init() {
	config.GlobalConfig.Register("image_ai_setting", &imageAISetting)
}

func GetImageAISetting() *ImageAISetting {
	return &imageAISetting
}
