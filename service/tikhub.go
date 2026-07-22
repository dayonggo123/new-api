package service

import (
	"bytes"
	"context"
	"encoding/json"
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
		baseURL = "https://heharse.cloud"
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

// FetchTikHubTweetDetail 请求 TikHub 获取 Twitter/X 推文详情（含视频地址）。
// endpoint: /api/v1/twitter/web/fetch_tweet_detail?tweet_id=xxx
func FetchTikHubTweetDetail(ctx context.Context, tweetID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/twitter/web/fetch_tweet_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("tweet_id", tweetID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub Twitter API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub tweet detail fetched: tweet_id=%s", tweetID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched tweet_id=%s", tweetID))

	return body, nil
}

// FetchTikHubCommentKeywords 请求 TikHub 获取视频评论关键词分析。
// endpoint: /api/v1/tiktok/analytics/fetch_comment_keywords?item_id=xxx
func FetchTikHubCommentKeywords(ctx context.Context, itemID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/analytics/fetch_comment_keywords")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("item_id", itemID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub comment keywords API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub comment keywords fetched: item_id=%s", itemID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched comment keywords for item_id=%s", itemID))

	return body, nil
}

// FetchTikHubSingleVideoByShareURL 请求 TikHub 根据分享链接获取单个 TikTok 视频数据。
// endpoint: /api/v1/tiktok/app/v3/fetch_one_video_by_share_url?share_url=xxx
func FetchTikHubSingleVideoByShareURL(ctx context.Context, shareURL string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_one_video_by_share_url")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("share_url", shareURL)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub API error (by share_url): status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub single video (by share_url) fetched: share_url=%s", shareURL))
	common.SysLog(fmt.Sprintf("[TikHub] fetched by share_url=%s", shareURL))

	return body, nil
}

// FetchTikHubMusicChartList 请求 TikHub 获取 TikTok 音乐排行榜。
// endpoint: /api/v1/tiktok/app/v3/fetch_music_chart_list?scene=0&cursor=0&count=50
func FetchTikHubMusicChartList(ctx context.Context, scene int, cursor int, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_music_chart_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("scene", fmt.Sprintf("%d", scene))
	q.Set("cursor", fmt.Sprintf("%d", cursor))
	q.Set("count", fmt.Sprintf("%d", count))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub music chart API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub music chart fetched: scene=%d, cursor=%d", scene, cursor))
	common.SysLog(fmt.Sprintf("[TikHub] fetched music chart: scene=%d", scene))

	return body, nil
}

