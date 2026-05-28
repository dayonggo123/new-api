package types

import "strings"

// ToExternalCode 将内部 ErrorCode 映射为对外标准格式。
// 将冒号分隔符替换为下划线，如 channel:no_available_key -> channel_no_available_key。
func (code ErrorCode) ToExternalCode() string {
	s := string(code)
	// 将冒号分隔符替换为下划线
	s = strings.ReplaceAll(s, ":", "_")
	return s
}
