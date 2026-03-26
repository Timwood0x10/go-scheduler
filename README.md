# AlgoGPU

A minimal GPU scheduler for AI agents and inference workloads. Simple. Deterministic. Reliable.

## Features

- **Dual Mode Architecture**: Run as standalone service or embedded plugin
- **Simple Scheduling Loop**: Channel-based, deterministic scheduling
- **GPU Packing**: Best Fit strategy with load thresholds
- **Token Bucket**: User-level rate limiting with daily quotas
- **Task Aging**: Prevents starvation with priority adjustment
- **Priority Queue**: Heap-based efficient task ordering
- **gRPC API**: Language-agnostic interface for Python integration
- **Plugin Interface**: Easy embedding in agent frameworks

## Architecture

AlgoGPU supports two deployment modes:

### Mode 1: Standalone Service
Independent gRPC service with HTTP monitoring, suitable for production environments.

```
┌───────────────────┐
│  Standalone Mode  │
├───────────────────┤
│ • gRPC Server     │
│ • HTTP Monitor    │
│ • Python SDK      │
│ • 独立部署        │
└───────────────────┘
        │
        ▼
  go-agent / Other Agents
```

### Mode 2: Plugin Mode
Lightweight embedded plugin for integration with agent frameworks.

```
┌───────────────────┐
│  Plugin Mode      │
├───────────────────┤
│ • Simple Interface │
│ • Direct Call     │
│ • No Network      │
│ • 嵌入式部署      │
└───────────────────┘
        │
        ▼
  go-agent / Other Agents
```

Both modes share the same core scheduling logic:
```
queue (heap) → scheduler (channel) → gpu_pool (Reservation) → executor (async)
                                  ↓
                               Loop（核心循环）
```

## Features

- **Two-level Scheduling**: Task-level scheduling for workflow tasks and token-level batching handled by inference engines
- **Multi-tenant Fairness**: Token Bucket with weighted fair queuing prevents single users from monopolizing GPU resources
- **Cost-aware Scheduling**: Priority based on recent GPU usage with sliding window decay
- **GPU Packing**: Best Fit strategy with load thresholds to maximize GPU utilization
- **Task Aging**: Prevents starvation of long-running tasks
- **Admission Control**: Protects system from request floods
- **gRPC API**: Language-agnostic interface for Python integration
- **HTTP Monitoring**: Built-in metrics and health check endpoints

## Architecture

```
User Requests
      │
      ▼
Agent Workflow Engine
      │
      ▼
Task-level GPU Scheduler
      │
      ├── Admission Control
      ├── Token Bucket
      ├── Cost-aware Scheduling
      ├── GPU Packing
      └── Task Aging
      │
      ▼
GPU Resource Manager
      │
      ▼
GPU Workers (LLM, Embedding, Diffusion)
```

## Quick Start

### Prerequisites

- Go 1.26+
- Python 3.8+ (for SDK)

### Build

```bash
make build
```

### Run Modes

#### Standalone Mode (Production)

```bash
make run
```

The server will start on `localhost:50051`.

#### Plugin Mode (Development)

```bash
go run ./cmd/plugin/main.go -gpus 4 -memory 8192
```

This runs the scheduler as an embedded plugin.

### Python SDK

```python
from gpu_scheduler import GPUClient

client = GPUClient(host="localhost:50051")

# Submit a task
task_id = client.submit_task(
    task_id="task-1",
    user_id="user-123",
    task_type="llm",
    gpu_memory_mb=8192,
    payload={"prompt": "Hello, world!"}
)

# Check status
status = client.get_status(task_id)
print(f"Status: {status['status']}")

# Wait for completion
result = client.wait(task_id, timeout=60)
```

### Go Agent Integration

```go
package main

import (
    "context"
    "algogpu/internal/plugin"
    "algogpu/internal/scheduler"
    "algogpu/internal/gpu"
    "algogpu/internal/queue"
    "algogpu/pkg/types"
)

func main() {
    // Create GPU pool
    gpuPool := gpu.NewPool()
    gpuPool.AddGPU(0, "NVIDIA-A100", 81920)

    // Create task queue
    taskQueue := queue.NewTaskQueue()

    // Create scheduler
    cfg := &scheduler.Config{
        MaxQueueSize:     100,
        TokenRefillRate:  100,
        TokenBucketSize:  1000,
        DailyTokenLimit:  1000000,
        GPULoadThreshold: 0.85,
        AgingFactor:      0.1,
        UsageWindowMinutes: 5,
    }

    sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
    pluginScheduler := plugin.NewPluginScheduler(sched, taskQueue, gpuPool)

    sched.Start()
    defer sched.Stop()

    // Submit task
    task := &types.Task{
        ID:                 "task-1",
        UserID:             "user-1",
        Type:               api.TaskType_TASK_TYPE_EMBEDDING,
        GPUMemoryRequired:  1024,
        GPUComputeRequired: 100,
        EstimatedRuntimeMs: 1000,
    }

    err := pluginScheduler.SubmitTask(context.Background(), task)
    if err != nil {
        panic(err)
    }
}
```

