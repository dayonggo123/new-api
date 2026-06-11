package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// ==================== SEO Monitor ====================

// GetSEOMonitorData 获取 SEO 监控数据
// GET /api/admin/seo/monitor
func GetSEOMonitorData(c *gin.Context) {
	data := service.GetSEOMonitorData()
	common.ApiSuccess(c, data)
}

// GetSEOMonitorHistory 获取 SEO 监控历史
// GET /api/admin/seo/monitor/history?days=30
func GetSEOMonitorHistory(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30
	}

	history := service.GetSEOMonitorHistory(days)
	common.ApiSuccess(c, history)
}

// GetSEOHealthSummary 获取 SEO 健康摘要
// GET /api/admin/seo/monitor/summary
func GetSEOHealthSummary(c *gin.Context) {
	summary := service.GetSEOHealthSummary()
	common.ApiSuccess(c, summary)
}

// SimulateSEOMonitorData 模拟 SEO 监控数据（用于测试）
// POST /api/admin/seo/monitor/simulate
func SimulateSEOMonitorData(c *gin.Context) {
	data := service.SimulateMonitorData()
	common.ApiSuccess(c, data)
}

// UpdateSEOMonitorData 手动更新 SEO 监控数据
// POST /api/admin/seo/monitor/update
func UpdateSEOMonitorData(c *gin.Context) {
	var data service.SEOMonitorData
	if err := c.ShouldBindJSON(&data); err != nil {
		common.ApiError(c, err)
		return
	}

	service.UpdateSEOMonitorData(&data)
	common.ApiSuccess(c, gin.H{"message": "监控数据已更新"})
}
