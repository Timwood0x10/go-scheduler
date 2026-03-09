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

**职责划分**：

| 组件 | 职责 |
|-----|------|
| Workflow Engine | 任务 DAG 执行 |
| GPU Scheduler | GPU 任务调度 |
| GPU Manager | GPU 资源分配 |
| GPU Workers | 执行具体模型 |

> **注意**：LLM 内部 token 调度由推理框架（vLLM、TensorRT-LLM 等）负责，不在调度系统范围内。

---

## 3 Task-level Scheduling

Task-level Scheduler 负责 GPU 任务调度。

调度策略由五部分组成：

- **Admission Control** — 准入控制
- **Token Bucket** — 用户配额
- **Cost-aware Scheduling** — 成本感知调度
- **GPU Packing** — GPU 装箱
- **Task Aging** — 任务老化

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

### 3.3 Cost-aware Scheduling（成本感知调度）

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

### 3.4 GPU Packing（GPU 装箱）

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

**作用**：减少 GPU 碎片，避免单 GPU 过载

---

### 3.5 Task Aging（任务老化）

防止长任务被短任务持续插队。

**优先级调整**：

```
priority = base_priority + wait_time * γ
```

**效果**：等待时间越长，优先级越高，保证任务最终会被执行。

---

## 4 GPU Scheduler 执行流程

GPU Scheduler 周期性运行调度循环。

**伪代码**：

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

## 5 GPU 任务类型

GPU Scheduler 不关心模型细节，只管理任务资源。

**任务示例**：

- LLM inference
- embedding generation
- diffusion image generation
- GPU data processing

**任务定义**：

```
task = {
    gpu_memory,
    gpu_compute,
    estimated_runtime
}
```

Scheduler 根据资源需求进行调度。

---

## 6 系统效果

| 指标 | 改善 |
|-----|-----|
| GPU 利用率 | ↑ |
| 平均延迟 | ↓ |
| 系统稳定性 | ↑ |
| 多用户公平 | ↑ |

系统可从「简单 AI 服务」升级为「多租户 AI 工作流计算平台」。

---

## 7 设计总结

**核心思想**：公平 + 成本感知 + 资源装箱

**调度策略**：

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

该设计适用于 AI Agent 工作流 GPU 资源调度，在保证公平性和稳定性的同时，提高 GPU 利用率和系统吞吐量。该调度系统目标是“工程实用调度”，而非最优调度算法。