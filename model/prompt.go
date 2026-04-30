package model

import (
	"errors"

	"gorm.io/gorm"
)

type Prompt struct {
	Id          int            `json:"id"`
	CategoryId  int            `json:"category_id" gorm:"index"`
	Title       string         `json:"title" gorm:"index"`
	Content     string         `json:"content" gorm:"type:text"`
	Description string         `json:"description"`
	Variables   string         `json:"variables" gorm:"type:text"` // JSON array of variable definitions
	Tags        string         `json:"tags" gorm:"type:text"`      // JSON array of tag strings
	SortOrder   int            `json:"sort_order" gorm:"default:0"`
	Status      int            `json:"status" gorm:"default:1"` // 1=enabled, 2=disabled
	UsageCount  int            `json:"usage_count" gorm:"default:0"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func GetAllPrompts(startIdx int, num int) (prompts []*Prompt, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&Prompt{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = tx.Order("sort_order asc, id desc").Limit(num).Offset(startIdx).Find(&prompts).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return prompts, total, nil
}

func SearchPrompts(keyword string, categoryId int, startIdx int, num int) (prompts []*Prompt, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Prompt{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR description LIKE ?", like, like, like)
	}
	if categoryId > 0 {
		query = query.Where("category_id = ?", categoryId)
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Order("sort_order asc, id desc").Limit(num).Offset(startIdx).Find(&prompts).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return prompts, total, nil
}

func GetPromptById(id int) (*Prompt, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	prompt := Prompt{Id: id}
	err := DB.First(&prompt, "id = ?", id).Error
	return &prompt, err
}

func GetPublicPrompts(categoryId int, keyword string, startIdx int, num int) (prompts []*Prompt, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Prompt{}).Where("status = ?", 1)
	if categoryId > 0 {
		query = query.Where("category_id = ?", categoryId)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ?", like, like)
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Order("sort_order asc, usage_count desc, id desc").Limit(num).Offset(startIdx).Find(&prompts).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return prompts, total, nil
}

func GetPublicPromptById(id int) (*Prompt, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	prompt := Prompt{Id: id}
	err := DB.Where("status = ?", 1).First(&prompt, "id = ?", id).Error
	return &prompt, err
}

func (prompt *Prompt) Insert() error {
	return DB.Create(prompt).Error
}

func (prompt *Prompt) Update() error {
	return DB.Model(prompt).Select("category_id", "title", "content", "description", "variables", "tags", "sort_order", "status").Updates(prompt).Error
}

func (prompt *Prompt) Delete() error {
	return DB.Delete(prompt).Error
}

func DeletePromptById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	prompt := Prompt{Id: id}
	err := DB.Where(prompt).First(&prompt).Error
	if err != nil {
		return err
	}
	return prompt.Delete()
}

func IncrementPromptUsageCount(id int) error {
	return DB.Model(&Prompt{}).Where("id = ?", id).UpdateColumn("usage_count", gorm.Expr("usage_count + ?", 1)).Error
}

// GetPromptsWithCategory 获取提示词列表并附带分类名称
func GetPromptsWithCategory(startIdx int, num int) ([]*PromptWithCategory, int64, error) {
	prompts, total, err := GetAllPrompts(startIdx, num)
	if err != nil {
		return nil, 0, err
	}
	return attachCategoryInfo(prompts), total, nil
}

// SearchPromptsWithCategory 搜索提示词并附带分类名称
func SearchPromptsWithCategory(keyword string, categoryId int, startIdx int, num int) ([]*PromptWithCategory, int64, error) {
	prompts, total, err := SearchPrompts(keyword, categoryId, startIdx, num)
	if err != nil {
		return nil, 0, err
	}
	return attachCategoryInfo(prompts), total, nil
}

type PromptWithCategory struct {
	*Prompt
	CategoryName string `json:"category_name"`
}

func attachCategoryInfo(prompts []*Prompt) []*PromptWithCategory {
	if len(prompts) == 0 {
		return []*PromptWithCategory{}
	}

	// Collect category IDs
	categoryIds := make(map[int]struct{})
	for _, p := range prompts {
		categoryIds[p.CategoryId] = struct{}{}
	}

	// Batch fetch categories
	var categories []*PromptCategory
	DB.Where("id IN ?", getCategoryIds(categoryIds)).Find(&categories)

	categoryMap := make(map[int]string)
	for _, c := range categories {
		categoryMap[c.Id] = c.Name
	}

	result := make([]*PromptWithCategory, len(prompts))
	for i, p := range prompts {
		result[i] = &PromptWithCategory{
			Prompt:       p,
			CategoryName: categoryMap[p.CategoryId],
		}
	}
	return result
}

func getCategoryIds(m map[int]struct{}) []int {
	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}
