package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ImageTaskWorkerPool consumes image generation tasks for a single channel.
// It runs a fixed number of goroutines that poll the database queue and execute
// tasks.
type ImageTaskWorkerPool struct {
	ChannelID    int
	Concurrency  int
	PollInterval time.Duration
	Timeout      time.Duration
	Queue        *ImageTaskQueue
	Executor     *ImageTaskExecutor

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewImageTaskWorkerPool creates a worker pool for a single channel.
func NewImageTaskWorkerPool(channelID int, queue *ImageTaskQueue, executor *ImageTaskExecutor) *ImageTaskWorkerPool {
	cfg := common.ImageTaskWorkerConfigSingleton
	return &ImageTaskWorkerPool{
		ChannelID:    channelID,
		Concurrency:  cfg.Concurrency,
		PollInterval: cfg.PollInterval,
		Timeout:      time.Duration(cfg.TimeoutSeconds) * time.Second,
		Queue:        queue,
		Executor:     executor,
	}
}

// Start launches the worker goroutines. It is safe to call only once.
func (p *ImageTaskWorkerPool) Start(ctx context.Context) {
	p.ctx, p.cancel = context.WithCancel(ctx)
	for i := 0; i < p.Concurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop signals the pool to stop and waits for all workers to finish.
func (p *ImageTaskWorkerPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}

func (p *ImageTaskWorkerPool) worker(index int) {
	defer p.wg.Done()
	common.SysLog(fmt.Sprintf("[ImageTaskWorkerPool] channel %d worker %d started", p.ChannelID, index))
	defer common.SysLog(fmt.Sprintf("[ImageTaskWorkerPool] channel %d worker %d stopped", p.ChannelID, index))

	ticker := time.NewTicker(p.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}

		tasks, err := p.Queue.Dequeue(p.ChannelID, 1)
		if err != nil {
			common.SysError(fmt.Sprintf("[ImageTaskWorkerPool] channel %d dequeue error: %v", p.ChannelID, err))
			continue
		}
		if len(tasks) == 0 {
			continue
		}
		task := tasks[0]
		p.processTask(task)
	}
}

func (p *ImageTaskWorkerPool) processTask(task *model.Task) {
	if task == nil {
		return
	}

	// Check for tasks that have been stuck in IN_PROGRESS for too long. The
	// global recovery loop should reset them, but as a safety net we skip them.
	if task.Status != model.TaskStatusQueued {
		common.SysLog(fmt.Sprintf("[ImageTaskWorkerPool] task %s has status %s, skipping", task.TaskID, task.Status))
		return
	}

	ok, err := p.Queue.MarkInProgress(task)
	if err != nil {
		common.SysError(fmt.Sprintf("[ImageTaskWorkerPool] MarkInProgress task %s error: %v", task.TaskID, err))
		return
	}
	if !ok {
		common.SysLog(fmt.Sprintf("[ImageTaskWorkerPool] task %s already taken by another worker", task.TaskID))
		return
	}

	common.SysLog(fmt.Sprintf("[ImageTaskWorkerPool] executing task %s on channel %d", task.TaskID, p.ChannelID))

	execCtx, cancel := context.WithTimeout(p.ctx, p.Timeout)
	defer cancel()

	if err := p.Executor.Execute(execCtx, task); err != nil {
		common.SysError(fmt.Sprintf("[ImageTaskWorkerPool] task %s execution error: %v", task.TaskID, err))
	}
}

// ImageTaskWorkerPoolManager manages per-channel worker pools. It starts pools
// for channels that have incomplete image tasks and stops them on shutdown.
type ImageTaskWorkerPoolManager struct {
	mu           sync.RWMutex
	pools        map[int]*ImageTaskWorkerPool
	queue        *ImageTaskQueue
	executor     *ImageTaskExecutor
	pollInterval time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewImageTaskWorkerPoolManager creates a manager with the default queue and executor.
func NewImageTaskWorkerPoolManager() *ImageTaskWorkerPoolManager {
	cfg := common.ImageTaskWorkerConfigSingleton
	queue := NewImageTaskQueue()
	executor := NewImageTaskExecutor(*queue)
	return &ImageTaskWorkerPoolManager{
		pools:        make(map[int]*ImageTaskWorkerPool),
		queue:        queue,
		executor:     executor,
		pollInterval: cfg.PollInterval,
	}
}

// Start recovers incomplete tasks, resets IN_PROGRESS tasks to QUEUED, starts
// pools for affected channels, and begins a background monitor for new channels.
func (m *ImageTaskWorkerPoolManager) Start(ctx context.Context) {
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Recover tasks that were in progress when the service stopped. This must
	// happen before starting any pool to avoid duplicate execution.
	if err := m.recoverTasks(); err != nil {
		common.SysError(fmt.Sprintf("[ImageTaskWorkerPoolManager] recover tasks failed: %v", err))
	}

	// Start pools for channels with existing queued tasks.
	if err := m.startPoolsForQueuedTasks(); err != nil {
		common.SysError(fmt.Sprintf("[ImageTaskWorkerPoolManager] start pools failed: %v", err))
	}

	m.wg.Add(1)
	go m.monitor()

	common.SysLog("[ImageTaskWorkerPoolManager] started")
}

// Stop signals all pools to stop and waits for them to finish.
func (m *ImageTaskWorkerPoolManager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}

	m.mu.Lock()
	for channelID, pool := range m.pools {
		pool.Stop()
		delete(m.pools, channelID)
	}
	m.mu.Unlock()

	m.wg.Wait()
	common.SysLog("[ImageTaskWorkerPoolManager] stopped")
}

// Queue returns the underlying queue used by the manager. Useful for tests.
func (m *ImageTaskWorkerPoolManager) Queue() *ImageTaskQueue {
	return m.queue
}

// Executor returns the underlying executor used by the manager. Useful for tests.
func (m *ImageTaskWorkerPoolManager) Executor() *ImageTaskExecutor {
	return m.executor
}

func (m *ImageTaskWorkerPoolManager) recoverTasks() error {
	tasks, err := m.queue.RecoverIncompleteTasks()
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	for _, task := range tasks {
		if task.Status == model.TaskStatusInProgress {
			common.SysLog(fmt.Sprintf("[ImageTaskWorkerPoolManager] recovering IN_PROGRESS task %s on channel %d", task.TaskID, task.ChannelId))
			if _, err := m.queue.MarkRetry(task, "服务重启后恢复", now); err != nil {
				common.SysError(fmt.Sprintf("[ImageTaskWorkerPoolManager] recover task %s failed: %v", task.TaskID, err))
			}
		}
	}
	return nil
}

func (m *ImageTaskWorkerPoolManager) startPoolsForQueuedTasks() error {
	channelIDs, err := m.listChannelsWithQueuedTasks()
	if err != nil {
		return err
	}
	for _, channelID := range channelIDs {
		m.startPool(channelID)
	}
	return nil
}

func (m *ImageTaskWorkerPoolManager) listChannelsWithQueuedTasks() ([]int, error) {
	var rows []struct {
		ChannelId int
	}
	err := model.DB.Model(&model.Task{}).
		Distinct("channel_id").
		Where("action = ?", "image_generation").
		Where("status = ?", model.TaskStatusQueued).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ChannelId)
	}
	return ids, nil
}

