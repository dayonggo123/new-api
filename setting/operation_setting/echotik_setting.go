package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type EchotikSetting struct {
	EchotikEnabled bool   `json:"echotik_enabled"` // 是否启用 EchoTik 接口代理
	EchotikBaseURL string `json:"echotik_base_url"` // EchoTik API 基础地址，如 https://open.echotik.live
	EchotikUsername string `json:"echotik_username"` // EchoTik 用户名
	EchotikPassword string `json:"echotik_password"` // EchoTik 密码
}

var echotikSetting = EchotikSetting{
	EchotikEnabled: false,
	EchotikBaseURL: "https://open.echotik.live",
}

func init() {
	config.GlobalConfig.Register("echotik_setting", &echotikSetting)
}

func GetEchotikSetting() *EchotikSetting {
	return &echotikSetting
}
