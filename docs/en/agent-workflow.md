# AI Agent Workflow System Design

## 1 System Architecture

```
┌──────────────────────────┐
│        User Client        │
└─────────────┬────────────┘
              │
              ▼
┌──────────────────────────┐
│        API Gateway       │
│  auth / rate limit       │
└─────────────┬────────────┘
              │
              ▼
┌─────────────────────────────────────┐
│      User Fairness Scheduler        │
│  token bucket / quota / priority    │
└─────────────┬───────────────────────┘
              │
              ▼
┌────────────────────────────────────┐
│     User Profiling + Small Model    │
│  intent extraction / user memory     │
└─────────────┬──────────────────────┘
              │
              ▼
┌────────────────────────────────────┐
│           Leader Agent              │
│  planning / task decomposition       │
└─────────────┬───────────────────────┘
              │
              ▼
┌───────────────────────────┐
│        Task Graph         │
│  DAG workflow representation │
└─────────────┬─────────────┘
              │
┌─────────────┼─────────────┬───────────────────────┐
▼             ▼             ▼                       ▼
┌──────────────────┐  ┌────────────────────┐  ┌──────────────────┐
│ Cost Estimator   │  │ Workflow Scheduler │  │ Resource Guard   │
│ historical model │  │ concurrency control│  │ token / request │
└─────────┬────────┘  └─────────┬──────────┘  └─────────┬────────┘
          │                      │                        │
          └──────────────┬───────┴────────────────┬───────┘
                         ▼                        ▼
               ┌───────────────────────────────────────┐
               │           GPU Scheduler                │
               │  GPU packing / fragmentation / fairness │
               └───────────────┬───────────────────────┘
                               │
      ┌────────────────────────┼────────────────────────┐
      ▼                        ▼                        ▼
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│ LLM Inference│      │ Image Model  │      │ Embedding    │
│ Engine       │      │ Engine       │      │ Engine       │
└──────┬───────┘      └──────┬───────┘      └──────┬───────┘
       │                      │                        │
       └──────────────────────┼────────────────────────┘
                              ▼
              ┌────────────────────────────────────┐
              │   Token-level Scheduler (Inference)  │
              │  dynamic batching / token interleaving│
              └─────────────────────┬──────────────┘
                                    │
                                    ▼
                            ┌───────────────┐
                            │ Result Validator │
                            │ output check    │
                            └───────┬────────┘
                                    ▼
                                Response
                                    │
                                    │ (后台异步)
                                    ▼
                           ┌───────────────┐
                           │Metrics Collector│
                           └───────┬───────┘
                                   │
                                   ▼
                          GPU Task Metrics DB
                                   │
                                   ▼
                          Cost Model Updater
                                   │
                                   ▼
                          Cost Estimator Cache
```

---

## 2 Module Details

### 2.1 API Gateway

**职责**:
- 认证 (Authentication)
- 限流 (Rate Limiting)
- 请求路由

```
请求 → 验证 Token → 限流检查 → 转发
```

### 2.2 User Fairness Scheduler

**职责**: 多用户公平性保证

**策略**:
- Token Bucket (用户配额)
- 优先级调度
- 每日/每小时限制

```
┌─────────────────────────────────────┐
│         User Fairness Scheduler      │
├─────────────────────────────────────┤
│ • Token Bucket: 用户配额             │
│ • Quota: 资源限制                   │
│ • Priority: 动态优先级               │
│ • Scheduler: 跨用户公平              │
└─────────────────────────────────────┘
```

### 2.3 User Profiling + Small Model

**职责**:
- 用户识别
- 画像管理
- 意图提取

```
User Input
    │
    ▼
Small Model (意图识别)
    │
    ▼
User Profiling (画像匹配)
    │
    ▼
UserContext
```

### 2.4 Leader Agent

**职责**:
- 需求理解
- 任务规划
- 工作流拆分

```
UserContext + User Input
    │
    ▼
需求理解
    │
    ▼
Task Decomposition
    │
    ▼
Task Graph (DAG)
```

### 2.5 Task Graph

**职责**: 任务依赖关系表示

```go
type TaskGraph struct {
    Tasks    []Task      // 任务列表
    Edges    []TaskEdge  // 依赖边
}

type Task struct {
    ID          string
    Type        TaskType  // "tool", "sub_agent"
    Engine      EngineType // "llm", "image", "embedding"
    Input       map[string]interface{}
    Dependencies []string
}
```

### 2.6 Cost Estimator

**职责**: 基于历史数据预估 GPU 成本

```
输入: Task特征 + 历史数据
    │
    ▼
预估计算
    │
    ▼
GPU Cost Estimate
```

**预估公式**:
```
cost = α × feature_based + β × historical_avg
```

### 2.7 Workflow Scheduler

