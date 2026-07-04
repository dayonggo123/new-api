package service

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

// ImageTaskRetryPolicy encapsulates the retry decisions for image generation
// tasks. It is safe for concurrent use.
type ImageTaskRetryPolicy struct {
	// MaxRetry is the maximum number of attempts before a task is considered
	// permanently failed. The first attempt counts as retry 0.
	MaxRetry int
	// BackoffSeconds is the base delay for exponential backoff.
	BackoffSeconds int
}

// NewDefaultImageTaskRetryPolicy creates a retry policy using the global worker
// configuration defaults.
func NewDefaultImageTaskRetryPolicy() ImageTaskRetryPolicy {
	cfg := common.ImageTaskWorkerConfigSingleton
	return ImageTaskRetryPolicy{
		MaxRetry:       cfg.MaxRetry,
		BackoffSeconds: cfg.RetryBackoffSeconds,
	}
}

// ShouldRetry returns true when a failed task is eligible for another attempt.
// Client-side errors (HTTP 4xx) are never retried because they will fail again
// with the same request.
func (p ImageTaskRetryPolicy) ShouldRetry(task *model.Task, err error) bool {
	if err == nil {
		return false
	}
	if task == nil {
		return false
	}
	if task.PrivateData.RetryCount >= p.MaxRetry {
		return false
	}
	return !IsImageTaskNonRetryableError(err)
}

// IsImageTaskNonRetryableError reports whether the error is a client-side or
// business error that should not be retried.
func IsImageTaskNonRetryableError(err error) bool {
	var apiErr *types.NewAPIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			return true
		}
	}
	return false
}

// NextRetryAt returns the Unix timestamp when the next retry attempt should be
// scheduled. The delay grows exponentially with the retry count: base * 2^count.
func (p ImageTaskRetryPolicy) NextRetryAt(retryCount int) int64 {
	if retryCount < 0 {
		retryCount = 0
	}
	backoff := int64(p.BackoffSeconds) << retryCount
	if backoff <= 0 {
		backoff = int64(p.BackoffSeconds)
	}
	return time.Now().Unix() + backoff
}
