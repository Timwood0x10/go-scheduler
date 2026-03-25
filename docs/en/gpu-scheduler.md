# AI Agent GPU Scheduling System Design

## 1 Design Goals

This GPU scheduling system is designed for GPU resource management in AI Agent workflow backends, addressing the following issues:

1. GPU being monopolized by a single user
2. Long tasks blocking short tasks
3. Low GPU utilization
4. Unfair resource distribution among multiple users
5. System instability during peak hours

**System Goals**:

- Fairness
- Stability
- High GPU Utilization

---

## 2 System Architecture

The GPU scheduling system sits at the Agent workflow execution layer.

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
      ├── Cost Model
      ├── Priority Scheduler
      ├── GPU Packing
      │
      ▼
GPU Resource Manager
      │
      ▼
GPU Workers
      │
      ├── LLM Service
      ├── Embedding Service
      └── Diffusion Service
```

**Component Responsibilities**:

| Component | Responsibility |
|-----------|----------------|
| Workflow Engine | Task DAG execution |
| GPU Scheduler | GPU task scheduling |
| GPU Resource Manager | GPU state management & allocation |
| GPU Workers | Model execution |

> **Note**: LLM internal token scheduling is handled by inference frameworks (vLLM, TensorRT-LLM, etc.) and is outside the scope of this scheduling system.

---

## 3 Task-level Scheduling

The Task-level Scheduler is responsible for GPU task scheduling.

The scheduling strategy consists of seven components:

- **Admission Control** — System protection
- **Token Bucket** — User quotas
- **Cost Model** — GPU cost prediction
- **Priority Scheduler** — Cost-aware scheduling
- **GPU Packing** — Resource bin packing
- **Fragmentation Management** — Memory defragmentation
- **Task Aging** — Starvation prevention

---

### 3.1 Admission Control

Prevents request floods from overwhelming the system.

**Strategy**:

```
max_queue_size = N
```

When `queue_size > N`, the system `reject request`.

Can be extended to a two-layer mechanism: `soft_limit / hard_limit`.

**Purpose**: Prevents GPU scheduler overload

---

### 3.2 Token Bucket

Each user has a GPU usage quota.

**Token Growth**:

```
token += refill_rate * time
```

**Task Consumption**:

```
token_cost = task_gpu_cost
```

When `token < cost`, the system rejects the request or pauses the workflow.

**Purpose**: Prevents single user from exhausting GPU resources

---

### 3.3 GPU Cost Model

The Cost Model predicts task GPU consumption to provide scheduling decisions.

GPU Scheduler depends on **predicted resource cost**, not just task type.

#### 3.3.1 Cost Dimensions

Task GPU cost consists of three parts:

```
task_cost = f(memory_cost, compute_cost, runtime)
```

| Cost Type | Description |
|-----------|-------------|
| memory_cost | GPU memory usage |
| compute_cost | GPU compute resources |
| runtime | Estimated execution time |

#### 3.3.2 Cost Estimation Input

```go
task = {
    model_type,
    input_size,
    output_size,
    parameters
}
```

**Example**: LLM task
```
model = llama70b
input_tokens = 1000
output_tokens = 200
```

#### 3.3.3 Cost Estimation Method

The system uses **Bucket Average** method.

Historical tasks are bucketed by input size:

| Token Bucket | Range |
|-------------|-------|
| small | 0-256 |
| medium | 256-1024 |
| large | 1024-4096 |

Each bucket records:
- avg_runtime
- avg_memory
- avg_gpu_util

**Prediction**:
```
cost(task) = bucket_avg
```

#### 3.3.4 Cost Model Update

After task execution, GPU metrics are recorded:

```go
task_metrics = {
    task_type,
    input_size,
    gpu_runtime,
    peak_memory,
}
```

The Cost Table is periodically updated:

```
cost_table[task_type][bucket]
```

**Purpose**: Improve prediction accuracy, prevent GPU packing failures

---

### 3.4 Priority Scheduler (Cost-aware)

Different tasks have different GPU costs.

| Task Type | GPU Cost |
|-----------|----------|
| embedding | 1 |
| LLM | 5 |
| diffusion | 10 |

**Scheduling Priority**:

```
priority = weight / (recent_gpu_usage + task_cost)
```

Where `recent_gpu_usage = GPU usage in the last 5 minutes`.

**Effect**:

- More recent GPU usage → lower priority
- No usage for a long time → priority recovers

**Purpose**: Achieve long-term fairness

---

### 3.5 GPU Packing

GPU should not run only one task.

**Scheduling Goal**: Maximize GPU utilization

**Resource Model**:

```
GPU Resource = (memory, compute)
```

**GPU Load Threshold**:

```
max_gpu_load = 85%
```

**Scheduling Logic**:

```
candidate = []
for gpu in gpus:
    if gpu.can_fit(task) and gpu.load < max_load:
        candidate.append(gpu)

gpu = best_fit(candidate)
```

**GPU Score**:

```
gpu_score = 0.7 * memory_util + 0.3 * compute_util
```

**Purpose**: Maximize GPU utilization while preventing overload

---

### 3.6 GPU Fragmentation Management

GPU Packing creates **memory fragmentation** problems.

#### 3.6.1 Problem Example

```
GPU memory = 40GB