See `examples/go-agent/main.go` for a complete example.

### Python SDK

```python
from gpu_scheduler import GPUClient

client = GPUClient(host="localhost:50051")

# Submit a task
task_id = client.submit_task(
    task_id="task-1",
    user_id="user-123",
    task_type="llm",
    gpu_memory_mb=8192,
    payload={"prompt": "Hello, world!"}
)

# Check status
status = client.get_status(task_id)
print(f"Status: {status['status']}")

# Wait for completion
result = client.wait(task_id, timeout=60)
```

## API

### gRPC Endpoints

| Method | Description |
|--------|-------------|
| `SubmitTask` | Submit a GPU task |
| `GetTaskStatus` | Get task status |
| `CancelTask` | Cancel a task |
| `GetGPUStatus` | Get GPU status |
| `TaskEvents` | Stream task events |

### HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Health check |
| `/metrics` | System metrics |
| `/gpu/metrics` | GPU metrics |
| `/queue/status` | Queue status |

## Configuration

Default configuration in `internal/scheduler/config.go`:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MaxQueueSize` | 1000 | Maximum queue size |
| `TokenRefillRate` | 100 | Tokens per second |
| `TokenBucketSize` | 1000 | Maximum tokens per user |
| `DailyTokenLimit` | 1000000 | Daily token limit |
| `GPULoadThreshold` | 0.85 | Max GPU load (85%) |
| `AgingFactor` | 0.1 | Task aging factor (γ) |
| `UsageWindowMinutes` | 5 | Usage tracking window |

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run static analysis
make static
```

## Project Structure

```
algogpu/
├── cmd/
│   ├── scheduler/main.go       # Standalone mode entry
│   └── plugin/main.go          # Plugin mode entry
├── internal/
│   ├── executor/               # Task execution (async)
│   ├── gpu/                   # GPU pool & reservation
│   ├── plugin/                # Plugin interface
│   ├── queue/                # Priority queue (heap)
│   ├── scheduler/            # Core scheduler loop
│   └── server/               # gRPC server
├── pkg/
│   └── types/                # Core type definitions
├── api/
│   └── gpu_scheduler.proto   # gRPC definitions
├── python/
│   └── gpu_scheduler/        # Python SDK
├── examples/
│   └── go-agent/main.go      # Plugin integration example
├── docs/                     # Documentation
└── Makefile
```

## Code Size

Target: **3000-5000 lines** (industrial-grade minimalism)

Current implementation:
- Core scheduler: ~600 lines
- GPU management: ~300 lines
- Task queue: ~200 lines
- Executor: ~200 lines
- Plugin interface: ~200 lines
- Tests: ~1000 lines
- **Total: ~3500 lines**

## Technical Summary

### Core Scheduling Loop

```go
func (s *Scheduler) Loop() {
    for task := range s.queueChan {
        // User rate limiting (Token Bucket)
        if !s.token.Allow(task.UserID) {
            s.queue.Push(task)
            continue
        }

        // GPU Packing (Best Fit)
        gpu := s.gpuPool.Allocate(task.GPUMem)
        if gpu == nil {
            s.queue.Push(task)
            continue
        }

        // Async execution
        go s.executor.Run(task, gpu)
    }
}
```

### Scheduling Strategies

1. **Admission Control**: Queue capacity check
2. **Token Bucket**: User-level rate limiting
3. **GPU Packing**: Best Fit strategy
4. **Task Aging**: Priority adjustment over time

### Key Design Principles

- **Simple**: Channel-driven, no busy-wait
- **Deterministic**: Same input = same output
- **Concurrent-safe**: Mutex-protected shared state
- **Async execution**: Non-blocking scheduler loop

### GPU Reservation

```
gpu.Allocate(taskID, memory)  // Allocate resources
gpu.Reserve(taskID)           // Prevent race conditions
... task executes ...
gpuPool.Release(gpu)          // Cleanup
```
