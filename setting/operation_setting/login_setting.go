package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type LoginSetting struct {
	EnableDeviceLimit bool `json:"login_enable_device_limit"` // 是否启用设备数量限制
	MaxOnlineDevices  int  `json:"login_max_online_devices"`  // 最大同时在线设备数
}

var loginSetting = LoginSetting{
	EnableDeviceLimit: false,
	MaxOnlineDevices:  3,
}

func init() {
	config.GlobalConfig.Register("login_setting", &loginSetting)
}

func GetLoginSetting() *LoginSetting {
	return &loginSetting
}
