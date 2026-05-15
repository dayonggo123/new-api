package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== Admin: Article Category ====================

func GetAllArticleCategories(c *gin.Context) {
	categories, _, err := model.GetAllArticleCategories(0, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetEnabledArticleCategories(c *gin.Context) {
	categories, err := model.GetEnabledArticleCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetArticleCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	category, err := model.GetArticleCategoryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    category,
	})
}

func AddArticleCategory(c *gin.Context) {
	category := model.ArticleCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(category.Name) == 0 || utf8.RuneCountInString(category.Name) > 50 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "分类名称不能为空且不能超过50个字符"})
		return
	}
	category.CreatedTime = common.GetTimestamp()
	category.UpdatedTime = common.GetTimestamp()
	err = category.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    category,
	})
}

func UpdateArticleCategory(c *gin.Context) {
	category := model.ArticleCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory, err := model.GetArticleCategoryById(category.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory.Name = category.Name
	cleanCategory.Description = category.Description
	cleanCategory.Icon = category.Icon
	cleanCategory.SortOrder = category.SortOrder
	cleanCategory.Status = category.Status
	cleanCategory.UpdatedTime = common.GetTimestamp()
	err = cleanCategory.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanCategory,
	})
}

func DeleteArticleCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteArticleCategoryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Admin: Article ====================

func GetArticles(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))
	status, _ := strconv.Atoi(c.Query("status"))

	articles, total, err := model.SearchArticles(keyword, categoryId, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := model.AttachArticleCategoryInfo(articles)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetArticle(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    article,
	})
}

func CreateArticle(c *gin.Context) {
	article := model.Article{}
	err := c.ShouldBindJSON(&article)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(article.Title) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "文章标题不能为空"})
		return
	}
	if utf8.RuneCountInString(article.Content) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "文章内容不能为空"})
		return
	}
	if article.Slug == "" {
		article.Slug = model.GenerateSlug(article.Title)
	}
	article.CreatedTime = common.GetTimestamp()
	article.UpdatedTime = common.GetTimestamp()
	err = article.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步生成 SEO 元数据
	go generateArticleSEO(&article)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    article,
	})
}

func UpdateArticle(c *gin.Context) {
	article := model.Article{}
	err := c.ShouldBindJSON(&article)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanArticle, err := model.GetArticleById(article.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanArticle.CategoryId = article.CategoryId
	cleanArticle.Title = article.Title
	cleanArticle.Slug = article.Slug
	cleanArticle.Content = article.Content
	cleanArticle.Summary = article.Summary
	cleanArticle.CoverImageUrl = article.CoverImageUrl
	cleanArticle.Author = article.Author
	cleanArticle.Tags = article.Tags
	cleanArticle.Status = article.Status
	cleanArticle.IsFeatured = article.IsFeatured
	cleanArticle.SeoTitle = article.SeoTitle
	cleanArticle.SeoDescription = article.SeoDescription
	cleanArticle.SeoKeywords = article.SeoKeywords
	cleanArticle.I18n = article.I18n
	cleanArticle.SeoI18n = article.SeoI18n
	cleanArticle.UpdatedTime = common.GetTimestamp()
	err = cleanArticle.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步生成 SEO 元数据
	go generateArticleSEO(cleanArticle)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanArticle,
	})
}

func DeleteArticle(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeleteArticleById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// generateArticleSEO 异步调用 AI 生成文章 SEO 元数据
func generateArticleSEO(article *model.Article) {
	result, err := service.GenerateSEOForArticle(article)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("generate seo for article %d failed: %v", article.Id, err))
		return
	}
	service.UpdateArticleSEO(article.Id, result)
}

// ==================== Public API (no auth required) ====================

func GetPublicArticles(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))

	articles, total, err := model.GetPublicArticles(categoryId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	items := model.AttachArticleCategoryInfo(articles)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetPublicArticle(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	article, err := model.GetPublicArticleById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步增加浏览量
	go model.IncrementArticleViewCount(id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    article,
	})
}

func GetPublicArticleBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		common.ApiErrorMsg(c, "slug is required")
		return
	}
	article, err := model.GetPublicArticleBySlug(slug)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步增加浏览量
	go model.IncrementArticleViewCount(article.Id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    article,
	})
}

