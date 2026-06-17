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

// SyncSEOMonitorFromGSC 从 Google Search Console 同步 SEO 监控数据
// POST /api/admin/seo/monitor/sync-gsc
func SyncSEOMonitorFromGSC(c *gin.Context) {
	var req struct {
		SiteURL   string `json:"site_url"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	// 绑定可选参数，即使为空也能继续
	_ = c.ShouldBindJSON(&req)

	data, err := service.FetchGSCSearchAnalytics(req.SiteURL, req.StartDate, req.EndDate)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	service.UpdateSEOMonitorData(data)
	common.ApiSuccess(c, gin.H{
		"message": "已从 Google Search Console 同步监控数据",
		"data":    data,
	})
}
