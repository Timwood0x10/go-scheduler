// Package executor provides task execution functionality with async execution.
// All task executions must be asynchronous to avoid blocking the scheduler.
package executor

import (
	"context"
	"log"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

// Runner executes GPU tasks asynchronously.
// It ensures GPU resources are released after execution.
type Runner struct {
	gpuPool *gpu.Pool
	ctx     context.Context
}

// NewRunner creates a new task executor.
func NewRunner(gpuPool *gpu.Pool) *Runner {
	return &Runner{
		gpuPool: gpuPool,
		ctx:     context.Background(),
	}
}

// Run executes a task asynchronously on the specified GPU.
// This method must be called with `go r.Run(task, gpu)` to avoid blocking the scheduler.
// The GPU is released automatically after execution completes (success or failure).
func (r *Runner) Run(task *types.Task, gpu *gpu.GPU) {
	// Ensure GPU is released even if execution fails
	defer func() {
		if err := r.gpuPool.Release(gpu); err != nil {
			log.Printf("Failed to release GPU %d: %v", gpu.ID, err)
		}
	}()

	startTime := time.Now()

	// Execute task based on type
	err := r.executeTask(task)

	duration := time.Since(startTime)

	// Update task status based on execution result
	if err != nil {
		log.Printf("Task %s failed on GPU %d: %v (duration: %v)", task.ID, gpu.ID, err, duration)
		task.Status = api.TaskStatus_TASK_STATUS_FAILED
		task.Message = err.Error()
	} else {
		log.Printf("Task %s completed on GPU %d (duration: %v)", task.ID, gpu.ID, duration)
		task.Status = api.TaskStatus_TASK_STATUS_COMPLETED
		task.Message = "Task completed successfully"
	}

	task.CompletedAt = time.Now()
}

// executeTask executes a task based on its type.
// This is a simplified implementation for demonstration.
// In production, this would call actual GPU kernels or container runtimes.
func (r *Runner) executeTask(task *types.Task) error {
	ctx, cancel := context.WithTimeout(r.ctx, 30*time.Minute)
	defer cancel()

	switch task.Type {
	case api.TaskType_TASK_TYPE_EMBEDDING:
		return r.runEmbeddingTask(ctx, task)
	case api.TaskType_TASK_TYPE_LLM:
		return r.runLLMTask(ctx, task)
	case api.TaskType_TASK_TYPE_DIFFUSION:
		return r.runDiffusionTask(ctx, task)
	default:
		return r.runGenericTask(ctx, task)
	}
}

// runEmbeddingTask simulates an embedding task execution.
func (r *Runner) runEmbeddingTask(ctx context.Context, task *types.Task) error {
	// Simulate embedding computation time
	select {
	case <-time.After(20 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runLLMTask simulates an LLM inference task execution.
func (r *Runner) runLLMTask(ctx context.Context, task *types.Task) error {
	// Simulate LLM inference time
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runDiffusionTask simulates a diffusion model task execution.
func (r *Runner) runDiffusionTask(ctx context.Context, task *types.Task) error {
	// Simulate diffusion model generation time
	select {
	case <-time.After(500 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// runGenericTask simulates a generic GPU task execution.
func (r *Runner) runGenericTask(ctx context.Context, task *types.Task) error {
	// Simulate generic task execution time
	select {
	case <-time.After(50 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
