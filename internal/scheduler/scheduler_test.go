package scheduler

import (
	"testing"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/pkg/types"
)

func TestNewScheduler(t *testing.T) {
	gpuPool := gpu.NewPool()
	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       100,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	if sched == nil {
		t.Fatal("NewScheduler() returned nil")
	}

	if sched.cfg != cfg {
		t.Error("NewScheduler() did not set config")
	}

	if sched.taskQueue != taskQueue {
		t.Error("NewScheduler() did not set task queue")
	}

	if sched.gpuPool != gpuPool {
		t.Error("NewScheduler() did not set GPU pool")
	}
}

func TestScheduler_SubmitTask(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	accepted, message, status := sched.SubmitTask(task)

	if !accepted {
		t.Errorf("SubmitTask() accepted = false, want true")
	}

	if message != "Task accepted" {
		t.Errorf("SubmitTask() message = %v, want 'Task accepted'", message)
	}

	if status != api.TaskStatus_TASK_STATUS_PENDING {
		t.Errorf("SubmitTask() status = %v, want PENDING", status)
	}

	// Verify task is in queue
	if taskQueue.Len() != 1 {
		t.Errorf("Queue length = %d, want 1", taskQueue.Len())
	}
}

func TestScheduler_StartStop(t *testing.T) {
	gpuPool := gpu.NewPool()
	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	// Initial state should not be running
	if sched.IsRunning() {
		t.Error("Scheduler should not be running initially")
	}

	// Start scheduler
	sched.Start()

	// Wait a bit for start
	time.Sleep(100 * time.Millisecond)

	if !sched.IsRunning() {
		t.Error("Scheduler should be running after Start()")
	}

	// Start again should be idempotent
	sched.Start()

	if !sched.IsRunning() {
		t.Error("Scheduler should still be running after second Start()")
	}

	// Stop scheduler
	sched.Stop()

	// Wait a bit for stop
	time.Sleep(100 * time.Millisecond)

	if sched.IsRunning() {
		t.Error("Scheduler should not be running after Stop()")
	}

	// Stop again should be idempotent
	sched.Stop()

	if sched.IsRunning() {
		t.Error("Scheduler should still not be running after second Stop()")
	}
}

func TestScheduler_GetSchedulerStatus(t *testing.T) {
	gpuPool := gpu.NewPool()
	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	status := sched.GetSchedulerStatus()

	if status == nil {
		t.Fatal("GetSchedulerStatus() returned nil")
	}

	if running, ok := status["running"].(bool); !ok || running {
		t.Error("GetSchedulerStatus() running should be false initially")
	}

	if _, ok := status["queue_size"]; !ok {
		t.Error("GetSchedulerStatus() should have queue_size")
	}

	if _, ok := status["running_tasks"]; !ok {
		t.Error("GetSchedulerStatus() should have running_tasks")
	}

	// Start scheduler
	sched.Start()
	defer sched.Stop()

	status = sched.GetSchedulerStatus()

	if running, ok := status["running"].(bool); !ok || !running {
		t.Error("GetSchedulerStatus() running should be true after Start()")
	}
}

func TestScheduler_TaskExecution(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)
	sched.Start()
	defer sched.Stop()

	// Submit a small task
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 100,
	}

	accepted, _, _ := sched.SubmitTask(task)
	if !accepted {
		t.Fatal("Task should be accepted")
	}

	// Wait for task to complete
	time.Sleep(500 * time.Millisecond)

	// Check task status
	retrievedTask, exists := taskQueue.Get("task-1")
	if !exists {
		t.Fatal("Task should exist in queue")
	}

	if retrievedTask.Status != api.TaskStatus_TASK_STATUS_COMPLETED {
		t.Errorf("Task status = %v, want COMPLETED", retrievedTask.Status)
	}
}

func TestScheduler_AdmissionControl(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       2, // Small queue
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	// Submit tasks until queue is full
	for i := 0; i < 2; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		accepted, _, _ := sched.SubmitTask(task)
		if !accepted {
			t.Errorf("Task %d should be accepted", i)
		}
	}

	// This task should be rejected (queue full)
	task := &types.Task{
		ID:                 "task-3",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	accepted, message, status := sched.SubmitTask(task)

	if accepted {
		t.Error("Task should be rejected when queue is full")
	}

	if status != api.TaskStatus_TASK_STATUS_REJECTED {
		t.Errorf("Task status = %v, want REJECTED", status)
	}

	if message == "" {
		t.Error("Rejection message should not be empty")
	}
}

func TestScheduler_GetGPUPool(t *testing.T) {
	gpuPool := gpu.NewPool()
	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	if sched.GetGPUPool() != gpuPool {
		t.Error("GetGPUPool() should return the same GPU pool")
	}
}

func TestScheduler_GetTaskQueue(t *testing.T) {
	gpuPool := gpu.NewPool()
	taskQueue := queue.NewTaskQueue()

	cfg := &Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := NewScheduler(cfg, taskQueue, gpuPool)

	if sched.GetTaskQueue() != taskQueue {
		t.Error("GetTaskQueue() should return the same task queue")
	}
}
