package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// ChannelHealthSetting 渠道健康与自动熔断配置
type ChannelHealthSetting struct {
	// AutoDisableConsecutiveFails 连续失败多少次后临时禁用该渠道+模型
	// 0 表示关闭自动禁用功能
	AutoDisableConsecutiveFails int `json:"auto_disable_consecutive_fails"`
	// AutoEnableMinutes 临时禁用后多少分钟自动恢复
	AutoEnableMinutes int `json:"auto_enable_minutes"`
}

var channelHealthSetting = ChannelHealthSetting{
	AutoDisableConsecutiveFails: 0, // 默认关闭
	AutoEnableMinutes:           10,
}

func init() {
	config.GlobalConfig.Register("channel_health_setting", &channelHealthSetting)
}

func GetChannelHealthSetting() *ChannelHealthSetting {
	return &channelHealthSetting
}
