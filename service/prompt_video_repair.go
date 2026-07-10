package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

// VideoUrlRepairResult 单条修复结果
type VideoUrlRepairResult struct {
	PromptID   int    `json:"prompt_id"`
	Title      string `json:"title"`
	OldVideoURL string `json:"old_video_url"`
	NewVideoURL string `json:"new_video_url"`
	Status     string `json:"status"` // "fixed" / "skipped" / "error"
	Message    string `json:"message"`
}

// RepairVideoUrlsResult 批量修复结果
type RepairVideoUrlsResult struct {
	Total   int                    `json:"total"`
	Fixed   int                    `json:"fixed"`
	Skipped int                    `json:"skipped"`
	Failed  int                    `json:"failed"`
	Items   []VideoUrlRepairResult `json:"items"`
}

// extractTweetIDFromSourceURL 从各种来源 URL 中提取 tweet_id
func extractTweetIDFromSourceURL(sourceURL string) string {
	if sourceURL == "" {
		return ""
	}
	// Twitter/X 直接链接: https://x.com/user/status/123456 或 https://twitter.com/user/status/123456
	for _, pattern := range []string{
		`x\.com/[^/]+/status/(\d+)`,
		`twitter\.com/[^/]+/status/(\d+)`,
		`x\.com/i/status/(\d+)`,
		`twitter\.com/i/status/(\d+)`,
	} {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(sourceURL)
		if len(matches) >= 2 {
			return matches[1]
		}
	}
	return ""
}

// extractVideoURLFromTikHubResponse 从 TikHub Twitter 响应中提取视频播放地址
func extractVideoURLFromTikHubResponse(body []byte) (string, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// TikHub 标准响应: { code: 200, data: "..." }
	dataRaw, ok := resp["data"]
	if !ok || dataRaw == nil {
		return "", fmt.Errorf("no data field in response")
	}

	// data 可能是 JSON 字符串或对象
	var dataObj map[string]interface{}
	switch d := dataRaw.(type) {
	case string:
		if err := json.Unmarshal([]byte(d), &dataObj); err != nil {
			return "", fmt.Errorf("parse data string: %w", err)
		}
	case map[string]interface{}:
		dataObj = d
	default:
		return "", fmt.Errorf("unexpected data type")
	}

	// 尝试从嵌套结构中提取视频地址
	// TikHub Twitter 返回结构通常包含 video_info 或类似字段
	videoURL := recursiveFindVideoURL(dataObj, "")
	if videoURL != "" {
		return videoURL, nil
	}

	return "", fmt.Errorf("video URL not found in tikhub response")
}

// recursiveFindVideoURL 递归搜索 JSON 对象中的视频播放地址
func recursiveFindVideoURL(obj interface{}, path string) string {
	switch v := obj.(type) {
	case map[string]interface{}:
		// 优先检查已知视频字段
		for _, field := range []string{"video_url", "playAddr", "play_addr", "stream_url", "url"} {
			if val, ok := v[field]; ok {
				if s, ok := val.(string); ok && isVideoURL(s) {
					return s
				}
			}
		}
		// 检查 url_list 数组（TikTok/Twitter 常见格式）
		if urlList, ok := v["url_list"].([]interface{}); ok && len(urlList) > 0 {
			if s, ok := urlList[0].(string); ok && isVideoURL(s) {
				return s
			}
		}
		// 递归子节点
		for key, child := range v {
			if result := recursiveFindVideoURL(child, path+"."+key); result != "" {
				return result
			}
		}
	case []interface{}:
		for i, item := range v {
			if result := recursiveFindVideoURL(item, fmt.Sprintf("%s[%d]", path, i)); result != "" {
				return result
			}
		}
	}
	return ""
}

// isVideoURL 判断字符串是否像视频播放地址
func isVideoURL(s string) bool {
	s = strings.ToLower(s)
	return (strings.Contains(s, ".mp4") ||
		strings.Contains(s, ".m3u8") ||
		strings.Contains(s, ".webm") ||
		strings.Contains(s, "videoplayback") ||
		strings.Contains(s, "ext_tw_video") ||
		strings.Contains(s, "video.twimg.com") ||
		strings.Contains(s, "cloudflarestream.com") && !strings.Contains(s, "thumb"))
}

// RepairPromptVideoUrls 批量修复提示词的视频 URL
// 扫描条件：media_type=video 且 (video_url 为空 或 video_url == cover_image_url)
func RepairPromptVideoUrls(ctx context.Context, dryRun bool) (*RepairVideoUrlsResult, error) {
	result := &RepairVideoUrlsResult{
		Items: []VideoUrlRepairResult{},
	}

	// 1. 查询所有需要修复的记录
	var prompts []model.Prompt
	err := model.DB.Where("media_type = ?", "video").
		Where("video_url = '' OR video_url IS NULL OR video_url = cover_image_url").
		Where("status = ?", 1).
		Find(&prompts).Error
	if err != nil {
		return nil, fmt.Errorf("query broken prompts failed: %w", err)
	}

	result.Total = len(prompts)
	logger.LogInfo(ctx, fmt.Sprintf("[VideoUrlRepair] found %d prompts needing video URL repair", result.Total))

	if result.Total == 0 {
		return result, nil
	}

	// 2. 逐条修复
	for _, p := range prompts {
		item := VideoUrlRepairResult{
			PromptID:    p.Id,
			Title:       p.Title,
			OldVideoURL: p.VideoUrl,
		}

		// 尝试从 source_url 提取 tweet_id
		tweetID := extractTweetIDFromSourceURL(p.SourceUrl)

		var newVideoURL string

		if tweetID != "" {
			// 方案 A：有 tweet_id，直接用 TikHub 获取推文详情
			body, err := FetchTikHubTweetDetail(ctx, tweetID)
			if err != nil {
				item.Status = "error"
				item.Message = fmt.Sprintf("tikhub fetch failed: %v", err)
				result.Failed++
				result.Items = append(result.Items, item)
				continue
			}
			newVideoURL, _ = extractVideoURLFromTikHubResponse(body)
		}

		if newVideoURL == "" && p.SourceUrl != "" {
			// 方案 B：尝试从 source_url 的 YouMind 页面推断
			// YouMind 聚合了 Twitter 内容，封面图可能包含 amplify_video_thumb 信息
			// 这里暂时跳过，后续可以加 YouMind 解析
			item.Status = "skipped"
			item.Message = "cannot extract tweet_id from source_url, and no fallback available"
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}

		if newVideoURL == "" {
			item.Status = "skipped"
			item.Message = "video URL not found in upstream response"
			result.Skipped++
			result.Items = append(result.Items, item)
			continue
		}

		// 3. 更新数据库
		item.NewVideoURL = newVideoURL
		if !dryRun {
			updateErr := model.DB.Model(&model.Prompt{}).Where("id = ?", p.Id).
				Update("video_url", newVideoURL).Error
			if updateErr != nil {
				item.Status = "error"
				item.Message = fmt.Sprintf("db update failed: %v", updateErr)
				result.Failed++
				result.Items = append(result.Items, item)
				continue
			}
		}

		item.Status = "fixed"
		result.Fixed++
		result.Items = append(result.Items, item)

		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}

	common.SysLog(fmt.Sprintf("[VideoUrlRepair] total=%d fixed=%d skipped=%d failed=%d (dryRun=%v)",
		result.Total, result.Fixed, result.Skipped, result.Failed, dryRun))

	return result, nil
}
