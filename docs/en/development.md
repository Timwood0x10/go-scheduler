# GPU Scheduler Project Development Document

## 1 Project Overview

### 1.1 Project Positioning

An independent high-performance GPU scheduling system that plugs into Python-based AI Agent workflows for unified GPU resource management.

### 1.2 System Goals

- **Fairness**: Fair resource distribution among users, preventing GPU monopolization
- **Stability**: Preventing request floods from overwhelming the system, stable operation during peak hours
- **High Utilization**: Maximizing resource utilization through GPU bin packing strategies
- **Low Latency**: Controllable task queuing time, short tasks get priority

### 1.3 Problems Solved

1. GPU being monopolized by a single user
2. Long tasks blocking short tasks
3. Low GPU utilization
4. Unfair resource distribution among multiple users
5. System instability during peak hours

---

## 2 Technology Stack

### 2.1 Core Language

- **Go 1.26+**: High-performance scheduling service

### 2.2 Communication

- **gRPC**: Go ↔ Python cross-language communication
  - Efficient binary serialization
  - Support for streaming calls
  - Multi-language client support

### 2.3 Storage

- **Memory**: Task queue, runtime state
- **Optional Persistence**: Task state checkpoint (future phase)

---

## 3 System Architecture

### 3.1 Overall Architecture

```
┌─────────────────┐       gRPC        ┌─────────────────┐
│  Python Agent  │ ───────────────▶  │   Go Scheduler  │
│  (Workflow)    │   SubmitTask()    │                 │
│                 │   GetStatus()     │                 │
│                 │   CancelTask()    │                 │
└─────────────────┘                   └────────┬────────┘
                                               │
                    GPU Resource               ▼
                    Management        ┌─────────────────┐
                                     │   GPU Manager   │
                                     └────────┬────────┘
                                              │
                    GPU Pool                 ▼
            ┌─────────────┬─────────────┐ ┌──────────┐
            │    GPU 0    │    GPU 1    │ │  GPU N   │
            └─────────────┴─────────────┘ └──────────┘
```

### 3.2 Component Responsibilities

| Component | Responsibility |
|-----------|----------------|
| Python Agent | Workflow execution, submit GPU tasks via SDK |
| gRPC API | Task submission, status query, cancellation |
| Task Queue | Task queuing, priority sorting |
| Scheduler | Scheduling strategy execution |
| GPU Manager | GPU resource tracking, best GPU selection |
| GPU Workers | Actual task execution (calls external services) |

---

## 4 Scheduling Strategy

### 4.1 Five-layer Scheduling Strategy

| Strategy | Purpose |
|----------|---------|
| Admission Control | Prevent system overload, limit queue length |
| Token Bucket | User quota management, prevent single user monopolization |
| Cost-aware Scheduling | Cost-aware scheduling with sliding window for long-term fairness |
| GPU Packing | GPU bin packing with Best Fit + load threshold |
| Task Aging | Prevent long tasks from starving |

### 4.2 Priority Calculation

```
priority = (weight / (recent_gpu_usage + task_cost)) + (wait_time * γ)
```

---

## 5 Task State Machine

### 5.1 State Definitions

| State | Meaning |
|-------|---------|
| PENDING | Task submitted, waiting for scheduling |
| RUNNING | Task is executing |
| COMPLETED | Task completed successfully |
| FAILED | Task execution failed |
| CANCELLED | Task was cancelled |
| REJECTED | Task rejected (quota/queue limit exceeded) |

### 5.2 State Transitions

```
        submit
          │
          ▼
      PENDING
          │
   scheduler dispatch
          ▼
      RUNNING
      /     \
     ▼       ▼
 COMPLETED  FAILED
     │
 CANCELLED (can be triggered at any time)
```

### 5.3 State Transition Rules

| Current State | Trigger Event | Target State | Description |
|---------------|---------------|--------------|-------------|
| PENDING | scheduler dispatch | RUNNING | Task dispatched for execution |
| RUNNING | task completed | COMPLETED | Task completed successfully |
| RUNNING | task error | FAILED | Task execution failed |
| PENDING | user cancel | CANCELLED | User cancelled task |
| RUNNING | user cancel | CANCELLED | User cancelled task |
| PENDING | admission reject | REJECTED | Quota or queue limit exceeded |
| Any | system shutdown | CANCELLED | System shutdown |

