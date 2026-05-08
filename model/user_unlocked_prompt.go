package model

import (
	"time"

	"gorm.io/gorm"
)

// UserUnlockedPrompt 用户已解锁提示词表
type UserUnlockedPrompt struct {
	UserId     int   `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	PromptId   int   `json:"prompt_id" gorm:"primaryKey;autoIncrement:false"`
	Cost       int   `json:"cost" gorm:"default:0"`
	UnlockedAt int64 `json:"unlocked_at" gorm:"bigint"`
}

func (uup *UserUnlockedPrompt) TableName() string {
	return "user_unlocked_prompts"
}

// IsPromptUnlocked 检查用户是否已解锁某提示词
func IsPromptUnlocked(userId int, promptId int) (bool, error) {
	var count int64
	err := DB.Model(&UserUnlockedPrompt{}).
		Where("user_id = ? AND prompt_id = ?", userId, promptId).
		Count(&count).Error
	return count > 0, err
}

// GetUnlockedPromptByUserId 分页获取用户已解锁的提示词ID列表
func GetUnlockedPromptByUserId(userId int, startIdx int, pageSize int) ([]*UserUnlockedPrompt, int64, error) {
	var unlocked []*UserUnlockedPrompt
	var total int64
	err := DB.Model(&UserUnlockedPrompt{}).Where("user_id = ?", userId).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DB.Where("user_id = ?", userId).Order("unlocked_at DESC").Limit(pageSize).Offset(startIdx).Find(&unlocked).Error
	return unlocked, total, err
}

// UnlockPrompt 解锁提示词（需外部事务控制）
func UnlockPrompt(tx *gorm.DB, userId int, promptId int, cost int) error {
	record := &UserUnlockedPrompt{
		UserId:     userId,
		PromptId:   promptId,
		Cost:       cost,
		UnlockedAt: time.Now().Unix(),
	}
	return tx.Create(record).Error
}

// GetUnlockedPromptIdsByUserId 获取用户已解锁的提示词ID列表（用于批量查询）
func GetUnlockedPromptIdsByUserId(userId int) ([]int, error) {
	var unlocked []*UserUnlockedPrompt
	err := DB.Where("user_id = ?", userId).Find(&unlocked).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(unlocked))
	for i, u := range unlocked {
		ids[i] = u.PromptId
	}
	return ids, nil
}
