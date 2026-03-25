# AI Agent GPU 调度系统设计

## 1 设计目标

该 GPU 调度系统用于 AI Agent 工作流后台 GPU 资源管理，解决以下问题：

1. GPU 被单用户占满
2. 长任务阻塞短任务
3. GPU 利用率低
4. 多用户资源不公平
5. 系统高峰期不稳定

**系统目标**：

- 公平性
- 稳定性
- 高 GPU 利用率

---

## 2 系统架构

GPU 调度系统位于 Agent 工作流执行层。

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

**组件职责**：

| 组件 | 职责 |
|-----|------|
| Workflow Engine | 任务 DAG 执行 |
| GPU Scheduler | GPU 任务调度 |
| GPU Resource Manager | GPU 状态管理与资源分配 |
| GPU Workers | 模型执行 |

> **注意**：LLM 内部 token 调度由推理框架（vLLM、TensorRT-LLM 等）负责，不在调度系统范围内。

---

## 3 Task-level Scheduling

Task-level Scheduler 负责 GPU 任务调度。

调度策略由七部分组成：

- **Admission Control** — 系统保护
- **Token Bucket** — 用户配额
- **Cost Model** — GPU 成本预测
- **Priority Scheduler** — 成本感知调度
- **GPU Packing** — 资源装箱
- **Fragmentation Management** — 碎片管理
- **Task Aging** — 防止饿死

---

### 3.1 Admission Control（准入控制）

防止请求洪峰压垮系统。

**策略**：

```
max_queue_size = N
```

当 `queue_size > N` 时，系统 `reject request`。

可扩展为 `soft_limit / hard_limit` 两层机制。

**作用**：防止 GPU scheduler 过载

---

### 3.2 Token Bucket（用户配额）

每个用户拥有 GPU 使用额度。

**Token 增长**：

```
token += refill_rate * time
```

**任务消耗**：

```
token_cost = task_gpu_cost
```

当 `token < cost` 时，系统拒绝请求或暂停 workflow。

**作用**：防止单用户耗尽 GPU

---

### 3.3 GPU Cost Model（GPU 成本模型）

Cost Model 用于预测任务 GPU 消耗，为调度器提供调度依据。

GPU Scheduler 不直接依赖任务类型，而是依赖 **预测资源成本**。

#### 3.3.1 成本维度

任务 GPU 成本由三部分组成：

```
task_cost = f(memory_cost, compute_cost, runtime)
```

| 成本类型 | 说明 |
|---------|------|
| memory_cost | GPU 显存占用 |
| compute_cost | GPU 计算资源 |
| runtime | 预计执行时间 |

#### 3.3.2 成本估计输入

```go
task = {
    model_type,
    input_size,
    output_size,
    parameters
}
```

**例如**：LLM task
```
model = llama70b
input_tokens = 1000
output_tokens = 200
```

#### 3.3.3 成本估计方法

系统使用 **Bucket Average** 方法。

历史任务按输入规模分桶：

| Token Bucket | 范围 |
|-------------|------|
| small | 0-256 |
| medium | 256-1024 |
| large | 1024-4096 |

每个 bucket 记录：

- avg_runtime
- avg_memory
- avg_gpu_util

**预测成本**：

```
cost(task) = bucket_avg
```

#### 3.3.4 成本模型更新

任务执行结束后记录 GPU metrics：

```go
task_metrics = {
    task_type,
    input_size,
    gpu_runtime,
    peak_memory,
}
```

定期更新 Cost Table：

```
cost_table[task_type][bucket]
```

**作用**：
- 提高调度预测准确度
- 避免 GPU packing 失效

---

### 3.4 Priority Scheduler（成本感知调度）

不同任务 GPU 成本不同。

| 任务类型 | GPU cost |
|---------|---------|
| embedding | 1 |
| LLM | 5 |
| diffusion | 10 |

**调度优先级**：

```
priority = weight / (recent_gpu_usage + task_cost)
```

其中 `recent_gpu_usage = 最近5分钟 GPU 使用量`。

**效果**：

- 最近用 GPU 多 → 优先级下降
- 长时间未使用 → 优先级恢复

**作用**：实现长期公平

---

### 3.5 GPU Packing（GPU 装箱）

