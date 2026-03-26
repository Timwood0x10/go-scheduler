# Types Module

## Overview

The Types module defines core data structures and enumerations used throughout the AlgoGPU system. It provides a common vocabulary for tasks, GPU resources, and system configuration.

## Core Types

### Task

Represents a GPU task with all required metadata.

```go
type Task struct {
    ID                 string           // Unique task identifier
    UserID             string           // User who submitted the task
    Type               TaskType         // Task type (embedding, LLM, etc.)
    GPUMemoryRequired  int64            // Required GPU memory in MB
    GPUComputeRequired int64            // Required GPU compute units
    EstimatedRuntimeMs int64            // Estimated runtime in milliseconds
    Priority           int              // Task priority (higher = more important)
    Status             TaskStatus       // Current task status
    Payload            []byte           // Task payload (input data)
    Result             []byte           // Task result (output data)
    Message            string           // Status message or error
    CreatedAt          time.Time        // Task creation timestamp
    StartedAt          time.Time        // Task start timestamp
    CompletedAt        time.Time        // Task completion timestamp
    GPUID              int              // ID of GPU running the task
}
```

### TaskType

Enumeration of supported GPU task types.

```go
const (
    TaskTypeUnspecified TaskType = 0  // Unspecified type
    TaskTypeEmbedding  TaskType = 1  // Text embedding generation
    TaskTypeLLM        TaskType = 2  // Large language model inference
    TaskTypeDiffusion  TaskType = 3  // Diffusion model generation
    TaskTypeOther      TaskType = 4  // Generic GPU task
)
```

**Usage:**
```go
task := &types.Task{
    Type: api.TaskType_TASK_TYPE_LLM,
}
```

### TaskStatus

Enumeration of task states.

```go
const (
    TaskStatusPending   TaskStatus = 0  // Waiting in queue
    TaskStatusRunning   TaskStatus = 1  // Currently executing
    TaskStatusCompleted TaskStatus = 2  // Finished successfully
    TaskStatusFailed    TaskStatus = 3  // Failed with error
    TaskStatusCancelled TaskStatus = 4  // Cancelled by user
    TaskStatusRejected  TaskStatus = 5  // Rejected by admission control
)
```

**State Transitions:**
```
PENDING → RUNNING → COMPLETED
    ↓
CANCELLED
    ↓
FAILED
```

## Configuration Types

### SchedulerConfig

Scheduler configuration parameters.

```go
type SchedulerConfig struct {
    MaxQueueSize       int     // Maximum queue size
    TokenRefillRate    int64   // Tokens per second
    TokenBucketSize    int64   // Maximum tokens per user
    DailyTokenLimit    int64   // Daily token limit
    GPULoadThreshold   float64 // Max GPU load (0-1)
    AgingFactor        float64 // Task aging factor
    MemoryWeight       float64 // Memory weight in load calculation
    ComputeWeight      float64 // Compute weight in load calculation
}
```

## GPU Types

GPU-related types are defined in the internal/gpu package.

### GPU

Represents a single GPU with its state.

```go
type GPU struct {
    ID            int       // GPU identifier
    Name          string    // GPU name
    MemoryTotal   int64     // Total memory (MB)
    MemoryUsed    int64     // Used memory (MB)
    ComputeUtil   int       // Compute utilization (0-100)
    MemoryUtil    int       // Memory utilization (0-100)
    Temperature   int       // Temperature (Celsius)
    ReservedBy    string    // Reserved by task ID
    RunningTasks  []string  // Running task IDs
}
```

## API Types

API types are defined in the api package using Protocol Buffers.

### TaskRequest

gRPC request for task submission.

### TaskResponse

gRPC response for task submission.

### TaskStatusRequest

gRPC request for task status query.

### TaskStatusResponse

gRPC response for task status query.

## Utility Functions

### GenerateID

Generate unique task IDs.

```go
func GenerateID(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
```

## Best Practices

1. Always set required fields (ID, UserID, Type, GPUMemoryRequired)
2. Use appropriate task types
3. Handle task status transitions properly
4. Validate task fields before submission
5. Use meaningful task IDs for debugging

## Usage Examples

### Create Task

```go
task := &types.Task{
    ID:                 "task-1",
    UserID:             "user-1",
    Type:               api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired:  8192,
    GPUComputeRequired: 200,
    EstimatedRuntimeMs: 5000,
    Priority:           100,
    CreatedAt:          time.Now(),
}
```

### Check Task Status

```go
if task.Status == api.TaskStatus_TASK_STATUS_COMPLETED {
    // Task completed successfully
    fmt.Printf("Result: %s\n", string(task.Result))
} else if task.Status == api.TaskStatus_TASK_STATUS_FAILED {
    // Task failed
    fmt.Printf("Error: %s\n", task.Message)
}
```

## Performance Considerations

- Use `[]byte` for payloads instead of `string` for efficiency
- Keep task metadata small for fast serialization
- Use appropriate data types (e.g., `int64` for timestamps)

## Testing

Run types tests:
```bash
go test ./pkg/types/...
```