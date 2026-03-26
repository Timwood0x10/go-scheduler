# Plugin Module

## Overview

The Plugin module provides a minimal interface for embedding the AlgoGPU scheduler as a plugin in agent frameworks. It offers two deployment modes: standalone service and embedded plugin.

## Components

### Scheduler Interface

Minimal interface with 6 core methods for plugin mode.

```go
type Scheduler interface {
    SubmitTask(ctx context.Context, task *types.Task) error
    GetStatus(ctx context.Context, taskID string) (*types.Task, error)
    GetQueueSize() int
    GetRunningCount() int
    GetGPUStatus(ctx context.Context) []*api.GPUInfo
    CancelTask(ctx context.Context, taskID string) error
}
```

### PluginScheduler

Implementation of the Scheduler interface that wraps the core scheduler.

**Key Features:**
- Simple validation
- Error handling
- Integration with core scheduler
- Direct access to GPU pool and task queue

**Key Methods:**
- `NewPluginScheduler(sched, taskQueue, gpuPool)` - Create plugin scheduler
- `SubmitTask(ctx, task)` - Submit task with validation
- `GetStatus(ctx, taskID)` - Get task status
- `GetQueueSize()` - Get pending task count
- `GetRunningCount()` - Get running task count
- `GetGPUStatus(ctx)` - Get all GPU status
- `CancelTask(ctx, taskID)` - Cancel a task

## Deployment Modes

### Standalone Mode

Run as a separate process with gRPC and HTTP APIs.

**Usage:**
```bash
make build-standalone
make run-standalone
```

**Features:**
- Full API support (gRPC + HTTP)
- Independent deployment
- Suitable for production
- Easy monitoring

### Plugin Mode

Embed directly in agent applications.

**Usage:**
```go
scheduler := plugin.NewPluginScheduler(coreScheduler, taskQueue, gpuPool)
err := scheduler.SubmitTask(ctx, task)
```

**Features:**
- No network overhead
- Simple interface
- Easy integration
- Fast startup

## Usage Examples

### Initialize Plugin Scheduler

```go
// Create core components
cfg := scheduler.DefaultConfig()
taskQueue := queue.NewTaskQueue()
gpuPool := gpu.NewPool()
coreScheduler := scheduler.NewScheduler(cfg, taskQueue, gpuPool)

// Create plugin scheduler
pluginScheduler := plugin.NewPluginScheduler(coreScheduler, taskQueue, gpuPool)

// Start scheduler
coreScheduler.Start()
```

### Submit Task

```go
task := &types.Task{
    ID:                 "task-1",
    UserID:             "user-1",
    Type:               api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired:  8192,
    GPUComputeRequired: 200,
    EstimatedRuntimeMs: 5000,
    Payload:            []byte("Hello, world!"),
}

err := pluginScheduler.SubmitTask(ctx, task)
if err != nil {
    // Handle error
}
```

### Get Task Status

```go
task, err := pluginScheduler.GetStatus(ctx, "task-1")
if err != nil {
    // Task not found
}

fmt.Printf("Status: %v\n", task.Status)
```

### Cancel Task

```go
err := pluginScheduler.CancelTask(ctx, "task-1")
if err != nil {
    // Task not found or already cancelled
}
```

### Monitor Queue

```go
queueSize := pluginScheduler.GetQueueSize()
runningCount := pluginScheduler.GetRunningCount()

fmt.Printf("Queue size: %d, Running: %d\n", queueSize, runningCount)
```

### Get GPU Status

```go
gpuStatus := pluginScheduler.GetGPUStatus(ctx)
for _, gpu := range gpuStatus {
    fmt.Printf("GPU %d: %s\n", gpu.Id, gpu.Name)
}
```

## Validation

The plugin scheduler validates all inputs:

- Task cannot be nil
- Task ID cannot be empty
- User ID cannot be empty
- GPU memory required must be positive

**Example:**
```go
err := pluginScheduler.SubmitTask(ctx, nil)
// Error: "task cannot be nil"
```

## Error Handling

All errors are returned for proper handling:

```go
err := pluginScheduler.SubmitTask(ctx, task)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "task cannot be nil"):
        // Handle nil task
    case strings.Contains(err.Error(), "task ID cannot be empty"):
        // Handle empty ID
    case strings.Contains(err.Error(), "GPU memory"):
        // Handle memory validation
    }
}
```

## Best Practices

1. Always validate input before submitting
2. Handle errors gracefully
3. Monitor queue size for load balancing
4. Use appropriate GPU memory requirements
5. Stop scheduler when done

## Testing

Run plugin tests:
```bash
go test ./internal/plugin/...
```

## Integration Example

See `examples/go-agent/main.go` for a complete integration example.

## Performance

- **Submission**: O(1) for validation + O(log n) for enqueue
- **Status Query**: O(1) hash lookup
- **Cancellation**: O(log n) for removal
- **GPU Status**: O(n) for all GPUs