---

## 6 GPU Status Collection

### 6.1 GPU Manager Architecture

```
GPU Manager
    │
    ├── GPU Pool
    │      ├── GPU 0
    │      ├── GPU 1
    │      └── GPU N
    │
    └── Metrics Collector
           │
           ├── memory_used
           ├── compute_util
           ├── temperature
           └── running_tasks
```

### 6.2 Collection Methods

| Method | Description |
|--------|-------------|
| nvidia-smi | Command-line tool, suitable for simple scenarios |
| NVML (NVIDIA Management Library) | C API, Go can call via cgo |
| DCGM (Data Center GPU Manager) | Enterprise-level monitoring, most feature-rich |

### 6.3 GPU Load Calculation

```
gpu_score = 0.7 * memory_util + 0.3 * compute_util
```

Where:

- `memory_util = memory_used / memory_total`
- `compute_util = compute_used / compute_total`

---

## 7 Scheduling Trigger Mechanism

### 7.1 Event-driven vs Polling

The current design uses **Event-driven** mode, where the scheduler responds to events instead of polling.

### 7.2 Trigger Events

| Event | Source | Trigger Action |
|-------|--------|----------------|
| new task arrival | gRPC SubmitTask | Trigger scheduling |
| task completion | Worker callback | Trigger scheduling |
| task failure | Worker callback | Trigger scheduling |
| gpu freed | Metrics Collector | Trigger scheduling |
| periodic tick | Timer (100ms) | Prevent task omission |

### 7.3 Scheduling Execution

```
on_event(event):
    run_scheduler()

run_scheduler():
    1. Get pending tasks from queue
    2. Calculate task priorities
    3. Sort tasks
    4. Find best GPU for each task
    5. Allocate GPU and update status
    6. Requeue failed tasks
```

---

## 8 gRPC Interface Design

### 8.1 Service Definition

```protobuf
service GPUScheduler {
    // Submit task
    rpc SubmitTask(TaskRequest) returns (TaskResponse);

    // Get task status
    rpc GetTaskStatus(TaskStatusRequest) returns (TaskStatusResponse);

    // Cancel task
    rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);

    // Get GPU status
    rpc GetGPUStatus(Empty) returns (GPUStatusResponse);

    // Task result stream (optional)
    rpc TaskEvents(Empty) returns (stream TaskEvent);
}
```

### 8.2 Core Data Types

```protobuf
enum TaskType {
    TASK_TYPE_UNSPECIFIED = 0;
    TASK_TYPE_EMBEDDING = 1;
    TASK_TYPE_LLM = 2;
    TASK_TYPE_DIFFUSION = 3;
    TASK_TYPE_OTHER = 4;
}

enum TaskStatus {
    TASK_STATUS_PENDING = 0;
    TASK_STATUS_RUNNING = 1;
    TASK_STATUS_COMPLETED = 2;
    TASK_STATUS_FAILED = 3;
    TASK_STATUS_CANCELLED = 4;
    TASK_STATUS_REJECTED = 5;
}

message TaskRequest {
    string task_id = 1;
    string user_id = 2;
    TaskType task_type = 3;
    int64 gpu_memory_required = 4;  // MB
    int64 gpu_compute_required = 5;  // TFLOPS estimate
    int64 estimated_runtime_ms = 6;
    bytes payload = 7;  // Task-specific data
}

message TaskResponse {
    bool accepted = 1;
    string message = 2;
    TaskStatus status = 3;
}
```

---

## 9 Python SDK Design

### 9.1 Usage

```python
from gpu_scheduler import GPUClient

client = GPUClient(host="localhost:50051")

# Submit task
task_id = client.submit_task(
    user_id="user_123",
    task_type="llm",
    gpu_memory_mb=8192,
    payload={"prompt": "..."}
)

# Get status
status = client.get_status(task_id)
print(f"Task status: {status}")

# Block and wait for result
result = client.wait(task_id)
print(f"Result: {result}")
```

