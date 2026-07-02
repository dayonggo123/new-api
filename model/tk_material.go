package model

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// TKMaterial represents an image material in the TikTok/e-commerce scene library.
type TKMaterial struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	Category     string `json:"category" gorm:"column:category;size:64;index"`
	URL          string `json:"url" gorm:"column:url;type:text"`
	ThumbnailURL string `json:"thumbnail_url" gorm:"column:thumbnail_url;type:text"`
	Filename     string `json:"filename" gorm:"column:filename;size:255"`
	FileType     string `json:"file_type" gorm:"column:file_type;size:64"`
	Size         int64  `json:"size" gorm:"column:size"`
	Width        int    `json:"width" gorm:"column:width"`
	Height       int    `json:"height" gorm:"column:height"`
	Source       string `json:"source" gorm:"column:source;size:32;index"` // upload / notion
	NotionPageID string `json:"notion_page_id" gorm:"column:notion_page_id;size:128;index"`
	Status       int    `json:"status" gorm:"column:status;default:1;index"`
	CreatedAt    int64  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    int64  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (TKMaterial) TableName() string {
	return "tk_materials"
}

// TKMaterialCategories returns all supported scene categories (including sub-categories).
func TKMaterialCategories() []string {
	return []string{
		"浴室", "客厅", "厨房", "卧室", "车库", "院子",
		"街景", "健身房", "车", "机场", "农村", "公园",
		"超市", "仓库",
		"分析 UGC/男", "分析 UGC/女",
	}
}

// TKMaterialCreate inserts a new material.
func TKMaterialCreate(m *TKMaterial) error {
	return DB.Create(m).Error
}

// TKMaterialCreateBatch inserts materials in batch.
func TKMaterialCreateBatch(materials []*TKMaterial) error {
	if len(materials) == 0 {
		return nil
	}
	return DB.CreateInBatches(materials, 100).Error
}

// TKMaterialGetByID returns a material by id.
func TKMaterialGetByID(id int) (*TKMaterial, error) {
	var m TKMaterial
	err := DB.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// TKMaterialList returns paginated materials with optional category and keyword filter.
func TKMaterialList(category, keyword string, page, pageSize int) ([]TKMaterial, int64, error) {
	var materials []TKMaterial
	var total int64

	query := DB.Model(&TKMaterial{}).Where("status = ?", 1)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if keyword != "" {
		like := fmt.Sprintf("%%%s%%", keyword)
		query = query.Where("filename LIKE ? OR category LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	err := query.Order("id DESC").Limit(pageSize).Offset(offset).Find(&materials).Error
	return materials, total, err
}

// TKMaterialListAll returns all materials for admin (with status filter).
func TKMaterialListAll(category, keyword string, status int, page, pageSize int) ([]TKMaterial, int64, error) {
	var materials []TKMaterial
	var total int64

	query := DB.Model(&TKMaterial{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if keyword != "" {
		like := fmt.Sprintf("%%%s%%", keyword)
		query = query.Where("filename LIKE ? OR category LIKE ?", like, like)
	}
	if status >= 0 {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	err := query.Order("id DESC").Limit(pageSize).Offset(offset).Find(&materials).Error
	return materials, total, err
}

// TKMaterialDeleteByID soft deletes a material by setting status = 0.
func TKMaterialDeleteByID(id int) error {
	return DB.Model(&TKMaterial{}).Where("id = ?", id).Update("status", 0).Error
}

// TKMaterialRandom returns up to limit random materials for a category.
func TKMaterialRandom(category string, limit int) ([]TKMaterial, error) {
	var materials []TKMaterial
	if limit <= 0 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}

	query := DB.Where("status = ?", 1)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// Use ORDER BY RANDOM() for SQLite/PostgreSQL, ORDER BY RAND() for MySQL
	orderSQL := "RANDOM()"
	if common.UsingMySQL {
		orderSQL = "RAND()"
	}

	err := query.Order(gorm.Expr(orderSQL)).Limit(limit).Find(&materials).Error
	return materials, err
}

// TKMaterialExistsByURL checks whether a material with the same URL and category already exists.
func TKMaterialExistsByURL(url, category string) (bool, error) {
	var count int64
	err := DB.Model(&TKMaterial{}).Where("url = ? AND category = ?", url, category).Count(&count).Error
	return count > 0, err
}

// TKMaterialCountByCategory returns total enabled count per category.
func TKMaterialCountByCategory() (map[string]int64, error) {
	type result struct {
		Category string
		Count    int64
	}
	var results []result
	err := DB.Model(&TKMaterial{}).Select("category, COUNT(*) as count").Where("status = ?", 1).Group("category").Scan(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(results))
	for _, r := range results {
		m[r.Category] = r.Count
	}
	return m, nil
}

// TKMaterialRandomByCategories returns random materials for each requested category.
// It tries to distribute evenly; for categories with fewer items than perCategory,
// it returns all available items.
func TKMaterialRandomByCategories(categories []string, perCategory int) (map[string][]TKMaterial, error) {
	result := make(map[string][]TKMaterial, len(categories))
	if perCategory <= 0 {
		perCategory = 1
	}
	if perCategory > 100 {
		perCategory = 100
	}

	orderSQL := "RANDOM()"
	if common.UsingMySQL {
		orderSQL = "RAND()"
	}

	for _, category := range categories {
		var items []TKMaterial
		if err := DB.Where("status = ? AND category = ?", 1, category).
			Order(gorm.Expr(orderSQL)).Limit(perCategory).Find(&items).Error; err != nil {
			return nil, err
		}
		result[category] = items
	}
	return result, nil
}

func init() {
	rand.Seed(time.Now().Unix())
}
