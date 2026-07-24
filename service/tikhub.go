package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
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

// FetchTikHubUserCountryByUsername 请求 TikHub 通过用户名获取用户账号国家地区。
// endpoint: /api/v1/tiktok/app/v3/fetch_user_country_by_username?username=xxx
func FetchTikHubUserCountryByUsername(ctx context.Context, username string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_user_country_by_username")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("username", username)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub user country API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub user country fetched: username=%s", username))
	common.SysLog(fmt.Sprintf("[TikHub] fetched user country: username=%s", username))

	return body, nil
}

// FetchTikHubGeneralSearchResult 请求 TikHub 获取综合搜索结果。
// endpoint: /api/v1/tiktok/app/v3/fetch_general_search_result?keyword=xxx
func FetchTikHubGeneralSearchResult(ctx context.Context, keyword string, offset, count, sortType, publishTime int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_general_search_result")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	if sortType > 0 {
		q.Set("sort_type", fmt.Sprintf("%d", sortType))
	}
	if publishTime > 0 {
		q.Set("publish_time", fmt.Sprintf("%d", publishTime))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub general search API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub general search result fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched general search: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubVideoSearchResult 请求 TikHub 获取视频搜索结果。
// endpoint: /api/v1/tiktok/app/v3/fetch_video_search_result?keyword=xxx
func FetchTikHubVideoSearchResult(ctx context.Context, keyword string, offset, count, sortType, publishTime int, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_video_search_result")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	if sortType > 0 {
		q.Set("sort_type", fmt.Sprintf("%d", sortType))
	}
	if publishTime > 0 {
		q.Set("publish_time", fmt.Sprintf("%d", publishTime))
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub video search API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub video search result fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched video search: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubUserSearchResult 请求 TikHub 获取用户搜索结果。
// endpoint: /api/v1/tiktok/app/v3/fetch_user_search_result?keyword=xxx
func FetchTikHubUserSearchResult(ctx context.Context, keyword string, offset, count int, followerCount, profileType, otherPref string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_user_search_result")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	if followerCount != "" {
		q.Set("user_search_follower_count", followerCount)
	}
	if profileType != "" {
		q.Set("user_search_profile_type", profileType)
	}
	if otherPref != "" {
		q.Set("user_search_other_pref", otherPref)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub user search API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub user search result fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched user search: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubMusicSearchResult 请求 TikHub 获取音乐搜索结果。
// endpoint: /api/v1/tiktok/app/v3/fetch_music_search_result?keyword=xxx
func FetchTikHubMusicSearchResult(ctx context.Context, keyword string, offset, count, filterBy, sortType int, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_music_search_result")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	if filterBy > 0 {
		q.Set("filter_by", fmt.Sprintf("%d", filterBy))
	}
	if sortType > 0 {
		q.Set("sort_type", fmt.Sprintf("%d", sortType))
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub music search API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub music search result fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched music search: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubHashtagSearchResult 请求 TikHub 获取话题搜索结果。
// endpoint: /api/v1/tiktok/app/v3/fetch_hashtag_search_result?keyword=xxx
func FetchTikHubHashtagSearchResult(ctx context.Context, keyword string, offset, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_hashtag_search_result")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub hashtag search API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub hashtag search result fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched hashtag search: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubMusicDetail 请求 TikHub 获取音乐详情。
// endpoint: /api/v1/tiktok/app/v3/fetch_music_detail?music_id=xxx
func FetchTikHubMusicDetail(ctx context.Context, musicID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_music_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("music_id", musicID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub music detail API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub music detail fetched: music_id=%s", musicID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched music detail: music_id=%s", musicID))

	return body, nil
}

// FetchTikHubMusicVideoList 请求 TikHub 获取音乐视频列表。
// endpoint: /api/v1/tiktok/app/v3/fetch_music_video_list?music_id=xxx
func FetchTikHubMusicVideoList(ctx context.Context, musicID string, cursor, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_music_video_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("music_id", musicID)
	if cursor > 0 {
		q.Set("cursor", fmt.Sprintf("%d", cursor))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub music video list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub music video list fetched: music_id=%s", musicID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched music video list: music_id=%s", musicID))

	return body, nil
}

// FetchTikHubHashtagDetail 请求 TikHub 获取话题详情。
// endpoint: /api/v1/tiktok/app/v3/fetch_hashtag_detail?ch_id=xxx
func FetchTikHubHashtagDetail(ctx context.Context, chID, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_hashtag_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("ch_id", chID)
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub hashtag detail API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub hashtag detail fetched: ch_id=%s", chID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched hashtag detail: ch_id=%s", chID))

	return body, nil
}

// FetchTikHubHashtagVideoList 请求 TikHub 获取话题视频列表。
// endpoint: /api/v1/tiktok/app/v3/fetch_hashtag_video_list?ch_id=xxx
func FetchTikHubHashtagVideoList(ctx context.Context, chID string, cursor, count int, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_hashtag_video_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("ch_id", chID)
	if cursor > 0 {
		q.Set("cursor", fmt.Sprintf("%d", cursor))
	}
	if count > 0 {
		q.Set("count", fmt.Sprintf("%d", count))
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub hashtag video list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub hashtag video list fetched: ch_id=%s", chID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched hashtag video list: ch_id=%s", chID))

	return body, nil
}