GPU 不应只运行一个任务。

**调度目标**：最大化 GPU 利用率

**资源模型**：

```
GPU Resource = (memory, compute)
```

**GPU 负载阈值**：

```
max_gpu_load = 85%
```

**调度逻辑**：

```
candidate = []
for gpu in gpus:
    if gpu.can_fit(task) and gpu.load < max_load:
        candidate.append(gpu)

gpu = best_fit(candidate)
```

**GPU Score**：

```
gpu_score = 0.7 * memory_util + 0.3 * compute_util
```

**作用**：在防止过载的前提下最大化 GPU 利用率

---

### 3.6 GPU Fragmentation Management（碎片管理）

GPU Packing 会产生 **显存碎片** 问题。

#### 3.6.1 问题示例

```
GPU memory = 40GB

TaskA = 12GB
TaskB = 10GB
TaskC = 8GB

剩余: 10GB

新任务需要: 15GB
→ 调度失败
```

#### 3.6.2 碎片检测

GPU Manager 定期计算：

```
fragmentation = free_memory / total_memory
```

如果 `fragmentation > threshold`，说明 GPU 碎片严重。

#### 3.6.3 碎片缓解策略

**1. Memory Reservation**

为大任务保留显存空间：

```
reserved_memory = 20%
```

防止 GPU 被小任务填满。

**2. GPU Class Scheduling**

任务按显存需求分类：

| 类型 | 显存 |
|------|------|
| small | < 4GB |
| medium | 4-16GB |
| large | > 16GB |

不同任务优先调度到不同 GPU。

**3. Spillover Scheduling**

当 GPU 无法容纳任务：

```
scheduler → 尝试下一个 GPU 节点
```

防止任务长时间排队。

---

### 3.7 Task Aging（任务老化）

防止长任务被短任务持续插队。

**优先级调整**：

```
priority = base_priority + wait_time * γ
```

**效果**：等待时间越长，优先级越高，保证任务最终会被执行。

---

## 4 GPU State Manager

GPU Resource Manager 负责维护 GPU 状态。

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

### 4.2 状态更新

GPU Worker 定期上报状态：

```
heartbeat_interval = 1s
```

更新内容：

- memory_usage
- gpu_utilization
- running_tasks

GPU Manager 更新 State Table。

### 4.3 调度器接口

调度器只读取 State Table：

```
调度器读取 → GPUStateTable → 做出决策
```

---

## 5 增强版调度器执行流程

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

GPU 调度系统必须具备 **反馈机制**。

### 6.1 系统结构

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

### 6.2 记录指标

| 指标 | 说明 |
|------|------|
| gpu_runtime | 实际 GPU 执行时间 |
| gpu_utilization | GPU 利用率 |
| memory_peak | 峰值显存使用 |
| task_latency | 端到端延迟 |
| queue_time | 排队时间 |

### 6.3 调度优化

调度器定期计算：

- GPU utilization
- queue latency
- task wait time

如果发现 `GPU utilization < target`：

→ 说明 packing strategy 需要调整

---

## 7 调度器数据结构

### 7.1 核心结构

```go
type Scheduler struct {
    TaskQueue
    UserTokenTable
    GPUStateTable
    CostModel
}
```

### 7.2 任务结构

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

## 8 调度复杂度

**调度循环复杂度**：

```
O(N log N)
```

其中：N = task_queue_size

**GPU Packing**：

```
O(G)
```

其中：G = GPU 数量

整体调度开销非常低。

---

## 9 系统稳定性设计

| 问题 | 解决方案 |
|------|---------|
| GPU 拥塞 | Admission Control |
| 用户垄断 | Token Bucket |
| 任务饿死 | Task Aging |
| GPU 碎片 | GPU Packing + Fragmentation Management |
| 成本失真 | Cost Model + Feedback Loop |

---

## 10 最终调度系统结构

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

## 11 设计总结

**核心原则**：

```
公平性
+
成本预测
+
资源装箱
+
碎片控制
+
反馈闭环
```

**完整调度策略**：

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

**系统目标**：

- 高 GPU 利用率
- 多用户公平
- 稳定推理服务

该设计适用于 AI Agent 工作流 GPU 资源调度，重点在于工程实用性与系统稳定性，而非理论最优调度。
