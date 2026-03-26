# 插件模块

## 概述

插件模块提供一个最小的接口，用于将 AlgoGPU 调度器作为插件嵌入到 agent 框架中。它提供两种部署模式：独立服务和嵌入式插件。

## 组件

### Scheduler 接口

插件模式的最小接口，包含 6 个核心方法。

```go
type Scheduler interface {
    SubmitTask(ctx context.Context, task *types.Task) error
    GetStatus(ctx context.Context, taskID string) (*types.Task, error)
    GetQueueSize() int
    GetRunningCount() int
    GetGPUStatus(ctx context.Context) []*api.GPUInfo
    CancelTask(ctx context.Context, taskID string) error
}
```

### PluginScheduler

实现 Scheduler 接口，包装核心调度器。

**关键特性:**
- 简单验证
- 错误处理
- 与核心调度器集成
- 直接访问 GPU 池和任务队列

**关键方法:**
- `NewPluginScheduler(sched, taskQueue, gpuPool)` - 创建插件调度器
- `SubmitTask(ctx, task)` - 提交任务并验证
- `GetStatus(ctx, taskID)` - 获取任务状态
- `GetQueueSize()` - 获取待处理任务数
- `GetRunningCount()` - 获取运行中任务数
- `GetGPUStatus(ctx)` - 获取所有 GPU 状态
- `CancelTask(ctx, taskID)` - 取消任务

## 部署模式

### 独立服务模式

作为独立进程运行，提供 gRPC 和 HTTP API。

**使用:**
```bash
make build-standalone
make run-standalone
```

**特性:**
- 完整 API 支持（gRPC + HTTP）
- 独立部署
- 适合生产环境
- 易于监控

### 插件模式

直接嵌入到 agent 应用程序中。

**使用:**
```go
scheduler := plugin.NewPluginScheduler(coreScheduler, taskQueue, gpuPool)
err := scheduler.SubmitTask(ctx, task)
```

**特性:**
- 无网络开销
- 简单接口
- 易于集成
- 快速启动

## 使用示例

### 初始化插件调度器

```go
// 创建核心组件
cfg := scheduler.DefaultConfig()
taskQueue := queue.NewTaskQueue()
gpuPool := gpu.NewPool()
coreScheduler := scheduler.NewScheduler(cfg, taskQueue, gpuPool)

// 创建插件调度器
pluginScheduler := plugin.NewPluginScheduler(coreScheduler, taskQueue, gpuPool)

// 启动调度器
coreScheduler.Start()
```

### 提交任务

```go
task := &types.Task{
    ID:                 "task-1",
    UserID:             "user-1",
    Type:               api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired:  8192,
    GPUComputeRequired: 200,
    EstimatedRuntimeMs: 5000,
    Payload:            []byte("Hello, world!"),
}

err := pluginScheduler.SubmitTask(ctx, task)
if err != nil {
    // 处理错误
}
```

### 获取任务状态

```go
task, err := pluginScheduler.GetStatus(ctx, "task-1")
if err != nil {
    // 未找到任务
}

fmt.Printf("Status: %v\n", task.Status)
```

### 取消任务

```go
err := pluginScheduler.CancelTask(ctx, "task-1")
if err != nil {
    // 未找到任务或已取消
}
```

### 监控队列

```go
queueSize := pluginScheduler.GetQueueSize()
runningCount := pluginScheduler.GetRunningCount()

fmt.Printf("Queue size: %d, Running: %d\n", queueSize, runningCount)
```

### 获取 GPU 状态

```go
gpuStatus := pluginScheduler.GetGPUStatus(ctx)
for _, gpu := range gpuStatus {
    fmt.Printf("GPU %d: %s\n", gpu.Id, gpu.Name)
}
```

## 验证

插件调度器验证所有输入:

- 任务不能为 nil
- 任务 ID 不能为空
- 用户 ID 不能为空
- GPU 内存需求必须为正数

**示例:**
```go
err := pluginScheduler.SubmitTask(ctx, nil)
// Error: "task cannot be nil"
```

## 错误处理

所有错误都被返回以进行适当处理:

```go
err := pluginScheduler.SubmitTask(ctx, task)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "task cannot be nil"):
        // 处理 nil 任务
    case strings.Contains(err.Error(), "task ID cannot be empty"):
        // 处理空 ID
    case strings.Contains(err.Error(), "GPU memory"):
        // 处理内存验证
    }
}
```

## 最佳实践

1. 提交前始终验证输入
2. 优雅处理错误
3. 监控队列大小以进行负载均衡
4. 使用适当的 GPU 内存需求
5. 完成后停止调度器

## 测试

运行插件测试:
```bash
go test ./internal/plugin/...
```

## 集成示例

查看 `examples/go-agent/main.go` 获取完整的集成示例。

## 性能

- **提交**: O(1) 验证 + O(log n) 入队
- **状态查询**: O(1) 哈希查找
- **取消**: O(log n) 移除
- **GPU 状态**: O(n) 获取所有 GPU