// FetchTikHubCreatorSearchInsights 请求 TikHub 获取创作者搜索洞察。
// endpoint: /api/v1/tiktok/app/v3/fetch_creator_search_insights
func FetchTikHubCreatorSearchInsights(ctx context.Context, offset, limit int, tab, languageFilters, categoryFilters, creatorSource string, forceRefresh bool) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_creator_search_insights")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if tab != "" {
		q.Set("tab", tab)
	}
	if languageFilters != "" {
		q.Set("language_filters", languageFilters)
	}
	if categoryFilters != "" {
		q.Set("category_filters", categoryFilters)
	}
	if creatorSource != "" {
		q.Set("creator_source", creatorSource)
	}
	if forceRefresh {
		q.Set("force_refresh", "true")
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
		logger.LogError(ctx, fmt.Sprintf("TikHub creator search insights API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, "TikHub creator search insights fetched")
	common.SysLog("[TikHub] fetched creator search insights")

	return body, nil
}

// FetchTikHubCreatorSearchInsightsDetail 请求 TikHub 获取创作者搜索洞察详情。
// endpoint: /api/v1/tiktok/app/v3/fetch_creator_search_insights_detail
func FetchTikHubCreatorSearchInsightsDetail(ctx context.Context, queryIDStr, timeRange string, startDate, endDate int64, dimensionList string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_creator_search_insights_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("query_id_str", queryIDStr)
	if timeRange != "" {
		q.Set("time_range", timeRange)
	}
	if startDate > 0 {
		q.Set("start_date", fmt.Sprintf("%d", startDate))
	}
	if endDate > 0 {
		q.Set("end_date", fmt.Sprintf("%d", endDate))
	}
	if dimensionList != "" {
		q.Set("dimension_list", dimensionList)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub creator search insights detail API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub creator search insights detail fetched: query_id=%s", queryIDStr))
	common.SysLog(fmt.Sprintf("[TikHub] fetched creator search insights detail: query_id=%s", queryIDStr))

	return body, nil
}

// FetchTikHubCreatorSearchInsightsVideos 请求 TikHub 获取创作者搜索洞察相关视频。
// endpoint: /api/v1/tiktok/app/v3/fetch_creator_search_insights_videos
func FetchTikHubCreatorSearchInsightsVideos(ctx context.Context, keyword string, offset, count int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/app/v3/fetch_creator_search_insights_videos")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("keyword", keyword)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub creator search insights videos API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub creator search insights videos fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched creator search insights videos: keyword=%s", keyword))

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

// FetchTikHubVideoListAnalytics 请求 TikHub 获取创作者视频列表分析。
// endpoint: /api/v1/tiktok/creator/get_video_list_analytics
func FetchTikHubVideoListAnalytics(ctx context.Context, cookie, startDate, rules, proxy string, page int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_video_list_analytics")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie":     cookie,
		"start_date": startDate,
		"page":       page,
	}
	if rules != "" {
		bodyMap["rules"] = rules
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
		logger.LogError(ctx, fmt.Sprintf("TikHub video list analytics API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub video list analytics fetched")
	common.SysLog("[TikHub] fetched video list analytics")

	return respBody, nil
}

// FetchTikHubProductAnalyticsList 请求 TikHub 获取创作者商品列表分析。
// endpoint: /api/v1/tiktok/creator/get_product_analytics_list
func FetchTikHubProductAnalyticsList(ctx context.Context, cookie, startDate, endDate, proxy string, page int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_product_analytics_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	// 构建请求体
	bodyMap := map[string]interface{}{
		"cookie":     cookie,
		"start_date": startDate,
		"end_date":   endDate,
		"page":       page,
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
		logger.LogError(ctx, fmt.Sprintf("TikHub product analytics list API error: status=%d, body=%s", resp.StatusCode, string(respBody)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.LogInfo(ctx, "TikHub product analytics list fetched")
	common.SysLog("[TikHub] fetched product analytics list")

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
// endpoint: /api/v1/tiktok/shop/web/fetch_product_detail_v2?product_id=xxx
func FetchTikHubProductDetail(ctx context.Context, productID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_detail_v2")
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

// FetchTikHubProductDetailV1 请求 TikHub 获取 TikTok 商品详情数据 V1 (桌面端-数据完整)。
// endpoint: /api/v1/tiktok/shop/web/fetch_product_detail?product_id=xxx
func FetchTikHubProductDetailV1(ctx context.Context, productID, sellerID, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("product_id", productID)
	if sellerID != "" {
		q.Set("seller_id", sellerID)
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub Product V1 API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub product detail V1 fetched: product_id=%s", productID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched product detail V1: product_id=%s", productID))

	return body, nil
}

// FetchTikHubProductReviewsV1 请求 TikHub 获取商品评论 V1。
// endpoint: /api/v1/tiktok/shop/web/fetch_product_reviews?product_id=xxx
func FetchTikHubProductReviewsV1(ctx context.Context, productID string, pageStart, pageSize, sortRule, filterType, filterValue int, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_reviews")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("product_id", productID)
	if pageStart > 0 {
		q.Set("page_start", fmt.Sprintf("%d", pageStart))
	}
	if pageSize > 0 {
		q.Set("page_size", fmt.Sprintf("%d", pageSize))
	}
	if sortRule > 0 {
		q.Set("sort_rule", fmt.Sprintf("%d", sortRule))
	}
	if filterType > 0 {
		q.Set("filter_type", fmt.Sprintf("%d", filterType))
	}
	if filterValue > 0 {
		q.Set("filter_value", fmt.Sprintf("%d", filterValue))
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub product reviews V1 API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub product reviews V1 fetched: product_id=%s", productID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched product reviews V1: product_id=%s", productID))

	return body, nil
}

// FetchTikHubProductReviewsV2 请求 TikHub 获取商品评论 V2。
// endpoint: /api/v1/tiktok/shop/web/fetch_product_reviews_v2?product_id=xxx
func FetchTikHubProductReviewsV2(ctx context.Context, productID string, pageStart, sortRule, filterType, filterValue int, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_reviews_v2")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("product_id", productID)
	if pageStart > 0 {
		q.Set("page_start", fmt.Sprintf("%d", pageStart))
	}
	if sortRule > 0 {
		q.Set("sort_rule", fmt.Sprintf("%d", sortRule))
	}
	if filterType > 0 {
		q.Set("filter_type", fmt.Sprintf("%d", filterType))
	}
	if filterValue > 0 {
		q.Set("filter_value", fmt.Sprintf("%d", filterValue))
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub product reviews V2 API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub product reviews V2 fetched: product_id=%s", productID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched product reviews V2: product_id=%s", productID))

	return body, nil
}

// FetchTikHubSellerProductsList 请求 TikHub 获取商家商品列表 V1。
// endpoint: /api/v1/tiktok/shop/web/fetch_seller_products_list
func FetchTikHubSellerProductsList(ctx context.Context, sellerID, searchParams, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_seller_products_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("seller_id", sellerID)
	if searchParams != "" {
		q.Set("search_params", searchParams)
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub seller products list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub seller products list fetched: seller_id=%s", sellerID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched seller products list: seller_id=%s", sellerID))

	return body, nil
}

// FetchTikHubSearchProductsList 请求 TikHub 搜索商品列表 V1。
// endpoint: /api/v1/tiktok/shop/web/fetch_search_products_list
func FetchTikHubSearchProductsList(ctx context.Context, searchWord string, offset int, pageToken, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_search_products_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("search_word", searchWord)
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if pageToken != "" {
		q.Set("page_token", pageToken)
	}
	if region != "" {
		q.Set("region", region)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub search products list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub search products list fetched: search_word=%s", searchWord))
	common.SysLog(fmt.Sprintf("[TikHub] fetched search products list: search_word=%s", searchWord))

	return body, nil
}

// FetchTikHubHotSellingProductsListV1 请求 TikHub 获取热卖商品列表。
// endpoint: /api/v1/tiktok/shop/web/fetch_hot_selling_products_list
func FetchTikHubHotSellingProductsListV1(ctx context.Context, region string, count int) ([]byte, error) {
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

	logger.LogInfo(ctx, "TikHub hot selling products list fetched")
	common.SysLog("[TikHub] fetched hot selling products list")

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

// =============================================================================
// TikTok Ads API - 广告搜索与分析
// =============================================================================

// FetchTikHubSearchAds 请求 TikHub 搜索广告。
// endpoint: /api/v1/tiktok/ads/search_ads
func FetchTikHubSearchAds(ctx context.Context, keyword string, objective, like, period int, industry, page, limit int, orderBy, countryCode string, adFormat int, adLanguage string, searchID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/search_ads")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	if objective > 0 {
		q.Set("objective", fmt.Sprintf("%d", objective))
	}
	if like > 0 {
		q.Set("like", fmt.Sprintf("%d", like))
	}
	if period > 0 {
		q.Set("period", fmt.Sprintf("%d", period))
	}
	if industry > 0 {
		q.Set("industry", fmt.Sprintf("%d", industry))
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if orderBy != "" {
		q.Set("order_by", orderBy)
	}
	if countryCode != "" {
		q.Set("country_code", countryCode)
	}
	if adFormat > 0 {
		q.Set("ad_format", fmt.Sprintf("%d", adFormat))
	}
	if adLanguage != "" {
		q.Set("ad_language", adLanguage)
	}
	if searchID != "" {
		q.Set("search_id", searchID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub search ads API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub search ads fetched: keyword=%s", keyword))
	common.SysLog(fmt.Sprintf("[TikHub] fetched search ads: keyword=%s", keyword))

	return body, nil
}

// FetchTikHubTopAdsSpotlight 请求 TikHub 获取热门广告聚光灯。
// endpoint: /api/v1/tiktok/ads/get_top_ads_spotlight
func FetchTikHubTopAdsSpotlight(ctx context.Context, industry string, page, limit int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_top_ads_spotlight")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	if industry != "" {
		q.Set("industry", industry)
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub top ads spotlight API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub top ads spotlight fetched: industry=%s", industry))
	common.SysLog(fmt.Sprintf("[TikHub] fetched top ads spotlight: industry=%s", industry))

	return body, nil
}

// FetchTikHubAdKeyframeAnalysis 请求 TikHub 获取广告关键帧分析。
// endpoint: /api/v1/tiktok/ads/get_ad_keyframe_analysis
func FetchTikHubAdKeyframeAnalysis(ctx context.Context, materialID, metric string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_ad_keyframe_analysis")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("material_id", materialID)
	if metric != "" {
		q.Set("metric", metric)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub ad keyframe analysis API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub ad keyframe analysis fetched: material_id=%s", materialID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched ad keyframe analysis: material_id=%s", materialID))

	return body, nil
}

// FetchTikHubAdPercentile 请求 TikHub 获取广告百分位数据。
// endpoint: /api/v1/tiktok/ads/get_ad_percentile
func FetchTikHubAdPercentile(ctx context.Context, materialID, metric string, periodType int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_ad_percentile")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("material_id", materialID)
	if metric != "" {
		q.Set("metric", metric)
	}
	if periodType > 0 {
		q.Set("period_type", fmt.Sprintf("%d", periodType))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub ad percentile API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub ad percentile fetched: material_id=%s", materialID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched ad percentile: material_id=%s", materialID))

	return body, nil
}

// FetchTikHubAdInteractiveAnalysis 请求 TikHub 获取广告互动分析。
// endpoint: /api/v1/tiktok/ads/get_ad_interactive_analysis
func FetchTikHubAdInteractiveAnalysis(ctx context.Context, materialID, metricType string, periodType int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_ad_interactive_analysis")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("material_id", materialID)
	if metricType != "" {
		q.Set("metric_type", metricType)
	}
	if periodType > 0 {
		q.Set("period_type", fmt.Sprintf("%d", periodType))
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
		logger.LogError(ctx, fmt.Sprintf("TikHub ad interactive analysis API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub ad interactive analysis fetched: material_id=%s", materialID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched ad interactive analysis: material_id=%s", materialID))

	return body, nil
}

// FetchTikHubTrendsHashtagDetail 请求 TikHub 获取热门标签详情。
// endpoint: /api/v1/tiktok/ads/get_trends_hashtag_detail
func FetchTikHubTrendsHashtagDetail(ctx context.Context, hashtagID string, timeRange int, countryCode string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/ads/get_trends_hashtag_detail")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("hashtag_id", hashtagID)
	if timeRange > 0 {
		q.Set("time_range", fmt.Sprintf("%d", timeRange))
	}
	if countryCode != "" {
		q.Set("country_code", countryCode)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub trends hashtag detail API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub trends hashtag detail fetched: hashtag_id=%s", hashtagID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched trends hashtag detail: hashtag_id=%s", hashtagID))

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

// FetchTikHubVideoMetrics 请求 TikHub 获取视频统计数据。
// endpoint: /api/v1/tiktok/analytics/fetch_video_metrics
func FetchTikHubVideoMetrics(ctx context.Context, itemID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/analytics/fetch_video_metrics")
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
		logger.LogError(ctx, fmt.Sprintf("TikHub video metrics API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub video metrics fetched: item_id=%s", itemID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched video metrics: item_id=%s", itemID))

	return body, nil
}

// FetchTikHubDetectFakeViews 请求 TikHub 检测视频虚假流量分析。
// endpoint: /api/v1/tiktok/analytics/detect_fake_views
func FetchTikHubDetectFakeViews(ctx context.Context, itemID string, contentCategory string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/analytics/detect_fake_views")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("item_id", itemID)
	if contentCategory != "" {
		q.Set("content_category", contentCategory)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub detect fake views API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub detect fake views fetched: item_id=%s", itemID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched detect fake views: item_id=%s", itemID))

	return body, nil
}

// FetchTikHubCreatorInfoAndMilestones 请求 TikHub 获取创作者信息和里程碑数据。
// endpoint: /api/v1/tiktok/analytics/fetch_creator_info_and_milestones
func FetchTikHubCreatorInfoAndMilestones(ctx context.Context, userID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/analytics/fetch_creator_info_and_milestones")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("user_id", userID)
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
		logger.LogError(ctx, fmt.Sprintf("TikHub creator info and milestones API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub creator info and milestones fetched: user_id=%s", userID))
	common.SysLog(fmt.Sprintf("[TikHub] fetched creator info and milestones: user_id=%s", userID))

	return body, nil
}

// FetchTikHubAccountViolationList 请求 TikHub 获取创作者账号违规记录列表。
// endpoint: /api/v1/tiktok/creator/get_account_violation_list
func FetchTikHubAccountViolationList(ctx context.Context, cookie string, proxy string, page int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/creator/get_account_violation_list")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	bodyMap := map[string]interface{}{
		"cookie": cookie,
	}
	if proxy != "" {
		bodyMap["proxy"] = proxy
	}
	if page > 0 {
		bodyMap["page"] = page
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read tikhub response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("TikHub account violation list API error: status=%d, body=%s", resp.StatusCode, string(body)))
		return nil, fmt.Errorf("tikhub api returned status %d: %s", resp.StatusCode, string(body))
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub account violation list fetched: page=%d", page))
	common.SysLog(fmt.Sprintf("[TikHub] fetched account violation list: page=%d", page))

	return body, nil
}

// =============================================================================
// 整合报告 API - Product Analysis Report
// =============================================================================

// FetchProductAnalysisReport 获取商品分析报告
// 整合商品详情、评论、关联视频等数据
func FetchProductAnalysisReport(ctx context.Context, productID string, region string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	// 并行获取多个接口数据
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		ProductDetail []byte
		ProductReviews []byte
		RelatedVideos []byte
		HotProducts []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 1. 获取商品详情
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := fetchProductDetail(ctx, productID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.ProductDetail = body
		}
	}()

	// 2. 获取商品评论
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := fetchProductReviews(ctx, productID, 1, 20)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.ProductReviews = body
		}
	}()

	// 3. 获取关联视频
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubProductRelatedVideos(ctx, "", "", "", productID, "")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.RelatedVideos = body
		}
	}()

	// 4. 获取热卖商品列表（同类目对比）
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubHotSellingProductsList(ctx, region, 20)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.HotProducts = body
		}
	}()

	wg.Wait()

	if hasError {
		return nil, fmt.Errorf("failed to fetch product analysis: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"product_detail":   json.RawMessage(result.ProductDetail),
		"product_reviews": json.RawMessage(result.ProductReviews),
		"related_videos":  json.RawMessage(result.RelatedVideos),
		"hot_products":    json.RawMessage(result.HotProducts),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub product analysis report generated: product_id=%s", productID))
	common.SysLog(fmt.Sprintf("[TikHub] generated product analysis report: product_id=%s", productID))

	return responseBytes, nil
}

// 辅助函数：获取商品详情
func fetchProductDetail(ctx context.Context, productID string) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_detail_v2")
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
		return nil, fmt.Errorf("tikhub api returned status %d", resp.StatusCode)
	}

	return body, nil
}

// 辅助函数：获取商品评论
func fetchProductReviews(ctx context.Context, productID string, page, limit int) ([]byte, error) {
	setting := operation_setting.GetTikHubSetting()
	baseURL := setting.TikHubBaseURL
	if baseURL == "" {
		baseURL = "https://heharse.cloud"
	}

	reqURL, err := url.Parse(baseURL + "/api/v1/tiktok/shop/web/fetch_product_reviews_v2")
	if err != nil {
		return nil, fmt.Errorf("invalid tikhub base url: %w", err)
	}

	q := reqURL.Query()
	q.Set("product_id", productID)
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
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
		return nil, fmt.Errorf("tikhub api returned status %d", resp.StatusCode)
	}

	return body, nil
}

// =============================================================================
// 整合报告 API - Creator Diagnosis Report
// =============================================================================

// FetchCreatorDiagnosisReport 获取创作者诊断报告
// 整合账号概览、健康状态、违规记录、视频列表、受众分析
func FetchCreatorDiagnosisReport(ctx context.Context, cookie string, proxy string) ([]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		AccountOverview  []byte
		HealthStatus     []byte
		ViolationList    []byte
		VideoAnalytics   []byte
		AudienceStats    []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 并行获取多个接口数据
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAccountInsightsOverview(ctx, cookie, "", proxy)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.AccountOverview = body
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAccountHealthStatus(ctx, cookie, proxy)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.HealthStatus = body
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAccountViolationList(ctx, cookie, proxy, 1)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.ViolationList = body
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubVideoListAnalytics(ctx, cookie, "", "", proxy, 1)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.VideoAnalytics = body
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		// 需要先获取视频ID才能分析受众，这里传空获取默认
		body, err := FetchTikHubVideoAudienceStats(ctx, cookie, "", "", proxy)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.AudienceStats = body
		}
	}()

	wg.Wait()

	if hasError {
		return nil, fmt.Errorf("failed to fetch creator diagnosis: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"account_overview":  json.RawMessage(result.AccountOverview),
		"health_status":    json.RawMessage(result.HealthStatus),
		"violation_list":   json.RawMessage(result.ViolationList),
		"video_analytics":  json.RawMessage(result.VideoAnalytics),
		"audience_stats":  json.RawMessage(result.AudienceStats),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, "TikHub creator diagnosis report generated")
	common.SysLog("[TikHub] generated creator diagnosis report")

	return responseBytes, nil
}

