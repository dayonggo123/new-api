package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== Content Generator ====================

// ContentGenerationRequest 内容生成请求
type ContentGenerationRequest struct {
	Type          string   `json:"type" binding:"required"` // article 或 prompt
	Keywords      []string `json:"keywords" binding:"required"`
	Language      string   `json:"language"`
	AutoSEO       bool     `json:"auto_seo"`
	AutoGEO       bool     `json:"auto_geo"`
	AutoTranslate bool     `json:"auto_translate"`
	AutoPublish   bool     `json:"auto_publish"`
}

// GenerateContent 生成内容（文章或 Prompt）
// POST /api/admin/content/generate
func GenerateContent(c *gin.Context) {
	var req ContentGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.GenerateContent(&service.ContentGenerationRequest{
		Type:          service.ContentGenerationType(req.Type),
		Keywords:      req.Keywords,
		Language:      req.Language,
		AutoSEO:       req.AutoSEO,
		AutoGEO:       req.AutoGEO,
		AutoTranslate: req.AutoTranslate,
		AutoPublish:   req.AutoPublish,
	})
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}
