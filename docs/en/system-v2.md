# AI Agent GPU Platform System Design v2

## 1 Overview

**Goal**: Build an AI inference platform supporting complex Agent workflows with:
- High GPU utilization
- Multi-tenant fairness
- Agent workflow scheduling
- Predictable costs
- Self-learning GPU patterns

---

## 2 Architecture (5 Layers)

```
┌────────────────────────────────────────────────────────────────┐
│                     Client Layer                                │
└─────────────────────────────┬──────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                      API Layer                                  │
│                   API Gateway                                   │
└─────────────────────────────┬──────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                   Workflow Layer                                 │
│                                                                │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │ User Profiling  │ │  Leader Agent   │ │ Workflow Store │  │
│  └────────┬────────┘ └────────┬────────┘ └────────┬────────┘  │
│           │                    │                    │            │
│           └────────────────────┼────────────────────┘            │
│                                ▼                               │
│              ┌─────────────────────────────┐                  │
│              │   Workflow Scheduler       │                  │
│              │   Task State Manager      │                  │
│              └─────────────┬─────────────┘                  │
└─────────────────────────────┬─────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                   Resource Layer                                │
│                                                                │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐  │
│  │ Cost Estimator │ │   GPU Scheduler │ │  Engine Router  │  │
│  └────────┬────────┘ └────────┬────────┘ └────────┬────────┘  │
│           │                    │                    │            │
│           └────────────────────┼────────────────────┘            │
│                                ▼                               │
│              ┌─────────────────────────────┐                  │
│              │  Admission Controller      │                  │
│              └─────────────────────────────┘                  │
└─────────────────────────────┬─────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                   Execution Layer                               │
│                                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │LLM Engine│  │Image Engine│ │Embedding  │                │
│  └──────────┘  └──────────┘  └──────────┘                │
└─────────────────────────────┬─────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                Observability Layer                              │
│                                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                │
│  │  Metrics │  │   Logs   │  │ Tracing  │                │
│  └──────────┘  └──────────┘  └──────────┘                │
└────────────────────────────────────────────────────────────────┘
```

---

## 3 Layer Details

### 3.1 API Layer

| Module | Responsibility |
|--------|---------------|
| API Gateway | Auth, rate limiting, routing |

### 3.2 Workflow Layer

| Module | Responsibility | Why |
|--------|---------------|-----|
| User Profiling | User context, history, preferences | Fair scheduling basis |
| Leader Agent | Task planning, DAG decomposition | Agent logic only |
| Workflow Scheduler | DAG execution, dependency management | Parallel task execution |
| Task State Manager | Task lifecycle, retry, recovery | Task failure handling |
| Workflow Store | Persistence, checkpoint, recovery | System restart resilience |

### 3.3 Resource Layer

| Module | Responsibility | Why |
|--------|---------------|-----|
| Cost Estimator | GPU cost prediction | Schedule by predicted cost |
| Admission Controller | System protection | Prevent overload |
| GPU Scheduler | GPU allocation, packing | Core scheduling |
| Engine Router | Model routing | Multi-model support |

### 3.4 Execution Layer

| Module | Responsibility |
|--------|---------------|
| LLM Engine | Text generation |
| Image Engine | Image generation |
| Embedding Engine | Vector embedding |

### 3.5 Observability Layer

| Module | Purpose |
|--------|---------|
| Metrics | GPU util, latency, throughput (Prometheus) |
| Logs | Execution logs, errors (ELK) |
| Tracing | Full workflow path (OpenTelemetry) |

---

## 4 Module Details

### 4.1 User Profiling

```
Input: User ID
Output: UserContext

Context includes:
- Historical tasks
- GPU usage
- User tier
- Preferences

Used for:
priority = weight / (gpu_usage + task_cost)
```

### 4.2 Leader Agent

```
Input: User request
Output: TaskGraph

TaskGraph:
├── LLM Task
├── Tool Task
└── Image Task

Note: Only planning, no resource scheduling
```

### 4.3 Workflow Scheduler

```
Task A → Task B
Task A → Task C

After A completes: B and C execute in parallel
```

### 4.4 Task State Manager

**States**:
```
PENDING → RUNNING → SUCCESS
              ↓
            FAILED → RETRY
              ↓
           CANCELLED
```

**Functions**:
- Track retry count
- Manage timeout
- Handle cancellation

### 4.5 Workflow Store

**Storage**:
- Workflow graph
- Task states
- Dependencies

**Recommendation**: PostgreSQL + Redis

### 4.6 Cost Estimator

**Input**:
```
- task_type
- model
- input_tokens
- historical_data
```

**Output**:
```
estimated_cost = token_cost + memory_cost + latency_cost
```

**Formula**:
```
cost = α × feature_based + β × historical_avg
```

### 4.7 Admission Controller

**Rules**:
```go
if user_quota_exceeded { reject }
if queue_length > threshold { reject }
if estimated_cost > max { reject }
```

### 4.8 GPU Scheduler

**Priority** (with time decay):
```
priority = weight / (user_gpu_usage × decay + task_cost)
```

**GPU Packing**:
```
Best Fit with load threshold (80%)
```

### 4.9 Engine Router

```
GPU Scheduler
      │
      ▼
Engine Router
      │
 ┌────┼─────┐
 ▼    ▼     ▼
LLM Image Embed
```

**Strategies**:
- Model version matching
- Load balancing
- Capability matching

---

## 5 Feedback Loop

```
Task Execution
       │
       ▼
Metrics Collector
       │
       ▼
GPU Usage DB
       │
       ▼
Cost Estimator ←── Self-learning
       │
       ▼
GPU Scheduler
```

---

## 6 Design Principles

| Principle | Implementation |
|-----------|---------------|
| Control/Execution Separation | Leader Agent → planning, Scheduler → resource, Engine → execution |
| Two-level Scheduling | Workflow Scheduler → GPU Scheduler |
| Cost-driven Scheduling | Cost Estimator → priority |
| Observability First | Metrics + Logs + Tracing |

---

## 7 Bottlenecks & Solutions

| Bottleneck | Solution |
|------------|----------|
| Leader Agent planning per request | Planning cache |
| Single scheduler instance | Distributed scheduler (future) |

---

## 8 System Features

- ✓ Agent Workflow support
- ✓ GPU Packing
- ✓ Cost-aware Scheduling
- ✓ Multi-tenant Fairness
- ✓ Self-learning Cost Model
- ✓ Task State Management
- ✓ Workflow Persistence
- ✓ Full Observability
