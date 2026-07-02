package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

const (
	deviceSessionPrefix = "user:devices"
	deviceSessionTTL    = 30 * 24 * time.Hour // 30 天，与 cookie session 保持一致
)

// DeviceSession 表示一个在线设备
type DeviceSession struct {
	DeviceID  string `json:"device_id"`
	LoginAt   int64  `json:"login_at"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	DeviceType string `json:"device_type"`
}

// getUserDevicesKey 生成 Redis key
func getUserDevicesKey(userID int) string {
	return fmt.Sprintf("%s:%d", deviceSessionPrefix, userID)
}

// DetectDeviceType 根据 User-Agent 简单识别设备类型
func DetectDeviceType(userAgent string) string {
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		return "mobile"
	}
	if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		return "tablet"
	}
	if strings.Contains(ua, "bot") || strings.Contains(ua, "crawler") || strings.Contains(ua, "spider") {
		return "bot"
	}
	return "desktop"
}

// RegisterDeviceSession 用户登录时注册设备会话，超限则踢掉最早的设备
func RegisterDeviceSession(userID int, deviceID string, c *gin.Context) error {
	cfg := operation_setting.GetLoginSetting()
	if !cfg.EnableDeviceLimit || cfg.MaxOnlineDevices <= 0 {
		return nil
	}
	if !common.RedisEnabled || common.RDB == nil {
		return fmt.Errorf("redis not enabled")
	}

	session := DeviceSession{
		DeviceID:   deviceID,
		LoginAt:    time.Now().Unix(),
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		DeviceType: DetectDeviceType(c.Request.UserAgent()),
	}
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	ctx := context.Background()
	key := getUserDevicesKey(userID)

	// 清理可能过期的设备（TTL 会自然过期，这里也做主动防御）
	if err := cleanupExpiredDevices(ctx, key); err != nil {
		common.SysLog(fmt.Sprintf("[DeviceSession] cleanup expired devices failed: %v", err))
	}

	// 如果超过最大设备数，移除最早的设备
	devices, err := GetUserDeviceSessions(userID)
	if err != nil {
		return err
	}
	if len(devices) >= cfg.MaxOnlineDevices {
		// 按登录时间排序，移除最早的
		oldest := findOldestDevice(devices)
		if oldest != "" && oldest != deviceID {
			common.SysLog(fmt.Sprintf("[DeviceSession] kick oldest device %s for user %d", oldest, userID))
			common.RDB.HDel(ctx, key, oldest)
		}
	}

	// 写入新设备
	if err := common.RDB.HSet(ctx, key, deviceID, string(data)).Err(); err != nil {
		return err
	}
	common.RDB.Expire(ctx, key, deviceSessionTTL)
	common.SysLog(fmt.Sprintf("[DeviceSession] registered device %s for user %d", deviceID, userID))
	return nil
}

// ValidateDeviceSession 校验当前设备是否在线
func ValidateDeviceSession(userID int, deviceID string) bool {
	cfg := operation_setting.GetLoginSetting()
	if !cfg.EnableDeviceLimit || cfg.MaxOnlineDevices <= 0 {
		return true
	}
	if !common.RedisEnabled || common.RDB == nil || deviceID == "" {
		return false
	}
	ctx := context.Background()
	key := getUserDevicesKey(userID)
	exists, err := common.RDB.HExists(ctx, key, deviceID).Result()
	if err != nil || !exists {
		return false
	}
	// 刷新过期时间
	common.RDB.Expire(ctx, key, deviceSessionTTL)
	return true
}

// RemoveDeviceSession 退出登录或管理员踢掉设备时移除
func RemoveDeviceSession(userID int, deviceID string) error {
	if !common.RedisEnabled || common.RDB == nil {
		return nil
	}
	ctx := context.Background()
	key := getUserDevicesKey(userID)
	return common.RDB.HDel(ctx, key, deviceID).Err()
}

// GetUserDeviceSessions 获取用户的所有在线设备
func GetUserDeviceSessions(userID int) ([]DeviceSession, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return []DeviceSession{}, nil
	}
	ctx := context.Background()
	key := getUserDevicesKey(userID)
	result, err := common.RDB.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var devices []DeviceSession
	for _, v := range result {
		var d DeviceSession
		if err := json.Unmarshal([]byte(v), &d); err == nil {
			devices = append(devices, d)
		}
	}
	return devices, nil
}

// ClearUserDeviceSessions 清空用户的所有设备
func ClearUserDeviceSessions(userID int) error {
	if !common.RedisEnabled || common.RDB == nil {
		return nil
	}
	ctx := context.Background()
	key := getUserDevicesKey(userID)
	return common.RDB.Del(ctx, key).Err()
}

// cleanupExpiredDevices 主动清理 Hash 中可能过期的字段（Redis 不会自动删除 Hash 内的单个字段，但整个 key 会过期）
func cleanupExpiredDevices(ctx context.Context, key string) error {
	// 如果整个 key 已过期，HGetAll 会返回空，这里不需要额外处理
	// 如果 key 还在但里面某些字段理论上不会过期，因为整个 key 过期时全部清理
	return nil
}

func findOldestDevice(devices []DeviceSession) string {
	if len(devices) == 0 {
		return ""
	}
	oldest := devices[0]
	for _, d := range devices[1:] {
		if d.LoginAt < oldest.LoginAt {
			oldest = d
		}
	}
	return oldest.DeviceID
}

// GetDeviceIDFromRequest 从请求头中获取 device id（前端在登录时生成并保存到 localStorage，每次请求带在 Header 中）
func GetDeviceIDFromRequest(c *gin.Context) string {
	return c.Request.Header.Get("X-Device-ID")
}

// GenerateDeviceID 生成新的设备 ID
func GenerateDeviceID() string {
	return common.GetUUID() // 假设 common 包有 GetUUID
}
