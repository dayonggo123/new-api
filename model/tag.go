package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Tag 标签定义表
type Tag struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"type:varchar(50);not null;uniqueIndex"`
	Color       string `json:"color" gorm:"type:varchar(20);default:'blue'"`
	Category    string `json:"category" gorm:"type:varchar(50);default:'manual'"` // manual / auto
	Description string `json:"description" gorm:"type:varchar(255)"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func (t *Tag) TableName() string {
	return "tags"
}

// UserTag 用户-标签关联表
type UserTag struct {
	UserId      int    `json:"user_id" gorm:"primaryKey;autoIncrement:false"`
	TagId       int    `json:"tag_id" gorm:"primaryKey;autoIncrement:false"`
	Source      string `json:"source" gorm:"type:varchar(50);default:'manual'"` // manual / auto
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
}

func (ut *UserTag) TableName() string {
	return "user_tags"
}

// ==================== Tag CRUD ====================

// GetAllTags 获取所有标签
func GetAllTags() ([]*Tag, error) {
	var tags []*Tag
	err := DB.Order("created_time DESC").Find(&tags).Error
	return tags, err
}

// GetTagById 根据 ID 获取标签
func GetTagById(id int) (*Tag, error) {
	var tag Tag
	err := DB.First(&tag, id).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// GetTagByName 根据名称获取标签
func GetTagByName(name string) (*Tag, error) {
	var tag Tag
	err := DB.Where("name = ?", name).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// CreateTag 创建标签
func CreateTag(tag *Tag) error {
	if tag.CreatedTime == 0 {
		tag.CreatedTime = time.Now().Unix()
	}
	return DB.Create(tag).Error
}

// DeleteTag 删除标签（同时删除用户关联）
func DeleteTag(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&UserTag{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Tag{}, id).Error
	})
}

// ==================== UserTag CRUD ====================

// GetUserTags 获取用户的所有标签
func GetUserTags(userId int) ([]*Tag, error) {
	var tags []*Tag
	err := DB.Model(&Tag{}).
		Joins("JOIN user_tags ON user_tags.tag_id = tags.id").
		Where("user_tags.user_id = ?", userId).
		Find(&tags).Error
	return tags, err
}

// GetUserTagIds 获取用户的标签 ID 列表
func GetUserTagIds(userId int) ([]int, error) {
	var tagIds []int
	err := DB.Model(&UserTag{}).Where("user_id = ?", userId).Pluck("tag_id", &tagIds).Error
	return tagIds, err
}

// AddUserTag 给用户添加标签
func AddUserTag(userId int, tagId int, source string) error {
	if userId == 0 || tagId == 0 {
		return errors.New("user_id and tag_id cannot be empty")
	}
	if source == "" {
		source = "manual"
	}
	ut := &UserTag{
		UserId:      userId,
		TagId:       tagId,
		Source:      source,
		CreatedTime: time.Now().Unix(),
	}
	return DB.Create(ut).Error
}

// RemoveUserTag 移除用户的某个标签
func RemoveUserTag(userId int, tagId int) error {
	return DB.Where("user_id = ? AND tag_id = ?", userId, tagId).Delete(&UserTag{}).Error
}

// SetUserTags 设置用户的标签（全量替换）
func SetUserTags(userId int, tagIds []int, source string) error {
	if userId == 0 {
		return errors.New("user_id cannot be empty")
	}
	if source == "" {
		source = "manual"
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userId).Delete(&UserTag{}).Error; err != nil {
			return err
		}
		if len(tagIds) == 0 {
			return nil
		}
		now := time.Now().Unix()
		userTags := make([]*UserTag, len(tagIds))
		for i, tagId := range tagIds {
			userTags[i] = &UserTag{
				UserId:      userId,
				TagId:       tagId,
				Source:      source,
				CreatedTime: now,
			}
		}
		return tx.CreateInBatches(userTags, 100).Error
	})
}

// GetUsersByTagId 根据标签获取用户 ID 列表
func GetUsersByTagId(tagId int) ([]int, error) {
	var userIds []int
	err := DB.Model(&UserTag{}).Where("tag_id = ?", tagId).Pluck("user_id", &userIds).Error
	return userIds, err
}

// GetUsersByTagIds 根据多个标签获取用户 ID 列表（并集）
func GetUsersByTagIds(tagIds []int) ([]int, error) {
	if len(tagIds) == 0 {
		return []int{}, nil
	}
	var userIds []int
	err := DB.Model(&UserTag{}).Where("tag_id IN ?", tagIds).Distinct().Pluck("user_id", &userIds).Error
	return userIds, err
}

// GetTagUserCount 获取标签下的用户数
func GetTagUserCount(tagId int) (int64, error) {
	var count int64
	err := DB.Model(&UserTag{}).Where("tag_id = ?", tagId).Count(&count).Error
	return count, err
}
