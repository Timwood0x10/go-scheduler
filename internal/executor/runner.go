// Package executor provides task execution functionality with async execution.
// All task executions must be asynchronous to avoid blocking the scheduler.
package executor

import (
	"context"
	"log"
	"time"

	"algogpu/api"
	"algogpu/internal/db"
	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

// Runner executes GPU tasks asynchronously.
// It ensures GPU resources are released after execution.
type Runner struct {
	gpuPool *gpu.Pool
	ctx     context.Context
	dbStore *db.SQLiteStore
}

// NewRunner creates a new task executor.
func NewRunner(gpuPool *gpu.Pool, dbStore *db.SQLiteStore) *Runner {
	return &Runner{
		gpuPool: gpuPool,
		ctx:     context.Background(),
		dbStore: dbStore,
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
	success := err == nil
	if success {
		log.Printf("Task %s completed on GPU %d (duration: %v)", task.ID, gpu.ID, duration)
		task.Status = api.TaskStatus_TASK_STATUS_COMPLETED
		task.Message = "Task completed successfully"
	} else {
		log.Printf("Task %s failed on GPU %d: %v (duration: %v)", task.ID, gpu.ID, err, duration)
		task.Status = api.TaskStatus_TASK_STATUS_FAILED
		task.Message = err.Error()
	}

	task.CompletedAt = time.Now()

	// Record execution metrics to database
	if r.dbStore != nil {
		queueWait := task.StartedAt.Sub(task.CreatedAt)

		// Calculate metrics from GPU
		avgGPUUtil := float64(gpu.ComputeUtil)
		maxGPUUtil := float64(gpu.ComputeUtil)
		avgMemUtil := float64(gpu.MemoryUtil)
		maxMemUtil := float64(gpu.MemoryUtil)
		gpuMemoryUsedMB := gpu.MemoryUsed

		// Create execution record
		record := &db.TaskExecutionRecord{
			TaskID:          task.ID,
			TaskType:        task.Type.String(),
			UserID:          task.UserID,
			GPUID:           gpu.ID,
			GPUModel:        gpu.Name,
			Priority:        int(task.Priority),
			QueueWaitMs:     queueWait.Milliseconds(),
			ExecutionTimeMs: duration.Milliseconds(),
			AvgGPUUtil:      avgGPUUtil,
			MaxGPUUtil:      maxGPUUtil,
			AvgMemUtil:      avgMemUtil,
			MaxMemUtil:      maxMemUtil,
			GPUMemoryUsedMB: gpuMemoryUsedMB,
			Success:         success,
			CreatedAt:       task.CreatedAt.Unix(),
		}

		if err := r.dbStore.RecordExecution(context.Background(), record); err != nil {

			log.Printf("Failed to record execution metrics for task %s: %v", task.ID, err)

		}

	}
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
