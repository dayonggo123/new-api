package controller

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== Admin: Prompt Category ====================

func GetAllPromptCategories(c *gin.Context) {
	categories, _, err := model.GetAllPromptCategories(0, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetEnabledPromptCategories(c *gin.Context) {
	categories, err := model.GetEnabledPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func GetPromptCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	category, err := model.GetPromptCategoryById(id)
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

func AddPromptCategory(c *gin.Context) {
	category := model.PromptCategory{}
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

func UpdatePromptCategory(c *gin.Context) {
	category := model.PromptCategory{}
	err := c.ShouldBindJSON(&category)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanCategory, err := model.GetPromptCategoryById(category.Id)
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

func DeletePromptCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeletePromptCategoryById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// ==================== Admin: Prompt ====================

func GetAllPrompts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))

	prompts, total, err := model.SearchPromptsWithCategory(keyword, categoryId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(prompts)
	common.ApiSuccess(c, pageInfo)
}

func GetPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func AddPrompt(c *gin.Context) {
	prompt := model.Prompt{}
	err := c.ShouldBindJSON(&prompt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if utf8.RuneCountInString(prompt.Title) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提示词标题不能为空"})
		return
	}
	if utf8.RuneCountInString(prompt.Content) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "提示词内容不能为空"})
		return
	}
	prompt.CreatedTime = common.GetTimestamp()
	prompt.UpdatedTime = common.GetTimestamp()
	err = prompt.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步生成 SEO 关键词和介绍文案
	go generatePromptSEO(&prompt)
	// 异步触发多语言自动翻译
	go service.StartAutoTranslate("prompt", prompt.Id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func UpdatePrompt(c *gin.Context) {
	prompt := model.Prompt{}
	err := c.ShouldBindJSON(&prompt)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPrompt, err := model.GetPromptById(prompt.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	cleanPrompt.CategoryId = prompt.CategoryId
	cleanPrompt.Title = prompt.Title
	cleanPrompt.Content = prompt.Content
	cleanPrompt.ContentEn = prompt.ContentEn
	cleanPrompt.Description = prompt.Description
	cleanPrompt.CoverImageUrl = prompt.CoverImageUrl
	cleanPrompt.VideoUrl = prompt.VideoUrl
	cleanPrompt.Author = prompt.Author
	cleanPrompt.Source = prompt.Source
	cleanPrompt.Model = prompt.Model
	cleanPrompt.MediaType = prompt.MediaType
	cleanPrompt.Variables = prompt.Variables
	cleanPrompt.Tags = prompt.Tags
	cleanPrompt.SortOrder = prompt.SortOrder
	cleanPrompt.Status = prompt.Status
	cleanPrompt.I18n = prompt.I18n
	cleanPrompt.TitleI18n = prompt.TitleI18n
	// 只有所有目标语言都翻译完整，才标记为已翻译
	cleanPrompt.IsTranslated = isPromptTranslationComplete(prompt.TitleI18n, prompt.I18n, prompt.ContentEn)
	if cleanPrompt.IsTranslated {
		cleanPrompt.TranslationError = ""
	}
	cleanPrompt.UpdatedTime = common.GetTimestamp()
	err = cleanPrompt.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	// 异步生成 SEO 关键词和介绍文案
	go generatePromptSEO(cleanPrompt.Prompt)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanPrompt,
	})
}

func DeletePrompt(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	err := model.DeletePromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// isPromptTranslationComplete 检查 Prompt 的 title_i18n + i18n 是否包含所有目标语言
func isPromptTranslationComplete(titleI18n, i18n, contentEn string) bool {
	targetLangs := []string{"en", "fr", "ru", "ja", "vi", "ko", "es", "de", "pt", "it", "ar"}
	var titleMap, contentMap map[string]string
	if titleI18n != "" {
		_ = common.Unmarshal([]byte(titleI18n), &titleMap)
	}
	if i18n != "" {
		_ = common.Unmarshal([]byte(i18n), &contentMap)
	}
	for _, lang := range targetLangs {
		hasTitle := titleMap[lang] != ""
		hasContent := contentMap[lang] != ""
		if lang == "en" && contentEn != "" {
			hasContent = true
		}
		if !hasTitle || !hasContent {
			return false
		}
	}
	return true
}

// ==================== Public API (no auth required) ====================

func GetPublicPrompts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	categoryId, _ := strconv.Atoi(c.Query("category_id"))
	lang := c.Query("lang")

	prompts, total, err := model.GetPublicPrompts(categoryId, keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 根据 lang 参数切换多语言内容
	if lang != "" {
		for _, p := range prompts {
			p.Prompt.ApplyLanguage(lang)
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(prompts)
	common.ApiSuccess(c, pageInfo)
}

// GetPublicPromptUpdates 公开接口：获取提示词库增量更新
// 支持 ?since= 参数（秒级时间戳），返回该时间之后有更新的启用提示词
// 不带 since 则返回全部启用提示词（但精简字段）
func GetPublicPromptUpdates(c *gin.Context) {
	sinceStr := c.Query("since")
	since := int64(0)
	if sinceStr != "" {
		if v, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = v
		}
	}

	prompts, err := model.GetPublicPromptsUpdatedSince(since)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 精简字段返回，只保留下游同步需要的核心信息
	type PromptUpdateItem struct {
		Id          int    `json:"id"`
		Title       string `json:"title"`
		Slug        string `json:"slug"`
		CategoryId  int    `json:"category_id"`
		Description string `json:"description"`
		Status      int    `json:"status"`
		SortOrder   int    `json:"sort_order"`
		UpdatedTime int64  `json:"updated_time"`
		CreatedTime int64  `json:"created_time"`
	}

	items := make([]PromptUpdateItem, 0, len(prompts))
	for i := range prompts {
		items = append(items, PromptUpdateItem{
			Id:          prompts[i].Id,
			Title:       prompts[i].Title,
			Slug:        prompts[i].Slug,
			CategoryId:  prompts[i].CategoryId,
			Description: prompts[i].Description,
			Status:      prompts[i].Status,
			SortOrder:   prompts[i].SortOrder,
			UpdatedTime: prompts[i].UpdatedTime,
			CreatedTime: prompts[i].CreatedTime,
		})
	}

	common.ApiSuccess(c, gin.H{
		"items":       items,
		"total":       len(items),
		"server_time": time.Now().Unix(),
	})
}

func GetPublicPrompt(c *gin.Context) {
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
	// Increment usage count asynchronously
	go model.IncrementPromptUsageCount(id)
	lang := c.Query("lang")
	if lang != "" {
		prompt.Prompt.ApplyLanguage(lang)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

func GetPublicPromptBySlug(c *gin.Context) {
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
	// Increment usage count asynchronously
	go model.IncrementPromptUsageCount(prompt.Id)
	lang := c.Query("lang")
	if lang != "" {
		prompt.Prompt.ApplyLanguage(lang)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

// GetPublicPromptsSitemap 公开接口：获取提示词站点地图数据（SEO 专用，轻量、高性能）
func GetPublicPromptsSitemap(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize < 1 {
		pageSize = 500
	}
	if pageSize > 5000 {
		pageSize = 5000
	}
	startIdx := (page - 1) * pageSize

	items, total, err := model.GetPublicPromptsForSitemap(startIdx, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items":      items,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (int(total) + pageSize - 1) / pageSize,
	})
}

func GetPublicPromptCategories(c *gin.Context) {
	categories, err := model.GetEnabledPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

// generatePromptSEO 异步调用 AI 生成 SEO 关键词和介绍文案
func generatePromptSEO(prompt *model.Prompt) {
	result, err := service.GenerateSEOForPrompt(prompt)
	if err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("generate seo for prompt %d failed: %v", prompt.Id, err))
		return
	}
	service.UpdatePromptSEO(prompt.Id, result)
}

// ==================== Admin: SEO Management ====================

func GetPromptSEOList(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	prompts, total, err := model.SearchPromptsWithCategory(keyword, 0, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 批量获取最新审计分数
	promptIds := make([]int, len(prompts))
	for i, p := range prompts {
		promptIds[i] = p.Id
	}
	auditScores, _ := model.GetLatestPromptSEOAuditScores(promptIds)

	// 只返回 SEO 相关字段
	type SEOItem struct {
		Id            int    `json:"id"`
		Title         string `json:"title"`
		CategoryName  string `json:"category_name"`
		SeoKeywords   string `json:"seo_keywords"`
		Intro         string `json:"intro"`
		Faq           string `json:"faq"`
		SeoI18n       string `json:"seo_i18n"`
		TitleI18n     string `json:"title_i18n"`
		I18n          string `json:"i18n"`
		AuditScore    int    `json:"audit_score"`
		Status        int    `json:"status"`
		CreatedTime   int64  `json:"created_time"`
		UpdatedTime   int64  `json:"updated_time"`
	}

	items := make([]*SEOItem, len(prompts))
	for i, p := range prompts {
		items[i] = &SEOItem{
			Id:           p.Id,
			Title:        p.Title,
			CategoryName: p.CategoryName,
			SeoKeywords:  p.SeoKeywords,
			Intro:        p.Intro,
			Faq:          p.Faq,
			SeoI18n:      p.SeoI18n,
			TitleI18n:    p.TitleI18n,
			I18n:         p.I18n,
			AuditScore:   auditScores[p.Id],
			Status:       p.Status,
			CreatedTime:  p.CreatedTime,
			UpdatedTime:  p.UpdatedTime,
		}
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetPromptSEODetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    prompt,
	})
}

// CheckPromptsExist 检查 source_url 是否已存在（用于扩展去重）
func CheckPromptsExist(c *gin.Context) {
	var req struct {
		Urls []string `json:"urls"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Urls) == 0 {
		common.ApiSuccess(c, map[string]interface{}{"existing": []string{}})
		return
	}

	// 限制一次最多查 100 个
	if len(req.Urls) > 100 {
		req.Urls = req.Urls[:100]
	}

	var existingUrls []string
	model.DB.Model(&model.Prompt{}).
		Where("source_url IN ?", req.Urls).
		Pluck("source_url", &existingUrls)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    map[string]interface{}{"existing": existingUrls},
	})
}

func UpdatePromptSEOFields(c *gin.Context) {
	var req struct {
		Id          int    `json:"id"`
		SeoKeywords string `json:"seo_keywords"`
		Intro       string `json:"intro"`
		Faq         string `json:"faq"`
		SeoI18n     string `json:"seo_i18n"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	updates := map[string]interface{}{
		"seo_keywords":          req.SeoKeywords,
		"intro":                 req.Intro,
		"faq":                   req.Faq,
		"seo_i18n":              req.SeoI18n,
		"seo_translation_error": "", // 保存时清空 SEO 翻译错误
	}
	if err := model.DB.Model(&model.Prompt{}).Where("id = ?", req.Id).Select("seo_keywords", "intro", "faq", "seo_i18n", "seo_translation_error").Updates(updates).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	// 保存后如果 seo_i18n 有值，自动触发 SEO 审计
	if req.SeoI18n != "" {
		if p, err := model.GetPromptById(req.Id); err == nil && p != nil && p.Prompt != nil {
			p.Prompt.SeoI18n = req.SeoI18n
			go service.AuditPromptSEO(p.Prompt)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func RegeneratePromptSEO(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	go generatePromptSEO(prompt.Prompt)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI 生成任务已启动，请稍后刷新查看结果",
	})
}

// AuditPromptSEOHandler 调用 AI 审计 Prompt 的 SEO 内容
func AuditPromptSEOHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.AuditPromptSEO(prompt.Prompt)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}

// GetPromptSEOAHistory 获取指定 Prompt 的 SEO 审计历史
func GetPromptSEOAHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	audits, err := model.GetPromptSEOAudits(id, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, audits)
}

// GetPromptSEOStats 获取 SEO 统计概览
func GetPromptSEOStats(c *gin.Context) {
	stats, err := model.GetPromptSEOAuditStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

// GetPromptSEOTrends 获取 SEO 审计趋势
func GetPromptSEOTrends(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 90 {
		days = 30
	}
	trends, err := model.GetSEOTrends(days)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, trends)
}

// GetLowScorePrompts 获取低分提示词列表
func GetLowScorePrompts(c *gin.Context) {
	thresholdStr := c.DefaultQuery("threshold", "60")
	threshold, _ := strconv.Atoi(thresholdStr)
	if threshold <= 0 || threshold > 100 {
		threshold = 60
	}
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	items, err := model.GetLowScorePrompts(threshold, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, items)
}

// GetPromptSEOReport 导出单个 Prompt 的 SEO 审计报告（Markdown）
func GetPromptSEOReport(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	prompt, err := model.GetPromptById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	latestAudit, err := model.GetLatestPromptSEOAudit(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var categories map[string]service.SEOAuditCategory
	var criticalIssues, quickWins []string
	common.Unmarshal([]byte(latestAudit.Categories), &categories)
	common.Unmarshal([]byte(latestAudit.CriticalIssues), &criticalIssues)
	common.Unmarshal([]byte(latestAudit.QuickWins), &quickWins)

	markdown := fmt.Sprintf("# SEO 审计报告：%s\n\n", prompt.Title)
	markdown += fmt.Sprintf("- **Prompt ID**: %d\n", id)
	markdown += fmt.Sprintf("- **审计日期**: %s\n", time.Unix(latestAudit.CreatedAt, 0).Format("2006-01-02 15:04:05"))
	markdown += fmt.Sprintf("- **总分**: %d / 100\n\n", latestAudit.OverallScore)

	markdown += "## 维度评分\n\n"
	for key, cat := range categories {
		labelMap := map[string]string{
			"completeness":    "完整性",
			"keyword_quality": "关键词质量",
			"intro_quality":   "介绍文案",
			"faq_quality":     "FAQ 质量",
			"structured_data": "结构化数据",
		}
		label := labelMap[key]
		if label == "" {
			label = key
		}
		markdown += fmt.Sprintf("### %s: %d/100\n\n", label, cat.Score)
		if len(cat.Issues) > 0 {
			markdown += "**问题**:\n"
			for _, issue := range cat.Issues {
				markdown += fmt.Sprintf("- %s\n", issue)
			}
			markdown += "\n"
		}
		if len(cat.Suggestions) > 0 {
			markdown += "**建议**:\n"
			for _, s := range cat.Suggestions {
				markdown += fmt.Sprintf("- %s\n", s)
			}
			markdown += "\n"
		}
	}

	if len(criticalIssues) > 0 {
		markdown += "## 关键问题\n\n"
		for _, issue := range criticalIssues {
			markdown += fmt.Sprintf("- %s\n", issue)
		}
		markdown += "\n"
	}

	if len(quickWins) > 0 {
		markdown += "## 快速改进\n\n"
		for _, win := range quickWins {
			markdown += fmt.Sprintf("- %s\n", win)
		}
		markdown += "\n"
	}

	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"seo-report-%d.md\"", id))
	c.String(http.StatusOK, markdown)
}

// GetAllSEOReport 导出全站 SEO 健康度报告（Markdown）
func GetAllSEOReport(c *gin.Context) {
	stats, err := model.GetPromptSEOAuditStats()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	lowScores, err := model.GetLowScorePrompts(60, 50)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	trends, err := model.GetSEOTrends(30)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	markdown := "# 全站 SEO 健康度报告\n\n"
	markdown += fmt.Sprintf("**生成时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	markdown += "## 概览\n\n"
	markdown += fmt.Sprintf("- **总提示词数**: %.0f\n", stats["total_prompts"])
	markdown += fmt.Sprintf("- **已配置 SEO**: %.0f (%.2f%%)\n", stats["with_seo"], stats["seo_coverage"])
	markdown += fmt.Sprintf("- **已审计**: %.0f (%.2f%%)\n", stats["with_audit"], stats["audit_coverage"])
	markdown += fmt.Sprintf("- **平均审计分**: %.1f\n\n", stats["average_score"])

	markdown += "## 分数分布\n\n"
	if dist, ok := stats["score_distribution"].([]struct {
		Range string `json:"range"`
		Count int64  `json:"count"`
	}); ok {
		labelMap := map[string]string{"excellent": "优秀", "good": "良好", "average": "中等", "poor": "较差"}
		for _, d := range dist {
			label := labelMap[d.Range]
			if label == "" {
				label = d.Range
			}
			markdown += fmt.Sprintf("- **%s**: %d\n", label, d.Count)
		}
	}
	markdown += "\n"

	markdown += "## 最近 30 天趋势\n\n"
	markdown += "| 日期 | 平均分 | 审计数 | 覆盖 Prompt 数 |\n"
	markdown += "|------|--------|--------|----------------|\n"
	for _, t := range trends {
		if t.AuditCount > 0 {
			markdown += fmt.Sprintf("| %s | %.1f | %d | %d |\n", t.Date, t.AvgScore, t.AuditCount, t.PromptCount)
		}
	}
	markdown += "\n"

	if len(lowScores) > 0 {
		markdown += "## 低分提示词（需改进）\n\n"
		markdown += "| ID | 标题 | 分类 | 分数 | 审计日期 |\n"
		markdown += "|----|------|------|------|----------|\n"
		for _, p := range lowScores {
			markdown += fmt.Sprintf("| %d | %s | %s | %d | %s |\n",
				p.Id, p.Title, p.CategoryName, p.AuditScore, time.Unix(p.AuditDate, 0).Format("2006-01-02"))
		}
		markdown += "\n"
	}

	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\"seo-report-all.md\"")
	c.String(http.StatusOK, markdown)
}

// BatchAuditPromptSEO 批量审计 Prompt SEO
func BatchAuditPromptSEO(c *gin.Context) {
	var req struct {
		Ids []int `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "ids is required")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "最多一次审计 50 个提示词")
		return
	}

	// 异步批量审计
	go func(ids []int) {
		for _, id := range ids {
			prompt, err := model.GetPromptById(id)
			if err != nil {
				continue
			}
			service.AuditPromptSEO(prompt.Prompt)
		}
	}(req.Ids)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("已启动 %d 个提示词的批量审计任务", len(req.Ids)),
	})
}

// AdminRegeneratePromptSlugs 管理员接口：批量为 slug 为空的提示词生成 slug
func AdminRegeneratePromptSlugs(c *gin.Context) {
	updated, skipped, err := model.BatchGeneratePromptSlugs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"updated": updated,
		"skipped": skipped,
		"message": fmt.Sprintf("成功生成 %d 个 slug，跳过 %d 个", updated, skipped),
	})
}

// BatchTranslatePromptSEO 启动批量 SEO 翻译异步任务
func BatchTranslatePromptSEO(c *gin.Context) {
	var req struct {
		Ids         []int    `json:"ids" binding:"required"`
		TargetLangs []string `json:"target_langs" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(req.Ids) == 0 {
		common.ApiErrorMsg(c, "请选择要翻译的提示词")
		return
	}
	if len(req.Ids) > 50 {
		common.ApiErrorMsg(c, "单次最多处理 50 个提示词")
		return
	}

	taskID := service.StartSEOBatchTranslate(req.Ids, req.TargetLangs)
	common.ApiSuccess(c, gin.H{
		"task_id": taskID,
		"status":  "running",
		"total":   len(req.Ids),
		"message":   fmt.Sprintf("已启动 %d 个提示词的批量 SEO 翻译任务", len(req.Ids)),
	})
}

// GetBatchTranslatePromptSEOStatus 查询批量 SEO 翻译任务状态
func GetBatchTranslatePromptSEOStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id 不能为空")
		return
	}

	task := service.GetSEOBatchTask(taskID)
	if task == nil {
		common.ApiErrorMsg(c, "任务不存在或已过期")
		return
	}

	common.ApiSuccess(c, task)
}

// GetAutoTranslateStatus 查询自动翻译任务状态
func GetAutoTranslateStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		common.ApiErrorMsg(c, "task_id 不能为空")
		return
	}

	task := service.GetAutoTranslateTask(taskID)
	if task == nil {
		common.ApiErrorMsg(c, "任务不存在或已过期")
		return
	}

	common.ApiSuccess(c, task)
}
