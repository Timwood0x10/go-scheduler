# Collector Module

## Overview

The Collector module automatically collects GPU metrics at regular intervals. It provides real-time monitoring of GPU utilization, memory usage, and temperature to support intelligent scheduling decisions.

## Components

### Collector

Automated GPU metrics collector that periodically queries GPU status.

**Key Features:**
- Automatic metrics collection
- nvidia-smi integration
- Fallback to dummy metrics when nvidia-smi unavailable
- Configurable collection interval
- Thread-safe updates

**Key Methods:**
- `NewCollector(pool, interval)` - Create new collector
- `Start()` - Start metrics collection
- `Stop()` - Stop metrics collection
- `GetGPUMetricsJSON()` - Get metrics as JSON

## Metrics Collected

The collector collects the following metrics for each GPU:

| Metric | Description | Unit |
|--------|-------------|------|
| ID | GPU identifier | integer |
| MemoryUsed | Used memory | MB |
| MemoryTotal | Total memory | MB |
| ComputeUtil | Compute utilization | % |
| MemoryUtil | Memory utilization | % |
| Temperature | GPU temperature | Celsius |

## Collection Methods

### nvidia-smi (Preferred)

When nvidia-smi is available, the collector uses it to get real metrics:

```bash
nvidia-smi --query-gpu=index,memory.used,memory.total,utilization.gpu,utilization.memory,temperature.gpu \
  --format=csv,noheader,nounits
```

**Advantages:**
- Real-time accurate metrics
- Standard NVIDIA tool
- Low overhead

### Dummy Metrics (Fallback)

When nvidia-smi is not available (e.g., development without GPUs), the collector provides dummy metrics:

```go
{
  ComputeUtil: 50,  // 50% utilization
  MemoryUtil:  50,  // 50% utilization
  Temperature: 60,  // 60 degrees Celsius
}
```

**Advantages:**
- Enables development without GPUs
- Consistent API
- Easy testing

## Usage Examples

### Create and Start Collector

```go
collector := gpu.NewCollector(gpuPool, 5*time.Second)
collector.Start()
```

### Stop Collector

```go
collector.Stop()
```

### Get Metrics as JSON

```go
metricsJSON, err := collector.GetGPUMetricsJSON()
if err != nil {
    // Handle error
}

fmt.Println(metricsJSON)
// Output: [{"gpu_id":0,"memory_total_mb":16384,"memory_used_mb":2048,...}]
```

## Integration

### Standalone Mode

The collector is automatically started in the gRPC server:

```go
server := server.NewServer()
// Collector started automatically with 5s interval
```

### Plugin Mode

The collector is explicitly started in plugin mode:

```go
collector := gpu.NewCollector(gpuPool, 5*time.Second)
collector.Start()
defer collector.Stop()
```

## Collection Interval

The collection interval determines how often metrics are updated:

```go
// Common intervals
5 * time.Second   // High frequency (recommended for production)
10 * time.Second  // Normal frequency
30 * time.Second  // Low frequency (reduced overhead)
```

**Trade-offs:**
- Shorter interval: More accurate, higher overhead
- Longer interval: Less accurate, lower overhead

**Recommendation:** 5 seconds for most use cases

## Thread Safety

The collector is thread-safe:

- Updates are protected by `sync.RWMutex`
- Multiple readers can access metrics simultaneously
- Updates do not block reads

## Error Handling

The collector handles errors gracefully:

```go
func (c *Collector) collect() {
    metrics, err := c.queryNvidiaSMI()
    if err != nil {
        log.Printf("Failed to collect GPU metrics: %v", err)
        return  // Skip this collection cycle
    }
    c.updateGPUPool(metrics)
}
```

**Error Scenarios:**
- nvidia-smi not found → Use dummy metrics
- nvidia-smi fails → Skip collection cycle
- GPU not found → Skip that GPU

## Performance

- **Memory**: Minimal (only stores metrics)
- **CPU**: Low (nvidia-smi every 5 seconds)
- **Network**: None (local only)
- **I/O**: Low (single command execution)

## Best Practices

1. Always stop collector when done
2. Use appropriate interval for your use case
3. Handle nvidia-smi failures gracefully
4. Monitor collection logs for errors
5. Test with dummy metrics in development

## Testing

Run collector tests:
```bash
go test ./internal/gpu/...
```

## Monitoring

The collector logs important events:

```log
GPU metrics collector started with interval 5s
Failed to collect GPU metrics: <error>
GPU metrics collector stopped
```

## Troubleshooting

### Metrics Not Updating

**Problem:** GPU metrics remain static

**Solutions:**
1. Check if collector is started
2. Verify nvidia-smi is available
3. Check logs for collection errors
4. Verify collection interval

### High CPU Usage

**Problem:** Collector using too much CPU

**Solutions:**
1. Increase collection interval
2. Check if nvidia-smi is slow
3. Monitor collector frequency

### nvidia-smi Not Found

**Problem:** nvidia-smi command not available

**Solutions:**
1. Install NVIDIA drivers
2. Use dummy metrics for development
3. Verify PATH includes nvidia-smi

## Notes

- Current implementation uses nvidia-smi
- Future versions may support other GPU vendors (AMD, Intel)
- Metrics are updated in-place in GPU objects
- No historical data is stored (only current state)