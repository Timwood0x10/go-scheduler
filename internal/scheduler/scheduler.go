// Package scheduler provides the core GPU scheduling logic with a simple, deterministic loop.
// The scheduler follows the principle of "大道至简" - simple, predictable, and reliable.
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/internal/db"
	"algogpu/internal/executor"
	"algogpu/internal/gpu"
	"algogpu/internal/policy"
	"algogpu/internal/predictor"
	"algogpu/internal/queue"
	"algogpu/pkg/types"
)

// Scheduler is the main GPU scheduler with a simple, deterministic loop.
// This is the core of the system - keep it minimal and focused.
type Scheduler struct {
	cfg       *Config
	taskQueue *queue.TaskQueue
	gpuPool   *gpu.Pool
	executor  *executor.Runner

	// Scheduling components (simplified to 3 core strategies)
	admission   *AdmissionControl
	tokenBucket *TokenBucketManager
	gpuPacking  *GPUPackingStrategy

	// Data-driven components
	dbStore      *db.SQLiteStore
	predictor    *predictor.ResourcePredictor
	policyEngine *policy.Engine

	// Scheduler state
	mu        sync.RWMutex
	running   bool
	queueChan chan *types.Task
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewScheduler creates a new Scheduler with minimal configuration.
func NewScheduler(cfg *Config, taskQueue *queue.TaskQueue, gpuPool *gpu.Pool, dbStore *db.SQLiteStore) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize data-driven components
	cache := predictor.NewStatsCache(5 * time.Minute)
	resPredictor := predictor.NewResourcePredictor(dbStore, cache)
	policyEngine := policy.NewEngine(resPredictor)

	return &Scheduler{
		cfg:       cfg,
		taskQueue: taskQueue,
		gpuPool:   gpuPool,
		executor:  executor.NewRunner(gpuPool, dbStore),

		admission:   NewAdmissionControl(cfg.MaxQueueSize, taskQueue),
		tokenBucket: NewTokenBucketManager(cfg),
		gpuPacking:  NewGPUPackingStrategy(gpuPool, cfg),

		running:   false,
		queueChan: make(chan *types.Task, 1000),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// SubmitTask submits a task with all scheduling checks.
// Returns whether accepted, message, and task status.
func (s *Scheduler) SubmitTask(task *types.Task) (bool, string, api.TaskStatus) {
	// 1. Admission Control (queue capacity check)
	if err := s.admission.Check(); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// 2. Token Bucket check (user-level rate limiting)
	if err := s.tokenBucket.CheckAndConsume(task.UserID, task.Type); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// 3. Policy engine evaluation (data-driven decision)
	queueSize := s.taskQueue.Len()
	decision, err := s.policyEngine.EvaluateTask(context.Background(), task, queueSize)
	if err != nil {
		log.Printf("Policy engine evaluation failed for task %s: %v", task.ID, err)
		// Continue with default decision
	} else {
		// Store policy decision in task metadata
		task.EstimatedRuntimeMs = decision.EstimatedDuration
		task.Priority = int64(decision.Priority)
		log.Printf("Task %s policy decision: duration=%dms, priority=%.2f, allow_packing=%v",
			task.ID, decision.EstimatedDuration, decision.Priority, decision.AllowPacking)
	}

	// 4. Enqueue task
	if err := s.taskQueue.Enqueue(task); err != nil {
		return false, err.Error(), api.TaskStatus_TASK_STATUS_REJECTED
	}

	// Send to scheduler channel for processing
	select {
	case s.queueChan <- task:
	default:
		// Channel full, task will be picked up by periodic check
	}

	return true, "Task accepted", api.TaskStatus_TASK_STATUS_PENDING
}

// Loop is the core scheduling loop - the soul of this project.
// It implements a simple, deterministic scheduling logic:
// 1. Get task from channel
// 2. Check token bucket for rate limiting
// 3. Find best GPU using packing strategy
// 4. Execute task asynchronously
func (s *Scheduler) Loop() {
	s.wg.Add(1)
	defer s.wg.Done()

	log.Println("Scheduler loop started")

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Scheduler loop stopped")
			return

		case task := <-s.queueChan:
			if task != nil {
				s.processTask(task)
			}

		case <-time.After(10 * time.Millisecond):
			// Periodic check for tasks that might have been skipped
			s.processNextTask()
		}
	}
}

// processTask processes a single task through the scheduling pipeline.
func (s *Scheduler) processTask(task *types.Task) {
	// User rate limiting (Token Bucket)
	if !s.tokenBucket.Allow(task.UserID) {
		// Requeue with exponential backoff
		time.Sleep(20 * time.Millisecond)
		s.requeueTask(task)
		return
	}

	// GPU Packing (Best Fit strategy)
	gpu, err := s.gpuPacking.FindBestGPU(task)
	if err != nil || gpu == nil {
		// No GPU available, requeue
		time.Sleep(50 * time.Millisecond)
		s.requeueTask(task)
		return
	}

	// Allocate GPU
	if err := gpu.Allocate(task.ID, task.GPUMemoryRequired); err != nil {
		log.Printf("Failed to allocate GPU %d for task %s: %v", gpu.ID, task.ID, err)
		s.requeueTask(task)
		return
	}

	// Update task status to running
	if err := s.taskQueue.UpdateStatus(task.ID, api.TaskStatus_TASK_STATUS_RUNNING); err != nil {
		log.Printf("Failed to update task status: %v", err)
		if releaseErr := s.gpuPool.Release(gpu); releaseErr != nil {
			log.Printf("Failed to release GPU %d: %v", gpu.ID, releaseErr)
		}
		return
	}

	log.Printf("Dispatched task %s to GPU %d", task.ID, gpu.ID)

	// Execute task asynchronously (must be async to avoid blocking scheduler)
	go s.executor.Run(task, gpu)
}

// processNextTask processes the next task from the queue.
// This is called periodically to ensure no tasks are stuck.
func (s *Scheduler) processNextTask() {
	// Dequeue the highest priority task from the heap
	task := s.taskQueue.Dequeue()
	if task == nil {
		return
	}

	// Reuse processTask logic
	s.processTask(task)
}

// requeueTask requeues a task for later processing.
func (s *Scheduler) requeueTask(task *types.Task) {
	if err := s.taskQueue.Requeue(task); err != nil {
		log.Printf("Failed to requeue task %s: %v", task.ID, err)
	}
}

// Start starts the scheduler loop.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	go s.Loop()

	log.Println("Scheduler started")
}

// Stop stops the scheduler loop gracefully.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.cancel()

	// Wait for loop to finish
	s.wg.Wait()

	log.Println("Scheduler stopped")
}

// GetSchedulerStatus returns scheduler status.
func (s *Scheduler) GetSchedulerStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"running":       s.running,
		"queue_size":    s.taskQueue.Len(),
		"running_tasks": s.taskQueue.RunningCount(),
	}
}

// GetGPUPool returns the GPU pool.
func (s *Scheduler) GetGPUPool() *gpu.Pool {
	return s.gpuPool
}

// GetTaskQueue returns the task queue.
func (s *Scheduler) GetTaskQueue() *queue.TaskQueue {
	return s.taskQueue
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
