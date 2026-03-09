# GPU Scheduler

A high-performance GPU scheduling system designed for AI Agent workflows, built with Go. Provides multi-tenant GPU resource management with fairness, stability, and high utilization.

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

### Run

```bash
make run
```

The server will start on `localhost:50051`.

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
│   └── scheduler/main.go       # Entry point
├── internal/
│   ├── executor/               # Task execution
│   ├── gpu/                   # GPU pool & metrics
│   ├── monitor/               # HTTP monitoring
│   ├── queue/                # Task queue
│   ├── scheduler/            # Scheduling strategies
│   ├── server/               # gRPC server
│   └── state/                # Task state machine
├── api/
│   └── gpu_scheduler.proto   # gRPC definitions
├── python/
│   └── gpu_scheduler/        # Python SDK
├── docs/                     # Documentation
└── Makefile
```

## Technical Summary

### Scheduling Algorithm

1. **Admission Control**: Checks if queue has capacity before accepting
2. **Token Bucket**: Rate limiting per user with daily quotas
3. **Cost-aware Priority**: `priority = weight / (recent_usage + task_cost)`
4. **GPU Packing**: Best Fit with load threshold (85%)
5. **Task Aging**: `priority += wait_time * γ` to prevent starvation

### Event-driven Design

The scheduler responds to events rather than polling:
- New task arrival
- Task completion
- GPU freed
- Periodic tick (100ms)

### GPU Metrics

Collected via nvidia-smi (when available) or simulated for testing:
- Memory usage
- Compute utilization
- Temperature
