package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type TikHubSetting struct {
	TikHubEnabled bool   `json:"tikhub_enabled"` // 是否启用 TikHub 接口代理
	TikHubBaseURL string `json:"tikhub_base_url"` // TikHub API 基础地址，如 https://api.tikhub.io
	TikHubAPIKey  string `json:"tikhub_api_key"`  // TikHub API Token
}

var tikHubSetting = TikHubSetting{
	TikHubEnabled: false,
	TikHubBaseURL: "https://api.tikhub.io",
	TikHubAPIKey:  "",
}

func init() {
	config.GlobalConfig.Register("tikhub_setting", &tikHubSetting)
}

func GetTikHubSetting() *TikHubSetting {
	return &tikHubSetting
}
