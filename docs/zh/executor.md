# 执行器模块

## 概述

执行器模块处理 GPU 上的异步任务执行。它确保正确的资源清理，并为运行不同类型的 GPU 任务提供简单的接口。

## 组件

### Runner

异步执行 GPU 任务，自动清理资源。

**关键特性:**
- 异步执行（非阻塞）
- 完成后自动释放 GPU
- 支持多种任务类型
- 超时处理
- 错误传播

**关键方法:**
- `NewRunner(gpuPool)` - 创建新的执行器
- `Run(task, gpu)` - 在 GPU 上异步执行任务

## 任务类型

执行器支持四种类型的 GPU 任务:

1. **TASK_TYPE_EMBEDDING** - 文本嵌入生成
2. **TASK_TYPE_LLM** - 大语言模型推理
3. **TASK_TYPE_DIFFUSION** - 扩散模型生成
4. **TASK_TYPE_OTHER** - 通用 GPU 任务

## 执行流程

```
Run(task, gpu) → executeTask(task, gpu) → 更新状态 → 释放 GPU
```

### 步骤

1. **延迟释放 GPU**: 确保即使执行失败也释放 GPU
2. **执行任务**: 根据类型调用适当的任务处理器
3. **测量持续时间**: 跟踪执行时间
4. **更新状态**: 根据结果设置任务状态
5. **记录结果**: 记录成功或失败

## 任务处理器

### runEmbeddingTask

执行嵌入任务，模拟 20ms 延迟。

```go
func (r *Runner) runEmbeddingTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runLLMTask

执行 LLM 推理任务，模拟 100ms 延迟。

```go
func (r *Runner) runLLMTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runDiffusionTask

执行扩散模型任务，模拟 500ms 延迟。

```go
func (r *Runner) runDiffusionTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

### runGenericTask

执行通用 GPU 任务，模拟 50ms 延迟。

```go
func (r *Runner) runGenericTask(ctx context.Context, task *types.Task, gpu *gpu.GPU) error
```

## 错误处理

执行器使用 Go 标准的错误处理:

- 错误记录任务 ID 和 GPU ID
- 出错时任务状态设置为 FAILED
- 无论成功或失败都释放 GPU
- 错误传播用于监控

**示例:**
```go
if err != nil {
    log.Printf("Task %s failed on GPU %d: %v", task.ID, gpu.ID, err)
    task.Status = api.TaskStatus_TASK_STATUS_FAILED
    task.Message = err.Error()
}
```

## 超时处理

每种任务类型都有 30 分钟超时:

```go
ctx, cancel := context.WithTimeout(r.ctx, 30*time.Minute)
defer cancel()
```

如果任务超过超时时间:
- Context 被取消
- 任务因上下文错误失败
- GPU 被释放

## 资源管理

### 自动清理

执行器使用 defer 确保正确清理:

```go
defer func() {
    if err := r.gpuPool.Release(gpu); err != nil {
        log.Printf("Failed to release GPU %d: %v", gpu.ID, err)
    }
}()
```

这保证了:
- 即使 panic 也释放 GPU
- 即使出错也释放 GPU
- 无资源泄漏

## 使用示例

```go
executor := executor.NewRunner(gpuPool)

task := &types.Task{
    ID: "task-1",
    Type: api.TaskType_TASK_TYPE_LLM,
}

gpu, _ := gpuPool.Allocate(8192)

// 异步执行
go executor.Run(task, gpu)
```

## 性能

- **并发**: 多个任务可以并行运行
- **内存**: 每个任务最小化分配
- **清理**: 保证资源释放

## 最佳实践

1. 始终使用 `go` 关键字进行异步执行
2. 处理 GPU 释放的错误
3. 监控任务执行时间
4. 为不同的任务类型使用适当的超时
5. 记录失败以进行调试

## 测试

运行执行器测试:
```bash
go test ./internal/executor/...
```

## 注意事项

- 当前实现使用模拟延迟
- 生产环境应集成实际的 GPU 运行时
- 考虑为瞬时故障添加重试逻辑
- 添加任务执行时间的指标