func GetPublicArticleCategories(c *gin.Context) {
	categories, err := model.GetEnabledArticleCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

// ==================== Admin: SEO Management ====================

func GetArticleSEOList(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	articles, total, err := model.SearchArticles(keyword, 0, 0, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	articlesWithCat := model.AttachArticleCategoryInfo(articles)

	// 批量获取最新审计分数
	articleIds := make([]int, len(articlesWithCat))
	for i, a := range articlesWithCat {
		articleIds[i] = a.Id
	}
	auditScores, _ := model.GetLatestArticleSEOAuditScores(articleIds)

	// 只返回 SEO 相关字段
	type SEOItem struct {
		Id             int    `json:"id"`
		Title          string `json:"title"`
		CategoryName   string `json:"category_name"`
		SeoTitle       string `json:"seo_title"`
		SeoDescription string `json:"seo_description"`
		SeoKeywords    string `json:"seo_keywords"`
		SeoI18n        string `json:"seo_i18n"`
		AuditScore     int    `json:"audit_score"`
		Status         int    `json:"status"`
		CreatedTime    int64  `json:"created_time"`
		UpdatedTime    int64  `json:"updated_time"`
	}

	items := make([]*SEOItem, len(articlesWithCat))
	for i, a := range articlesWithCat {
		items[i] = &SEOItem{
			Id:             a.Id,
			Title:          a.Title,
			CategoryName:   a.CategoryName,
			SeoTitle:       a.SeoTitle,
			SeoDescription: a.SeoDescription,
			SeoKeywords:    a.SeoKeywords,
			SeoI18n:        a.SeoI18n,
			AuditScore:     auditScores[a.Id],
			Status:         a.Status,
			CreatedTime:    a.CreatedTime,
			UpdatedTime:    a.UpdatedTime,
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetArticleSEO(c *gin.Context) {
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
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    article,
	})
}

func UpdateArticleSEOFields(c *gin.Context) {
	var req struct {
		Id             int    `json:"id"`
		SeoTitle       string `json:"seo_title"`
		SeoDescription string `json:"seo_description"`
		SeoKeywords    string `json:"seo_keywords"`
		Intro          string `json:"intro"`
		Faq            string `json:"faq"`
		SeoI18n        string `json:"seo_i18n"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	updates := map[string]interface{}{
		"seo_title":       req.SeoTitle,
		"seo_description": req.SeoDescription,
		"seo_keywords":    req.SeoKeywords,
		"intro":           req.Intro,
		"faq":             req.Faq,
		"seo_i18n":        req.SeoI18n,
	}
	if err := model.DB.Model(&model.Article{}).Where("id = ?", req.Id).Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func RegenerateArticleSEO(c *gin.Context) {
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
	go generateArticleSEO(article)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI 生成任务已启动，请稍后刷新查看结果",
	})
}

// ==================== Admin: AI Write Article ====================

func GenerateArticle(c *gin.Context) {
	var req service.ArticleWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Prompt == "" && req.Title == "" && req.ReferenceURL == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请至少提供标题、写作要求或参考链接之一"})
		return
	}
	if req.Language == "" {
		req.Language = "zh"
	}

	result, err := service.GenerateArticle(&req)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("generate article failed: %v", err))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "AI 生成失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

// ==================== Admin: Article SEO Audit ====================

func AuditArticleSEOHandler(c *gin.Context) {
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
	result, err := service.AuditArticleSEO(article)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "SEO 审核失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}

func GetArticleSEOAHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	audits, err := model.GetArticleSEOAudits(id, 20)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    audits,
	})
}

func GetArticleSEOReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	audit, err := model.GetLatestArticleSEOAudit(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "暂无审核记录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    audit,
	})
}

func GetAllArticleSEOReport(c *gin.Context) {
	stats, err := model.GetArticleSEOAuditStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}

func GetLowScoreArticlesHandler(c *gin.Context) {
	threshold, _ := strconv.Atoi(c.Query("threshold"))
	if threshold <= 0 {
		threshold = 60
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 20
	}
	articles, err := model.GetLowScoreArticles(threshold, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    articles,
	})
}

// ==================== Admin: AI Generate Images ====================

type GenerateArticleImagesRequest struct {
	Prompt string `json:"prompt" binding:"required"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

func GenerateArticleImages(c *gin.Context) {
	var req GenerateArticleImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.GenerateImagesForArticle(req.Prompt, req.N, req.Size)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("generate article images failed: %v", err))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "AI 生成图片失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    result,
	})
}
