package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// TikHubSingleVideo 代理 TikHub 获取单个 TikTok 作品数据 V2
// GET /api/public/tikhub/tiktok/video?aweme_id=7350810998023949599
func TikHubSingleVideo(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	if !setting.TikHubEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "TikHub 接口未启用",
		})
		return
	}

	if setting.TikHubAPIKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "TikHub API Key 未配置",
		})
		return
	}

	awemeID := c.Query("aweme_id")
	if awemeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "aweme_id 不能为空",
		})
		return
	}

	body, err := service.FetchTikHubSingleVideo(c.Request.Context(), awemeID)
	if err != nil {
		logger.LogError(c.Request.Context(), err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 直接透传上游原始 JSON
	c.Data(http.StatusOK, "application/json", body)
}

// TikHubSettingStatus 返回 TikHub 配置状态（仅管理员）
func TikHubSettingStatus(c *gin.Context) {
	setting := operation_setting.GetTikHubSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":      setting.TikHubEnabled,
			"base_url":     setting.TikHubBaseURL,
			"api_key_set":  setting.TikHubAPIKey != "",
		},
	})
}
