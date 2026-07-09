package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

var (
	asyncImageTasks   = make(map[string]*AsyncImageTask)
	asyncImageTasksMu sync.RWMutex
	asyncTaskTTL      = 24 * time.Hour
)

type AsyncImageTask struct {
	TaskID         string
	UpstreamTaskID string // 上游真实的 task ID（APIMart 等 task 渠道）
	ChannelURL     string
	ChannelKey     string
	ChannelType    int
	ModelName      string
	CreatedAt      time.Time
}

func RegisterAsyncImageTask(taskID string, info *relaycommon.RelayInfo) {
	if info == nil || info.ChannelMeta == nil {
		return
	}
	asyncImageTasksMu.Lock()
	defer asyncImageTasksMu.Unlock()
	// Clean up expired entries opportunistically on registration
	cleanupExpiredAsyncImageTasksLocked()
	asyncImageTasks[taskID] = &AsyncImageTask{
		TaskID:      taskID,
		ChannelURL:  info.ChannelBaseUrl,
		ChannelKey:  info.ApiKey,
		ChannelType: info.ChannelType,
		ModelName:   info.OriginModelName,
		CreatedAt:   time.Now(),
	}
}

func SetAsyncImageTaskUpstreamID(taskID, upstreamID string) {
	asyncImageTasksMu.Lock()
	defer asyncImageTasksMu.Unlock()
	if task, ok := asyncImageTasks[taskID]; ok {
		task.UpstreamTaskID = upstreamID
	}

	// Also persist to DB so recovery after restart has the UpstreamTaskID
	var dbTask model.Task
	if err := model.DB.Where("task_id = ?", taskID).First(&dbTask).Error; err == nil {
		dbTask.PrivateData.UpstreamTaskID = upstreamID
		if err := model.DB.Model(&dbTask).Update("private_data", dbTask.PrivateData).Error; err != nil {
			common.SysError(fmt.Sprintf("[SetAsyncImageTaskUpstreamID] failed to persist to DB: %v", err))
		}
	}
}

func cleanupExpiredAsyncImageTasksLocked() {
	for id, task := range asyncImageTasks {
		if time.Since(task.CreatedAt) > asyncTaskTTL {
			delete(asyncImageTasks, id)
		}
	}
}

func GetAsyncImageTask(taskID string) *AsyncImageTask {
	// Fast path: check in-memory map
	asyncImageTasksMu.RLock()
	task := asyncImageTasks[taskID]
	asyncImageTasksMu.RUnlock()
	if task != nil {
		if time.Since(task.CreatedAt) > asyncTaskTTL {
			common.SysLog(fmt.Sprintf("[GetAsyncImageTask] task %s expired, removing", taskID))
			asyncImageTasksMu.Lock()
			delete(asyncImageTasks, taskID)
			asyncImageTasksMu.Unlock()
			return nil
		}
		return task
	}

	// Slow path: try to recover from DB (handles New-API restart)
	task = RecoverAsyncImageTaskFromDB(taskID)
	if task != nil {
		common.SysLog(fmt.Sprintf("[GetAsyncImageTask] recovered task %s from DB", taskID))
		asyncImageTasksMu.Lock()
		asyncImageTasks[taskID] = task
		asyncImageTasksMu.Unlock()
	}
	return task
}

// StoreAsyncImageTask stores a task directly into the in-memory map.
// Used for recovering tasks from DB after a restart.
func StoreAsyncImageTask(task *AsyncImageTask) {
	asyncImageTasksMu.Lock()
	defer asyncImageTasksMu.Unlock()
	asyncImageTasks[task.TaskID] = task
}

