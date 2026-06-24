package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
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
}

func cleanupExpiredAsyncImageTasksLocked() {
	for id, task := range asyncImageTasks {
		if time.Since(task.CreatedAt) > asyncTaskTTL {
			delete(asyncImageTasks, id)
		}
	}
}

func GetAsyncImageTask(taskID string) *AsyncImageTask {
	asyncImageTasksMu.RLock()
	defer asyncImageTasksMu.RUnlock()
	task := asyncImageTasks[taskID]
	if task != nil && time.Since(task.CreatedAt) > asyncTaskTTL {
		return nil
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
	err := model.DB.Where("id = ?", taskID).First(&dbTask).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			common.SysLog(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] task %s not found in DB", taskID))
		} else {
			common.SysError(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] DB error for task %s: %v", taskID, err))
		}
		return nil
	}

	channel, err := model.GetChannelById(dbTask.ChannelID, true)
	if err != nil {
		common.SysError(fmt.Sprintf("[RecoverAsyncImageTaskFromDB] channel %d not found for task %s: %v", dbTask.ChannelID, taskID, err))
		return nil
	}

	task := &AsyncImageTask{
		TaskID:      taskID,
		ChannelURL:  channel.BaseURL,
		ChannelKey:  channel.Key,
		ChannelType: channel.Type,
		ModelName:   dbTask.Properties.OriginModelName,
		CreatedAt:   dbTask.CreatedAt,
	}

	// Extract UpstreamTaskID from PrivateData
	if dbTask.PrivateData.UpstreamTaskID != "" {
		task.UpstreamTaskID = dbTask.PrivateData.UpstreamTaskID
	}

	// Also extract from request_payload if available
	if dbTask.PrivateData.RequestPayload != nil && task.UpstreamTaskID == "" {
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(dbTask.PrivateData.RequestPayload, &payloadMap); err == nil {
			// Try to find upstream task ID in the payload
			// (this is channel-specific, may not always work)
		}
	}

	return task
}
func PollAsyncImageTask(task *AsyncImageTask) ([]byte, int, error) {
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
