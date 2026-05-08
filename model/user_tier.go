package model

import (
	"errors"
	"time"
)

// UserTier 用户层级定义表
type UserTier struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	Level       int    `json:"level" gorm:"uniqueIndex;not null"` // 1-4
	Name        string `json:"name" gorm:"type:varchar(50);not null"`
	Color       string `json:"color" gorm:"type:varchar(20);default:'blue'"`
	Description string `json:"description" gorm:"type:varchar(255)"`
	MinPoints   int    `json:"min_points" gorm:"default:0"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func (ut *UserTier) TableName() string {
	return "user_tiers"
}

// GetAllTiers 获取所有层级（按 level 升序）
func GetAllTiers() ([]*UserTier, error) {
	var tiers []*UserTier
	err := DB.Order("level ASC").Find(&tiers).Error
	return tiers, err
}

// GetTierById 根据 ID 获取层级
func GetTierById(id int) (*UserTier, error) {
	var tier UserTier
	err := DB.First(&tier, id).Error
	if err != nil {
		return nil, err
	}
	return &tier, nil
}

// GetTierByLevel 根据 level 获取层级
func GetTierByLevel(level int) (*UserTier, error) {
	var tier UserTier
	err := DB.Where("level = ?", level).First(&tier).Error
	if err != nil {
		return nil, err
	}
	return &tier, nil
}

// CreateTier 创建层级
func CreateTier(tier *UserTier) error {
	if tier.CreatedTime == 0 {
		tier.CreatedTime = time.Now().Unix()
	}
	return DB.Create(tier).Error
}

// UpdateTier 更新层级
func UpdateTier(tier *UserTier) error {
	return DB.Save(tier).Error
}

// DeleteTier 删除层级
func DeleteTier(id int) error {
	return DB.Delete(&UserTier{}, id).Error
}

// GetUserTierLevel 获取用户当前层级
func GetUserTierLevel(userId int) (int, error) {
	var up UserPoints
	err := DB.Select("tier_level").First(&up, userId).Error
	if err != nil {
		return 1, err // 默认层级 1
	}
	if up.TierLevel == 0 {
		return 1, nil
	}
	return up.TierLevel, nil
}

// SetUserTierLevel 设置用户层级
func SetUserTierLevel(userId int, tierLevel int) error {
	if userId == 0 {
		return errors.New("user id is empty")
	}
	now := time.Now().Unix()
	return DB.Model(&UserPoints{}).Where("user_id = ?", userId).Updates(map[string]interface{}{
		"tier_level":  tierLevel,
		"updated_time": now,
	}).Error
}

// GetUsersByTierLevel 按层级获取用户列表（返回 user_id 列表）
func GetUsersByTierLevel(tierLevel int) ([]int, error) {
	var userIds []int
	err := DB.Model(&UserPoints{}).Where("tier_level = ?", tierLevel).Pluck("user_id", &userIds).Error
	return userIds, err
}

// GetUsersByTierLevels 按多个层级获取用户列表
func GetUsersByTierLevels(tierLevels []int) ([]int, error) {
	if len(tierLevels) == 0 {
		return []int{}, nil
	}
	var userIds []int
	err := DB.Model(&UserPoints{}).Where("tier_level IN ?", tierLevels).Pluck("user_id", &userIds).Error
	return userIds, err
}

// AutoUpdateUserTierByPoints 根据积分自动更新用户层级
func AutoUpdateUserTierByPoints(userId int) error {
	tiers, err := GetAllTiers()
	if err != nil {
		return err
	}
	up, err := GetUserPointsById(userId)
	if err != nil {
		return err
	}

	// 从最高层级向下匹配
	newLevel := 1
	for i := len(tiers) - 1; i >= 0; i-- {
		if up.TotalPoints >= tiers[i].MinPoints {
			newLevel = tiers[i].Level
			break
		}
	}

	if up.TierLevel != newLevel {
		return SetUserTierLevel(userId, newLevel)
	}
	return nil
}
