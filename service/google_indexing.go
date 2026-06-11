package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const indexingTimeout = 30 * time.Second

// GoogleIndexingRequest 索引提交请求
type GoogleIndexingRequest struct {
	URL  string `json:"url" binding:"required"`
	Type string `json:"type"` // "URL_UPDATED" or "URL_DELETED", default "URL_UPDATED"
}

// GoogleIndexingResult 索引提交结果
type GoogleIndexingResult struct {
	URL       string `json:"url"`
	Status    string `json:"status"`    // success / failed
	Message   string `json:"message"`   // 成功或失败信息
	ErrorCode string `json:"error_code,omitempty"`
}

// SubmitToGoogleIndexing 提交 URL 到 Google Indexing API
func SubmitToGoogleIndexing(url string, notifyType string) (*GoogleIndexingResult, error) {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GoogleIndexingAPIKey == "" {
		return nil, fmt.Errorf("google indexing api not configured")
	}

	if notifyType == "" {
		notifyType = "URL_UPDATED"
	}

	reqBody := map[string]interface{}{
		"url":  url,
		"type": notifyType,
	}

	jsonData, err := common.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: indexingTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		"https://indexing.googleapis.com/v3/urlNotifications:publish",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.GoogleIndexingAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := &GoogleIndexingResult{
		URL:    url,
		Status: "success",
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		common.DecodeJson(resp.Body, &errResp)
		result.Status = "failed"
		result.Message = errResp.Error.Message
		result.ErrorCode = errResp.Error.Status
		logger.LogError(context.Background(), fmt.Sprintf("google indexing failed for %s: %s", url, errResp.Error.Message))
		return result, fmt.Errorf("google indexing api error: %s", errResp.Error.Message)
	}

	result.Message = "URL submitted successfully"
	return result, nil
}

// BatchSubmitToGoogleIndexing 批量提交 URL
func BatchSubmitToGoogleIndexing(urls []string) []GoogleIndexingResult {
	var results []GoogleIndexingResult

	for _, url := range urls {
		result, err := SubmitToGoogleIndexing(url, "URL_UPDATED")
		if err != nil {
			results = append(results, GoogleIndexingResult{
				URL:     url,
				Status:  "failed",
				Message: err.Error(),
			})
		} else {
			results = append(results, *result)
		}
		// Google Indexing API 有速率限制，简单延迟
		time.Sleep(500 * time.Millisecond)
	}

	return results
}

// AutoSubmitAfterTranslation 翻译完成后自动提交索引
func AutoSubmitAfterTranslation(recordType string, recordID int, slug string) {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GoogleIndexingAPIKey == "" || cfg.SiteDomain == "" {
		return // 未配置则跳过
	}

	baseURL := cfg.SiteDomain
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	var url string
	switch recordType {
	case "article":
		url = fmt.Sprintf("%s/article/%s", baseURL, slug)
	case "prompt":
		url = fmt.Sprintf("%s/prompt/%s", baseURL, slug)
	default:
		return
	}

	// 异步提交，不阻塞主流程
	go func() {
		_, _ = SubmitToGoogleIndexing(url, "URL_UPDATED")
	}()
}

// GetIndexingStatus 获取 URL 的索引状态
func GetIndexingStatus(url string) (string, error) {
	cfg := operation_setting.GetSEOSetting()
	if cfg.GoogleIndexingAPIKey == "" {
		return "", fmt.Errorf("google indexing api not configured")
	}

	reqURL := fmt.Sprintf("https://indexing.googleapis.com/v3/urlNotifications/metadata?url=%s", url)

	client := &http.Client{Timeout: indexingTimeout}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+cfg.GoogleIndexingAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "not_indexed", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get indexing status: %d", resp.StatusCode)
	}

	var metadata struct {
		LatestUpdate struct {
			NotifyTime string `json:"notifyTime"`
			Type       string `json:"type"`
		} `json:"latestUpdate"`
		URL string `json:"url"`
	}

	if err := common.DecodeJson(resp.Body, &metadata); err != nil {
		return "", err
	}

	if metadata.LatestUpdate.Type == "URL_UPDATED" {
		return "indexed", nil
	}

	return "unknown", nil
}
