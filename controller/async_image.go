package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func AsyncImageTaskFetch(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	task := service.GetAsyncImageTask(taskID)
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found or expired"})
		return
	}

	body, statusCode, err := service.PollAsyncImageTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Try to parse JSON and rewrite image URLs with proxy
	var result map[string]any
	if err := common.Unmarshal(body, &result); err == nil {
		// APIMart / DuoYuanTanSuo 查询响应转换为 OpenAI Video 格式
		if task.ChannelType == constant.ChannelTypeAPIMart || task.ChannelType == constant.ChannelTypeDuoYuanTanSuo {
			result = convertTaskQueryToOpenAIVideo(result, task.TaskID)
		}
		rewriteImageURLsInResponse(result, c)
		c.JSON(statusCode, result)
		return
	}

	c.Data(statusCode, "application/json", body)
}

// convertTaskQueryToOpenAIVideo 将 APIMart/DuoYuanTanSuo 的查询响应 {code, data, error}
// 转换为下游客户端期望的 OpenAI Video 格式。
func convertTaskQueryToOpenAIVideo(result map[string]any, publicTaskID string) map[string]any {
	code, _ := result["code"].(float64)
	if code != 200 {
		return result
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return result
	}

	video := map[string]any{
		"id":         publicTaskID,
		"object":     "video",
		"status":     "unknown",
		"progress":   0,
		"created_at": 0,
	}

	if uuid, ok := data["id"].(string); ok {
		video["uuid"] = uuid
	}
	if status, ok := data["status"].(string); ok {
		switch status {
		case "completed":
			video["status"] = "completed"
			video["progress"] = 100
		case "failed":
			video["status"] = "failed"
			video["progress"] = 100
		case "pending", "processing":
			video["status"] = "in_progress"
		}
	}
	if progress, ok := data["progress"].(float64); ok {
		video["progress"] = int(progress)
	}
	if created, ok := data["created"].(float64); ok {
		video["created_at"] = int64(created)
	}
	if completed, ok := data["completed"].(float64); ok {
		video["completed_at"] = int64(completed)
	}

	// 提取结果 URL（image 优先，其次 video）
	if resultData, ok := data["result"].(map[string]any); ok {
		var resultURL string
		if images, ok := resultData["images"].([]any); ok && len(images) > 0 {
			if img, ok := images[0].(map[string]any); ok {
				if urls, ok := img["url"].([]any); ok && len(urls) > 0 {
					if url, ok := urls[0].(string); ok {
						resultURL = url
					}
				}
			}
		}
		if resultURL == "" {
			if videos, ok := resultData["videos"].([]any); ok && len(videos) > 0 {
				if v, ok := videos[0].(map[string]any); ok {
					if urls, ok := v["url"].([]any); ok && len(urls) > 0 {
						if url, ok := urls[0].(string); ok {
							resultURL = url
						}
					}
				}
			}
		}
		if resultURL != "" {
			video["metadata"] = map[string]any{
				"url": resultURL,
			}
		}
	}

	// 错误信息
	if errData, ok := data["error"].(map[string]any); ok {
		if msg, ok := errData["message"].(string); ok {
			video["error"] = map[string]any{
				"message": msg,
				"code":    "task_failed",
			}
		}
	}

	return video
}

func rewriteImageURLsInResponse(v any, c *gin.Context) {
	switch val := v.(type) {
	case map[string]any:
		for k, item := range val {
			if str, ok := item.(string); ok && (k == "url" || k == "image_url") && str != "" {
				val[k] = buildProxyURL(str, c)
			} else {
				rewriteImageURLsInResponse(item, c)
			}
		}
	case []any:
		for i, item := range val {
			rewriteImageURLsInResponse(item, c)
			val[i] = item
		}
	}
}

func buildProxyURL(upstreamURL string, c *gin.Context) string {
	proxyID := service.RegisterImageProxyURL(upstreamURL)
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return fmt.Sprintf("%s://%s/image-proxy/%s.png", scheme, host, proxyID)
}
