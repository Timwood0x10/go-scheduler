package scheduler

import (
	"log"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/pkg/types"
)

// Scheduler is the main GPU scheduler
type Scheduler struct {
	cfg       *Config
	taskQueue *queue.TaskQueue
	gpuPool   *gpu.Pool

	// Scheduling components
	admission    *AdmissionControl
	tokenBucket  *TokenBucketManager
	costAware    *CostAwareScheduler
	usageTracker *UsageTracker
	gpuPacking   *GPUPackingStrategy
	taskAging    *TaskAging

	// Scheduler state
	mu              sync.RWMutex
	running         bool
	schedulerLoopCh chan struct{}
}

// NewScheduler creates a new Scheduler
func NewScheduler(cfg *Config, taskQueue *queue.TaskQueue, gpuPool *gpu.Pool) *Scheduler {
	usageTracker := NewUsageTracker(cfg.UsageWindowMinutes)

	return &Scheduler{
		cfg:       cfg,
		taskQueue: taskQueue,
		gpuPool:   gpuPool,

		admission:    NewAdmissionControl(cfg.MaxQueueSize, taskQueue),
		tokenBucket:  NewTokenBucketManager(cfg),
		costAware:    NewCostAwareScheduler(usageTracker),
		usageTracker: usageTracker,
		gpuPacking:   NewGPUPackingStrategy(gpuPool, cfg),
		taskAging:    NewTaskAging(cfg.AgingFactor),

		running:         false,
		schedulerLoopCh: make(chan struct{}, 1),
	}
}

// SubmitTask submits a task with all scheduling checks
func (s *Scheduler) SubmitTask(task *types.Task) (bool, string, api.TaskStatus) {
	// 1. Admission Control
	if err := s.admission.Check(); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// 2. Token Bucket check
	if err := s.tokenBucket.CheckAndConsume(task.UserID, task.Type); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// 3. Enqueue task
	if err := s.taskQueue.Enqueue(task); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// Trigger scheduler
	s.triggerScheduler()

	return true, "Task accepted", api.TaskStatus_TASK_STATUS_PENDING
}

// DispatchNext dispatches the next task if possible
func (s *Scheduler) DispatchNext() *types.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all pending tasks
	pendingTasks := s.taskQueue.GetAllPending()
	if len(pendingTasks) == 0 {
		return nil
	}

	// Calculate priorities with cost-aware scheduling
	tasksWithPriority := SortByPriority(pendingTasks, s.costAware)

	// Apply aging
	tasksWithPriority = s.taskAging.ApplyAging(tasksWithPriority, pendingTasks)

	// Try to dispatch tasks in priority order
	for _, twp := range tasksWithPriority {
		task := twp.Task

		// Check GPU availability
		gpu, err := s.gpuPacking.FindBestGPU(task)
		if err != nil {
			continue // try next task
		}

		// Allocate GPU
		if err := gpu.Allocate(task.ID, task.GPUMemoryRequired); err != nil {
			log.Printf("Failed to allocate GPU %d for task %s: %v", gpu.ID, task.ID, err)
			continue
		}

		// Update task status to running
		s.taskQueue.UpdateStatus(task.ID, api.TaskStatus_TASK_STATUS_RUNNING)

		log.Printf("Dispatched task %s to GPU %d", task.ID, gpu.ID)

		return task
	}

	return nil
}

// triggerScheduler triggers a scheduling cycle
func (s *Scheduler) triggerScheduler() {
	select {
	case s.schedulerLoopCh <- struct{}{}:
	default:
	}
}

// Start starts the scheduler loop
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	go s.runSchedulerLoop()

	log.Println("Scheduler started")
}

// Stop stops the scheduler loop
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.schedulerLoopCh)

	log.Println("Scheduler stopped")
}

// runSchedulerLoop runs the main scheduler loop
func (s *Scheduler) runSchedulerLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.schedulerLoopCh:
			s.DispatchNext()
		case <-ticker.C:
			s.DispatchNext()
		case <-time.After(10 * time.Second):
			// Periodic full scan
			s.DispatchNext()
		}
	}
}

// GetSchedulerStatus returns scheduler status
func (s *Scheduler) GetSchedulerStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"running":       s.running,
		"queue_size":    s.taskQueue.Len(),
		"running_tasks": s.taskQueue.RunningCount(),
	}
}

// RecordUsage records GPU usage for a task
func (s *Scheduler) RecordUsage(userID string, taskType api.TaskType) {
	cost := types.GetTaskCost(taskType)
	s.usageTracker.AddUsage(userID, cost)
}

// GetGPUPool returns the GPU pool
func (s *Scheduler) GetGPUPool() *gpu.Pool {
	return s.gpuPool
}

// GetTaskQueue returns the task queue
func (s *Scheduler) GetTaskQueue() *queue.TaskQueue {
	return s.taskQueue
}
