# 队列模块

## 概述

队列模块使用 Go 的 `container/heap` 包实现基于优先级的任务队列。它提供 O(log n) 的入队和出队操作，支持任务老化和并发访问。

## 组件

### TaskQueue

GPU 任务的线程安全优先级队列。

**关键特性:**
- 基于优先级的排序
- 支持老化（等待时间优先级）
- 线程安全操作
- 高效的 O(log n) 操作
- 任务状态跟踪

**关键方法:**
- `NewTaskQueue()` - 创建新任务队列
- `Enqueue(task)` - 将任务添加到队列
- `Dequeue()` - 移除并返回最高优先级任务
- `Get(taskID)` - 按 ID 获取任务
- `Cancel(taskID)` - 取消任务
- `UpdateStatus(taskID, status)` - 更新任务状态
- `Requeue(task)` - 重新排队任务
- `GetAllPending()` - 获取所有待处理任务
- `GetAllRunning()` - 获取所有运行中任务
- `Len()` - 获取队列长度
- `RunningCount()` - 获取运行中任务数

### TaskHeap

使用 `container/heap` 的内部堆实现。

**方法:**
- `Len()` - 获取堆大小
- `Less(i, j)` - 比较任务优先级
- `Swap(i, j)` - 交换堆中的任务
- `Push(x)` - 将任务添加到堆
- `Pop()` - 从堆中移除任务

## 优先级计算

任务按优先级排序，考虑老化因素:

```
priority = base_priority + (wait_time / 1000) * 0.1
```

其中:
- `base_priority`: 任务的优先级（越高越重要）
- `wait_time`: 任务创建以来的时间（毫秒）
- `0.1`: 老化因子

**优先级顺序:**
1. 任务类型 (LLM > Diffusion > Embedding > Other)
2. 用户优先级（如果配置）
3. 等待时间（老化）

## 任务状态

任务通过以下状态转换:

```
PENDING → RUNNING → COMPLETED
    ↓
CANCELLED
    ↓
FAILED
```

## 线程安全

所有操作都由 `sync.RWMutex` 保护:

- `sync.RLock()` 用于读操作（Get, Len 等）
- `sync.Lock()` 用于写操作（Enqueue, Dequeue 等）

## 使用示例

### 入队任务

```go
task := &types.Task{
    ID: "task-1",
    UserID: "user-1",
    Type: api.TaskType_TASK_TYPE_LLM,
    GPUMemoryRequired: 8192,
    Priority: 100,
}

err := queue.Enqueue(task)
if err != nil {
    // 处理错误
}
```

### 出队任务

```go
task := queue.Dequeue()
if task == nil {
    // 队列为空
}
```

### 取消任务

```go
err := queue.Cancel("task-1")
if err != nil {
    // 未找到任务或已取消
}
```

### 更新任务状态

```go
err := queue.UpdateStatus("task-1", api.TaskStatus_TASK_STATUS_RUNNING)
if err != nil {
    // 未找到任务
}
```

## 错误处理

模块定义了特定的错误:

- `ErrQueueFull` - 队列已满
- `ErrTaskExists` - 任务已在队列中
- `ErrTaskNotFound` - 队列中未找到任务

## 性能

| 操作 | 时间复杂度 | 空间复杂度 |
|------|-----------|-----------|
| 入队 | O(log n)  | O(1)      |
| 出队 | O(log n)  | O(1)      |
| 获取 | O(1)      | O(1)      |
| 取消 | O(log n)  | O(1)      |
| 长度 | O(1)      | O(1)      |

## 最佳实践

1. 优雅处理 `ErrQueueFull`
2. 使用 `GetAllPending()` 进行监控
3. 始终为不需要的任务调用 `Cancel()`
4. 随任务进展更新任务状态
5. 使用适当的优先级值

## 测试

运行队列测试:
```bash
go test ./internal/queue/...
```

## 并发

队列专为高并发设计:
- 多个读取者可以同时访问
- 写入者会阻塞其他写入者和读取者
- 监控操作使用无锁读取