# GPU Scheduler 项目开发文档

## 1 项目概述

### 1.1 项目定位

独立的高性能 GPU 调度系统，以插件形式接入 Python 编写的 AI Agent 工作流，对 GPU 资源进行统一调度管理。

### 1.2 系统目标

- **公平性**：多用户资源分配公平，防止单用户占满 GPU
- **稳定性**：防止请求洪峰压垮系统，高峰期稳定运行
- **高利用率**：通过 GPU 装箱策略最大化资源利用率
- **低延迟**：任务排队时间可控，短任务优先执行

### 1.3 解决的问题

1. GPU 被单用户占满
2. 长任务阻塞短任务
3. GPU 利用率低
4. 多用户资源不公平
5. 系统高峰期不稳定

---

## 2 技术选型

### 2.1 核心语言

- **Go 1.26+**：高性能调度服务

### 2.2 通信方式

- **gRPC**：Go ↔ Python 跨语言通信
  - 高效的二进制序列化
  - 支持流式调用
  - 多语言客户端支持

### 2.3 存储

- **内存**：任务队列、运行时状态
- **可选持久化**：任务状态 checkpoint（后续阶段）

---

## 3 系统架构

### 3.1 整体架构

```
┌─────────────────┐       gRPC        ┌─────────────────┐
│  Python Agent  │ ───────────────▶  │   Go Scheduler  │
│   (工作流引擎)   │   SubmitTask()    │   (调度服务)    │
│                 │   GetStatus()     │                 │
│                 │   CancelTask()    │                 │
└─────────────────┘                   └────────┬────────┘
                                               │
                    GPU Resource               ▼
                    Management        ┌─────────────────┐
                                     │   GPU Manager   │
                                     │  (资源追踪/分配) │
                                     └────────┬────────┘
                                              │
                    GPU Pool                 ▼
            ┌─────────────┬─────────────┐ ┌──────────┐
            │    GPU 0    │    GPU 1    │ │  GPU N   │
            └─────────────┴─────────────┘ └──────────┘
```

### 3.2 组件职责

| 组件 | 职责 |
|-----|------|
| Python Agent | 工作流执行，调用 SDK 提交 GPU 任务 |
| gRPC API | 任务提交、状态查询、取消等接口 |
| Task Queue | 任务排队、优先级排序 |
| Scheduler | 调度策略执行（Admission Control、Token Bucket 等）|
| GPU Manager | GPU 资源追踪、最佳 GPU 选择 |
| GPU Workers | 实际任务执行（调用外部服务）|

---

## 4 调度策略

### 4.1 五层调度策略

| 策略 | 作用 |
|-----|------|
| Admission Control | 防止系统过载，限制队列长度 |
| Token Bucket | 用户配额管理，防止单用户占满 GPU |
| Cost-aware Scheduling | 成本感知调度，滑动窗口实现长期公平 |
| GPU Packing | GPU 装箱，Best Fit + 负载阈值 |
| Task Aging | 任务老化，防止长任务饿死 |

### 4.2 优先级计算

```
priority = (weight / (recent_gpu_usage + task_cost)) + (wait_time * γ)
```

---

## 5 任务状态机

### 5.1 状态定义

| 状态 | 含义 |
|-----|------|
| PENDING | 任务已提交，等待调度 |
| RUNNING | 任务正在执行 |
| COMPLETED | 任务成功完成 |
| FAILED | 任务执行失败 |
| CANCELLED | 任务被取消 |
| REJECTED | 任务被拒绝（超出配额/队列限制）|

### 5.2 状态流转

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
 CANCELLED (任意时刻可触发)
```

### 5.3 状态转换规则

| 当前状态 | 触发事件 | 目标状态 | 说明 |
|---------|---------|---------|------|
| PENDING | scheduler dispatch | RUNNING | 任务被调度执行 |
| RUNNING | task completed | COMPLETED | 任务成功完成 |
| RUNNING | task error | FAILED | 任务执行失败 |
| PENDING | user cancel | CANCELLED | 用户取消任务 |
| RUNNING | user cancel | CANCELLED | 用户取消任务 |
| PENDING | admission reject | REJECTED | 超出配额或队列限制 |
| 任意 | system shutdown | CANCELLED | 系统关闭 |

---

## 6 GPU 状态采集

### 6.1 GPU Manager 架构

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

### 6.2 采集方式

| 采集方式 | 说明 |
|---------|------|
| nvidia-smi | 命令行工具，简单场景可用 |
| NVML (NVIDIA Management Library) | C API，Go 可通过 cgo 调用 |
| DCGM (Data Center GPU Manager) | 企业级监控，功能最全 |

### 6.3 GPU 负载计算

```
gpu_score = 0.7 * memory_util + 0.3 * compute_util
```

其中：

- `memory_util = memory_used / memory_total`
- `compute_util = compute_used / compute_total`

---

## 7 调度触发机制

### 7.1 Event-driven vs Polling

当前设计采用 **Event-driven** 模式，调度器响应事件而非轮询。

### 7.2 触发事件

| 事件 | 来源 | 触发动作 |
|-----|------|---------|
| new task arrival | gRPC SubmitTask | 触发调度 |
| task completion | Worker 回调 | 触发调度 |
| task failure | Worker 回调 | 触发调度 |
| gpu freed | Metrics Collector | 触发调度 |
| periodic tick | Timer (100ms) | 防止任务遗漏 |

### 7.3 调度执行

```
on_event(event):
    run_scheduler()

