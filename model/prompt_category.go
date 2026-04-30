package model

import (
	"errors"

	"gorm.io/gorm"
)

type PromptCategory struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"index"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	Status      int            `json:"status" gorm:"default:1"` // 1=enabled, 2=disabled
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func GetAllPromptCategories(startIdx int, num int) (categories []*PromptCategory, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&PromptCategory{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Order("sort_order asc, id desc").Limit(num).Offset(startIdx).Find(&categories).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func GetEnabledPromptCategories() (categories []*PromptCategory, err error) {
	err = DB.Where("status = ?", 1).Order("sort_order asc, id desc").Find(&categories).Error
	return categories, err
}

func GetPromptCategoryById(id int) (*PromptCategory, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	category := PromptCategory{Id: id}
	err := DB.First(&category, "id = ?", id).Error
	return &category, err
}

func (category *PromptCategory) Insert() error {
	return DB.Create(category).Error
}

func (category *PromptCategory) Update() error {
	return DB.Model(category).Select("name", "description", "icon", "sort_order", "status").Updates(category).Error
}

func (category *PromptCategory) Delete() error {
	return DB.Delete(category).Error
}

func DeletePromptCategoryById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	category := PromptCategory{Id: id}
	err := DB.Where(category).First(&category).Error
	if err != nil {
		return err
	}
	return category.Delete()
}
