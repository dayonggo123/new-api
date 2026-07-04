package service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func insertImageTask(t *testing.T, task *model.Task) {
	t.Helper()
	task.CreatedAt = time.Now().Unix()
	task.UpdatedAt = time.Now().Unix()
	if task.Action == "" {
		task.Action = constant.TaskActionImageGenerate
	}
	if task.Status == "" {
		task.Status = model.TaskStatusQueued
	}
	if task.TaskID == "" {
		task.TaskID = model.GenerateTaskID()
	}
	require.NoError(t, model.DB.Create(task).Error)
}

func reloadTask(t *testing.T, id int64) *model.Task {
	t.Helper()
	var task model.Task
	require.NoError(t, model.DB.First(&task, id).Error)
	return &task
}

func makeTestRelayInfo() *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:     1,
		UsingGroup: "default",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   1,
			ChannelType: constant.ChannelTypeOpenAI,
		},
		OriginModelName: "dall-e-3",
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		PriceData: types.PriceData{
			ModelPrice: 0.02,
			ModelRatio: 1,
			UsePrice:   false,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
			OtherRatios: map[string]float64{},
		},
		BillingSource:  "wallet",
		SubscriptionId: 0,
		TokenId:        1,
		Request: &dto.ImageRequest{
			Model:  "dall-e-3",
			Prompt: "a cute cat",
			Size:   "1024x1024",
		},
	}
}

// ---------------------------------------------------------------------------
// ImageTaskQueue unit tests
// ---------------------------------------------------------------------------

func TestImageTaskQueue_CreateTask(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	info := makeTestRelayInfo()
	payload := []byte(`{"model":"dall-e-3","prompt":"a cute cat"}`)

	task, err := q.CreateTask(info, payload, 1000)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.EqualValues(t, model.TaskStatusQueued, task.Status)
	assert.Equal(t, "0%", task.Progress)
	assert.Equal(t, constant.TaskActionImageGenerate, task.Action)
	assert.Equal(t, 1000, task.Quota)
	assert.Equal(t, "a cute cat", task.Properties.Input)
	assert.Equal(t, string(payload), task.PrivateData.RequestPayload)
	assert.NotEmpty(t, task.TaskID)
	assert.EqualValues(t, model.TaskStatusQueued, reloadTask(t, task.ID).Status)
}

func TestImageTaskQueue_CreateTask_WithPublicTaskID(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	info := makeTestRelayInfo()
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{PublicTaskID: "task_custom_public"}

	task, err := q.CreateTask(info, []byte(`{}`), 500)
	require.NoError(t, err)

	assert.Equal(t, "task_custom_public", task.TaskID)
	assert.EqualValues(t, model.TaskStatusQueued, reloadTask(t, task.ID).Status)
}

func TestImageTaskQueue_Dequeue(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()

	// eligible tasks for channel 1
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued})
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued})
	// not eligible: different channel
	insertImageTask(t, &model.Task{ChannelId: 2, Status: model.TaskStatusQueued})
	// not eligible: already in progress
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusInProgress})
	// not eligible: future retry
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued, PrivateData: model.TaskPrivateData{NextRetryAt: time.Now().Unix() + 1000}})

	tasks, err := q.Dequeue(1, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.Equal(t, 1, task.ChannelId)
		assert.EqualValues(t, model.TaskStatusQueued, task.Status)
	}

	// limit <= 0 should default to 1
	single, err := q.Dequeue(1, 0)
	require.NoError(t, err)
	assert.Len(t, single, 1)
}

func TestImageTaskQueue_MarkInProgress(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued})
	task := &model.Task{}
	require.NoError(t, model.DB.Where("channel_id = ?", 1).First(task).Error)

	ok, err := q.MarkInProgress(task)
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	assert.EqualValues(t, model.TaskStatusInProgress, reloaded.Status)
	assert.Greater(t, reloaded.StartTime, int64(0))
	assert.Greater(t, reloaded.UpdatedAt, int64(0))

	// second CAS attempt from QUEUED should lose
	ok2, err2 := q.MarkInProgress(task)
	require.NoError(t, err2)
	assert.False(t, ok2)
}