// FetchTikHubTrendingSearchWords 请求 TikHub 获取每日趋势搜索关键词。
// endpoint: /api/v1/tiktok/web/fetch_trending_searchwords
func FetchTikHubTrendingSearchWords(ctx context.Context) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/web/fetch_trending_searchwords")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

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
		logger.LogError(ctx, fmt.Sprintf("TikHub trending search words API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, "TikHub trending search words fetched")
	common.SysLog("[TikHub] fetched trending search words")

	return body, nil
}

// FetchTikHubAccountHealthStatus 请求 TikHub 获取创作者账号健康状态。
// endpoint: /api/v1/tiktok/creator/get_account_health_status
func FetchTikHubAccountHealthStatus(ctx context.Context, cookie string, proxy string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_account_health_status")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie": cookie,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub account health status API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub account health status fetched")
	common.SysLog("[TikHub] fetched account health status")

	return respBody, nil
}

// FetchTikHubAccountInsightsOverview 请求 TikHub 获取创作者账号概览。
// endpoint: /api/v1/tiktok/creator/get_account_insights_overview
func FetchTikHubAccountInsightsOverview(ctx context.Context, cookie, startDate, proxy string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_account_insights_overview")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie":     cookie,
		"start_date": startDate,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub account insights overview API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub account insights overview fetched")
	common.SysLog("[TikHub] fetched account insights overview")

	return respBody, nil
}

// FetchTikHubVideoAnalyticsSummary 请求 TikHub 获取创作者视频概览。
// endpoint: /api/v1/tiktok/creator/get_video_analytics_summary
func FetchTikHubVideoAnalyticsSummary(ctx context.Context, cookie, proxy string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_video_analytics_summary")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie": cookie,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub video analytics summary API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub video analytics summary fetched")
	common.SysLog("[TikHub] fetched video analytics summary")

	return respBody, nil
}

// FetchTikHubProductRelatedVideos 请求 TikHub 获取同款商品关联视频。
// endpoint: /api/v1/tiktok/creator/get_product_related_videos
func FetchTikHubProductRelatedVideos(ctx context.Context, cookie, startDate, itemID, productID, proxy string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_product_related_videos")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie":     cookie,
		"start_date": startDate,
		"item_id":    itemID,
		"product_id": productID,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub product related videos API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub product related videos fetched")
	common.SysLog("[TikHub] fetched product related videos")

	return respBody, nil
}

// FetchTikHubProductDetail 请求 TikHub 获取 TikTok 商品详情数据 V2。
// endpoint: /api/v1/tiktok/app/v3/fetch_product_detail_v2?product_id=xxx
func FetchTikHubProductDetail(ctx context.Context, productID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_product_detail_v2")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("product_id", productID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub Product API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub product detail fetched: product_id=%s", productID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched product_id=%s", productID))

	return body, nil
}

// FetchTikHubTrendsHashtagList 请求 TikHub 获取热门标签榜单。
// endpoint: /api/v1/tiktok/ads/get_trends_hashtag_list
func FetchTikHubTrendsHashtagList(ctx context.Context, timeRange int, countryCode string, page int, limit int, industryID int64) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_trends_hashtag_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	if timeRange > 0 {
		q.Set("time_range", fmt.Sprintf("%d", timeRange))
	}
	if countryCode != "" {
		q.Set("country_code", countryCode)
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if industryID > 0 {
		q.Set("industry_id", fmt.Sprintf("%d", industryID))
	}
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
		logger.LogError(ctx, fmt.Sprintf("TikHub trends hashtag list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub trends hashtag list fetched: country=%s, time_range=%d", countryCode, timeRange))
	common.SysLog(fmt.Sprintf("[TikHub] fetched trends hashtag list: country=%s", countryCode))

	return body, nil
}

// FetchTikHubHotSellingProductsList 请求 TikHub 获取热卖商品列表。
// endpoint: /api/v1/tiktok/shop/web/fetch_hot_selling_products_list
func FetchTikHubHotSellingProductsList(ctx context.Context, region string, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_hot_selling_products_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	if region != "" {
		q.Set("region", region)
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
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
		logger.LogError(ctx, fmt.Sprintf("TikHub hot selling products list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub hot selling products list fetched: region=%s", region))
	common.SysLog(fmt.Sprintf("[TikHub] fetched hot selling products list: region=%s", region))

	return body, nil
}

// FetchTikHubVideoComments 请求 TikHub 获取单个视频评论数据。
// endpoint: /api/v1/tiktok/app/v3/fetch_video_comments
func FetchTikHubVideoComments(ctx context.Context, awemeID string, cursor int, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_video_comments")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("aweme_id", awemeID)
	q.Set("cursor", fmt.Sprintf("%d", cursor))
	q.Set("count", fmt.Sprintf("%d", count))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub video comments API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub video comments fetched: aweme_id=%s, cursor=%d", awemeID, cursor))
	common.SysLog(fmt.Sprintf("[TikHub] fetched video comments: aweme_id=%s", awemeID))

	return body, nil
}

// FetchTikHubVideoAudienceStats 请求 TikHub 获取视频受众分析数据。
// endpoint: /api/v1/tiktok/creator/get_video_audience_stats
func FetchTikHubVideoAudienceStats(ctx context.Context, cookie, startDate, itemID, proxy string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_video_audience_stats")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	bodyMap := map[string]interface{}{
		"cookie":     cookie,
		"start_date": startDate,
		"item_id":    itemID,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("marshal request body failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+setting.TikHubAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request tikhub failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub video audience stats API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub video audience stats fetched: item_id=%s", itemID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched video audience stats: item_id=%s", itemID))

	return respBody, nil
}

// FetchTikHubPostComment 请求 TikHub 获取作品评论列表。
// endpoint: /api/v1/tiktok/web/fetch_post_comment
func FetchTikHubPostComment(ctx context.Context, awemeID string, cursor int, count int, currentRegion string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/web/fetch_post_comment")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("aweme_id", awemeID)
	q.Set("cursor", fmt.Sprintf("%d", cursor))
	q.Set("count", fmt.Sprintf("%d", count))
	if currentRegion != "" {
		q.Set("current_region", currentRegion)
	}
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
		logger.LogError(ctx, fmt.Sprintf("TikHub post comment API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub post comment fetched: aweme_id=%s", awemeID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched post comment: aweme_id=%s", awemeID))

	return body, nil
}
