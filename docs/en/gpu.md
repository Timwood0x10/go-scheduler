# GPU Module

## Overview

The GPU module manages GPU resources with a simple, deterministic allocation strategy. It provides a reservation mechanism to prevent allocation races and implements the Best Fit allocation algorithm.

## Components

### GPU

Represents a single GPU with its state and resources.

**Fields:**
- `ID` - GPU identifier
- `Name` - GPU name
- `MemoryTotal` - Total memory in MB
- `MemoryUsed` - Used memory in MB
- `ComputeUtil` - Compute utilization (0-100%)
- `MemoryUtil` - Memory utilization (0-100%)
- `Temperature` - Temperature in Celsius
- `ReservedBy` - Task ID that reserved this GPU
- `RunningTasks` - List of running task IDs

**Key Methods:**
- `CanFit(memoryRequired)` - Check if task can fit
- `Allocate(taskID, memory)` - Allocate memory for a task
- `Reserve(taskID)` - Reserve GPU for a task
- `Free(taskID, memory)` - Free memory for a task
- `GetMemoryFree()` - Get free memory
- `GetMemoryUsed()` - Get used memory
- `GetLoad(memoryWeight, computeWeight)` - Get load score (0-1)
- `IsReserved()` - Check if GPU is reserved

### Pool

Manages a pool of GPUs with thread-safe operations.

**Key Methods:**
- `NewPool()` - Create new GPU pool
- `AddGPU(id, name, memoryTotal)` - Add GPU to pool
- `GetGPU(id)` - Get GPU by ID
- `GetAllGPUs()` - Get all GPUs
- `Allocate(memoryRequired)` - Find and allocate best GPU
- `Release(gpu)` - Release GPU reservation
- `UpdateMetrics(gpuID, memoryUtil, computeUtil, temperature)` - Update GPU metrics

## Allocation Strategy

### Best Fit Algorithm

The pool uses the Best Fit strategy to allocate GPUs:

1. Filter GPUs that can fit the task
2. Filter GPUs that are not reserved
3. Select GPU with minimum remaining memory after allocation

**Benefits:**
- Maximizes GPU utilization
- Minimizes memory fragmentation
- Simple and deterministic

### Reservation Mechanism

Prevents allocation races in concurrent scenarios:

```go
gpu.Allocate(taskID, memory)  // Allocate resources
gpu.Reserve(taskID)           // Prevent race conditions
... task executes ...
pool.Release(gpu)             // Cleanup
```

## Thread Safety

All operations are thread-safe:

- `Pool` uses `sync.RWMutex` for thread-safe operations
- `GPU` uses `sync.RWMutex` for state consistency
- `canFitUnsafe()` method for lock-free checks in critical paths

## Memory Management

### Allocation

```go
gpu, err := pool.Allocate(memoryRequired)
if err != nil || gpu == nil {
    // No GPU available
}
```

### Reservation

```go
gpu.Allocate(taskID, memory)
gpu.Reserve(taskID)
```

### Release

```go
pool.Release(gpu)  // Automatically frees resources
```

## Metrics

GPU metrics can be updated periodically:

```go
pool.UpdateMetrics(gpuID, memoryUtil, computeUtil, temperature)
```

## Error Handling

The module defines specific errors:

- `ErrInsufficientMemory` - Not enough memory on GPU
- `ErrGPUNotFound` - GPU not found in pool

## Best Practices

1. Always call `Release()` after task completion
2. Handle allocation errors gracefully
3. Use reservation mechanism for concurrent access
4. Monitor GPU metrics for load balancing
5. Update metrics regularly for accurate scheduling

## Testing

Run GPU tests:
```bash
go test ./internal/gpu/...
```

## Performance

- **Allocation**: O(n) where n is number of GPUs
- **Release**: O(1)
- **Memory**: Minimal allocations
- **Concurrency**: Lock-free reads in critical paths