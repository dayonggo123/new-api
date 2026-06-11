package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== SEO Keyword Research ====================

// SEOResearchRequest 关键词研究请求
type SEOResearchRequest struct {
	SeedKeyword string `json:"seed_keyword" binding:"required"`
	Language    string `json:"language"`
}

// ResearchSEOKeywords 执行 AI 关键词研究
// POST /api/admin/seo/research
func ResearchSEOKeywords(c *gin.Context) {
	var req SEOResearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.ResearchKeywords(&service.SEOResearchRequest{
		SeedKeyword: req.SeedKeyword,
		Language:    req.Language,
	})
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}

// GetSEOQuickTemplates 获取预设的快速研究模板
// GET /api/admin/seo/research/templates
func GetSEOQuickTemplates(c *gin.Context) {
	templates := model.GetSEOQuickTemplates()
	common.ApiSuccess(c, templates)
}