func TestImageTaskQueue_MarkSuccess(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusInProgress})
	task := &model.Task{}
	require.NoError(t, model.DB.Where("channel_id = ?", 1).First(task).Error)

	data := []byte(`{"url":"https://example.com/image.png"}`)
	ok, err := q.MarkSuccess(task, data, "https://example.com/image.png")
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	assert.EqualValues(t, model.TaskStatusSuccess, reloaded.Status)
	assert.Equal(t, "100%", reloaded.Progress)
	assert.Greater(t, reloaded.FinishTime, int64(0))
	assert.Equal(t, "https://example.com/image.png", reloaded.PrivateData.ResultURL)
	assert.JSONEq(t, string(data), string(reloaded.Data))
}

func TestImageTaskQueue_MarkFailure(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusInProgress})
	task := &model.Task{}
	require.NoError(t, model.DB.Where("channel_id = ?", 1).First(task).Error)

	ok, err := q.MarkFailure(task, "upstream request failed")
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Equal(t, "100%", reloaded.Progress)
	assert.Equal(t, "upstream request failed", reloaded.FailReason)
	assert.Greater(t, reloaded.FinishTime, int64(0))
}

func TestImageTaskQueue_MarkRetry(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{
		ChannelId: 1,
		Status:    model.TaskStatusInProgress,
		PrivateData: model.TaskPrivateData{
			RetryCount: 0,
		},
	})
	task := &model.Task{}
	require.NoError(t, model.DB.Where("channel_id = ?", 1).First(task).Error)

	nextRetryAt := time.Now().Unix() + 60
	ok, err := q.MarkRetry(task, "upstream busy", nextRetryAt)
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	assert.EqualValues(t, model.TaskStatusQueued, reloaded.Status)
	assert.Equal(t, 1, reloaded.PrivateData.RetryCount)
	assert.Equal(t, nextRetryAt, reloaded.PrivateData.NextRetryAt)
	assert.Equal(t, "upstream busy", reloaded.FailReason)
}

func TestImageTaskQueue_GetTaskByID(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{UserId: 1, TaskID: "task_get_by_id", ChannelId: 1})

	found, exists, err := q.GetTaskByID(1, "task_get_by_id")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, "task_get_by_id", found.TaskID)

	_, exists2, err2 := q.GetTaskByID(2, "task_get_by_id")
	require.NoError(t, err2)
	assert.False(t, exists2)

	_, exists3, err3 := q.GetTaskByID(1, "not_exists")
	require.NoError(t, err3)
	assert.False(t, exists3)
}

func TestImageTaskQueue_CancelTask(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{UserId: 1, TaskID: "task_cancel", ChannelId: 1, Status: model.TaskStatusQueued})

	ok, err := q.CancelTask(1, "task_cancel")
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, 1)
	assert.EqualValues(t, model.TaskStatusFailure, reloaded.Status)
	assert.Equal(t, "任务已取消", reloaded.FailReason)

	// terminal task cannot be cancelled again
	ok2, err2 := q.CancelTask(1, "task_cancel")
	require.NoError(t, err2)
	assert.False(t, ok2)
}

func TestImageTaskQueue_RecoverIncompleteTasks(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued})
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusInProgress})
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusSuccess})
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusFailure})
	insertImageTask(t, &model.Task{ChannelId: 1, Status: model.TaskStatusQueued, Action: "other_action"})

	tasks, err := q.RecoverIncompleteTasks()
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	for _, task := range tasks {
		assert.Equal(t, constant.TaskActionImageGenerate, task.Action)
		assert.True(t, task.Status == model.TaskStatusQueued || task.Status == model.TaskStatusInProgress)
	}
}

// ---------------------------------------------------------------------------
// Integration tests for the full lifecycle
// ---------------------------------------------------------------------------

