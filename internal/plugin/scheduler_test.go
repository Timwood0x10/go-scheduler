package plugin

import (
	"context"
	"testing"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/internal/scheduler"
	"algogpu/pkg/types"
)

func TestPluginScheduler_SubmitTask(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	tests := []struct {
		name    string
		task    *types.Task
		wantErr bool
	}{
		{
			name: "valid task",
			task: &types.Task{
				ID:                 "task-1",
				UserID:             "user-1",
				Type:               api.TaskType_TASK_TYPE_EMBEDDING,
				GPUMemoryRequired:  1024,
				GPUComputeRequired: 100,
				EstimatedRuntimeMs: 1000,
			},
			wantErr: false,
		},
		{
			name:    "nil task",
			task:    nil,
			wantErr: true,
		},
		{
			name: "empty task ID",
			task: &types.Task{
				ID:                 "",
				UserID:             "user-1",
				Type:               api.TaskType_TASK_TYPE_EMBEDDING,
				GPUMemoryRequired:  1024,
				GPUComputeRequired: 100,
			},
			wantErr: true,
		},
		{
			name: "empty user ID",
			task: &types.Task{
				ID:                 "task-2",
				UserID:             "",
				Type:               api.TaskType_TASK_TYPE_EMBEDDING,
				GPUMemoryRequired:  1024,
				GPUComputeRequired: 100,
			},
			wantErr: true,
		},
		{
			name: "zero memory required",
			task: &types.Task{
				ID:                 "task-3",
				UserID:             "user-1",
				Type:               api.TaskType_TASK_TYPE_EMBEDDING,
				GPUMemoryRequired:  0,
				GPUComputeRequired: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pluginScheduler.SubmitTask(ctx, tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPluginScheduler_GetStatus(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	// Submit a task
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := pluginScheduler.SubmitTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Get task status
	retrievedTask, err := pluginScheduler.GetStatus(ctx, "task-1")
	if err != nil {
		t.Errorf("GetStatus() error = %v", err)
	}

	if retrievedTask == nil {
		t.Fatal("GetStatus() returned nil task")
	}

	if retrievedTask.ID != "task-1" {
		t.Errorf("GetStatus() ID = %v, want task-1", retrievedTask.ID)
	}

	// Try to get non-existent task
	_, err = pluginScheduler.GetStatus(ctx, "non-existent")
	if err == nil {
		t.Error("GetStatus() expected error for non-existent task")
	}

	// Try empty task ID
	_, err = pluginScheduler.GetStatus(ctx, "")
	if err == nil {
		t.Error("GetStatus() expected error for empty task ID")
	}
}

func TestPluginScheduler_CancelTask(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	// Submit a task
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := pluginScheduler.SubmitTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Cancel the task
	err = pluginScheduler.CancelTask(ctx, "task-1")
	if err != nil {
		t.Errorf("CancelTask() error = %v", err)
	}

	// Verify task is removed from queue
	_, err = pluginScheduler.GetStatus(ctx, "task-1")
	if err == nil {
		t.Error("GetStatus() should return error for cancelled task")
	}

	// Try to cancel non-existent task
	err = pluginScheduler.CancelTask(ctx, "non-existent")
	if err == nil {
		t.Error("CancelTask() expected error for non-existent task")
	}

	// Try empty task ID
	err = pluginScheduler.CancelTask(ctx, "")
	if err == nil {
		t.Error("CancelTask() expected error for empty task ID")
	}
}

func TestPluginScheduler_GetQueueSize(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	// Initial queue size should be 0
	if size := pluginScheduler.GetQueueSize(); size != 0 {
		t.Errorf("GetQueueSize() = %v, want 0", size)
	}

	// Submit tasks
	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}
		err := pluginScheduler.SubmitTask(ctx, task)
		if err != nil {
			t.Fatalf("Failed to submit task: %v", err)
		}
	}

	// Queue size should be 3
	if size := pluginScheduler.GetQueueSize(); size != 3 {
		t.Errorf("GetQueueSize() = %v, want 3", size)
	}
}

func TestPluginScheduler_GetRunningCount(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	sched.Start()
	defer sched.Stop()

	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	// Initial running count should be 0
	if count := pluginScheduler.GetRunningCount(); count != 0 {
		t.Errorf("GetRunningCount() = %v, want 0", count)
	}

	// Submit a small task
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 100,
	}

	err := pluginScheduler.SubmitTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to submit task: %v", err)
	}

	// Wait a bit for task to start
	time.Sleep(200 * time.Millisecond)

	// Running count should be 1 (or 0 if task completed quickly)
	count := pluginScheduler.GetRunningCount()
	if count < 0 || count > 1 {
		t.Errorf("GetRunningCount() = %v, want 0 or 1", count)
	}
}

func TestPluginScheduler_GetGPUStatus(t *testing.T) {
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "GPU-0", 8192)
	gpuPool.AddGPU(1, "GPU-1", 16384)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       10,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := NewPluginScheduler(sched, taskQueue, gpuPool)

	ctx := context.Background()

	gpus := pluginScheduler.GetGPUStatus(ctx)
	if len(gpus) != 2 {
		t.Errorf("GetGPUStatus() returned %d GPUs, want 2", len(gpus))
	}

	if gpus[0].Name != "GPU-0" {
		t.Errorf("GPU 0 name = %v, want GPU-0", gpus[0].Name)
	}

	if gpus[0].MemoryTotal != 8192 {
		t.Errorf("GPU 0 memory = %v, want 8192", gpus[0].MemoryTotal)
	}
}
