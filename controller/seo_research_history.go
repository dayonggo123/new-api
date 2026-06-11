package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ==================== SEO Research History ====================

// ListSEOResearchHistories 获取关键词研究历史列表
// GET /api/admin/seo/research/history
func ListSEOResearchHistories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	histories, total, err := model.GetSEOResearchHistories(page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"histories": histories,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSEOResearchHistory 获取单条研究历史详情
// GET /api/admin/seo/research/history/:id
func GetSEOResearchHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	history, err := model.GetSEOResearchHistoryByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, history)
}

// DeleteSEOResearchHistory 删除研究历史
// DELETE /api/admin/seo/research/history/:id
func DeleteSEOResearchHistory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := model.DeleteSEOResearchHistory(id); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"message": "deleted"})
}
