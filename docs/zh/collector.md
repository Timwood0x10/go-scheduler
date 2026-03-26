# 收集器模块

## 概述

收集器模块定期自动收集 GPU 指标。它提供实时监控 GPU 利用率、内存使用和温度，以支持智能调度决策。

## 组件

### Collector

自动收集 GPU 指标的收集器，定期查询 GPU 状态。

**关键特性:**
- 自动指标收集
- nvidia-smi 集成
- nvidia-smi 不可用时回退到虚拟指标
- 可配置的收集间隔
- 线程安全更新

**关键方法:**
- `NewCollector(pool, interval)` - 创建新收集器
- `Start()` - 启动指标收集
- `Stop()` - 停止指标收集
- `GetGPUMetricsJSON()` - 获取 JSON 格式指标

## 收集的指标

收集器为每个 GPU 收集以下指标:

| 指标 | 描述 | 单位 |
|------|------|------|
| ID | GPU 标识符 | 整数 |
| MemoryUsed | 已用内存 | MB |
| MemoryTotal | 总内存 | MB |
| ComputeUtil | 计算利用率 | % |
| MemoryUtil | 内存利用率 | % |
| Temperature | GPU 温度 | 摄氏度 |

## 收集方法

### nvidia-smi（首选）

当 nvidia-smi 可用时，收集器使用它获取真实指标:

```bash
nvidia-smi --query-gpu=index,memory.used,memory.total,utilization.gpu,utilization.memory,temperature.gpu \
  --format=csv,noheader,nounits
```

**优势:**
- 实时准确指标
- 标准 NVIDIA 工具
- 低开销

### 虚拟指标（回退）

当 nvidia-smi 不可用时（例如没有 GPU 的开发环境），收集器提供虚拟指标:

```go
{
  ComputeUtil: 50,  // 50% 利用率
  MemoryUtil:  50,  // 50% 利用率
  Temperature: 60,  // 60 摄氏度
}
```

**优势:**
- 支持无 GPU 开发
- 一致的 API
- 易于测试

## 使用示例

### 创建并启动收集器

```go
collector := gpu.NewCollector(gpuPool, 5*time.Second)
collector.Start()
```

### 停止收集器

```go
collector.Stop()
```

### 获取 JSON 格式指标

```go
metricsJSON, err := collector.GetGPUMetricsJSON()
if err != nil {
    // 处理错误
}

fmt.Println(metricsJSON)
// 输出: [{"gpu_id":0,"memory_total_mb":16384,"memory_used_mb":2048,...}]
```

## 集成

### 独立服务模式

收集器在 gRPC 服务器中自动启动:

```go
server := server.NewServer()
// 收集器自动启动，间隔 5 秒
```

### 插件模式

收集器在插件模式中显式启动:

```go
collector := gpu.NewCollector(gpuPool, 5*time.Second)
collector.Start()
defer collector.Stop()
```

## 收集间隔

收集间隔决定指标更新的频率:

```go
// 常用间隔
5 * time.Second   // 高频（推荐用于生产环境）
10 * time.Second  // 正常频率
30 * time.Second  // 低频（减少开销）
```

**权衡:**
- 更短间隔：更准确，更高开销
- 更长间隔：不太准确，更低开销

**建议:** 大多数用例使用 5 秒

## 线程安全

收集器是线程安全的:

- 更新由 `sync.RWMutex` 保护
- 多个读取者可以同时访问指标
- 更新不会阻塞读取

## 错误处理

收集器优雅地处理错误:

```go
func (c *Collector) collect() {
    metrics, err := c.queryNvidiaSMI()
    if err != nil {
        log.Printf("Failed to collect GPU metrics: %v", err)
        return  // 跳过此收集周期
    }
    c.updateGPUPool(metrics)
}
```

**错误场景:**
- 未找到 nvidia-smi → 使用虚拟指标
- nvidia-smi 失败 → 跳过收集周期
- 未找到 GPU → 跳过该 GPU

## 性能

- **内存**: 最小（仅存储指标）
- **CPU**: 低（每 5 秒执行 nvidia-smi）
- **网络**: 无（仅本地）
- **I/O**: 低（单次命令执行）

## 最佳实践

1. 完成后始终停止收集器
2. 为您的用例使用适当的间隔
3. 优雅处理 nvidia-smi 故障
4. 监控收集日志中的错误
5. 在开发中使用虚拟指标进行测试

## 测试

运行收集器测试:
```bash
go test ./internal/gpu/...
```

## 监控

收集器记录重要事件:

```log
GPU metrics collector started with interval 5s
Failed to collect GPU metrics: <error>
GPU metrics collector stopped
```

## 故障排除

### 指标未更新

**问题:** GPU 指标保持静态

**解决方案:**
1. 检查收集器是否已启动
2. 验证 nvidia-smi 是否可用
3. 检查日志中的收集错误
4. 验证收集间隔

### CPU 使用率高

**问题:** 收集器使用过多 CPU

**解决方案:**
1. 增加收集间隔
2. 检查 nvidia-smi 是否缓慢
3. 监控收集器频率

### 未找到 nvidia-smi

**问题:** nvidia-smi 命令不可用

**解决方案:**
1. 安装 NVIDIA 驱动
2. 在开发中使用虚拟指标
3. 验证 PATH 包含 nvidia-smi

## 注意事项

- 当前实现使用 nvidia-smi
- 未来版本可能支持其他 GPU 厂商（AMD、Intel）
- 指标在 GPU 对象中就地更新
- 不存储历史数据（仅当前状态）