TaskA = 12GB
TaskB = 10GB
TaskC = 8GB

Remaining: 10GB

New task requires: 15GB
→ Scheduling fails
```

#### 3.6.2 Fragmentation Detection

GPU Manager periodically calculates:

```
fragmentation = free_memory / total_memory
```

If `fragmentation > threshold`, GPU has serious fragmentation.

#### 3.6.3 Mitigation Strategies

**1. Memory Reservation**

Reserve memory for large tasks:

```
reserved_memory = 20% of total
```

Prevents GPU from being filled with small tasks.

**2. GPU Class Scheduling**

Classify tasks by memory requirement:

| Class | Memory |
|-------|--------|
| small | < 4GB |
| medium | 4-16GB |
| large | > 16GB |

Schedule different classes to different GPUs.

**3. Spillover Scheduling**

When current GPU cannot fit task:

```
scheduler → try next GPU node
```

Prevents task from waiting too long.

---

### 3.7 Task Aging

Prevents long tasks from being continuously overtaken by short tasks.

**Priority Adjustment**:

```
priority = base_priority + wait_time * γ
```

**Effect**: The longer the wait time, the higher the priority, ensuring tasks eventually get executed.

---

## 4 GPU State Manager

GPU Resource Manager maintains GPU state.

### 4.1 GPU State Table

```go
type GPUState struct {
    GPUID         int
    TotalMemory   int64
    UsedMemory    int64
    ComputeUtil   int
    RunningTasks  []string
    LastHeartbeat time.Time
}
```

### 4.2 State Updates

GPU Workers periodically report status:

```
heartbeat_interval = 1s
```

Update contents:
- memory_usage
- gpu_utilization
- running_tasks

GPU Manager updates the State Table.

### 4.3 Scheduler Interface

Scheduler only reads the State Table:

```
Scheduler reads → GPUStateTable → makes decisions
```

---

## 5 Enhanced Scheduler Execution Flow

```
Task Queue
    │
    ▼
Admission Control
    │
    ▼
Token Bucket Check
    │
    ▼
Cost Estimation (Cost Model)
    │
    ▼
Priority Scheduling
    │
    ▼
GPU Packing + Fragmentation Management
    │
    ▼
Dispatch to GPU Worker
```

---

## 6 Metrics Feedback Loop

The GPU scheduling system must have a feedback mechanism.

### 6.1 System Structure

```
GPU Worker
     │
     ▼
Metrics Collector
     │
     ▼
Metrics Database
     │
     ▼
Cost Model Update
     │
     ▼
Scheduler
```

### 6.2 Metrics Recorded

| Metric | Description |
|--------|-------------|
| gpu_runtime | Actual GPU execution time |
| gpu_utilization | GPU utilization percentage |
| memory_peak | Peak memory usage |
| task_latency | End-to-end latency |
| queue_time | Time in queue |

### 6.3 Scheduler Optimization

Scheduler periodically calculates:

- GPU utilization
- queue latency
- task wait time

If `GPU utilization < target`:

→ packing strategy needs adjustment

---

## 7 Scheduler Data Structures

### 7.1 Core Structures

```go
type Scheduler struct {
    TaskQueue
    UserTokenTable
    GPUStateTable
    CostModel
}
```

### 7.2 Task Structure

```go
type Task struct {
    TaskID       string
    UserID       string
    GPUMemory    int64
    GPUCompute   int64
    EstRuntime   int64
}
```

---

## 8 Scheduling Complexity

**Scheduling Loop**:

```
O(N log N)
```

Where: N = task_queue_size

**GPU Packing**:

```
O(G)
```

Where: G = GPU count

Overall scheduling overhead is very low.

---

## 9 System Stability Design

| Problem | Solution |
|---------|----------|
| GPU congestion | Admission Control |
| User monopoly | Token Bucket |
| Task starvation | Task Aging |
| GPU fragmentation | GPU Packing + Fragmentation Management |
| Cost inaccuracy | Cost Model + Feedback Loop |

---

## 10 Final Scheduler Architecture

```
User Request
     │
     ▼
Agent Workflow Engine
     │
     ▼
Task Queue
     │
     ▼
GPU Scheduler
     │
     ├── Admission Control
     ├── Token Bucket
     ├── Cost Model
     ├── Priority Scheduler
     ├── GPU Packing
     └── Fragmentation Management
     │
     ▼
GPU Resource Manager
     │
     ▼
GPU Workers
```

---

## 11 Design Summary

**Core Principles**:

```
Fairness
+
Cost Prediction
+
Resource Packing
+
Fragmentation Control
+
Feedback Loop
```

**Complete Scheduling Strategy**:

```
Admission Control
+
Token Bucket
+
Cost Model
+
Cost-aware Scheduling
+
GPU Packing
+
Fragmentation Management
+
Task Aging
+
Metrics Feedback
```

**System Goals**:

- High GPU utilization
- Multi-user fairness
- Stable inference service

This design is suitable for AI Agent workflow GPU scheduling, focusing on engineering practicality and system stability rather than theoretical optimal scheduling.