func (m *ImageTaskWorkerPoolManager) startPool(channelID int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pools[channelID]; exists {
		return
	}

	pool := NewImageTaskWorkerPool(channelID, m.queue, m.executor)
	pool.Start(m.ctx)
	m.pools[channelID] = pool
	common.SysLog(fmt.Sprintf("[ImageTaskWorkerPoolManager] started pool for channel %d", channelID))
}

func (m *ImageTaskWorkerPoolManager) stopPool(channelID int) {
	m.mu.Lock()
	pool, exists := m.pools[channelID]
	if exists {
		delete(m.pools, channelID)
	}
	m.mu.Unlock()

	if exists {
		pool.Stop()
		common.SysLog(fmt.Sprintf("[ImageTaskWorkerPoolManager] stopped pool for channel %d", channelID))
	}
}

func (m *ImageTaskWorkerPoolManager) monitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
		}

		channelIDs, err := m.listChannelsWithQueuedTasks()
		if err != nil {
			common.SysError(fmt.Sprintf("[ImageTaskWorkerPoolManager] monitor list channels failed: %v", err))
			continue
		}
		for _, channelID := range channelIDs {
			m.startPool(channelID)
		}
	}
}

var imageTaskWorkerPoolManager *ImageTaskWorkerPoolManager
var imageTaskWorkerPoolManagerOnce sync.Once

// GetImageTaskWorkerPoolManager returns the singleton worker pool manager.
func GetImageTaskWorkerPoolManager() *ImageTaskWorkerPoolManager {
	imageTaskWorkerPoolManagerOnce.Do(func() {
		imageTaskWorkerPoolManager = NewImageTaskWorkerPoolManager()
	})
	return imageTaskWorkerPoolManager
}

// StartImageTaskWorkers starts the global image task worker pool manager. It is
// safe to call multiple times; subsequent calls are no-ops.
func StartImageTaskWorkers(ctx context.Context) {
	GetImageTaskWorkerPoolManager().Start(ctx)
}

// StopImageTaskWorkers stops the global image task worker pool manager.
func StopImageTaskWorkers() {
	GetImageTaskWorkerPoolManager().Stop()
}