**职责**: 工作流级别的调度

**功能**:
- 并发控制 (Concurrency Control)
- 任务依赖排序
- DAG 拓扑排序

### 2.8 Resource Guard

**职责**: 资源保护

**功能**:
- 单次请求 Token 限制
- 请求大小限制
- 并发请求数限制

### 2.9 GPU Scheduler

**职责**: GPU 资源分配

**功能**:
- GPU Packing (装箱)
- 碎片优化
- 多租户公平

**调度策略**:
```
1. Admission Control
2. Token Bucket
3. Cost-aware (Agent 预估 + 历史)
4. GPU Packing (Best Fit)
5. Task Aging
```

### 2.10 Model Engines

**执行层**:

| Engine | 任务类型 |
|--------|---------|
| LLM Inference | 文本生成 |
| Image Model | 图像生成/处理 |
| Embedding | 向量嵌入 |

### 2.11 Token-level Scheduler (Inference)

**职责**: 推理引擎内部调度

**功能**:
- Dynamic Batching (动态批处理)
- Token Interleaving (Token 交织)
- KV Cache 管理

> 注意: 这个在 Inference Engine 内部，不在 GPU Scheduler 范围内

### 2.12 Result Validator

**职责**: 输出校验

```
执行结果
    │
    ▼
需求匹配检查
    │
    ▼
格式检查
    │
    ▼
重试判断
```

### 2.13 Metrics Collector (后台)

**职责**: 收集执行指标

```
Task 完成
    │
    ▼
收集 Metrics
    │
    ▼
存储 DB
```

**收集的数据**:
```go
type TaskMetrics struct {
    TaskID         string
    TaskType       string
    Engine         string
    
    // 预估 vs 实际
    EstimatedCost  int64
    ActualCost     int64
    
    // GPU 指标
    GPUMemoryMB    int64
    GPUUtilPercent float64
    
    // 时间
    QueueTimeMs    int64
    ExecTimeMs    int64
}
```

### 2.14 Cost Model Updater

**职责**: 更新成本预估模型

```
Metrics 到达
    │
    ▼
计算误差
    │
    ▼
更新参数 (α, β)
    │
    ▼
更新缓存
```

---

## 3 Data Flow Summary

### 3.1 Request Flow

```
User → API Gateway → User Fairness → Profiling
    → Leader Agent → Task Graph → Cost Estimator
    → Workflow Scheduler + Resource Guard
    → GPU Scheduler → Engines
    → Token-level Scheduler (in Engine)
    → Result Validator → Response
```

### 3.2 Feedback Flow

```
Task Complete → Metrics Collector → DB
                            │
                            ▼
                    Cost Model Updater
                            │
                            ▼
                    Cost Estimator Cache
```

---

## 4 Module Responsibilities

| Module | Layer | Responsibility |
|--------|-------|---------------|
| API Gateway | Gateway | Auth, Rate Limit |
| User Fairness Scheduler | Scheduling | Multi-tenant fairness |
| User Profiling + Small Model | Entry | User context |
| Leader Agent | Planning | Task decomposition |
| Task Graph | Data | Workflow representation |
| Cost Estimator | Estimation | GPU cost prediction |
| Workflow Scheduler | Scheduling | Concurrency, DAG |
| Resource Guard | Protection | Limits enforcement |
| GPU Scheduler | Scheduling | GPU allocation |
| Model Engines | Execution | Task execution |
| Token-level Scheduler | Execution | Inference batching |
| Result Validator | Validation | Output check |
| Metrics Collector | Observability | Metrics collection |
| Cost Model Updater | Learning | Model update |

---

## 5 Key Design Points

### 5.1 Two-level Scheduling

1. **Workflow-level**: Task Graph → Workflow Scheduler
2. **GPU-level**: GPU Scheduler

### 5.2 Two-level Fairness

1. **User-level**: User Fairness Scheduler
2. **Task-level**: GPU Scheduler

### 5.3 Feedback Loop

```
Metrics → Cost Model Updater → Cost Estimator → GPU Scheduler
```

### 5.4 Separation of Concerns

- Leader Agent: 任务规划 (不含成本预估)
- Cost Estimator: 独立成本预估
- GPU Scheduler: 只负责调度
- Token-level Scheduler: Inference 内部 (不实现)

---

## 6 Interaction with GPU Scheduler

```
┌─────────────────────────────────────────────────────────────┐
│                     Cost Estimator                          │
│  Input: Task features, historical data                       │
│  Output: GPU cost estimate                                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    GPU Scheduler                            │
│  • Receives: Agent estimate + historical cost                │
│  • Policy: priority = f(estimate, historical, weight)          │
│  • Output: GPU allocation                                   │
└─────────────────────────────────────────────────────────────┘
```
