package service

import (
	"testing"
	"time"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestRegisterAndGetAsyncImageTask(t *testing.T) {
	// Clear map for clean test state
	asyncImageTasksMu.Lock()
	asyncImageTasks = make(map[string]*AsyncImageTask)
	asyncImageTasksMu.Unlock()

	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.example.com",
			ApiKey:         "sk-test-key",
			ChannelType:    1,
		},
	}

	RegisterAsyncImageTask("task_abc123", info)

	task := GetAsyncImageTask("task_abc123")
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.TaskID != "task_abc123" {
		t.Errorf("taskID = %s, want task_abc123", task.TaskID)
	}
	if task.ChannelURL != "https://api.example.com" {
		t.Errorf("channelURL = %s, want https://api.example.com", task.ChannelURL)
	}
	if task.ChannelKey != "sk-test-key" {
		t.Errorf("channelKey = %s, want sk-test-key", task.ChannelKey)
	}
	if task.ModelName != "gpt-image-2" {
		t.Errorf("modelName = %s, want gpt-image-2", task.ModelName)
	}

	// Non-existent task
	if GetAsyncImageTask("nonexistent") != nil {
		t.Error("expected nil for non-existent task")
	}
}

func TestAsyncImageTaskTTL(t *testing.T) {
	// Clear map
	asyncImageTasksMu.Lock()
	asyncImageTasks = make(map[string]*AsyncImageTask)
	asyncImageTasksMu.Unlock()

	// Temporarily reduce TTL for testing
	originalTTL := asyncTaskTTL
	asyncTaskTTL = 100 * time.Millisecond
	defer func() { asyncTaskTTL = originalTTL }()

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.example.com",
			ApiKey:         "sk-key",
		},
	}
	RegisterAsyncImageTask("task_old", info)

	// Task should exist immediately
	if GetAsyncImageTask("task_old") == nil {
		t.Error("expected task to exist before TTL expiry")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	if GetAsyncImageTask("task_old") != nil {
		t.Error("expected task to be expired after TTL")
	}
}

func TestCleanupExpiredAsyncImageTasks(t *testing.T) {
	asyncImageTasksMu.Lock()
	asyncImageTasks = make(map[string]*AsyncImageTask)
	asyncImageTasksMu.Unlock()

	originalTTL := asyncTaskTTL
	asyncTaskTTL = 100 * time.Millisecond
	defer func() { asyncTaskTTL = originalTTL }()

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "https://api.example.com",
			ApiKey:         "sk-key",
		},
	}

	// Register expired task
	asyncImageTasksMu.Lock()
	asyncImageTasks["task_expired"] = &AsyncImageTask{
		TaskID:    "task_expired",
		CreatedAt: time.Now().Add(-200 * time.Millisecond),
	}
	asyncImageTasksMu.Unlock()

	// Register new task should trigger cleanup
	RegisterAsyncImageTask("task_new", info)

	asyncImageTasksMu.RLock()
	_, exists := asyncImageTasks["task_expired"]
	asyncImageTasksMu.RUnlock()

	if exists {
		t.Error("expected expired task to be cleaned up")
	}

	if GetAsyncImageTask("task_new") == nil {
		t.Error("expected new task to exist")
	}
}
