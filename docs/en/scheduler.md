# Scheduler Module

## Overview

The Scheduler module is the core of AlgoGPU, implementing a simple, deterministic scheduling loop for GPU task management. It follows the principle of "大道至简" (Simplicity is the Ultimate Sophistication).

## Architecture

```
Task Queue (heap) → Scheduler Loop → GPU Pool (Reservation) → Executor (async)
```

## Components

### Scheduler

The main scheduler with a channel-based loop.

**Key Features:**
- Channel-driven, no busy-wait polling
- Deterministic scheduling behavior
- Graceful shutdown support

**Core Methods:**
- `NewScheduler()` - Create new scheduler instance
- `SubmitTask()` - Submit task with admission and rate limiting checks
- `Loop()` - Core scheduling loop
- `Start()` - Start scheduler
- `Stop()` - Stop scheduler gracefully

### AdmissionControl

Protects system from request floods by checking queue capacity.

**Methods:**
- `NewAdmissionControl()` - Create admission controller
- `Check()` - Check if queue has capacity

### TokenBucketManager

Implements user-level rate limiting with daily quotas.

**Features:**
- Token bucket per user
- Daily usage tracking
- Automatic token refill

**Methods:**
- `CheckAndConsume()` - Check and consume tokens for a task
- `Allow()` - Check if user has available tokens
- `GetTokenBalance()` - Get current token balance

### GPUPackingStrategy

Implements Best Fit GPU allocation strategy.

**Features:**
- Best Fit algorithm for GPU selection
- Load threshold consideration
- Maximum GPU utilization

**Methods:**
- `FindBestGPU()` - Find best GPU for a task
- `IsGPUAvailable()` - Check if any GPU can fit the task

### TaskAging

Prevents task starvation by adjusting priority over time.

**Formula:**
```
priority = base_priority + wait_time * γ
```

**Methods:**
- `CalculateAgingPriority()` - Calculate priority with aging
- `GetWaitTime()` - Get task wait time

## Scheduling Flow

1. **Task Submission**
   - Admission control check
   - Token bucket check
   - Enqueue to priority queue

2. **Scheduling Loop**
   - Get task from channel
   - Check token bucket for rate limiting
   - Find best GPU using packing strategy
   - Allocate GPU with reservation
   - Execute task asynchronously

3. **Task Execution**
   - Executor runs task on GPU
   - GPU released after completion
   - Task status updated

## Configuration

```go
type Config struct {
    MaxQueueSize       int     // Maximum queue size
    TokenRefillRate    int64   // Tokens per second
    TokenBucketSize    int64   // Maximum tokens per user
    DailyTokenLimit    int64   // Daily token limit
    GPULoadThreshold   float64 // Max GPU load (0-1)
    AgingFactor        float64 // Task aging factor
    UsageWindowMinutes int     // Usage tracking window
}
```

## Error Handling

The scheduler uses Go's standard error handling pattern:

```go
if err := scheduler.SubmitTask(task); err != nil {
    // Handle error
}
```

## Concurrency Safety

All shared state is protected by mutexes:
- `sync.RWMutex` for read-heavy operations
- `sync.Mutex` for write operations
- No race conditions

## Performance

- **Memory**: Minimal allocations in hot paths
- **CPU**: Channel-driven, no busy-wait
- **Scalability**: Tested with 1000+ concurrent tasks

## Testing

Run tests with:
```bash
make test
```

Run with race detector:
```bash
make test-race
```

## Best Practices

1. Always call `Stop()` before program exit
2. Handle errors from `SubmitTask()`
3. Monitor queue size for load balancing
4. Use appropriate token limits per user