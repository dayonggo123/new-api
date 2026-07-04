package common

import (
	"os"
	"time"
)

// ImageTaskWorkerConfig holds environment-driven configuration for the
// asynchronous image generation task queue worker pool.
type ImageTaskWorkerConfig struct {
	// Concurrency is the number of worker goroutines per channel pool.
	Concurrency int
	// PollInterval is how often each worker polls the database for new tasks.
	PollInterval time.Duration
	// MaxRetry is the maximum number of execution attempts before a task is
	// marked as permanently failed.
	MaxRetry int
	// TimeoutSeconds is the maximum time a task is allowed to run before it is
	// considered timed out.
	TimeoutSeconds int
	// RetryBackoffSeconds is the base for exponential backoff between retries.
	RetryBackoffSeconds int
}

// ImageTaskWorkerConfigSingleton is the lazily loaded global configuration for
// image task workers. It is safe to read concurrently after the first call.
var ImageTaskWorkerConfigSingleton = LoadImageTaskWorkerConfig()

// LoadImageTaskWorkerConfig reads image-task worker configuration from
// environment variables and returns a populated ImageTaskWorkerConfig with
// sensible defaults.
func LoadImageTaskWorkerConfig() ImageTaskWorkerConfig {
	return ImageTaskWorkerConfig{
		Concurrency:         GetEnvOrDefault("IMAGE_TASK_WORKER_CONCURRENCY", 5),
		PollInterval:        GetEnvOrDefaultDuration("IMAGE_TASK_WORKER_POLL_INTERVAL", 5*time.Second),
		MaxRetry:            GetEnvOrDefault("IMAGE_TASK_WORKER_MAX_RETRY", 3),
		TimeoutSeconds:      GetEnvOrDefault("IMAGE_TASK_WORKER_TIMEOUT_SECONDS", 600),
		RetryBackoffSeconds: GetEnvOrDefault("IMAGE_TASK_WORKER_RETRY_BACKOFF_SECONDS", 10),
	}
}

// GetEnvOrDefaultDuration reads a duration-valued environment variable. If the
// variable is missing or invalid, it returns the provided default and logs a
// system error.
func GetEnvOrDefaultDuration(env string, defaultValue time.Duration) time.Duration {
	if env == "" || os.Getenv(env) == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(os.Getenv(env))
	if err != nil {
		SysError("failed to parse " + env + ": " + err.Error() + ", using default value: " + defaultValue.String())
		return defaultValue
	}
	return d
}