### 9.2 SDK Core Interface

```python
class GPUClient:
    def submit_task(self, user_id, task_type, gpu_memory_mb, payload) -> str
    def get_status(self, task_id) -> TaskStatus
    def cancel_task(self, task_id) -> bool
    def get_gpu_status(self) -> List[GPUInfo]
    def wait(self, task_id, timeout=None) -> TaskResult
```

---

## 10 Project Phase Plan

### 10.1 Phase 1: Basic Framework Setup

**Goal**: Build a runnable minimum scheduling system

**Tasks**:

1. Project structure initialization
2. gRPC service definition and generation
3. Basic task queue implementation
4. Simple task submission/query interface
5. Python SDK basic wrapper

**Milestone**: Python Agent can submit tasks and get task IDs

### 10.2 Phase 2: Scheduling Strategy Implementation

**Goal**: Implement core scheduling strategies

**Tasks**:

1. Admission Control implementation
2. Token Bucket user quota implementation
3. Cost-aware Scheduling implementation (with sliding window)
4. Basic GPU Manager implementation
5. GPU Packing strategy implementation
6. Task Aging mechanism implementation

**Milestone**: Multiple task submissions can be scheduled according to strategy

### 10.3 Phase 3: Task Execution and Integration

**Goal**: Tasks can actually execute

**Tasks**:

1. Task executor abstraction
2. GPU status collection (nvidia-smi/NVML)
3. Task lifecycle management
4. Event-driven scheduling trigger mechanism
5. Result callback mechanism
6. Error handling and retry

**Milestone**: Tasks can complete execution and return results

### 10.4 Phase 4: Monitoring and Operations

**Goal**: Production-ready

**Tasks**:

1. Logging and metrics exposure (Prometheus)
2. Health check interface
3. Configuration hot reload
4. Graceful shutdown
5. Performance stress testing

**Milestone**: Deployable to production environment

---

## 11 Directory Structure

```
algogpu/
├── cmd/
│   └── scheduler/
│       └── main.go           # Entry point
├── internal/
│   ├── scheduler/
│   │   ├── scheduler.go      # Scheduler core (Event-driven)
│   │   ├── admission.go      # Admission control
│   │   ├── token_bucket.go   # Token Bucket
│   │   ├── cost_aware.go     # Cost-aware scheduling
│   │   ├── gpu_packing.go    # GPU packing
│   │   └── aging.go          # Task aging
│   ├── gpu/
│   │   ├── manager.go        # GPU management
│   │   ├── pool.go           # GPU pool
│   │   └── collector.go     # GPU status collection
│   ├── queue/
│   │   └── task_queue.go    # Task queue
│   ├── state/
│   │   └── task_state.go     # Task state machine
│   └── server/
│       └── grpc.go           # gRPC service
├── pkg/
│   └── types/
│       └── types.go          # Common types
├── api/
│   └── gpu_scheduler.proto   # gRPC definitions
├── python/
│   └── gpu_scheduler/        # Python SDK
│       ├── __init__.py
│       └── client.py
├── docs/
│   ├── gpu-scheduler.md      # Design document
│   └── development.md        # This document
└── go.mod
```

---

## 12 Acceptance Criteria

### 12.1 Functional Acceptance

- [ ] Python Agent can submit tasks via SDK
- [ ] Tasks are scheduled in priority order
- [ ] Tasks exceeding quota are rejected
- [ ] Tasks are rejected when queue limit exceeded
- [ ] GPU load is balanced
- [ ] Tasks can complete execution and return results
- [ ] Task state machine transitions correctly

### 12.2 Performance Acceptance

- [ ] Single task submission latency < 10ms
- [ ] Scheduling cycle < 100ms
- [ ] Support 100+ concurrent tasks

### 12.3 Stability Acceptance

- [ ] No memory leaks
- [ ] Graceful handling of service restart
- [ ] Clear and traceable error messages

---

## 13 Future Extensions

- [ ] Multi-node GPU scheduling
- [ ] Dynamic task priority adjustment
- [ ] Configurable scheduling strategies
- [ ] Task checkpoint and recovery
- [ ] Scheduling visualization Dashboard
