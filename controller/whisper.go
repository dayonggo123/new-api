package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

const whisperMaxMemory = 32 << 20 // 32 MB

// WhisperTranscriptions 代理 APIMart Whisper-1 音频转录接口
// POST /api/public/audio/transcriptions
func WhisperTranscriptions(c *gin.Context) {
	setting := operation_setting.GetWhisperSetting()
	if !setting.WhisperEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "Whisper 接口未启用",
		})
		return
	}

	if setting.WhisperApiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "Whisper API Key 未配置",
		})
		return
	}

	baseURL := setting.WhisperBaseURL
	if baseURL == "" {
		baseURL = "https://api.apimart.ai"
	}

	upstreamURL := baseURL + "/v1/audio/transcriptions"

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(upstreamURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": fmt.Sprintf("request blocked: %v", err),
		})
		return
	}

	if err := c.Request.ParseMultipartForm(whisperMaxMemory); err != nil && err != http.ErrNotMultipart {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to parse Whisper multipart form: %s", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "failed to parse multipart form",
		})
		return
	}

	if c.Request.MultipartForm == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "request must be multipart/form-data",
		})
		return
	}

	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "missing file field",
		})
		return
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create Whisper form file: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to build upstream request",
		})
		return
	}
	if _, err := io.Copy(part, file); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to copy Whisper file: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to copy file",
		})
		return
	}

	for key, values := range c.Request.MultipartForm.Value {
		if key == "file" {
			continue
		}
		for _, value := range values {
			if err := writer.WriteField(key, value); err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to write Whisper field %s: %s", key, err.Error()))
			}
		}
	}

	if err := writer.Close(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to close Whisper multipart writer: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to build upstream request",
		})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, &body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create Whisper request: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to create upstream request",
		})
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+setting.WhisperApiKey)
	req.Header.Set("Accept", "*/*")

	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch Whisper: %s", err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "failed to fetch upstream",
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to read Whisper response: %s", err.Error()))
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "failed to read upstream response",
		})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

// WhisperSettingStatus 返回 Whisper 配置状态（仅管理员）
func WhisperSettingStatus(c *gin.Context) {
	setting := operation_setting.GetWhisperSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled":     setting.WhisperEnabled,
			"base_url":    setting.WhisperBaseURL,
			"api_key_set": setting.WhisperApiKey != "",
		},
	})
}
