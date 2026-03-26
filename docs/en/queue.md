# Queue Module

## Overview

The Queue module implements a priority-based task queue using Go's `container/heap` package. It provides O(log n) operations for enqueue and dequeue, with support for task aging and concurrent access.

## Components

### TaskQueue

A thread-safe priority queue for GPU tasks.

**Key Features:**
- Priority-based ordering
- Aging support (wait time priority)
- Thread-safe operations
- Efficient O(log n) operations
- Task state tracking

**Key Methods:**
- `NewTaskQueue()` - Create new task queue
- `Enqueue(task)` - Add task to queue
- `Dequeue()` - Remove and return highest priority task
- `Get(taskID)` - Get task by ID
- `Cancel(taskID)` - Cancel a task
- `UpdateStatus(taskID, status)` - Update task status
- `Requeue(task)` - Requeue a task
- `GetAllPending()` - Get all pending tasks
- `GetAllRunning()` - Get all running tasks
- `Len()` - Get queue length
- `RunningCount()` - Get running task count

### TaskHeap

Internal heap implementation using `container/heap`.

**Methods:**
- `Len()` - Get heap size
- `Less(i, j)` - Compare tasks for priority
- `Swap(i, j)` - Swap tasks in heap
- `Push(x)` - Add task to heap
- `Pop()` - Remove task from heap

## Priority Calculation

Tasks are ordered by priority with aging consideration:

```
priority = base_priority + (wait_time / 1000) * 0.1
```

Where:
- `base_priority`: Task's priority (higher = more important)
- `wait_time`: Time since task creation (milliseconds)
- `0.1`: Aging factor

**Priority Order:**
1. Task type (LLM > Diffusion > Embedding > Other)
2. User priority (if configured)
3. Wait time (aging)

## Task States

Tasks transition through these states:

```
PENDING → RUNNING → COMPLETED
    ↓
CANCELLED
    ↓
FAILED
```

## Thread Safety

All operations are protected by `sync.RWMutex`:

- `sync.RLock()` for read operations (Get, Len, etc.)
- `sync.Lock()` for write operations (Enqueue, Dequeue, etc.)

## Usage Examples

### Enqueue Task

```go
task := &types.Task{
    ID: "task-1",
    UserID: "user-1",
    Type: api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired: 8192,
    Priority: 100,
}

err := queue.Enqueue(task)
if err != nil {
    // Handle error
}
```

### Dequeue Task

```go
task := queue.Dequeue()
if task == nil {
    // Queue is empty
}
```

### Cancel Task

```go
err := queue.Cancel("task-1")
if err != nil {
    // Task not found or already cancelled
}
```

### Update Task Status

```go
err := queue.UpdateStatus("task-1", api.TaskStatus_TASK_STATUS_RUNNING)
if err != nil {
    // Task not found
}
```

## Error Handling

The module defines specific errors:

- `ErrQueueFull` - Queue is at capacity
- `ErrTaskExists` - Task already in queue
- `ErrTaskNotFound` - Task not found in queue

## Performance

| Operation | Time Complexity | Space Complexity |
|-----------|----------------|------------------|
| Enqueue   | O(log n)       | O(1)             |
| Dequeue   | O(log n)       | O(1)             |
| Get       | O(1)           | O(1)             |
| Cancel    | O(log n)       | O(1)             |
| Len       | O(1)           | O(1)             |

## Best Practices

1. Handle `ErrQueueFull` gracefully
2. Use `GetAllPending()` for monitoring
3. Always call `Cancel()` for tasks you don't need
4. Update task status as it progresses
5. Use appropriate priority values

## Testing

Run queue tests:
```bash
go test ./internal/queue/...
```

## Concurrency

The queue is designed for high concurrency:
- Multiple readers can access simultaneously
- Writers block other writers and readers
- Lock-free reads for monitoring operations