package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// PromptSEO18n SEO 多语言翻译项
type PromptSEO18n struct {
	SeoKeywords string `json:"seo_keywords,omitempty"`
	Intro       string `json:"intro,omitempty"`
	Faq         string `json:"faq,omitempty"`
}

type Prompt struct {
	Id            int            `json:"id"`
	CategoryId    int            `json:"category_id" gorm:"index"`
	Title         string         `json:"title" gorm:"index"`
	Slug          string         `json:"slug" gorm:"uniqueIndex;size:255"` // URL 友好的路径标识
	Content       string         `json:"content" gorm:"type:text"`
	ContentEn     string         `json:"content_en" gorm:"type:text"`
	Description   string         `json:"description"`
	CoverImageUrl string         `json:"cover_image_url"`
	VideoUrl      string         `json:"video_url"`
	Author        string         `json:"author"`                     // 来源/作者，如 @username
	Source        string         `json:"source"`                     // 采集来源平台，如 opennana / tiktok
	Model         string         `json:"model"`                      // 使用的AI模型，如 ChatGPT
	Variables     string         `json:"variables" gorm:"type:text"` // JSON array of variable definitions
	Tags          string         `json:"tags" gorm:"type:text"`      // JSON array of tag strings
	SortOrder     int            `json:"sort_order" gorm:"default:0"`
	Status        int            `json:"status" gorm:"default:1;index"` // 1=enabled, 2=disabled
	UsageCount    int            `json:"usage_count" gorm:"default:0"`
	SeoKeywords   string         `json:"seo_keywords" gorm:"type:text"` // AI 生成的 SEO 关键词
	Intro         string         `json:"intro" gorm:"type:text"`        // AI 生成的介绍文案
	Faq           string         `json:"faq" gorm:"type:text"`          // AI 生成的 FAQ 问答（JSON）
	MediaType     string         `json:"media_type" gorm:"default:'image'"` // 内容类型: image / video
	IsPremium     bool           `json:"is_premium" gorm:"default:false"`
	UnlockCost    int            `json:"unlock_cost" gorm:"default:0"`
	I18n          string         `json:"i18n" gorm:"type:text"`         // 多语言 JSON（内容）
	TitleI18n     string         `json:"title_i18n" gorm:"type:text"`   // 标题多语言 JSON
	SeoI18n       string         `json:"seo_i18n" gorm:"type:text"`     // SEO 多语言 JSON
	IsTranslated  bool           `json:"is_translated" gorm:"default:false"` // 是否已完成多语言翻译
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime   int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// ApplyLanguage 根据语言代码替换 title/content/SEO 字段内容（缺失则保持默认中文）
func (p *Prompt) ApplyLanguage(lang string) {
	if lang == "" || lang == "zh" || lang == "zh-CN" || lang == "zh-TW" {
		return
	}

	// 处理 title_i18n
	if p.TitleI18n != "" {
		var titleMap map[string]string
		if err := common.Unmarshal([]byte(p.TitleI18n), &titleMap); err == nil {
			if t, ok := titleMap[lang]; ok && t != "" {
				p.Title = t
			}
		}
	}

	// 处理 content：en 优先用 content_en，其他从 i18n 取
	if lang == "en" && p.ContentEn != "" {
		p.Content = p.ContentEn
	} else if p.I18n != "" {
		var contentMap map[string]string
		if err := common.Unmarshal([]byte(p.I18n), &contentMap); err == nil {
			if c, ok := contentMap[lang]; ok && c != "" {
				p.Content = c
			}
		}
	}

	// 处理 seo_i18n
	if p.SeoI18n == "" {
		return
	}
	var i18nMap map[string]PromptSEO18n
	if err := common.Unmarshal([]byte(p.SeoI18n), &i18nMap); err != nil {
		return
	}
	if t, ok := i18nMap[lang]; ok {
		if t.SeoKeywords != "" {
			p.SeoKeywords = t.SeoKeywords
		}
		if t.Intro != "" {
			p.Intro = t.Intro
		}
		if t.Faq != "" {
			p.Faq = t.Faq
		}
	}
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

func GetPromptById(id int) (*PromptWithCategory, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	prompt := Prompt{Id: id}
	err := DB.First(&prompt, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	result := attachCategoryInfo([]*Prompt{&prompt})
	if len(result) == 0 {
		return nil, errors.New("prompt not found")
	}
	return result[0], nil
}

// GetPublicPrompts 获取公开的提示词列表（优化版：限制字段、无事务）
func GetPublicPrompts(categoryId int, keyword string, startIdx int, num int) (prompts []*PromptWithCategory, total int64, err error) {
	// 列表页只需要这些字段，避免加载大文本字段（content, variables, tags 等）
	selectFields := []string{
		"id", "category_id", "title", "description", "slug",
		"cover_image_url", "video_url", "author", "source", "model",
		"media_type", "is_premium", "unlock_cost",
		"sort_order", "status", "usage_count",
		"created_time", "updated_time",
		"i18n", "title_i18n", "content_en",
	}

	query := DB.Model(&Prompt{}).Select(selectFields).Where("status = ?", 1)
	if categoryId > 0 {
		query = query.Where("category_id = ?", categoryId)
	}
	if keyword != "" {
		// 只搜索 title，避免对 TEXT 类型的 content 做 LIKE（无索引时全表扫描）
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}

	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var rawPrompts []*Prompt
	err = query.Order("sort_order asc, usage_count desc, id desc").Limit(num).Offset(startIdx).Find(&rawPrompts).Error
	if err != nil {
		return nil, 0, err
	}

	return attachCategoryInfo(rawPrompts), total, nil
}

// PromptSitemapItem 提示词站点地图条目（精简字段，高性能）
type PromptSitemapItem struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	UpdatedTime int64  `json:"updated_time"`
	CreatedTime int64  `json:"created_time"`
}

// GetPublicPromptsForSitemap 获取公开提示词站点地图数据（只返回 SEO 需要的字段）
func GetPublicPromptsForSitemap(startIdx int, num int) (items []*PromptSitemapItem, total int64, err error) {
	query := DB.Model(&Prompt{}).
		Select("id", "title", "slug", "updated_time", "created_time").
		Where("status = ?", 1)

	err = query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("updated_time desc, id desc").
		Limit(num).Offset(startIdx).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetPublicPromptById(id int) (*PromptWithCategory, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	prompt := Prompt{Id: id}
	err := DB.Where("status = ?", 1).First(&prompt, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	result := attachCategoryInfo([]*Prompt{&prompt})
	if len(result) == 0 {
		return nil, errors.New("prompt not found")
	}
	return result[0], nil
}

func GetPublicPromptBySlug(slug string) (*PromptWithCategory, error) {
	if slug == "" {
		return nil, errors.New("slug is empty")
	}
	var prompt Prompt
	err := DB.Where("status = ? AND slug = ?", 1, slug).First(&prompt).Error
	if err != nil {
		return nil, err
	}
	result := attachCategoryInfo([]*Prompt{&prompt})
	if len(result) == 0 {
		return nil, errors.New("prompt not found")
	}
	return result[0], nil
}

func (prompt *Prompt) Insert() error {
	if prompt.Slug == "" {
		prompt.Slug = GenerateSlug(prompt.Title)
	}
	if prompt.Slug == "" {
		prompt.Slug = fmt.Sprintf("prompt-%d", time.Now().Unix())
	}
	return DB.Create(prompt).Error
}

func (prompt *Prompt) Update() error {
	return DB.Model(prompt).Select("category_id", "title", "slug", "content", "content_en", "description", "cover_image_url", "video_url", "author", "source", "model", "variables", "tags", "sort_order", "status", "media_type", "i18n", "title_i18n", "is_translated").Updates(prompt).Error
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

// BatchGeneratePromptSlugs 批量为 slug 为空的提示词生成 slug
// 从 title 生成，如果冲突则加 "-{id}" 后缀保证唯一性
func BatchGeneratePromptSlugs() (updated int, skipped int, err error) {
	var prompts []Prompt
	err = DB.Where("slug = ? OR slug IS NULL", "").Find(&prompts).Error
	if err != nil {
		return 0, 0, err
	}

	for _, p := range prompts {
		slug := GenerateSlug(p.Title)
		if slug == "" {
			slug = fmt.Sprintf("prompt-%d", p.Id)
		}

		// 检查是否已存在相同 slug（其他记录）
		var count int64
		DB.Model(&Prompt{}).Where("slug = ? AND id != ?", slug, p.Id).Count(&count)
		if count > 0 {
			slug = fmt.Sprintf("%s-%d", slug, p.Id)
		}

		if err := DB.Model(&Prompt{}).Where("id = ?", p.Id).Update("slug", slug).Error; err != nil {
			common.SysLog(fmt.Sprintf("批量生成 slug 失败: prompt_id=%d, err=%v", p.Id, err))
			skipped++
			continue
		}
		updated++
	}
	return updated, skipped, nil
}

// GetPublicPromptsUpdatedSince 获取自指定时间后有更新的公开提示词
func GetPublicPromptsUpdatedSince(since int64) ([]*Prompt, error) {
	var prompts []*Prompt
	tx := DB.Where("status = ?", 1)
	if since > 0 {
		tx = tx.Where("updated_time > ?", since)
	}
	err := tx.Order("updated_time desc, id desc").Find(&prompts).Error
	return prompts, err
}
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
	if err := DB.Where("id IN ?", getCategoryIds(categoryIds)).Find(&categories).Error; err != nil {
		// 如果分类查询失败，仍然返回结果（category_name 为空）
		common.SysLog("attachCategoryInfo query categories failed: " + err.Error())
	}

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