func TestImageTaskQueue_FullLifecycle(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	info := makeTestRelayInfo()
	payload := []byte(`{"model":"dall-e-3","prompt":"a cat"}`)

	// Create
	task, err := q.CreateTask(info, payload, 1500)
	require.NoError(t, err)
	assert.EqualValues(t, model.TaskStatusQueued, task.Status)

	// Dequeue
	tasks, err := q.Dequeue(1, 1)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	dequeued := tasks[0]
	assert.Equal(t, task.TaskID, dequeued.TaskID)
	assert.EqualValues(t, model.TaskStatusQueued, dequeued.Status)

	// MarkInProgress
	ok, err := q.MarkInProgress(dequeued)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.EqualValues(t, model.TaskStatusInProgress, reloadTask(t, dequeued.ID).Status)

	// MarkSuccess
	resultData := []byte(`{"url":"https://example.com/result.png"}`)
	ok, err = q.MarkSuccess(dequeued, resultData, "https://example.com/result.png")
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, dequeued.ID)
	assert.EqualValues(t, model.TaskStatusSuccess, reloaded.Status)
	assert.NotEmpty(t, reloaded.Data)
	assert.Equal(t, "https://example.com/result.png", reloaded.PrivateData.ResultURL)
}

func TestImageTaskQueue_RecoverResetsInProgress(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	insertImageTask(t, &model.Task{
		ChannelId: 1,
		Status:    model.TaskStatusInProgress,
		PrivateData: model.TaskPrivateData{
			RetryCount: 0,
		},
	})

	tasks, err := q.RecoverIncompleteTasks()
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// Simulate recovery loop behavior: MarkRetry IN_PROGRESS tasks back to QUEUED
	task := tasks[0]
	ok, err := q.MarkRetry(task, "服务重启后恢复", time.Now().Unix())
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	assert.EqualValues(t, model.TaskStatusQueued, reloaded.Status)
	assert.Equal(t, 1, reloaded.PrivateData.RetryCount)
}

// ---------------------------------------------------------------------------
// ImageTaskRetryPolicy tests
// ---------------------------------------------------------------------------

func TestImageTaskRetryPolicy_ShouldRetry(t *testing.T) {
	p := ImageTaskRetryPolicy{MaxRetry: 3, BackoffSeconds: 10}
	task := &model.Task{PrivateData: model.TaskPrivateData{RetryCount: 0}}

	assert.False(t, p.ShouldRetry(task, nil), "nil error should not retry")
	assert.False(t, p.ShouldRetry(nil, errors.New("err")), "nil task should not retry")

	err4xx := types.NewErrorWithStatusCode(errors.New("bad request"), types.ErrorCodeInvalidRequest, http.StatusBadRequest)
	assert.False(t, p.ShouldRetry(task, err4xx), "4xx errors should not retry")

	err5xx := types.NewErrorWithStatusCode(errors.New("server error"), types.ErrorCodeBadResponseStatusCode, http.StatusInternalServerError)
	assert.True(t, p.ShouldRetry(task, err5xx), "5xx errors should retry")

	plainErr := errors.New("network timeout")
	assert.True(t, p.ShouldRetry(task, plainErr), "plain errors should retry")

	// exhausted retries
	task.PrivateData.RetryCount = 3
	assert.False(t, p.ShouldRetry(task, plainErr), "exhausted retries should not retry")
}

func TestImageTaskRetryPolicy_NextRetryAt(t *testing.T) {
	p := ImageTaskRetryPolicy{BackoffSeconds: 10}
	now := time.Now().Unix()

	t0 := p.NextRetryAt(0)
	assert.InDelta(t, now+10, t0, 2)

	t1 := p.NextRetryAt(1)
	assert.InDelta(t, now+20, t1, 2)

	t2 := p.NextRetryAt(2)
	assert.InDelta(t, now+40, t2, 2)

	// negative retry count should be treated as 0
	tNeg := p.NextRetryAt(-1)
	assert.InDelta(t, now+10, tNeg, 2)
}

// ---------------------------------------------------------------------------
// ImageTaskContext tests
// ---------------------------------------------------------------------------

