package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func AsyncImageTaskFetch(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	// 同步图片异步化渠道（OpenAI/Gemini/VolcEngine）直接读 DB，返回统一任务包装格式。
	userID := c.GetInt("id")
	if userID > 0 {
		dbTask, exists, err := service.GetImageTaskWorkerPoolManager().Queue().GetTaskByID(userID, taskID)
		if err == nil && exists && dbTask != nil {
			channel, channelErr := model.GetChannelById(dbTask.ChannelId, true)
			if channelErr == nil && channel != nil && isSyncImageAsyncChannel(channel.Type) {
				c.JSON(http.StatusOK, buildImageGenerationTaskResponse(dbTask))
				return
			}
		}
	}

	task := service.GetAsyncImageTask(taskID)
	if task == nil {
		// New-API 重启后内存丢失，尝试从 DB 恢复
		common.SysLog(fmt.Sprintf("[AsyncImageTaskFetch] task %s not in memory, trying DB recovery", taskID))
		task = service.RecoverAsyncImageTaskFromDB(taskID)
		if task == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found or expired"})
			return
		}
		service.StoreAsyncImageTask(task)
		common.SysLog(fmt.Sprintf("[AsyncImageTaskFetch] task %s recovered from DB, channelType=%d, upstreamTaskID=%s", taskID, task.ChannelType, task.UpstreamTaskID))
	}

	body, statusCode, err := service.PollAsyncImageTask(task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Try to parse JSON and rewrite image URLs with proxy
	var result map[string]any
	if err := common.Unmarshal(body, &result); err == nil {
		// APIMart / DuoYuanTanSuo / 章鱼哥 查询响应转换为 OpenAI Video 格式
		if task.ChannelType == constant.ChannelTypeAPIMart || task.ChannelType == constant.ChannelTypeDuoYuanTanSuo || task.ChannelType == constant.ChannelTypeZhangyuge {
			result = convertTaskQueryToOpenAIVideo(result, task.TaskID)
		}
		// 非同步图片异步化渠道需要重写 URL。
		if !isSyncImageAsyncChannel(task.ChannelType) {
			rewriteImageURLsInResponse(result, c)
		}
		c.JSON(statusCode, result)
		return
	}

	c.Data(statusCode, "application/json", body)
}

// AsyncImageTaskCancel cancels an image generation task that is still in
// QUEUED or IN_PROGRESS. It refunds the pre-consumed quota and returns the
// updated task status.
func AsyncImageTaskCancel(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	userID := c.GetInt("id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	queue := service.GetImageTaskWorkerPoolManager().Queue()
	task, exists, err := queue.GetTaskByID(userID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	ok, err := queue.CancelTask(userID, taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !ok {
		c.JSON(http.StatusConflict, gin.H{"error": "task cannot be cancelled"})
		return
	}

	// Refund the pre-consumed quota.
	if task.Status == model.TaskStatusQueued || task.Status == model.TaskStatusInProgress {
		service.RefundTaskQuota(c, task, "任务已取消")
	}

	c.JSON(http.StatusOK, buildImageGenerationTaskResponse(task))
}

// buildImageGenerationTaskResponse builds a unified image.generation task
// response from a Task model record.
func buildImageGenerationTaskResponse(task *model.Task) map[string]any {
	progress := 0
	if task.Progress != "" {
		p, _ := strconv.Atoi(task.Progress)
		progress = p
	}

	status := task.Status.ToVideoStatus()
	if task.Status == model.TaskStatusQueued {
		status = dto.VideoStatusQueued
	} else if task.Status == model.TaskStatusInProgress {
		status = dto.VideoStatusInProgress
	} else if task.Status == model.TaskStatusFailure {
		status = dto.VideoStatusFailed
	} else if task.Status == model.TaskStatusSuccess {
		status = dto.VideoStatusCompleted
	}

	resp := map[string]any{
		"id":         task.TaskID,
		"object":     "image.generation",
		"status":     status,
		"progress":   progress,
		"created_at": task.CreatedAt,
	}

	if task.Status == model.TaskStatusSuccess {
		resp["completed_at"] = task.FinishTime
		if task.PrivateData.ResultURL != "" {
			resp["metadata"] = map[string]any{
				"url": task.PrivateData.ResultURL,
			}
		}
		if len(task.Data) > 0 {
			resp["data"] = task.Data
		}
	}

	if task.Status == model.TaskStatusFailure && task.FailReason != "" {
		resp["error"] = map[string]any{
			"message": task.FailReason,
			"code":    "task_failed",
		}
	}

	return resp
}

// isSyncImageAsyncChannel returns true for channels whose image generation is
// wrapped as a sync-to-async task.
func isSyncImageAsyncChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeGemini, constant.ChannelTypeVolcEngine:
		return true
	}
	return false
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
			video["status"] = "SUCCESS"
			video["progress"] = 100
		case "failed":
			video["status"] = "FAILURE"
			video["progress"] = 100
		case "pending", "processing":
			video["status"] = "PENDING"
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
