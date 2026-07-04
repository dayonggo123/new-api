package service

import (
	"fmt"
	"net/url"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

// ImageTaskQueue provides database-backed queue operations for asynchronous
// image generation tasks.
type ImageTaskQueue struct{}

// NewImageTaskQueue creates a new ImageTaskQueue instance.
func NewImageTaskQueue() *ImageTaskQueue {
	return &ImageTaskQueue{}
}

// CreateTask persists a new image generation task as QUEUED.
// It should be called after pre-consuming billing so that quota is already locked.
func (q *ImageTaskQueue) CreateTask(relayInfo *relaycommon.RelayInfo, requestPayload []byte, quota int) (*model.Task, error) {
	platform := constant.TaskPlatform("image")
	task := model.InitTask(platform, relayInfo)
	if relayInfo.TaskRelayInfo != nil && relayInfo.TaskRelayInfo.PublicTaskID != "" {
		task.TaskID = relayInfo.TaskRelayInfo.PublicTaskID
	}

	task.Status = model.TaskStatusQueued
	task.Progress = "0%"
	task.Action = constant.TaskActionImageGenerate
	task.Quota = quota
	task.PrivateData.RequestPayload = string(requestPayload)
	task.PrivateData.DownstreamBaseURL = downstreamBaseURLFromContext(relayInfo)
	task.PrivateData.BillingSource = relayInfo.BillingSource
	task.PrivateData.SubscriptionId = relayInfo.SubscriptionId
	task.PrivateData.TokenId = relayInfo.TokenId
	task.PrivateData.RelayMode = relayInfo.RelayMode

	bc := relayInfo.PriceData
	task.PrivateData.BillingContext = &model.TaskBillingContext{
		ModelPrice:      bc.ModelPrice,
		GroupRatio:      bc.GroupRatioInfo.GroupRatio,
		ModelRatio:      bc.ModelRatio,
		OtherRatios:     bc.OtherRatios,
		OriginModelName: relayInfo.OriginModelName,
		PerCallBilling:  common.StringsContains(constant.TaskPricePatches, relayInfo.OriginModelName) || bc.UsePrice,
	}

	if relayInfo.Request != nil {
		if imageReq, ok := relayInfo.Request.(*dto.ImageRequest); ok {
			task.Properties.Input = imageReq.Prompt
		}
	}

	if err := task.Insert(); err != nil {
		return nil, fmt.Errorf("insert image task failed: %w", err)
	}
	return task, nil
}

// Dequeue returns up to limit image tasks that are ready to be processed for the
// given channel. A task is ready when its status is QUEUED and its next retry
// time has passed (or is unset).
func (q *ImageTaskQueue) Dequeue(channelID int, limit int) ([]*model.Task, error) {
	if limit <= 0 {
		limit = 1
	}
	now := time.Now().Unix()
	var tasks []*model.Task
	err := model.DB.Where("channel_id = ?", channelID).
		Where("action = ?", constant.TaskActionImageGenerate).
		Where("status = ?", model.TaskStatusQueued).
		Where(nextRetryAtJSONFilter(), now).
		Order("submit_time").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("dequeue image tasks failed: %w", err)
	}
	return tasks, nil
}

// nextRetryAtJSONFilter returns a database-aware WHERE clause that filters on
// private_data.next_retry_at. It treats unset/missing values as 0 (ready).
// Supports SQLite, MySQL, and PostgreSQL.
func nextRetryAtJSONFilter() string {
	if common.UsingPostgreSQL {
		return `COALESCE((private_data->>'next_retry_at')::bigint, 0) <= ?`
	}
	if common.UsingMySQL {
		return `COALESCE(JSON_EXTRACT(private_data, '$.next_retry_at'), 0) <= ?`
	}
	return `COALESCE(json_extract(private_data, '$.next_retry_at'), 0) <= ?`
}

// MarkInProgress attempts to CAS a task from QUEUED to IN_PROGRESS. It returns
// true if the caller won the transition and should execute the task.
func (q *ImageTaskQueue) MarkInProgress(task *model.Task) (bool, error) {
	now := time.Now().Unix()
	task.Status = model.TaskStatusInProgress
	task.StartTime = now
	task.UpdatedAt = now
	ok, err := task.UpdateWithStatus(model.TaskStatusQueued)
	if err != nil {
		return false, fmt.Errorf("mark image task in_progress failed: %w", err)
	}
	return ok, nil
}

// MarkSuccess attempts to CAS a task from IN_PROGRESS to SUCCESS and stores the
// result data and result URL.
func (q *ImageTaskQueue) MarkSuccess(task *model.Task, data []byte, resultURL string) (bool, error) {
	now := time.Now().Unix()
	task.Status = model.TaskStatusSuccess
	task.Progress = "100%"
	task.FinishTime = now
	task.Data = data
	task.PrivateData.ResultURL = resultURL
	task.UpdatedAt = now
	ok, err := task.UpdateWithStatus(model.TaskStatusInProgress)
	if err != nil {
		return false, fmt.Errorf("mark image task success failed: %w", err)
	}
	return ok, nil
}

