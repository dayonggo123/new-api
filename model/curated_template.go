package model

import (
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/dto"
)

// CuratedTemplate 一键同款精选模板表
type CuratedTemplate struct {
	Id              int             `json:"-" gorm:"primaryKey;autoIncrement"`
	TemplateId      string          `json:"id" gorm:"column:template_id;size:64;uniqueIndex"`
	Title           string          `json:"title" gorm:"not null"`
	Category        string          `json:"category" gorm:"size:64;index"`
	CoverImageUrl   string          `json:"coverImageUrl" gorm:"column:cover_image_url"`
	PreviewMediaUrl string          `json:"previewMediaUrl" gorm:"column:preview_media_url"`
	Description     string          `json:"description" gorm:"type:text"`
	Prompt          string          `json:"prompt" gorm:"type:text"`
	InputSlots      json.RawMessage `json:"inputSlots" gorm:"column:input_slots;type:json"`
	Params          json.RawMessage `json:"params" gorm:"column:params;type:json"`
	ExecutionPlan   json.RawMessage `json:"executionPlan,omitempty" gorm:"column:execution_plan;type:json"`
	EstimatedPrice  float64         `json:"estimatedPrice" gorm:"column:estimated_price;type:decimal(10,6);default:0"`
	SortOrder       int             `json:"sortOrder" gorm:"column:sort_order;default:0;index"`
	Enabled         bool            `json:"enabled" gorm:"default:true"`
	HotScore        int             `json:"hotScore" gorm:"column:hot_score;default:0"`
	CreatedAt       int64           `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       int64           `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

func (CuratedTemplate) TableName() string {
	return "curated_templates"
}

// Insert 创建模板
func (template *CuratedTemplate) Insert() error {
	return DB.Create(template).Error
}

// Update 保存模板（全字段更新）
func (template *CuratedTemplate) Update() error {
	return DB.Save(template).Error
}

// Delete 删除模板
func (template *CuratedTemplate) Delete() error {
	return DB.Delete(template).Error
}

// GetCuratedTemplateById 根据自增 ID 获取模板
func GetCuratedTemplateById(id int) (*CuratedTemplate, error) {
	var template CuratedTemplate
	err := DB.First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetCuratedTemplateByTemplateId 根据业务模板 ID 获取模板
func GetCuratedTemplateByTemplateId(templateId string) (*CuratedTemplate, error) {
	var template CuratedTemplate
	err := DB.Where("template_id = ?", templateId).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetCuratedTemplates 获取已启用模板列表（公开接口）
func GetCuratedTemplates(query *dto.CuratedTemplateListQuery, page, pageSize int) ([]*CuratedTemplate, int64, error) {
	var templates []*CuratedTemplate
	var total int64

	db := DB.Model(&CuratedTemplate{}).Where("enabled = ?", true)
	if query.Category != "" && query.Category != "all" {
		db = db.Where("category = ?", query.Category)
	}
	if query.Keyword != "" {
		keyword := "%" + strings.TrimSpace(query.Keyword) + "%"
		db = db.Where("title LIKE ? OR description LIKE ? OR prompt LIKE ?", keyword, keyword, keyword)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	switch query.SortBy {
	case "newest":
		db = db.Order("created_at DESC")
	case "price_asc":
		db = db.Order("estimated_price ASC")
	case "price_desc":
		db = db.Order("estimated_price DESC")
	default:
		db = db.Order("hot_score DESC, sort_order ASC, id DESC")
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

// AdminListCuratedTemplates 管理后台获取全部模板（含启用/禁用）
func AdminListCuratedTemplates(page, pageSize int) ([]*CuratedTemplate, int64, error) {
	var templates []*CuratedTemplate
	var total int64

	db := DB.Model(&CuratedTemplate{}).Order("sort_order ASC, id DESC")
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

// AdminUpdateStatus 更新模板启用状态
func AdminUpdateStatus(id int, enabled bool) error {
	return DB.Model(&CuratedTemplate{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// CuratedTemplateExistsByCategory 检查是否存在使用该分类的模板
func CuratedTemplateExistsByCategory(category string) (bool, error) {
	var count int64
	err := DB.Model(&CuratedTemplate{}).Where("category = ?", category).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