run_scheduler():
    1. 从队列获取待调度任务
    2. 计算任务优先级
    3. 排序任务
    4. 为每个任务寻找最佳 GPU
    5. 分配 GPU 并更新状态
    6. 调度失败的任务重新入队
```

---

## 8 gRPC 接口设计

### 8.1 服务定义

```protobuf
service GPUScheduler {
    // 提交任务
    rpc SubmitTask(TaskRequest) returns (TaskResponse);
    
    // 查询任务状态
    rpc GetTaskStatus(TaskStatusRequest) returns (TaskStatusResponse);
    
    // 取消任务
    rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);
    
    // 获取 GPU 状态
    rpc GetGPUStatus(Empty) returns (GPUStatusResponse);
    
    // 任务结果流（可选）
    rpc TaskEvents(Empty) returns (stream TaskEvent);
}
```

### 8.2 核心数据类型

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
    bytes payload = 7;  // 任务具体数据
}

message TaskResponse {
    bool accepted = 1;
    string message = 2;
    TaskStatus status = 3;
}
```

---

## 9 Python SDK 设计

### 9.1 使用方式

```python
from gpu_scheduler import GPUClient

client = GPUClient(host="localhost:50051")

# 提交任务
task_id = client.submit_task(
    user_id="user_123",
    task_type="llm",
    gpu_memory_mb=8192,
    payload={"prompt": "..."}
)

# 查询状态
status = client.get_status(task_id)
print(f"Task status: {status}")

# 阻塞等待结果
result = client.wait(task_id)
print(f"Result: {result}")
```

### 9.2 SDK 核心接口

```python
class GPUClient:
    def submit_task(self, user_id, task_type, gpu_memory_mb, payload) -> str
    def get_status(self, task_id) -> TaskStatus
    def cancel_task(self, task_id) -> bool
    def get_gpu_status(self) -> List[GPUInfo]
    def wait(self, task_id, timeout=None) -> TaskResult
```

---

## 10 项目阶段规划

### 10.1 阶段一：基础框架搭建

**目标**：搭建可运行的最小调度系统

**任务**：

1. 项目结构初始化
2. gRPC 服务定义与生成
3. 基础任务队列实现
4. 简单的任务提交/查询接口
5. Python SDK 基础封装

**里程碑**：Python Agent 能提交任务并获取任务 ID

### 10.2 阶段二：调度策略实现

**目标**：实现核心调度策略

**任务**：

1. Admission Control 实现
2. Token Bucket 用户配额实现
3. Cost-aware Scheduling 实现（含滑动窗口）
4. GPU Manager 基础实现
5. GPU Packing 策略实现
6. Task Aging 机制实现

**里程碑**：多任务提交时能按策略调度

### 10.3 阶段三：任务执行与集成

**目标**：任务能真正执行

**任务**：

1. 任务执行器抽象
2. GPU 状态采集（nvidia-smi/NVML）
3. 任务生命周期管理
4. Event-driven 调度触发机制
5. 结果回调机制
6. 错误处理与重试

**里程碑**：任务能执行完成并返回结果

### 10.4 阶段四：监控与运维

**目标**：生产可用

**任务**：

1. 日志与指标暴露（Prometheus）
2. 健康检查接口
3. 配置热加载
4. 优雅关闭
5. 性能压测

**里程碑**：可部署生产环境

---

## 11 目录结构

```
algogpu/
├── cmd/
│   └── scheduler/
│       └── main.go           # 入口
├── internal/
│   ├── scheduler/
│   │   ├── scheduler.go      # 调度器核心 (Event-driven)
│   │   ├── admission.go      # 准入控制
│   │   ├── token_bucket.go   # Token Bucket
│   │   ├── cost_aware.go     # 成本感知调度
│   │   ├── gpu_packing.go    # GPU 装箱
│   │   └── aging.go          # 任务老化
│   ├── gpu/
│   │   ├── manager.go        # GPU 管理
│   │   ├── pool.go           # GPU 池
│   │   └── collector.go     # GPU 状态采集
│   ├── queue/
│   │   └── task_queue.go    # 任务队列
│   ├── state/
│   │   └── task_state.go     # 任务状态机
│   └── server/
│       └── grpc.go           # gRPC 服务
├── pkg/
│   └── types/
│       └── types.go          # 公共类型
├── api/
│   └── gpu_scheduler.proto   # gRPC 定义
├── python/
│   └── gpu_scheduler/        # Python SDK
│       ├── __init__.py
│       └── client.py
├── docs/
│   ├── gpu-scheduler.md      # 设计文档
│   └── development.md         # 本文档
└── go.mod
```

---

## 12 验收标准

### 12.1 功能验收

- [ ] Python Agent 能通过 SDK 提交任务
- [ ] 任务按优先级顺序调度
- [ ] 超出配额的任务被拒绝
- [ ] 超出队列限制时被拒绝
- [ ] GPU 负载均衡分配
- [ ] 任务能执行完成并返回结果
- [ ] 任务状态机流转正确

### 12.2 性能验收

- [ ] 单任务提交延迟 < 10ms
- [ ] 调度周期 < 100ms
- [ ] 支持 100+ 并发任务

### 12.3 稳定性验收

- [ ] 无内存泄漏
- [ ] 优雅处理服务重启
- [ ] 错误信息清晰可查

---

## 13 后续扩展

- [ ] 多节点 GPU 调度
- [ ] 任务优先级动态调整
- [ ] 调度策略可配置化
- [ ] 任务 checkpoint 与恢复
- [ ] 调度可视化 Dashboard