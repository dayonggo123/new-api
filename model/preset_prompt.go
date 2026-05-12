package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// PresetPromptI18n 多语言翻译项
type PresetPromptI18n struct {
	Name         string `json:"name,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
	UserPrompt   string `json:"user_prompt,omitempty"`
	Description  string `json:"description,omitempty"`
	Category     string `json:"category,omitempty"`
}

type PresetPrompt struct {
	Id           int            `json:"id"`
	Name         string         `json:"name" gorm:"index"`
	SystemPrompt string         `json:"system_prompt" gorm:"type:text"`
	UserPrompt   string         `json:"user_prompt" gorm:"type:text"`
	Description  string         `json:"description"`
	I18n         string         `json:"i18n,omitempty" gorm:"type:text"` // JSON: {"en": {"name": "..."}, "fr": {...}}
	Category     string         `json:"category"`
	Status       int            `json:"status" gorm:"default:1"` // 1=enabled, 2=disabled
	SortOrder    int            `json:"sort_order" gorm:"default:0"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// ApplyLanguage 根据语言代码替换字段内容（缺失则保持默认中文）
func (p *PresetPrompt) ApplyLanguage(lang string) {
	if lang == "" || lang == "zh" || lang == "zh-CN" || lang == "zh-TW" {
		return
	}
	if p.I18n == "" {
		return
	}
	var i18nMap map[string]PresetPromptI18n
	if err := common.Unmarshal([]byte(p.I18n), &i18nMap); err != nil {
		return
	}
	if t, ok := i18nMap[lang]; ok {
		if t.Name != "" {
			p.Name = t.Name
		}
		if t.SystemPrompt != "" {
			p.SystemPrompt = t.SystemPrompt
		}
		if t.UserPrompt != "" {
			p.UserPrompt = t.UserPrompt
		}
		if t.Description != "" {
			p.Description = t.Description
		}
		if t.Category != "" {
			p.Category = t.Category
		}
	}
}

func (p *PresetPrompt) BeforeCreate(tx *gorm.DB) (err error) {
	now := time.Now().Unix()
	if p.CreatedTime == 0 {
		p.CreatedTime = now
	}
	if p.UpdatedTime == 0 {
		p.UpdatedTime = now
	}
	return
}

func (p *PresetPrompt) BeforeUpdate(tx *gorm.DB) (err error) {
	p.UpdatedTime = time.Now().Unix()
	return
}

func (p *PresetPrompt) Insert() error {
	return DB.Create(p).Error
}

func (p *PresetPrompt) Update() error {
	return DB.Model(p).Updates(p).Error
}

func (p *PresetPrompt) Delete() error {
	return DB.Delete(p).Error
}

func GetPresetPromptById(id int) (*PresetPrompt, error) {
	prompt := &PresetPrompt{Id: id}
	err := DB.First(prompt, "id = ?", id).Error
	return prompt, err
}

func GetAllPresetPrompts(startIdx int, num int) ([]*PresetPrompt, int64, error) {
	var prompts []*PresetPrompt
	var total int64

	err := DB.Model(&PresetPrompt{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = DB.Order("sort_order desc, id desc").Limit(num).Offset(startIdx).Find(&prompts).Error
	return prompts, total, err
}

func SearchPresetPrompts(keyword string, category string, status int, startIdx int, num int) ([]*PresetPrompt, int64, error) {
	var prompts []*PresetPrompt
	var total int64

	tx := DB.Model(&PresetPrompt{})
	if keyword != "" {
		tx = tx.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if category != "" {
		tx = tx.Where("category = ?", category)
	}
	if status > 0 {
		tx = tx.Where("status = ?", status)
	}

	err := tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = tx.Order("sort_order desc, id desc").Limit(num).Offset(startIdx).Find(&prompts).Error
	return prompts, total, err
}

func GetEnabledPresetPrompts() ([]*PresetPrompt, error) {
	var prompts []*PresetPrompt
	err := DB.Where("status = ?", 1).Order("sort_order desc, id desc").Find(&prompts).Error
	return prompts, err
}

func GetPresetPromptCategories() ([]string, error) {
	var categories []string
	err := DB.Model(&PresetPrompt{}).Where("category != ?", "").Distinct("category").Pluck("category", &categories).Error
	return categories, err
}

func (p *PresetPrompt) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.SystemPrompt == "" && p.UserPrompt == "" {
		return errors.New("system_prompt or user_prompt is required")
	}
	return nil
}
