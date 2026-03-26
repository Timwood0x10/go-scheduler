# AlgoGPU

A minimal GPU scheduler for AI agents and inference workloads. Simple. Deterministic. Reliable.

## Features

- **Dual Mode Architecture**: Run as standalone service or embedded plugin
- **Data-Driven Scheduling**: Learn from historical GPU execution data
- **Resource Prediction**: Estimate task duration and memory requirements
- **Policy Engine**: Dynamic priority adjustment based on real-time metrics
- **Simple Scheduling Loop**: Channel-based, deterministic scheduling
- **GPU Packing**: Best Fit strategy with load thresholds
- **Token Bucket**: User-level rate limiting with daily quotas
- **Task Aging**: Prevents starvation with priority adjustment
- **Priority Queue**: Heap-based efficient task ordering
- **gRPC API**: Language-agnostic interface for Python integration
- **Plugin Interface**: Easy embedding in agent frameworks

## Architecture

### System Overview

AlgoGPU supports two deployment modes with shared core logic:

```
┌─────────────────────────────────────────────────────────┐
│                    AlgoGPU Core                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Simple Scheduler Loop (项目灵魂)                │  │
│  │  for {                                            │  │
│  │    task := queue.Pop()                            │  │
│  │                                                   │  │
│  │    if !token.Allow(user) { continue }            │  │
│  │                                                   │  │
│  │    gpu := gpuPool.Allocate(task.GPUMem)          │  │
│  │    if gpu == nil { continue }                    │  │
│  │                                                   │  │
│  │    executor.Run(task, gpu)                       │  │
│  │  }                                                │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
│  Core Modules (共享)                                     │
│  • scheduler/   - 调度逻辑                              │
│  • gpu/         - GPU 管理                              │
│  • queue/       - 任务队列                              │
│  • executor/    - 任务执行                              │
│  • db/          - 数据存储                              │
│  • predictor/   - 资源预测                              │
│  • policy/      - 策略引擎                              │
└─────────────────────────────────────────────────────────┘
                            │
                            │ 简单包装
                            ▼
        ┌───────────────────┴───────────────────┐
        │                                       │
        ▼                                       ▼
┌───────────────────┐               ┌───────────────────┐
│  Standalone Mode  │               │   Plugin Mode     │
│  独立服务模式      │               │   插件模式        │
├───────────────────┤               ├───────────────────┤
│ • gRPC Server     │               │ • Simple Interface │
│ • HTTP Monitor    │               │ • Direct Call     │
│ • Python SDK      │               │ • No Network      │
│ • 独立部署        │               │ • 嵌入式部署      │
└───────────────────┘               └───────────────────┘
```

### Data-Driven Scheduling Loop

```
Task Submission
      │
      ▼
Policy Engine Evaluation (基于历史数据)
      │
      ├── Resource Prediction (运行时间 + 显存)
      ├── Priority Calculation (动态优先级)
      └── GPU Packing Decision (是否适合打包)
      │
      ▼
Queue (Priority + Aging)
      │
      ▼
Scheduler Loop
      │
      ├── Token Bucket Check (用户限速)
      ├── GPU Allocation (Best Fit)
      └── Async Execution
      │
      ▼
Metrics Recording (写入 SQLite)
      │
      ▼
SQLite Database (task_execution 表)
      │
      ▼
Stats Query (资源预测数据源)
      │
      ▼
Better Future Decisions (更优调度)
```

### Module Breakdown

