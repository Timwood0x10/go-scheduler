// Package plugin provides a simple interface for embedding the scheduler as a plugin.
// This mode is designed for lightweight integration with agent frameworks.
package plugin

import (
	"context"
	"fmt"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/internal/scheduler"
	"algogpu/pkg/types"
)

// Scheduler defines the minimal interface for GPU scheduling in plugin mode.
// This interface provides the core methods needed for task scheduling and monitoring.
type Scheduler interface {
	// SubmitTask submits a GPU task for scheduling.
	// Returns error if task submission fails (e.g., queue full, rate limit).
	SubmitTask(ctx context.Context, task *types.Task) error

	// GetStatus returns the current status of a task.
	// Returns the task if found, nil if not found, and any error encountered.
	GetStatus(ctx context.Context, taskID string) (*types.Task, error)

	// GetQueueSize returns the current number of pending tasks.
	GetQueueSize() int

	// GetRunningCount returns the current number of running tasks.
	GetRunningCount() int

	// GetGPUStatus returns status of all GPUs.
	GetGPUStatus(ctx context.Context) []*api.GPUInfo

	// CancelTask cancels a pending or running task.
	CancelTask(ctx context.Context, taskID string) error
}

// PluginScheduler implements the Scheduler interface by wrapping the core scheduler.
type PluginScheduler struct {
	scheduler *scheduler.Scheduler
	taskQueue *queue.TaskQueue
	gpuPool   *gpu.Pool
}

// NewPluginScheduler creates a new plugin scheduler instance.
// This is the recommended way to embed the scheduler in agent frameworks.
func NewPluginScheduler(sched *scheduler.Scheduler, taskQueue *queue.TaskQueue, gpuPool *gpu.Pool) *PluginScheduler {
	return &PluginScheduler{
		scheduler: sched,
		taskQueue: taskQueue,
		gpuPool:   gpuPool,
	}
}

// SubmitTask submits a task to the scheduler with validation.
func (p *PluginScheduler) SubmitTask(ctx context.Context, task *types.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if task.UserID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if task.GPUMemoryRequired <= 0 {
		return fmt.Errorf("GPU memory required must be positive")
	}

	// Submit to core scheduler
	accepted, message, status := p.scheduler.SubmitTask(task)
	if !accepted {
		return fmt.Errorf("task rejected: %s (status: %s)", message, status)
	}

	return nil
}

// GetStatus retrieves the current status of a task.
func (p *PluginScheduler) GetStatus(ctx context.Context, taskID string) (*types.Task, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task ID cannot be empty")
	}

	task, exists := p.taskQueue.Get(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetGPUPool returns the underlying GPU pool for advanced use cases.
// Most plugin users should not need this.
func (p *PluginScheduler) GetGPUPool() *gpu.Pool {
	return p.gpuPool
}

// GetTaskQueue returns the underlying task queue for advanced use cases.
// Most plugin users should not need this.
func (p *PluginScheduler) GetTaskQueue() *queue.TaskQueue {
	return p.taskQueue
}

// CancelTask cancels a pending or running task.
// This is a convenience method for plugin mode.
func (p *PluginScheduler) CancelTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	return p.taskQueue.Cancel(taskID)
}

// GetQueueSize returns the current number of pending tasks.
func (p *PluginScheduler) GetQueueSize() int {
	return p.taskQueue.Len()
}

// GetRunningCount returns the current number of running tasks.
func (p *PluginScheduler) GetRunningCount() int {
	return p.taskQueue.RunningCount()
}

// GetGPUStatus returns status of all GPUs.
func (p *PluginScheduler) GetGPUStatus(ctx context.Context) []*api.GPUInfo {
	gpus := p.gpuPool.GetAllGPUs()
	infos := make([]*api.GPUInfo, len(gpus))

	for i, gpu := range gpus {
		infos[i] = gpu.ToProto()
	}

	return infos
}
