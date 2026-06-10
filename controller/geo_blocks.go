package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// GeneratePromptGeoBlocks 为单个 Prompt 生成 GEO 结构化内容
// POST /api/admin/prompts/:id/geo-blocks
func GeneratePromptGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.GeneratePromptGeoBlocks(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      id,
		"message": "Prompt GEO 结构化内容生成成功",
	})
}

// GenerateArticleGeoBlocks 为单篇文章生成 GEO 结构化内容
// POST /api/admin/articles/:id/geo-blocks
func GenerateArticleGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.GenerateArticleGeoBlocks(id); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":      id,
		"message": "文章 GEO 结构化内容生成成功",
	})
}

// BatchGeneratePromptGeoBlocksRequest 批量生成请求
type BatchGeneratePromptGeoBlocksRequest struct {
	Ids []int `json:"ids" binding:"required"`
}

// BatchGeneratePromptGeoBlocks 批量为 Prompt 生成 GEO 结构化内容
// POST /api/admin/prompts/geo-blocks/batch
func BatchGeneratePromptGeoBlocks(c *gin.Context) {
	var req BatchGeneratePromptGeoBlocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ids 不能为空")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "单次最多 50 个")
		return
	}

	taskID := service.StartGeoBlocksGeneration("prompt", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"message": fmt.Sprintf("已启动 %d 个 Prompt 的 GEO 结构化内容生成任务", len(req.Ids)),
	})
}

// BatchGenerateArticleGeoBlocks 批量为文章生成 GEO 结构化内容
// POST /api/admin/articles/geo-blocks/batch
func BatchGenerateArticleGeoBlocks(c *gin.Context) {
	var req BatchGeneratePromptGeoBlocksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ids 不能为空")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "单次最多 50 个")
		return
	}

	taskID := service.StartGeoBlocksGeneration("article", req.Ids)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"message": fmt.Sprintf("已启动 %d 篇文章的 GEO 结构化内容生成任务", len(req.Ids)),
	})
}

// GetGeoBlocksBatchStatus 查询批量 GEO 结构化内容生成任务状态
// GET /api/admin/geo-blocks/batch/:task_id
func GetGeoBlocksBatchStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id 不能为空")
		return
	}

	task := service.GetGeoBlocksTask(taskID)
	if task == nil {
		common.ApiErrorMsg(c, "任务不存在或已过期")
		return
	}

	common.ApiSuccess(c, task)
}

// ========== 下游对接接口（公开 API） ==========

// GetPublicPromptGeoBlocks 公开接口：获取提示词的 GEO 结构化内容
// GET /api/public/prompts/:id/geo-blocks?lang=ko
func GetPublicPromptGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	prompt, err := model.GetPublicPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang != "" {
		prompt.Prompt.ApplyLanguage(lang)
	}

	var geoData interface{}
	if prompt.Prompt.GeoBlocks != "" {
		_ = common.Unmarshal([]byte(prompt.Prompt.GeoBlocks), &geoData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":         id,
			"title":      prompt.Prompt.Title,
			"slug":       prompt.Prompt.Slug,
			"geo_blocks": geoData,
		},
	})
}

// GetPublicPromptGeoBlocksBySlug 公开接口：通过 slug 获取提示词的 GEO 结构化内容
// GET /api/public/prompts/slug/:slug/geo-blocks?lang=ko
func GetPublicPromptGeoBlocksBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		common.ApiErrorMsg(c, "slug is required")
		return
	}

	prompt, err := model.GetPublicPromptBySlug(slug)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang != "" {
		prompt.Prompt.ApplyLanguage(lang)
	}

	var geoData interface{}
	if prompt.Prompt.GeoBlocks != "" {
		_ = common.Unmarshal([]byte(prompt.Prompt.GeoBlocks), &geoData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":         prompt.Prompt.Id,
			"title":      prompt.Prompt.Title,
			"slug":       prompt.Prompt.Slug,
			"geo_blocks": geoData,
		},
	})
}

// GetPublicArticleGeoBlocks 公开接口：获取文章的 GEO 结构化内容
// GET /api/public/articles/:id/geo-blocks?lang=ko
func GetPublicArticleGeoBlocks(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	article, err := model.GetArticleById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang != "" {
		article.ApplyLanguage(lang)
	}

	var geoData interface{}
	if article.GeoBlocks != "" {
		_ = common.Unmarshal([]byte(article.GeoBlocks), &geoData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":         id,
			"title":      article.Title,
			"slug":       article.Slug,
			"geo_blocks": geoData,
		},
	})
}

// GetPublicArticleGeoBlocksBySlug 公开接口：通过 slug 获取文章的 GEO 结构化内容
// GET /api/public/articles/slug/:slug/geo-blocks?lang=ko
func GetPublicArticleGeoBlocksBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		common.ApiErrorMsg(c, "slug is required")
		return
	}

	article, err := model.GetArticleBySlug(slug)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lang := c.Query("lang")
	if lang != "" {
		article.ApplyLanguage(lang)
	}

	var geoData interface{}
	if article.GeoBlocks != "" {
		_ = common.Unmarshal([]byte(article.GeoBlocks), &geoData)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"id":         article.Id,
			"title":      article.Title,
			"slug":       article.Slug,
			"geo_blocks": geoData,
		},
	})
}

