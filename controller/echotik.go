package controller

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

// EchotikVideoRanklist 代理 EchoTik 视频榜单接口
// GET /api/public/echotik/video/ranklist
func EchotikVideoRanklist(c *gin.Context) {
	setting := operation_setting.GetEchotikSetting()
	if !setting.EchotikEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "EchoTik 接口未启用",
		})
		return
	}

	if setting.EchotikUsername == "" || setting.EchotikPassword == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "EchoTik 认证信息未配置",
		})
		return
	}

	baseURL := setting.EchotikBaseURL
	if baseURL == "" {
		baseURL = "https://open.echotik.live"
	}

	// 透传查询参数
	query := c.Request.URL.Query()
	upstreamURL := baseURL + "/api/v3/echotik/video/ranklist"
	if len(query) > 0 {
		upstreamURL = upstreamURL + "?" + query.Encode()
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(upstreamURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": fmt.Sprintf("request blocked: %v", err),
		})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create Echotik request: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to create upstream request",
		})
		return
	}

	// Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(setting.EchotikUsername + ":" + setting.EchotikPassword))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch Echotik: %s", err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "failed to fetch upstream",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to read Echotik response: %s", err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "failed to read upstream response",
		})
		return
	}

	// 透传状态码和响应体（保持 EchoTik 原始响应格式）
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// EchotikSettingStatus 返回 EchoTik 配置状态（仅管理员）
func EchotikSettingStatus(c *gin.Context) {
	setting := operation_setting.GetEchotikSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled": setting.EchotikEnabled,
			"base_url": setting.EchotikBaseURL,
			"username_set": setting.EchotikUsername != "",
			"password_set": setting.EchotikPassword != "",
		},
	})
}