// =============================================================================
// 整合报告 API - Ad Creative Analysis Report
// =============================================================================

// FetchAdCreativeAnalysisReport 获取广告创意分析报告
func FetchAdCreativeAnalysisReport(ctx context.Context, materialID string, industry string) ([]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		TopSpotlight    []byte
		KeyframeAnalysis []byte
		PercentileData  []byte
		InteractiveData  []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 1. 热门广告聚光灯
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubTopAdsSpotlight(ctx, industry, 1, 20)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.TopSpotlight = body
		}
	}()

	// 2. 关键帧分析
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAdKeyframeAnalysis(ctx, materialID, "retain_ctr")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.KeyframeAnalysis = body
		}
	}()

	// 3. 百分位数据
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAdPercentile(ctx, materialID, "ctr_percentile", 180)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.PercentileData = body
		}
	}()

	// 4. 互动分析
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubAdInteractiveAnalysis(ctx, materialID, "remain", 180)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.InteractiveData = body
		}
	}()

	wg.Wait()

	if hasError {
		return nil, fmt.Errorf("failed to fetch ad creative analysis: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"top_spotlight":     json.RawMessage(result.TopSpotlight),
		"keyframe_analysis": json.RawMessage(result.KeyframeAnalysis),
		"percentile_data":   json.RawMessage(result.PercentileData),
		"interactive_data":  json.RawMessage(result.InteractiveData),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub ad creative analysis report generated: material_id=%s", materialID))
	common.SysLog(fmt.Sprintf("[TikHub] generated ad creative analysis report: material_id=%s", materialID))

	return responseBytes, nil
}

// =============================================================================
// 整合报告 API - Content Trends Report
// =============================================================================

// FetchContentTrendsReport 获取内容趋势报告
func FetchContentTrendsReport(ctx context.Context, countryCode string, timeRange int, industryID int) ([]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		HashtagList    []byte
		TrendingWords  []byte
		MusicChart     []byte
		SearchResults  []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 1. 热门标签
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubTrendsHashtagList(ctx, timeRange, countryCode, 1, 30, int64(industryID))
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.HashtagList = body
		}
	}()

	// 2. 趋势搜索词
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubTrendingSearchWords(ctx)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.TrendingWords = body
		}
	}()

	// 3. 音乐排行
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubMusicChartList(ctx, 1, 0, 20)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.MusicChart = body
		}
	}()

	// 4. 综合搜索（可选关键词）
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubGeneralSearchResult(ctx, "", 0, 20, 0, 0)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.SearchResults = body
		}
	}()

	wg.Wait()

	if hasError {
		return nil, fmt.Errorf("failed to fetch content trends: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"hashtag_list":    json.RawMessage(result.HashtagList),
		"trending_words":  json.RawMessage(result.TrendingWords),
		"music_chart":     json.RawMessage(result.MusicChart),
		"search_results":  json.RawMessage(result.SearchResults),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, "TikHub content trends report generated")
	common.SysLog("[TikHub] generated content trends report")

	return responseBytes, nil
}