// GetPublicPromptGeoBlocksList 公开接口：获取有 GEO 的提示词列表（下游批量对接用）
// GET /api/public/prompts/geo-blocks/list?category_id=0&keyword=&p=1&page_size=20&lang=ko
func GetPublicPromptGeoBlocksList(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))
	lang := c.Query("lang")

	prompts, total, err := model.GetPublicPromptsWithGeo(categoryId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	type GeoListItem struct {
		Id               int         `json:"id"`
		Title            string      `json:"title"`
		Slug             string      `json:"slug"`
		Description      string      `json:"description"`
		CategoryName     string      `json:"category_name"`
		MediaType        string      `json:"media_type"`
		CoverImageUrl    string      `json:"cover_image_url"`
		GeoBlocks        interface{} `json:"geo_blocks"`
		GeoTranslateProgress string  `json:"geo_translate_progress"`
		UpdatedTime      int64       `json:"updated_time"`
	}

	items := make([]*GeoListItem, len(prompts))
	for i, p := range prompts {
		if lang != "" {
			p.Prompt.ApplyLanguage(lang)
		}

		var geoData interface{}
		if p.Prompt.GeoBlocks != "" {
			_ = common.Unmarshal([]byte(p.Prompt.GeoBlocks), &geoData)
		}

		// 计算 GEO 翻译进度
		geoProgress := "0/11"
		if p.Prompt.GeoBlocksI18n != "" {
			var i18nMap map[string]string
			if err := common.Unmarshal([]byte(p.Prompt.GeoBlocksI18n), &i18nMap); err == nil {
				completed := 0
				targetLangs := []string{"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar"}
				for _, l := range targetLangs {
					if v, ok := i18nMap[l]; ok && v != "" {
						completed++
					}
				}
				geoProgress = fmt.Sprintf("%d/%d", completed, len(targetLangs))
			}
		}

		items[i] = &GeoListItem{
			Id:               p.Prompt.Id,
			Title:            p.Prompt.Title,
			Slug:             p.Prompt.Slug,
			Description:      p.Prompt.Description,
			CategoryName:     p.CategoryName,
			MediaType:        p.Prompt.MediaType,
			CoverImageUrl:    p.Prompt.CoverImageUrl,
			GeoBlocks:        geoData,
			GeoTranslateProgress: geoProgress,
			UpdatedTime:      p.Prompt.UpdatedTime,
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// GetPublicArticleGeoBlocksList 公开接口：获取有 GEO 的文章列表（下游批量对接用）
// GET /api/public/articles/geo-blocks/list?category_id=0&keyword=&p=1&page_size=20&lang=ko
func GetPublicArticleGeoBlocksList(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))
	lang := c.Query("lang")

	articles, total, err := model.GetPublicArticlesWithGeo(categoryId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	articlesWithCat := model.AttachArticleCategoryInfo(articles)

	type GeoListItem struct {
		Id               int         `json:"id"`
		Title            string      `json:"title"`
		Slug             string      `json:"slug"`
		Summary          string      `json:"summary"`
		CategoryName     string      `json:"category_name"`
		CoverImageUrl    string      `json:"cover_image_url"`
		GeoBlocks        interface{} `json:"geo_blocks"`
		GeoTranslateProgress string  `json:"geo_translate_progress"`
		UpdatedTime      int64       `json:"updated_time"`
	}

	items := make([]*GeoListItem, len(articlesWithCat))
	for i, a := range articlesWithCat {
		if lang != "" {
			a.ApplyLanguage(lang)
		}

		var geoData interface{}
		if a.GeoBlocks != "" {
			_ = common.Unmarshal([]byte(a.GeoBlocks), &geoData)
		}

		// 计算 GEO 翻译进度
		geoProgress := "0/11"
		if a.GeoBlocksI18n != "" {
			var i18nMap map[string]string
			if err := common.Unmarshal([]byte(a.GeoBlocksI18n), &i18nMap); err == nil {
				completed := 0
				targetLangs := []string{"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar"}
				for _, l := range targetLangs {
					if v, ok := i18nMap[l]; ok && v != "" {
						completed++
					}
				}
				geoProgress = fmt.Sprintf("%d/%d", completed, len(targetLangs))
			}
		}

		items[i] = &GeoListItem{
			Id:               a.Id,
			Title:            a.Title,
			Slug:             a.Slug,
			Summary:          a.Summary,
			CategoryName:     a.CategoryName,
			CoverImageUrl:    a.CoverImageUrl,
			GeoBlocks:        geoData,
			GeoTranslateProgress: geoProgress,
			UpdatedTime:      a.UpdatedTime,
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