func TestSerializeImageRequest(t *testing.T) {
	// Both nil => empty object
	assert.Equal(t, "{}", string(SerializeImageRequest(nil, nil)))

	info := makeTestRelayInfo()

	// Falls back to info.Request when req is nil
	b := SerializeImageRequest(info, nil)
	assert.Contains(t, string(b), "a cute cat")
	assert.Contains(t, string(b), "dall-e-3")

	// Explicit request takes precedence
	req := &dto.ImageRequest{Model: "gpt-image-2", Prompt: "a dog", Size: "1024x1024"}
	b2 := SerializeImageRequest(info, req)
	assert.Contains(t, string(b2), "a dog")
	assert.Contains(t, string(b2), "gpt-image-2")
}

func TestDownstreamBaseURLFromContext(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RequestHeaders: map[string]string{
			"X-Forwarded-Proto": "https",
			"X-Forwarded-Host":  "api.example.com",
		},
	}
	assert.Equal(t, "https://api.example.com", downstreamBaseURLFromContext(info))

	// Fallback to Host header
	info2 := &relaycommon.RelayInfo{
		RequestHeaders: map[string]string{
			"Host": "fallback.example.com",
		},
	}
	assert.Equal(t, "https://fallback.example.com", downstreamBaseURLFromContext(info2))

	// No host => empty
	assert.Equal(t, "", downstreamBaseURLFromContext(&relaycommon.RelayInfo{RequestHeaders: map[string]string{}}))
	assert.Equal(t, "", downstreamBaseURLFromContext(nil))
}

func TestDownstreamBaseURLFromGinContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "gin.example.com")
	c.Request = req

	assert.Equal(t, "https://gin.example.com", DownstreamBaseURLFromGinContext(c))
}

func TestImageTaskTimeoutReached(t *testing.T) {
	now := time.Now().Unix()

	assert.False(t, ImageTaskTimeoutReached(nil, 10))
	assert.False(t, ImageTaskTimeoutReached(&model.Task{SubmitTime: now - 20}, 0))
	assert.False(t, ImageTaskTimeoutReached(&model.Task{StartTime: now - 5}, 10))
	assert.False(t, ImageTaskTimeoutReached(&model.Task{StartTime: 0, SubmitTime: now - 5}, 10))
	assert.True(t, ImageTaskTimeoutReached(&model.Task{StartTime: now - 15}, 10))
	assert.True(t, ImageTaskTimeoutReached(&model.Task{StartTime: 0, SubmitTime: now - 15}, 10))
}

func TestImageTaskLockTimeoutReached(t *testing.T) {
	now := time.Now().Unix()

	assert.False(t, ImageTaskLockTimeoutReached(nil, 10))
	assert.False(t, ImageTaskLockTimeoutReached(&model.Task{Status: model.TaskStatusQueued, StartTime: now - 100}, 10))
	assert.False(t, ImageTaskLockTimeoutReached(&model.Task{Status: model.TaskStatusInProgress, StartTime: now - 5}, 10))
	assert.True(t, ImageTaskLockTimeoutReached(&model.Task{Status: model.TaskStatusInProgress, StartTime: now - 15}, 10))
}

// ---------------------------------------------------------------------------
// Regression helpers: ensure the new functions fit the existing task model
// ---------------------------------------------------------------------------

func TestImageTaskQueue_TaskDataRoundtrip(t *testing.T) {
	truncate(t)

	q := NewImageTaskQueue()
	info := makeTestRelayInfo()
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{PublicTaskID: "task_roundtrip"}

	payload := []byte(`{"model":"dall-e-3","prompt":"roundtrip"}`)
	task, err := q.CreateTask(info, payload, 2000)
	require.NoError(t, err)

	assert.Empty(t, task.Data)

	ok, err := q.MarkInProgress(task)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = q.MarkSuccess(task, []byte(`{"url":"https://x.com/1.png"}`), "https://x.com/1.png")
	require.NoError(t, err)
	assert.True(t, ok)

	reloaded := reloadTask(t, task.ID)
	require.NotEmpty(t, reloaded.Data)
	assert.Equal(t, "https://x.com/1.png", reloaded.PrivateData.ResultURL)
}