// MarkRetry attempts to CAS a task from IN_PROGRESS back to QUEUED, increasing
// the retry counter and recording the next retry time.
func (q *ImageTaskQueue) MarkRetry(task *model.Task, reason string, nextRetryAt int64) (bool, error) {
	now := time.Now().Unix()
	task.Status = model.TaskStatusQueued
	task.PrivateData.RetryCount++
	task.PrivateData.NextRetryAt = nextRetryAt
	task.FailReason = reason
	task.UpdatedAt = now
	ok, err := task.UpdateWithStatus(model.TaskStatusInProgress)
	if err != nil {
		return false, fmt.Errorf("mark image task retry failed: %w", err)
	}
	return ok, nil
}

// MarkFailure attempts to CAS a task from IN_PROGRESS to FAILURE. After this
// the caller should refund the pre-consumed quota.
func (q *ImageTaskQueue) MarkFailure(task *model.Task, reason string) (bool, error) {
	now := time.Now().Unix()
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = now
	task.FailReason = reason
	task.UpdatedAt = now
	ok, err := task.UpdateWithStatus(model.TaskStatusInProgress)
	if err != nil {
		return false, fmt.Errorf("mark image task failure failed: %w", err)
	}
	return ok, nil
}

// GetTaskByID returns a task belonging to the given user by its public task ID.
func (q *ImageTaskQueue) GetTaskByID(userID int, taskID string) (*model.Task, bool, error) {
	return model.GetByTaskId(userID, taskID)
}

// RecoverIncompleteTasks returns all image tasks that are not in a terminal
// state. This is used on service startup to restore work after a restart.
func (q *ImageTaskQueue) RecoverIncompleteTasks() ([]*model.Task, error) {
	var tasks []*model.Task
	err := model.DB.Where("action = ?", constant.TaskActionImageGenerate).
		Where("status IN ?", []string{string(model.TaskStatusQueued), string(model.TaskStatusInProgress)}).
		Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("recover incomplete image tasks failed: %w", err)
	}
	return tasks, nil
}

// CancelTask cancels a task that is still in QUEUED or IN_PROGRESS and marks it
// as FAILED. It returns true if the task existed and was cancellable.
func (q *ImageTaskQueue) CancelTask(userID int, taskID string) (bool, error) {
	task, exists, err := model.GetByTaskId(userID, taskID)
	if err != nil || !exists {
		return false, err
	}
	if task.Status != model.TaskStatusQueued && task.Status != model.TaskStatusInProgress {
		return false, nil
	}

	fromStatus := task.Status
	now := time.Now().Unix()
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = now
	task.FailReason = "任务已取消"
	task.UpdatedAt = now
	ok, err := task.UpdateWithStatus(fromStatus)
	if err != nil {
		return false, fmt.Errorf("cancel image task failed: %w", err)
	}
	return ok, nil
}

// downstreamBaseURLFromContext derives the downstream base URL used for image
// proxy URL rewriting from the original request headers stored in RelayInfo.
func downstreamBaseURLFromContext(relayInfo *relaycommon.RelayInfo) string {
	if relayInfo == nil {
		return ""
	}
	scheme := "https"
	host := ""
	if relayInfo.RequestHeaders != nil {
		if v := relayInfo.RequestHeaders["X-Forwarded-Proto"]; v != "" {
			scheme = v
		}
		if v := relayInfo.RequestHeaders["X-Forwarded-Host"]; v != "" {
			host = v
		}
		if host == "" {
			if v := relayInfo.RequestHeaders["Host"]; v != "" {
				host = v
			}
		}
	}
	if host == "" {
		return ""
	}
	u := url.URL{
		Scheme: scheme,
		Host:   host,
	}
	return u.String()
}

// downstreamBaseURLFromGinContext derives a base URL (scheme + host) from the
// incoming request. This is stored in the task so that the worker can generate
// proxy URLs without requiring the original gin context.
func DownstreamBaseURLFromGinContext(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	scheme := "https"
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return ""
	}
	u := url.URL{
		Scheme: scheme,
		Host:   host,
	}
	return u.String()
}

// ImageTaskTimeoutReached returns true if the task has exceeded the configured
// timeout. It is safe to call on any task.
func ImageTaskTimeoutReached(task *model.Task, timeoutSeconds int) bool {
	if task == nil || timeoutSeconds <= 0 {
		return false
	}
	startTime := task.StartTime
	if startTime == 0 {
		startTime = task.SubmitTime
	}
	return time.Now().Unix()-startTime > int64(timeoutSeconds)
}

// ImageTaskLockTimeoutReached returns true if a task has been IN_PROGRESS for
// longer than the configured lock timeout, indicating a crashed worker.
func ImageTaskLockTimeoutReached(task *model.Task, timeoutSeconds int) bool {
	if task == nil || task.Status != model.TaskStatusInProgress || timeoutSeconds <= 0 {
		return false
	}
	return time.Now().Unix()-task.StartTime > int64(timeoutSeconds)
}
