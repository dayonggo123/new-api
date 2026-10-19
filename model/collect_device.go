package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// CollectDevice 桌面端设备令牌。
// 安全约束：TokenHash 只存哈希（sha256(token)），任何接口不得返回明文令牌（生成接口除外，仅一次）。
// token 为 c_ + 32 位随机 hex（128bit 熵），无需加盐即可抗预计算。
type CollectDevice struct {
	Id         int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId     int    `json:"user_id" gorm:"index:idx_collect_device_user;not null"`
	TokenHash  string `json:"-" gorm:"type:varchar(64);uniqueIndex:uk_collect_device_token_hash;not null"`
	TokenHint  string `json:"token_hint" gorm:"type:varchar(8);not null;default:''"`
	Name       string `json:"name" gorm:"type:varchar(64);not null;default:''"`
	LastSeen   int64  `json:"last_seen" gorm:"not null;default:0"`
	Revoked    bool   `json:"revoked" gorm:"not null;default:false"`
	CreateTime int64  `json:"create_time" gorm:"bigint;not null"`
}

func (CollectDevice) TableName() string {
	return "collect_devices"
}

// GenerateCollectDeviceToken 生成 c_ + 32 位 hex 的设备令牌
func GenerateCollectDeviceToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "c_" + hex.EncodeToString(buf), nil
}

func HashCollectToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// CreateCollectDevice 生成设备并返回明文令牌（仅此一次可见）
func CreateCollectDevice(userId int, name string) (*CollectDevice, string, error) {
	token, err := GenerateCollectDeviceToken()
	if err != nil {
		return nil, "", err
	}
	device := &CollectDevice{
		UserId:     userId,
		TokenHash:  HashCollectToken(token),
		TokenHint:  token[len(token)-4:],
		Name:       name,
		CreateTime: time.Now().Unix(),
	}
	if err := DB.Create(device).Error; err != nil {
		return nil, "", err
	}
	return device, token, nil
}

// GetCollectDeviceByTokenHash 按哈希查找设备（令牌校验用）
func GetCollectDeviceByTokenHash(tokenHash string) (*CollectDevice, error) {
	device := &CollectDevice{}
	err := DB.Where("token_hash = ?", tokenHash).First(device).Error
	if err != nil {
		return nil, err
	}
	return device, nil
}

func GetCollectDevicesByUser(userId int) ([]*CollectDevice, error) {
	devices := make([]*CollectDevice, 0)
	err := DB.Where("user_id = ?", userId).Order("id DESC").Find(&devices).Error
	return devices, err
}

// RevokeCollectDevice 吊销设备（仅本人），幂等
func RevokeCollectDevice(id int64, userId int) error {
	res := DB.Model(&CollectDevice{}).
		Where("id = ? AND user_id = ?", id, userId).
		Update("revoked", true)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}

// TouchCollectDeviceLastSeen 更新活跃时间，best-effort，失败不影响主流程
func TouchCollectDeviceLastSeen(id int64, now int64) {
	DB.Model(&CollectDevice{}).Where("id = ?", id).Update("last_seen", now)
}
