package controller

import (
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SitemapArticleItem 文章站点地图条目（SEO 专用格式）
type SitemapArticleItem struct {
	Id        string `json:"id"`
	Slug      string `json:"slug"`
	UpdatedAt string `json:"updatedAt"` // ISO 8601 格式
}

// SitemapPromptItem 提示词站点地图条目（SEO 专用格式）
type SitemapPromptItem struct {
	Id        string `json:"id"`
	Slug      string `json:"slug"`
	UpdatedAt string `json:"updatedAt"` // ISO 8601 格式
}

// GetSitemapArticles 公开接口：获取文章站点地图数据
// GET /api/articles?page={page}&pageSize={pageSize}
func GetSitemapArticles(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}
	startIdx := (page - 1) * pageSize

	items, total, err := model.GetPublicArticlesForSitemap(startIdx, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	list := make([]SitemapArticleItem, 0, len(items))
	for _, item := range items {
		list = append(list, SitemapArticleItem{
			Id:        strconv.Itoa(item.Id),
			Slug:      item.Slug,
			UpdatedAt: time.Unix(item.UpdatedTime, 0).UTC().Format(time.RFC3339),
		})
	}

	common.ApiSuccess(c, gin.H{
		"list":       list,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}

// GetSitemapPrompts 公开接口：获取提示词站点地图数据
// GET /api/prompts?page={page}&pageSize={pageSize}
func GetSitemapPrompts(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize < 1 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}
	startIdx := (page - 1) * pageSize

	items, total, err := model.GetPublicPromptsForSitemap(startIdx, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	list := make([]SitemapPromptItem, 0, len(items))
	for _, item := range items {
		slug := item.Slug
		if slug == "" {
			slug = model.GenerateSlug(item.Title)
		}
		if slug == "" {
			slug = strconv.Itoa(item.Id)
		}
		list = append(list, SitemapPromptItem{
			Id:        strconv.Itoa(item.Id),
			Slug:      slug,
			UpdatedAt: time.Unix(item.UpdatedTime, 0).UTC().Format(time.RFC3339),
		})
	}

	common.ApiSuccess(c, gin.H{
		"list":       list,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": (int(total) + pageSize - 1) / pageSize,
	})
}
