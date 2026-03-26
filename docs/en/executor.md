# Executor Module

## Overview

The Executor module handles asynchronous task execution on GPUs. It ensures proper resource cleanup and provides a simple interface for running different types of GPU tasks.

## Components

### Runner

Executes GPU tasks asynchronously with automatic resource cleanup.

**Key Features:**
- Asynchronous execution (non-blocking)
- Automatic GPU release on completion
- Support for multiple task types
- Timeout handling
- Error propagation

**Key Methods:**
- `NewRunner(gpuPool)` - Create new executor
- `Run(task, gpu)` - Execute task asynchronously on GPU

## Task Types

The executor supports four types of GPU tasks:

1. **TASK_TYPE_EMBEDDING** - Text embedding generation
2. **TASK_TYPE_LLM** - Large language model inference
3. **TASK_TYPE_DIFFUSION** - Diffusion model generation
4. **TASK_TYPE_OTHER** - Generic GPU tasks

## Execution Flow

```
Run(task, gpu) → executeTask(task, gpu) → Update Status → Release GPU
```

### Steps

1. **Defer GPU Release**: Ensure GPU is released even if execution fails
2. **Execute Task**: Call appropriate task handler based on type
3. **Measure Duration**: Track execution time
4. **Update Status**: Set task status based on result
5. **Log Result**: Log success or failure

## Task Handlers

### runEmbeddingTask

Executes embedding tasks with a simulated 20ms latency.

```go
func (r *Runner) runEmbeddingTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runLLMTask

Executes LLM inference tasks with a simulated 100ms latency.

```go
func (r *Runner) runLLMTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runDiffusionTask

Executes diffusion model tasks with a simulated 500ms latency.

```go
func (r *Runner) runDiffusionTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runGenericTask

Executes generic GPU tasks with a simulated 50ms latency.

```go
func (r *Runner) runGenericTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

## Error Handling

The executor uses Go's standard error handling:

- Errors are logged with task ID and GPU ID
- Task status is set to FAILED on error
- GPU is released regardless of success or failure
- Errors are propagated for monitoring

**Example:**
```go
if err != nil {
    log.Printf("Task %s failed on GPU %d: %v", task.ID, gpu.ID, err)
    task.Status = api.TaskStatus_TASK_STATUS_FAILED
    task.Message = err.Error()
}
```

## Timeout Handling

Each task type has a 30-minute timeout:

```go
ctx, cancel := context.WithTimeout(r.ctx, 30*time.Minute)
defer cancel()
```

If a task exceeds the timeout:
- Context is cancelled
- Task fails with context error
- GPU is released

## Resource Management

### Automatic Cleanup

The executor ensures proper cleanup using defer:

```go
defer func() {
    if err := r.gpuPool.Release(gpu); err != nil {
        log.Printf("Failed to release GPU %d: %v", gpu.ID, err)
    }
}()
```

This guarantees:
- GPU is released even on panic
- GPU is released even on error
- No resource leaks

## Usage Example

```go
executor := executor.NewRunner(gpuPool)

task := &types.Task{
    ID: "task-1",
    Type: api.TaskType_TASK_TYPE_LLM,
}

gpu, _ := gpuPool.Allocate(8192)

// Execute asynchronously
go executor.Run(task, gpu)
```

## Performance

- **Concurrency**: Multiple tasks can run in parallel
- **Memory**: Minimal allocations per task
- **Cleanup**: Guaranteed resource release

## Best Practices

1. Always call with `go` keyword for async execution
2. Handle errors from GPU release
3. Monitor task execution time
4. Use appropriate timeouts for different task types
5. Log failures for debugging

## Testing

Run executor tests:
```bash
go test ./internal/executor/...
```

## Notes

- Current implementation uses simulated latencies
- Production should integrate with actual GPU runtimes
- Consider adding retry logic for transient failures
- Add metrics for task execution time