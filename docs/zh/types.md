# 类型模块

## 概述

类型模块定义了 AlgoGPU 系统中使用的核心数据结构和枚举。它为任务、GPU 资源和系统配置提供了通用的词汇表。

## 核心类型

### Task

代表 GPU 任务及其所有必需的元数据。

```go
type Task struct {
    ID                 string           // 唯一任务标识符
    UserID             string           // 提交任务的用户
    Type               TaskType         // 任务类型（嵌入、LLM 等）
    GPUMemoryRequired  int64            // 所需 GPU 内存（MB）
    GPUComputeRequired int64            // 所需 GPU 计算单元
    EstimatedRuntimeMs int64            // 预计运行时间（毫秒）
    Priority           int              // 任务优先级（越高越重要）
    Status             TaskStatus       // 当前任务状态
    Payload            []byte           // 任务负载（输入数据）
    Result             []byte           // 任务结果（输出数据）
    Message            string           // 状态消息或错误
    CreatedAt          time.Time        // 任务创建时间戳
    StartedAt          time.Time        // 任务开始时间戳
    CompletedAt        time.Time        // 任务完成时间戳
    GPUID              int              // 运行任务的 GPU ID
}
```

### TaskType

支持的 GPU 任务类型枚举。

```go
const (
    TaskTypeUnspecified TaskType = 0  // 未指定类型
    TaskTypeEmbedding  TaskType = 1  // 文本嵌入生成
    TaskTypeLLM        TaskType = 2  // 大语言模型推理
    TaskTypeDiffusion  TaskType = 3  // 扩散模型生成
    TaskTypeOther      TaskType = 4  // 通用 GPU 任务
)
```

**使用:**
```go
task := &types.Task{
    Type: api.TaskType_TASK_TYPE_LLM,
}
```

### TaskStatus

任务状态枚举。

```go
const (
    TaskStatusPending   TaskStatus = 0  // 在队列中等待
    TaskStatusRunning   TaskStatus = 1  // 当前正在执行
    TaskStatusCompleted TaskStatus = 2  // 成功完成
    TaskStatusFailed    TaskStatus = 3  // 失败并报错
    TaskStatusCancelled TaskStatus = 4  // 被用户取消
    TaskStatusRejected  TaskStatus = 5  // 被准入控制拒绝
)
```

**状态转换:**
```
PENDING → RUNNING → COMPLETED
    ↓
CANCELLED
    ↓
FAILED
```

## 配置类型

### SchedulerConfig

调度器配置参数。

```go
type SchedulerConfig struct {
    MaxQueueSize       int     // 最大队列大小
    TokenRefillRate    int64   // 每秒令牌数
    TokenBucketSize    int64   // 每用户最大令牌数
    DailyTokenLimit    int64   // 每日令牌限制
    GPULoadThreshold   float64 // 最大 GPU 负载 (0-1)
    AgingFactor        float64 // 任务老化因子
    MemoryWeight       float64 // 负载计算中的内存权重
    ComputeWeight      float64 // 负载计算中的计算权重
}
```

## GPU 类型

GPU 相关类型在 internal/gpu 包中定义。

### GPU

代表单个 GPU 及其状态。

```go
type GPU struct {
    ID            int       // GPU 标识符
    Name          string    // GPU 名称
    MemoryTotal   int64     // 总内存（MB）
    MemoryUsed    int64     // 已用内存（MB）
    ComputeUtil   int       // 计算利用率 (0-100)
    MemoryUtil    int       // 内存利用率 (0-100)
    Temperature   int       // 温度（摄氏度）
    ReservedBy    string    // 被任务 ID 预留
    RunningTasks  []string  // 运行中的任务 ID
}
```

## API 类型

API 类型使用 Protocol Buffers 在 api 包中定义。

### TaskRequest

任务提交的 gRPC 请求。

### TaskResponse

任务提交的 gRPC 响应。

### TaskStatusRequest

任务状态查询的 gRPC 请求。

### TaskStatusResponse

任务状态查询的 gRPC 响应。

## 工具函数

### GenerateID

生成唯一的任务 ID。

```go
func GenerateID(prefix string) string {
    return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
```

## 最佳实践

1. 始终设置必需字段（ID、UserID、Type、GPUMemoryRequired）
2. 使用适当的任务类型
3. 正确处理任务状态转换
4. 提交前验证任务字段
5. 使用有意义的任务 ID 以便调试

## 使用示例

### 创建任务

```go
task := &types.Task{
    ID:                 "task-1",
    UserID:             "user-1",
    Type:               api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired:  8192,
    GPUComputeRequired: 200,
    EstimatedRuntimeMs: 5000,
    Priority:           100,
    CreatedAt:          time.Now(),
}
```

### 检查任务状态

```go
if task.Status == api.TaskStatus_TASK_STATUS_COMPLETED {
    // 任务成功完成
    fmt.Printf("Result: %s\n", string(task.Result))
} else if task.Status == api.TaskStatus_TASK_STATUS_FAILED {
    // 任务失败
    fmt.Printf("Error: %s\n", task.Message)
}
```

## 性能考虑

- 使用 `[]byte` 而不是 `string` 作为负载以提高效率
- 保持任务元数据小以便快速序列化
- 使用适当的数据类型（例如，时间戳使用 `int64`）

## 测试

运行类型测试:
```bash
go test ./pkg/types/...
```