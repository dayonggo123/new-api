package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ArticleCategory 文章分类
type ArticleCategory struct {
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

func GetAllArticleCategories(startIdx int, num int) (categories []*ArticleCategory, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	err = tx.Model(&ArticleCategory{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if num > 0 {
		err = tx.Order("sort_order asc, id desc").Limit(num).Offset(startIdx).Find(&categories).Error
	} else {
		err = tx.Order("sort_order asc, id desc").Find(&categories).Error
	}
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func GetEnabledArticleCategories() (categories []*ArticleCategory, err error) {
	err = DB.Where("status = ?", 1).Order("sort_order asc, id desc").Find(&categories).Error
	return categories, err
}

func GetArticleCategoryById(id int) (*ArticleCategory, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	category := ArticleCategory{Id: id}
	err := DB.First(&category, "id = ?", id).Error
	return &category, err
}

func (category *ArticleCategory) Insert() error {
	return DB.Create(category).Error
}

func (category *ArticleCategory) Update() error {
	return DB.Model(category).Select("name", "description", "icon", "sort_order", "status").Updates(category).Error
}

func (category *ArticleCategory) Delete() error {
	return DB.Delete(category).Error
}

func DeleteArticleCategoryById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	category := ArticleCategory{Id: id}
	err := DB.Where(category).First(&category).Error
	if err != nil {
		return err
	}
	return category.Delete()
}

// ArticleSEO18n SEO 多语言翻译项
type ArticleSEO18n struct {
	SeoTitle       string `json:"seo_title,omitempty"`
	SeoDescription string `json:"seo_description,omitempty"`
	SeoKeywords    string `json:"seo_keywords,omitempty"`
	Intro          string `json:"intro,omitempty"`
	Faq            string `json:"faq,omitempty"`
}

// ArticleContent18n 内容多语言翻译项
type ArticleContent18n struct {
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Content string `json:"content,omitempty"`
}

// Article 文章模型
type Article struct {
	Id             int            `json:"id"`
	CategoryId     int            `json:"category_id" gorm:"index"`
	Title          string         `json:"title" gorm:"index"`
	Slug           string         `json:"slug" gorm:"uniqueIndex;size:255"`
	Content        string         `json:"content" gorm:"type:text"`
	Summary        string         `json:"summary" gorm:"type:text"`
	CoverImageUrl  string         `json:"cover_image_url"`
	VideoUrl       string         `json:"video_url"`
	MediaType      string         `json:"media_type" gorm:"default:'image'"` // image | video
	Author         string         `json:"author"`
	Tags           string         `json:"tags" gorm:"type:text"` // JSON array
	Status         int            `json:"status" gorm:"default:1"`          // 1=enabled, 2=disabled
	IsFeatured     bool           `json:"is_featured" gorm:"default:false"` // 置顶/精选
	ViewCount      int            `json:"view_count" gorm:"default:0"`
	SeoTitle       string         `json:"seo_title" gorm:"type:text"`
	SeoDescription string         `json:"seo_description" gorm:"type:text"`
	SeoKeywords    string         `json:"seo_keywords" gorm:"type:text"`
	Intro          string         `json:"intro" gorm:"type:text"`        // AI 生成的介绍文案
	Faq            string         `json:"faq" gorm:"type:text"`          // AI 生成的 FAQ 问答（JSON）
	I18n           string         `json:"i18n" gorm:"type:longtext"`     // 内容多语言 JSON
	SeoI18n        string         `json:"seo_i18n" gorm:"type:longtext"` // SEO 多语言 JSON
	CreatedTime    int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime    int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// ApplyLanguage 根据语言代码替换内容和 SEO 字段（缺失则保持默认中文）
func (a *Article) ApplyLanguage(lang string) {
	if lang == "" || lang == "zh" || lang == "zh-CN" || lang == "zh-TW" {
		return
	}
	if a.SeoI18n != "" {
		var seoMap map[string]ArticleSEO18n
		if err := common.Unmarshal([]byte(a.SeoI18n), &seoMap); err == nil {
			if t, ok := seoMap[lang]; ok {
				if t.SeoTitle != "" {
					a.SeoTitle = t.SeoTitle
				}
				if t.SeoDescription != "" {
					a.SeoDescription = t.SeoDescription
				}
				if t.SeoKeywords != "" {
					a.SeoKeywords = t.SeoKeywords
				}
				if t.Intro != "" {
					a.Intro = t.Intro
				}
				if t.Faq != "" {
					a.Faq = t.Faq
				}
			}
		}
	}
	if a.I18n != "" {
		var contentMap map[string]ArticleContent18n
		if err := common.Unmarshal([]byte(a.I18n), &contentMap); err == nil {
			if c, ok := contentMap[lang]; ok {
				if c.Title != "" {
					a.Title = c.Title
				}
				if c.Summary != "" {
					a.Summary = c.Summary
				}
				if c.Content != "" {
					a.Content = c.Content
				}
			}
		}
	}
}

// SearchArticles 搜索文章（admin 用，支持 keyword + categoryId + status 筛选）
func SearchArticles(keyword string, categoryId int, status int, startIdx int, num int) (articles []*Article, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Article{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", like, like, like)
	}
	if categoryId > 0 {
		query = query.Where("category_id = ?", categoryId)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Order("is_featured desc, id desc").Limit(num).Offset(startIdx).Find(&articles).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

// ArticleSitemapItem 文章站点地图条目（精简字段，高性能）
type ArticleSitemapItem struct {
	Id          int    `json:"id"`
	Slug        string `json:"slug"`
	UpdatedTime int64  `json:"updated_time"`
	CreatedTime int64  `json:"created_time"`
}

// GetPublicArticlesForSitemap 获取公开文章站点地图数据（只返回 SEO 需要的字段）
func GetPublicArticlesForSitemap(startIdx int, num int) (items []*ArticleSitemapItem, total int64, err error) {
	query := DB.Model(&Article{}).
		Select("id", "slug", "updated_time", "created_time").
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

// GetPublicArticles 获取公开文章列表
func GetPublicArticles(categoryId int, keyword string, startIdx int, num int) (articles []*Article, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&Article{}).Where("status = ?", 1)
	if categoryId > 0 {
		query = query.Where("category_id = ?", categoryId)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR summary LIKE ?", like, like)
	}

	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	err = query.Order("is_featured desc, id desc").Limit(num).Offset(startIdx).Find(&articles).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func GetArticleById(id int) (*Article, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	article := Article{Id: id}
	err := DB.First(&article, "id = ?", id).Error
	return &article, err
}

func GetArticleBySlug(slug string) (*Article, error) {
	if slug == "" {
		return nil, errors.New("slug is empty")
	}
	var article Article
	err := DB.Where("slug = ?", slug).First(&article).Error
	return &article, err
}

func GetPublicArticleById(id int) (*Article, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	article := Article{Id: id}
	err := DB.Where("status = ?", 1).First(&article, "id = ?", id).Error
	return &article, err
}

func GetPublicArticleBySlug(slug string) (*Article, error) {
	if slug == "" {
		return nil, errors.New("slug is empty")
	}
	var article Article
	err := DB.Where("status = ? AND slug = ?", 1, slug).First(&article).Error
	return &article, err
}

func (article *Article) Insert() error {
	return DB.Create(article).Error
}

func (article *Article) Update() error {
	return DB.Model(article).Select(
		"category_id", "title", "slug", "content", "summary",
		"cover_image_url", "video_url", "media_type", "author", "tags", "status", "is_featured",
		"seo_title", "seo_description", "seo_keywords", "i18n", "seo_i18n",
	).Updates(article).Error
}

func (article *Article) Delete() error {
	return DB.Delete(article).Error
}

func DeleteArticleById(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}
	article := Article{Id: id}
	err := DB.Where(article).First(&article).Error
	if err != nil {
		return err
	}
	return article.Delete()
}

func IncrementArticleViewCount(id int) error {
	return DB.Model(&Article{}).Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

// ArticleWithCategory 附带分类名称的文章
type ArticleWithCategory struct {
	*Article
	CategoryName string `json:"category_name"`
}

// AttachArticleCategoryInfo 为文章列表附加分类名称
func AttachArticleCategoryInfo(articles []*Article) []*ArticleWithCategory {
	if len(articles) == 0 {
		return []*ArticleWithCategory{}
	}

	categoryIds := make(map[int]struct{})
	for _, a := range articles {
		categoryIds[a.CategoryId] = struct{}{}
	}

	ids := make([]int, 0, len(categoryIds))
	for id := range categoryIds {
		ids = append(ids, id)
	}

	var categories []*ArticleCategory
	DB.Where("id IN ?", ids).Find(&categories)

	categoryMap := make(map[int]string)
	for _, c := range categories {
		categoryMap[c.Id] = c.Name
	}

	result := make([]*ArticleWithCategory, len(articles))
	for i, a := range articles {
		result[i] = &ArticleWithCategory{
			Article:      a,
			CategoryName: categoryMap[a.CategoryId],
		}
	}
	return result
}

// GenerateSlug 从标题生成 URL-friendly slug
func GenerateSlug(title string) string {
	if title == "" {
		return ""
	}
	// 简单处理：替换空格为连字符，移除特殊字符，转小写
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// 保留中文、英文、数字、连字符
	var sb strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r >= 0x4e00 && r <= 0x9fff {
			sb.WriteRune(r)
		}
	}
	slug = sb.String()
	slug = strings.Trim(slug, "-")
	// 去除连续连字符
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}