// =============================================================================
// 整合报告 API - Video Deep Analysis Report
// =============================================================================

// FetchVideoAnalysisReport 获取视频深度分析报告
func FetchVideoAnalysisReport(ctx context.Context, awemeID string, cookie, proxy string) ([]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		VideoData     []byte
		VideoMetrics  []byte
		FakeViews     []byte
		Comments      []byte
		Keywords      []byte
		AudienceStats []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 1. 视频数据
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubSingleVideo(ctx, awemeID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.VideoData = body
		}
	}()

	// 2. 视频统计
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubVideoMetrics(ctx, awemeID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.VideoMetrics = body
		}
	}()

	// 3. 虚假流量检测
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubDetectFakeViews(ctx, awemeID, "")
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.FakeViews = body
		}
	}()

	// 4. 视频评论
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubVideoComments(ctx, awemeID, 0, 50)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.Comments = body
		}
	}()

	// 5. 评论关键词
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubCommentKeywords(ctx, awemeID)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.Keywords = body
		}
	}()

	// 6. 受众分析（可选，需要 cookie）
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubVideoAudienceStats(ctx, cookie, "", awemeID, proxy)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			// 不阻塞，受众分析失败不报错
			result.AudienceStats = []byte("{}")
		} else {
			result.AudienceStats = body
		}
	}()

	wg.Wait()

	if hasError {
		return nil, fmt.Errorf("failed to fetch video analysis: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"video_data":      json.RawMessage(result.VideoData),
		"video_metrics":   json.RawMessage(result.VideoMetrics),
		"fake_views":      json.RawMessage(result.FakeViews),
		"comments":        json.RawMessage(result.Comments),
		"keywords":        json.RawMessage(result.Keywords),
		"audience_stats":  json.RawMessage(result.AudienceStats),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub video analysis report generated: aweme_id=%s", awemeID))
	common.SysLog(fmt.Sprintf("[TikHub] generated video analysis report: aweme_id=%s", awemeID))

	return responseBytes, nil
}

// =============================================================================
// 整合报告 API - Competitor Monitor Report
// =============================================================================

// FetchCompetitorMonitorReport 获取竞品监控报告
func FetchCompetitorMonitorReport(ctx context.Context, sellerID string, region string) ([]byte, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	type PartialResult struct {
		SellerProducts   []byte
		HashtagVideos   []byte
		ProductRelated  []byte
		SearchProducts  []byte
		HotProducts     []byte
	}

	result := PartialResult{}
	hasError := false
	var errorMsg string

	// 1. 商家商品列表
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubSellerProductsList(ctx, sellerID, "", region)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.SellerProducts = body
		}
	}()

	// 2. 搜索商品列表（获取竞品关键词）
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 使用热门类目关键词搜索
		body, err := FetchTikHubSearchProductsList(ctx, "", 0, "", region)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.SearchProducts = body
		}
	}()

	// 3. 热卖商品（市场基准）
	wg.Add(1)
	go func() {
		defer wg.Done()
		body, err := FetchTikHubHotSellingProductsList(ctx, region, 50)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			hasError = true
			errorMsg = err.Error()
		} else {
			result.HotProducts = body
		}
	}()

	wg.Wait()

	// 注意：hashtag_videos 和 product_related 需要额外参数，这里简化处理

	if hasError {
		return nil, fmt.Errorf("failed to fetch competitor monitor: %s", errorMsg)
	}

	// 整合数据
	response := map[string]interface{}{
		"seller_products":   json.RawMessage(result.SellerProducts),
		"search_products":  json.RawMessage(result.SearchProducts),
		"hot_products":     json.RawMessage(result.HotProducts),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	logger.LogInfo(ctx, fmt.Sprintf("TikHub competitor monitor report generated: seller_id=%s", sellerID))
	common.SysLog(fmt.Sprintf("[TikHub] generated competitor monitor report: seller_id=%s", sellerID))

	return responseBytes, nil
}
