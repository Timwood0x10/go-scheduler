# AI Agent GPU 平台系统设计 v2

## 1 概述

**目标**: 构建支持复杂 Agent 工作流的 AI 推理平台，具备：
- 高 GPU 利用率
- 多用户公平
- Agent 工作流调度
- 成本可预测
- 自动学习 GPU 使用模式

---

## 2 架构 (5 层)

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

## 3 各层详情

### 3.1 API 层

| 模块 | 职责 |
|------|------|
| API Gateway | 认证、限流、路由 |

### 3.2 Workflow 层

| 模块 | 职责 | 存在理由 |
|------|------|---------|
| User Profiling | 用户上下文、历史、偏好 | 调度公平性基础 |
| Leader Agent | 任务规划、DAG 分解 | Agent 逻辑 |
| Workflow Scheduler | DAG 执行、依赖管理 | 并行任务执行 |
| Task State Manager | 任务生命周期、重试、恢复 | 任务失败处理 |
| Workflow Store | 持久化、检查点、恢复 | 系统重启恢复 |

### 3.3 Resource 层

| 模块 | 职责 | 存在理由 |
|------|------|---------|
| Cost Estimator | GPU 成本预测 | 按预测成本调度 |
| Admission Controller | 系统保护 | 防止过载 |
| GPU Scheduler | GPU 分配、装箱 | 核心调度 |
| Engine Router | 模型路由 | 多模型支持 |

### 3.4 Execution 层

| 模块 | 职责 |
|------|------|
| LLM Engine | 文本生成 |
| Image Engine | 图像生成 |
| Embedding Engine | 向量嵌入 |

### 3.5 Observability 层

| 模块 | 用途 |
|------|------|
| Metrics | GPU 利用率、延迟、吞吐量 (Prometheus) |
| Logs | 执行日志、错误 (ELK) |
| Tracing | 完整工作流路径 (OpenTelemetry) |

---

## 4 模块详情

### 4.1 User Profiling

```
输入: User ID
输出: UserContext

包含:
- 历史任务
- GPU 使用量
- 用户等级
- 用户偏好

用于:
priority = weight / (gpu_usage + task_cost)
```

### 4.2 Leader Agent

```
输入: 用户请求
输出: TaskGraph

TaskGraph:
├── LLM Task
├── Tool Task
└── Image Task

注意: 仅负责规划，不负责资源调度
```

### 4.3 Workflow Scheduler

```
Task A → Task B
Task A → Task C

A 完成后: B 和 C 并行执行
```

### 4.4 Task State Manager

**状态**:
```
PENDING → RUNNING → SUCCESS
              ↓
            FAILED → RETRY
              ↓
           CANCELLED
```

**功能**:
- 重试计数
- 超时管理
- 取消处理

### 4.5 Workflow Store

**存储**:
- 工作流图
- 任务状态
- 依赖关系

**推荐**: PostgreSQL + Redis

### 4.6 Cost Estimator

**输入**:
```
- task_type
- model
- input_tokens
- historical_data
```

**输出**:
```
estimated_cost = token_cost + memory_cost + latency_cost
```

**公式**:
```
cost = α × feature_based + β × historical_avg
```

### 4.7 Admission Controller

**规则**:
```go
if user_quota_exceeded { reject }
if queue_length > threshold { reject }
if estimated_cost > max { reject }
```

### 4.8 GPU Scheduler

**优先级** (含时间衰减):
```
priority = weight / (user_gpu_usage × decay + task_cost)
```

**GPU 装箱**:
```
Best Fit + 负载阈值 (80%)
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

**策略**:
- 模型版本匹配
- 负载均衡
- 能力匹配

---

## 5 反馈闭环

```
任务执行
       │
       ▼
Metrics Collector
       │
       ▼
GPU Usage DB
       │
       ▼
Cost Estimator ←── 自学习
       │
       ▼
GPU Scheduler
```

---

## 6 设计原则

| 原则 | 实现 |
|------|------|
| 控制与执行分离 | Leader Agent → 规划, Scheduler → 资源, Engine → 执行 |
| 两级调度 | Workflow Scheduler → GPU Scheduler |
| 成本驱动调度 | Cost Estimator → 优先级 |
| 可观测优先 | Metrics + Logs + Tracing |

---

## 7 瓶颈与解决方案

| 瓶颈 | 解决方案 |
|------|---------|
| Leader Agent 每次请求都规划 | 规划缓存 |
| 单调度器实例 | 分布式调度 (未来) |

---

## 8 系统特性

- ✓ Agent 工作流支持
- ✓ GPU 装箱
- ✓ 成本感知调度
- ✓ 多租户公平
- ✓ 自学习成本模型
- ✓ 任务状态管理
- ✓ 工作流持久化
- ✓ 完整可观测性