// RecoverAsyncImageTaskFromDB attempts to reconstruct an AsyncImageTask from the database.
// Returns nil if the task is not found or the channel is not found.
func RecoverAsyncImageTaskFromDB(taskID string) *AsyncImageTask {
	var dbTask model.Task
	err := model.DB.Where("task_id = ?", taskID).First(&dbTask).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.SysLog(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] task %s not found in DB", taskID))
		} else {
			common.SysError(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] DB error for task %s: %v", taskID, err))
		}
		return nil
	}

	channel, err := model.GetChannelById(dbTask.ChannelId, true)
	if err != nil {
		common.SysError(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] channel %d not found for task %s: %v", dbTask.ChannelId, taskID, err))
		return nil
	}

	// channel.BaseURL is *string
	channelBaseURL := ""
	if channel.BaseURL != nil {
		channelBaseURL = *channel.BaseURL
	}

	task := &AsyncImageTask{
		TaskID:      taskID,
		ChannelURL:  channelBaseURL,
		ChannelKey:  channel.Key,
		ChannelType: channel.Type,
		ModelName:   dbTask.Properties.OriginModelName,
		CreatedAt:   time.Unix(dbTask.CreatedAt, 0),
	}

	// Extract UpstreamTaskID from PrivateData
	if dbTask.PrivateData.UpstreamTaskID != "" {
		task.UpstreamTaskID = dbTask.PrivateData.UpstreamTaskID
	}

	// Also extract from request_payload if available
	if dbTask.PrivateData.RequestPayload != "" && task.UpstreamTaskID == "" {
		var payloadMap map[string]interface{}
		if err := json.Unmarshal([]byte(dbTask.PrivateData.RequestPayload), &payloadMap); err == nil {
			// Try to find upstream task ID in the payload
			// (this is channel-specific, may not always work)
		}
	}

	return task
}
func PollAsyncImageTask(task *AsyncImageTask) ([]byte, int, error) {
	// For synchronous image channels (OpenAI/Gemini/VolcEngine), the result is
	// already stored in the DB task record. Return it directly.
	if task.UpstreamTaskID == "" && isSyncImageAsyncChannel(task.ChannelType) {
		return pollSyncImageTaskFromDB(task)
	}

	queryID := task.TaskID
	if task.UpstreamTaskID != "" {
		queryID = task.UpstreamTaskID
	}
	var upstreamURL string
	switch task.ChannelType {
	case 60, 61, 64: // DuoYuanTanSuo, APIMart, 章鱼哥
		upstreamURL = fmt.Sprintf("%s/v1/tasks/%s", strings.TrimSuffix(task.ChannelURL, "/"), queryID)
	default:
		upstreamURL = fmt.Sprintf("%s/v1/images/tasks/%s", strings.TrimSuffix(task.ChannelURL, "/"), queryID)
	}
	req, err := http.NewRequest("GET", upstreamURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+task.ChannelKey)

	client := GetHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, resp.StatusCode, nil
}

func isSyncImageAsyncChannel(channelType int) bool {
	switch channelType {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeGemini, constant.ChannelTypeEasyRouter, constant.ChannelTypeVolcEngine:
		return true
	}
	return false
}

func pollSyncImageTaskFromDB(task *AsyncImageTask) ([]byte, int, error) {
	var dbTask model.Task
	if err := model.DB.Where("task_id = ?", task.TaskID).First(&dbTask).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, fmt.Errorf("task not found")
		}
		return nil, http.StatusInternalServerError, err
	}

	// If the task is not completed yet, return a pending status response.
	if dbTask.Status != model.TaskStatusSuccess {
		statusStr := string(dbTask.Status)
		switch dbTask.Status {
		case model.TaskStatusQueued, model.TaskStatusSubmitted, model.TaskStatusInProgress:
			statusStr = "in_progress"
		case model.TaskStatusFailure:
			statusStr = "failed"
		}
		body, _ := common.Marshal(map[string]any{
			"id":         task.TaskID,
			"object":     "video",
			"status":     statusStr,
			"progress":   0,
			"created_at": dbTask.CreatedAt,
		})
		return body, http.StatusOK, nil
	}

	// Return the stored image response directly
	body := dbTask.Data
	if len(body) == 0 {
		body = []byte("{}")
	}
	return body, http.StatusOK, nil
}