```
algogpu/
├── cmd/
│   ├── scheduler/main.go       # Standalone mode entry
│   └── plugin/main.go          # Plugin mode entry
├── internal/
│   ├── db/                     # SQLite database layer
│   │   └── store.go            # Metrics storage & queries
│   ├── executor/               # Task execution (async)
│   │   └── runner.go           # Task runner with metrics
│   ├── gpu/                   # GPU pool & reservation
│   │   ├── pool.go             # GPU allocation + reservation
│   │   ├── collector.go        # GPU metrics collection
│   │   └── state.go            # GPU state
│   ├── plugin/                # Plugin interface
│   │   └── scheduler.go        # Simple plugin API
│   ├── policy/                # Policy engine (600 lines)
│   │   ├── engine.go           # Task evaluation & decisions
│   │   ├── rules.go            # Scheduling rules
│   │   └── stats.go            # Statistics queries
│   ├── predictor/             # Resource predictor (250 lines)
│   │   ├── predictor.go        # Resource prediction logic
│   │   └── cache.go            # Stats cache (5s TTL)
│   ├── queue/                 # Priority queue (200 lines)
│   │   └── task_queue.go       # Heap-based queue
│   ├── scheduler/             # Core scheduler (600 lines)
│   │   ├── scheduler.go        # Channel-based loop
│   │   ├── config.go           # Configuration
│   │   ├── token_bucket.go     # Rate limiting
│   │   └── gpu_packing.go      # Best fit strategy
│   └── server/                # gRPC server
│       └── grpc.go             # gRPC service implementation
├── pkg/
│   └── types/                 # Core type definitions
│       └── types.go            # Task, metrics, predictions
├── api/
│   └── gpu_scheduler.proto    # gRPC definitions
├── python/
│   └── gpu_scheduler/         # Python SDK
├── examples/
│   └── go-agent/main.go       # Plugin integration example
├── docs/                      # Documentation
│   ├── en/                    # English docs
│   └── zh/                    # Chinese docs
└── Makefile
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

The server will start on `localhost:50051` with:
- gRPC API on port 50051
- HTTP monitoring on port 8080
- SQLite database: `./algogpu.db`

#### Plugin Mode (Development)

```bash
go run ./cmd/plugin/main.go
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
    "algogpu/pkg/types"
)

func main() {
    // Create plugin scheduler
    sched := plugin.NewPluginScheduler(...)
    
    // Submit task
    task := &types.Task{
        ID:                 "task-1",
        UserID:             "user-1",
        Type:               api.TaskType_TASK_TYPE_EMBEDDING,
        GPUMemoryRequired:  1024,
    }
    
    err := sched.SubmitTask(context.Background(), task)
    if err != nil {
        panic(err)
    }
}
```

See `examples/go-agent/main.go` for a complete example.

## Database

### SQLite Schema

```sql
CREATE TABLE task_execution (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    user_id TEXT,
    gpu_id INTEGER,
    gpu_model TEXT,
    priority INTEGER,
    queue_wait_ms INTEGER,
    execution_time_ms INTEGER,
    avg_gpu_util REAL,
    max_gpu_util REAL,
    avg_mem_util REAL,
    max_mem_util REAL,
    gpu_memory_used_mb INTEGER,
    success INTEGER,
    created_at INTEGER
);

CREATE INDEX idx_task_type ON task_execution(task_type);
CREATE INDEX idx_user_id ON task_execution(user_id);
CREATE INDEX idx_gpu_id ON task_execution(gpu_id);
CREATE INDEX idx_created_at ON task_execution(created_at);
```

### Queries

```go
// Get task type statistics
stats, err := store.GetTaskTypeStats(ctx, "llm")

// Get user statistics
stats, err := store.GetUserStats(ctx, "user-123")

// Get GPU statistics
stats, err := store.GetGPUStats(ctx, 0)

// Get queue wait statistics
waitStats, err := store.GetQueueWaitStats(ctx, "llm")
```

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

# Run with race detector
make test-race

# Run static checks
make static-check
```

## Code Size

Target: **3000-5000 lines** (industrial-grade minimalism)

Current implementation:
- Core scheduler: ~600 lines
- GPU management: ~300 lines
- Task queue: ~200 lines
- Executor: ~200 lines
- Database layer: ~400 lines
- Predictor: ~250 lines
- Policy engine: ~600 lines
- Plugin interface: ~200 lines
- Tests: ~1000 lines
- **Total: ~3750 lines** ✅

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

### Data-Driven Decision

```go
// Policy engine evaluates task
decision, err := policyEngine.EvaluateTask(ctx, task, queueSize)

// Predictor estimates resources
prediction := predictor.Predict(ctx, taskType)

// Adjust priority based on historical data
task.Priority = decision.Priority
task.EstimatedRuntimeMs = decision.EstimatedDuration
```

### Key Design Principles

- **Simple**: Channel-driven, no busy-wait
- **Deterministic**: Same input = same output
- **Data-Driven**: Learn from historical executions
- **Concurrent-safe**: Mutex-protected shared state
- **Async execution**: Non-blocking scheduler loop

### GPU Reservation

```
gpu.Allocate(taskID, memory)  // Allocate resources
gpu.Reserve(taskID)           // Prevent race conditions
... task executes ...
gpuPool.Release(gpu)          // Cleanup + record metrics
```

## License

MIT License