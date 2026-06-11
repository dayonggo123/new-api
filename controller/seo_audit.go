package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== SEO Audit ====================

// GetSEOAudit 获取指定内容的 SEO 审计报告
// GET /api/admin/seo/audit?type=article&id=123
func GetSEOAudit(c *gin.Context) {
	recordType := c.Query("type")
	idStr := c.Query("id")

	if recordType == "" || idStr == "" {
		common.ApiErrorMsg(c, "type 和 id 参数必填")
		return
	}

	recordID, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.AuditSEO(recordType, recordID)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, result)
}

// ==================== Internal Link Suggestions ====================

// GetInternalLinkSuggestions 获取内链建议
// GET /api/admin/seo/internal-links?type=article&id=123&limit=5
func GetInternalLinkSuggestions(c *gin.Context) {
	recordType := c.Query("type")
	idStr := c.Query("id")
	limitStr := c.DefaultQuery("limit", "5")

	if recordType == "" || idStr == "" {
		common.ApiErrorMsg(c, "type 和 id 参数必填")
		return
	}

	recordID, err := strconv.Atoi(idStr)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 5
	}

	suggestions, err := service.SuggestInternalLinks(recordType, recordID, limit)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	common.ApiSuccess(c, suggestions)
}
