package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// FetchTikHubSingleVideo 请求 TikHub 获取单个 TikTok 视频数据。
// endpoint: /api/v1/tiktok/app/v3/fetch_one_video_v2?aweme_id=xxx
func FetchTikHubSingleVideo(ctx context.Context, awemeID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://api.tikhub.io"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_one_video_v2")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("aweme_id", awemeID)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	// 记录原始请求路由，便于调试
	logger.LogInfo(ctx, fmt.Sprintf("TikHub single video fetched: aweme_id=%s", awemeID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched aweme_id=%s", awemeID))

	return body, nil
}
