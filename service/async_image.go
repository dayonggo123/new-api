package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

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

// PollAsyncImageTask queries the upstream for the async image task status.
func PollAsyncImageTask(task *AsyncImageTask) ([]byte, int, error) {
	queryID := task.TaskID
	if task.UpstreamTaskID != "" {
		queryID = task.UpstreamTaskID
	}
	var upstreamURL string
	switch task.ChannelType {
	case 60, 61: // DuoYuanTanSuo, APIMart
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
