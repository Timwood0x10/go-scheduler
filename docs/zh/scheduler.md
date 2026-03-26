# 调度器模块

## 概述

调度器模块是 AlgoGPU 的核心，实现了一个简单、确定性的 GPU 任务调度循环。它遵循"大道至简"的原则。

## 架构

```
任务队列 (堆) → 调度循环 → GPU 池 (预留) → 执行器 (异步)
```

## 组件

### Scheduler

主调度器，使用基于 channel 的循环。

**关键特性:**
- channel 驱动，无忙等待轮询
- 确定性的调度行为
- 支持优雅关闭

**核心方法:**
- `NewScheduler()` - 创建新的调度器实例
- `SubmitTask()` - 提交任务，进行准入控制和速率限制检查
- `Loop()` - 核心调度循环
- `Start()` - 启动调度器
- `Stop()` - 优雅停止调度器

### AdmissionControl

通过检查队列容量来保护系统免受请求洪水的冲击。

**方法:**
- `NewAdmissionControl()` - 创建准入控制器
- `Check()` - 检查队列是否有容量

### TokenBucketManager

实现用户级别的速率限制，支持每日配额。

**特性:**
- 每个用户一个令牌桶
- 每日使用量追踪
- 自动令牌补充

**方法:**
- `CheckAndConsume()` - 检查并消耗任务的令牌
- `Allow()` - 检查用户是否有可用令牌
- `GetTokenBalance()` - 获取当前令牌余额

### GPUPackingStrategy

实现 Best Fit GPU 分配策略。

**特性:**
- Best Fit 算法选择 GPU
- 考虑负载阈值
- 最大化 GPU 利用率

**方法:**
- `FindBestGPU()` - 为任务找到最佳 GPU
- `IsGPUAvailable()` - 检查是否有 GPU 可用

### TaskAging

通过随时间调整优先级来防止任务饥饿。

**公式:**
```
priority = base_priority + wait_time * γ
```

**方法:**
- `CalculateAgingPriority()` - 计算带老化的优先级
- `GetWaitTime()` - 获取任务等待时间

## 调度流程

1. **任务提交**
   - 准入控制检查
   - 令牌桶检查
   - 入队到优先级队列

2. **调度循环**
   - 从 channel 获取任务
   - 检查令牌桶进行速率限制
   - 使用打包策略找到最佳 GPU
   - 预留并分配 GPU
   - 异步执行任务

3. **任务执行**
   - 执行器在 GPU 上运行任务
   - 完成后释放 GPU
   - 更新任务状态

## 配置

```go
type Config struct {
    MaxQueueSize       int     // 最大队列大小
    TokenRefillRate    int64   // 每秒令牌数
    TokenBucketSize    int64   // 每用户最大令牌数
    DailyTokenLimit    int64   // 每日令牌限制
    GPULoadThreshold   float64 // 最大 GPU 负载 (0-1)
    AgingFactor        float64 // 任务老化因子
    UsageWindowMinutes int     // 使用追踪窗口
}
```

## 错误处理

调度器使用 Go 标准的错误处理模式:

```go
if err := scheduler.SubmitTask(task); err != nil {
    // 处理错误
}
```

## 并发安全

所有共享状态都由互斥锁保护:
- `sync.RWMutex` 用于读密集型操作
- `sync.Mutex` 用于写操作
- 无竞态条件

## 性能

- **内存**: 热路径中最小化分配
- **CPU**: channel 驱动，无忙等待
- **可扩展性**: 已测试 1000+ 并发任务

## 测试

运行测试:
```bash
make test
```

使用竞态检测器运行:
```bash
make test-race
```

## 最佳实践

1. 程序退出前始终调用 `Stop()`
2. 处理 `SubmitTask()` 返回的错误
3. 监控队列大小以进行负载均衡
4. 为每个用户设置适当的令牌限制