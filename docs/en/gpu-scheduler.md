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
| GPU Manager | GPU resource allocation |
| GPU Workers | Model execution |

> **Note**: LLM internal token scheduling is handled by inference frameworks (vLLM, TensorRT-LLM, etc.) and is outside the scope of this scheduling system.

---

## 3 Task-level Scheduling

The Task-level Scheduler is responsible for GPU task scheduling.

The scheduling strategy consists of five components:

- **Admission Control** — Admission control
- **Token Bucket** — User quotas
- **Cost-aware Scheduling** — Cost-aware scheduling
- **GPU Packing** — GPU bin packing
- **Task Aging** — Task aging

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

### 3.3 Cost-aware Scheduling

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

### 3.4 GPU Packing

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

**Purpose**: Reduce GPU fragmentation and prevent single GPU overload

---

### 3.5 Task Aging

Prevents long tasks from being continuously overtaken by short tasks.

**Priority Adjustment**:

```
priority = base_priority + wait_time * γ
```

**Effect**: The longer the wait time, the higher the priority, ensuring tasks eventually get executed.

---

## 4 GPU Scheduler Execution Flow

The GPU Scheduler runs a periodic scheduling loop.

**Pseudocode**:

```
while scheduler_running:

    tasks = task_queue.pop_all()

    for task in tasks:
        priority = calculate_priority(task)

    sort(tasks by priority)

    for task in tasks:
        gpu = find_best_gpu(task)
        if gpu available:
            dispatch(task, gpu)
        else:
            requeue(task)
```

---

## 5 GPU Task Types

The GPU Scheduler does not care about model details; it only manages task resources.

**Task Examples**:

- LLM inference
- embedding generation
- diffusion image generation
- GPU data processing

**Task Definition**:

```
task = {
    gpu_memory,
    gpu_compute,
    estimated_runtime
}
```

Scheduler schedules based on resource requirements.

---

## 6 System Effects

| Metric | Improvement |
|--------|-------------|
| GPU Utilization | ↑ |
| Average Latency | ↓ |
| System Stability | ↑ |
| Multi-user Fairness | ↑ |

The system can evolve from a "simple AI service" to a "multi-tenant AI workflow computing platform".

---

## 7 Design Summary

**Core Philosophy**: Fairness + Cost-awareness + Resource Packing

**Scheduling Strategy**:

```
Admission Control
+
Token Bucket
+
Cost-aware Scheduling
+
GPU Packing
+
Task Aging
```

This design is suitable for AI Agent workflow GPU resource scheduling, improving GPU utilization and system throughput while ensuring fairness and stability. The goal of this scheduling system is "engineering-practical scheduling," not optimal scheduling algorithms.
