package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// SigninPointsSetting 积分签到配置
type SigninPointsSetting struct {
	Enabled          bool  `json:"enabled"`           // 是否启用积分签到
	BasePoints       int   `json:"base_points"`       // 基础签到积分
	ConsecutiveBonus []int `json:"consecutive_bonus"` // 连续签到额外奖励（按天数索引，从第1天开始）
}

var signinPointsSetting = SigninPointsSetting{
	Enabled:          true,
	BasePoints:       10,
	ConsecutiveBonus: []int{0, 1, 2, 2, 2, 5, 5}, // 第1天0, 第2天1, 第3天2, 第4-6天2, 第7天+ 5
}

func init() {
	config.GlobalConfig.Register("signin_points_setting", &signinPointsSetting)
}

// GetSigninPointsSetting 获取积分签到配置
func GetSigninPointsSetting() *SigninPointsSetting {
	return &signinPointsSetting
}

// IsSigninPointsEnabled 是否启用积分签到
func IsSigninPointsEnabled() bool {
	return signinPointsSetting.Enabled
}

// GetSigninBasePoints 获取基础签到积分
func GetSigninBasePoints() int {
	return signinPointsSetting.BasePoints
}

// GetConsecutiveBonus 获取连续签到奖励
// day 从 1 开始（第1天=1）
func GetConsecutiveBonus(day int) int {
	if day <= 0 {
		return 0
	}
	bonus := signinPointsSetting.ConsecutiveBonus
	if len(bonus) == 0 {
		return 0
	}
	if day > len(bonus) {
		return bonus[len(bonus)-1] // 超过最大天数，取最后一档
	}
	return bonus[day-1]
}
