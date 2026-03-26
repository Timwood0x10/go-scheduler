# GPU 模块

## 概述

GPU 模块管理 GPU 资源，使用简单、确定的分配策略。它提供预留机制以防止分配竞态，并实现 Best Fit 分配算法。

## 组件

### GPU

表示单个 GPU 及其状态和资源。

**字段:**
- `ID` - GPU 标识符
- `Name` - GPU 名称
- `MemoryTotal` - 总内存 (MB)
- `MemoryUsed` - 已用内存 (MB)
- `ComputeUtil` - 计算利用率 (0-100%)
- `MemoryUtil` - 内存利用率 (0-100%)
- `Temperature` - 温度 (摄氏度)
- `ReservedBy` - 预留此 GPU 的任务 ID
- `RunningTasks` - 运行中的任务 ID 列表

**关键方法:**
- `CanFit(memoryRequired)` - 检查任务是否能容纳
- `Allocate(taskID, memory)` - 为任务分配内存
- `Reserve(taskID)` - 为任务预留 GPU
- `Free(taskID, memory)` - 释放任务的内存
- `GetMemoryFree()` - 获取空闲内存
- `GetMemoryUsed()` - 获取已用内存
- `GetLoad(memoryWeight, computeWeight)` - 获取负载分数 (0-1)
- `IsReserved()` - 检查 GPU 是否被预留

### Pool

管理 GPU 池，提供线程安全操作。

**关键方法:**
- `NewPool()` - 创建新的 GPU 池
- `AddGPU(id, name, memoryTotal)` - 向池中添加 GPU
- `GetGPU(id)` - 按 ID 获取 GPU
- `GetAllGPUs()` - 获取所有 GPU
- `Allocate(memoryRequired)` - 查找并分配最佳 GPU
- `Release(gpu)` - 释放 GPU 预留
- `UpdateMetrics(gpuID, memoryUtil, computeUtil, temperature)` - 更新 GPU 指标

## 分配策略

### Best Fit 算法

池使用 Best Fit 策略分配 GPU:

1. 筛选能容纳任务的 GPU
2. 筛选未被预留的 GPU
3. 选择分配后剩余内存最少的 GPU

**优势:**
- 最大化 GPU 利用率
- 最小化内存碎片
- 简单且确定

### 预留机制

防止并发场景中的分配竞态:

```go
gpu.Allocate(taskID, memory)  // 分配资源
gpu.Reserve(taskID)           // 防止竞态条件
... 任务执行 ...
pool.Release(gpu)             // 清理
```

## 线程安全

所有操作都是线程安全的:

- `Pool` 使用 `sync.RWMutex` 进行线程安全操作
- `GPU` 使用 `sync.RWMutex` 保持状态一致性
- `canFitUnsafe()` 方法用于关键路径中的无锁检查

## 内存管理

### 分配

```go
gpu, err := pool.Allocate(memoryRequired)
if err != nil || gpu == nil {
    // 没有可用的 GPU
}
```

### 预留

```go
gpu.Allocate(taskID, memory)
gpu.Reserve(taskID)
```

### 释放

```go
pool.Release(gpu)  // 自动释放资源
```

## 指标

可以定期更新 GPU 指标:

```go
pool.UpdateMetrics(gpuID, memoryUtil, computeUtil, temperature)
```

## 错误处理

模块定义了特定的错误:

- `ErrInsufficientMemory` - GPU 内存不足
- `ErrGPUNotFound` - 池中找不到 GPU

## 最佳实践

1. 任务完成后始终调用 `Release()`
2. 优雅处理分配错误
3. 使用预留机制进行并发访问
4. 监控 GPU 指标以进行负载均衡
5. 定期更新指标以获得准确的调度

## 测试

运行 GPU 测试:
```bash
go test ./internal/gpu/...
```

## 性能

- **分配**: O(n)，其中 n 是 GPU 数量
- **释放**: O(1)
- **内存**: 最小化分配
- **并发**: 关键路径中的无锁读取