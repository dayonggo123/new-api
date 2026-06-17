package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== SEO Optimization Queue ====================

// GetLowCTROpportunities 获取低 CTR 回流机会
// GET /api/admin/seo/optimization-queue/low-ctr
func GetLowCTROpportunities(c *gin.Context) {
	ctrStr := c.DefaultQuery("ctr_threshold", "5.0")
	ctrThreshold, _ := strconv.ParseFloat(ctrStr, 64)
	if ctrThreshold <= 0 {
		ctrThreshold = 5.0
	}

	minImpressionsStr := c.DefaultQuery("min_impressions", "100")
	minImpressions, _ := strconv.Atoi(minImpressionsStr)
	if minImpressions <= 0 {
		minImpressions = 100
	}

	items := service.GetLowCTROpportunities(ctrThreshold, minImpressions)
	common.ApiSuccess(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

// GetRankingDropOpportunities 获取排名下降/低位回流机会
// GET /api/admin/seo/optimization-queue/ranking-drop
func GetRankingDropOpportunities(c *gin.Context) {
	positionStr := c.DefaultQuery("position_threshold", "10.0")
	positionThreshold, _ := strconv.ParseFloat(positionStr, 64)
	if positionThreshold <= 0 {
		positionThreshold = 10.0
	}

	changeStr := c.DefaultQuery("change_threshold", "-1.0")
	changeThreshold, _ := strconv.ParseFloat(changeStr, 64)
	// changeThreshold 应为负数，表示下降；如果传了正数则取反
	if changeThreshold > 0 {
		changeThreshold = -changeThreshold
	}

	items := service.GetRankingDropOpportunities(positionThreshold, changeThreshold)
	common.ApiSuccess(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

// AddSEOOptimizationQueueItem 添加优化队列项
// POST /api/admin/seo/optimization-queue
func AddSEOOptimizationQueueItem(c *gin.Context) {
	var req service.OptimizationQueueAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	item, err := service.AddToOptimizationQueue(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, item)
}

// ListSEOOptimizationQueue 获取优化队列列表
// GET /api/admin/seo/optimization-queue
func ListSEOOptimizationQueue(c *gin.Context) {
	status := c.DefaultQuery("status", "all")
	limitStr := c.DefaultQuery("limit", "20")
	pageStr := c.DefaultQuery("page", "1")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 20
	}
	page, _ := strconv.Atoi(pageStr)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	items, total, err := service.ListOptimizationQueue(status, limit, offset)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// UpdateSEOOptimizationQueueItem 更新优化队列项状态
// PUT /api/admin/seo/optimization-queue/:id/status
func UpdateSEOOptimizationQueueItem(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req struct {
		Status     string `json:"status" binding:"required"`
		ScoreAfter int    `json:"score_after"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.UpdateOptimizationQueueStatus(id, req.Status, req.ScoreAfter); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"message": "状态已更新"})
}
