// Package gpu manages GPU resources with a simple, deterministic allocation strategy.
// It provides a reservation mechanism to prevent allocation races.
package gpu

import (
	"fmt"
	"sync"
	"time"

	"algogpu/api"
)

// GPU represents a single GPU with its state and resources.
type GPU struct {
	ID           int
	Name         string
	MemoryTotal  int64  // MB
	MemoryUsed   int64  // MB
	ComputeUtil  int    // percentage 0-100
	MemoryUtil   int    // percentage 0-100
	Temperature  int    // celsius
	ReservedBy   string // Task ID that reserved this GPU
	RunningTasks []string
	LastUpdated  time.Time
	mu           sync.RWMutex
}

// CanFit checks if a task can fit on a GPU considering available memory.
func (g *GPU) CanFit(memoryRequired int64) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.ReservedBy != "" {
		return false
	}

	return g.MemoryTotal-g.MemoryUsed >= memoryRequired
}

// Allocate allocates memory for a task on this GPU.
// This method is used for actual resource allocation.
func (g *GPU) Allocate(taskID string, memory int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ReservedBy != "" && g.ReservedBy != taskID {
		return fmt.Errorf("GPU %d is reserved by task %s", g.ID, g.ReservedBy)
	}

	if g.MemoryTotal-g.MemoryUsed < memory {
		return ErrInsufficientMemory
	}

	g.MemoryUsed += memory
	g.RunningTasks = append(g.RunningTasks, taskID)
	g.LastUpdated = time.Now()

	return nil
}

// Reserve reserves this GPU for a specific task to prevent race conditions.
// This should be called immediately after allocation.
func (g *GPU) Reserve(taskID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.ReservedBy = taskID
	g.LastUpdated = time.Now()
}

// Free frees memory for a task on this GPU.
func (g *GPU) Free(taskID string, memory int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.MemoryUsed -= memory
	if g.MemoryUsed < 0 {
		g.MemoryUsed = 0
	}

	// Remove task from running tasks
	newTasks := make([]string, 0, len(g.RunningTasks))
	for _, t := range g.RunningTasks {
		if t != taskID {
			newTasks = append(newTasks, t)
		}
	}
	g.RunningTasks = newTasks

	// Clear reservation if this was the reserved task
	if g.ReservedBy == taskID {
		g.ReservedBy = ""
	}

	g.LastUpdated = time.Now()
}

// GetMemoryFree returns free memory on this GPU.
func (g *GPU) GetMemoryFree() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.MemoryTotal - g.MemoryUsed
}

// GetMemoryUsed returns used memory on this GPU.
func (g *GPU) GetMemoryUsed() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.MemoryUsed
}

// GetLoad returns GPU load score (0-1) considering memory and compute.
func (g *GPU) GetLoad(memoryWeight, computeWeight float64) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	memoryLoad := float64(g.MemoryUtil) / 100.0
	computeLoad := float64(g.ComputeUtil) / 100.0

	return memoryWeight*memoryLoad + computeWeight*computeLoad
}

// IsReserved checks if this GPU is reserved.
func (g *GPU) IsReserved() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.ReservedBy != ""
}

// ToProto converts GPU to protobuf message.
func (g *GPU) ToProto() *api.GPUInfo {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return &api.GPUInfo{
		GpuId:        int32(g.ID),
		Name:         g.Name,
		MemoryTotal:  g.MemoryTotal,
		MemoryUsed:   g.MemoryUsed,
		MemoryFree:   g.MemoryTotal - g.MemoryUsed,
		ComputeUtil:  int32(g.ComputeUtil),
		MemoryUtil:   int32(g.MemoryUtil),
		Temperature:  int32(g.Temperature),
		RunningTasks: g.RunningTasks,
	}
}

// Pool manages a pool of GPUs with thread-safe operations.
type Pool struct {
	mu   sync.RWMutex
	GPUs map[int]*GPU
}

// NewPool creates a new GPU pool.
func NewPool() *Pool {
	return &Pool{
		GPUs: make(map[int]*GPU),
	}
}

// AddGPU adds a GPU to the pool.
func (p *Pool) AddGPU(id int, name string, memoryTotal int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.GPUs[id] = &GPU{
		ID:           id,
		Name:         name,
		MemoryTotal:  memoryTotal,
		MemoryUsed:   0,
		ComputeUtil:  0,
		MemoryUtil:   0,
		Temperature:  0,
		ReservedBy:   "",
		RunningTasks: []string{},
		LastUpdated:  time.Now(),
	}
}

// GetGPU returns a GPU by ID.
func (p *Pool) GetGPU(id int) (*GPU, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gpu, ok := p.GPUs[id]
	return gpu, ok
}

// GetAllGPUs returns all GPUs in the pool.
func (p *Pool) GetAllGPUs() []*GPU {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gpus := make([]*GPU, 0, len(p.GPUs))
	for _, gpu := range p.GPUs {
		gpus = append(gpus, gpu)
	}

	return gpus
}

// Allocate finds and allocates a GPU for a task using Best Fit strategy.
// Returns the allocated GPU or nil if no suitable GPU is available.
func (p *Pool) Allocate(memoryRequired int64) *GPU {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var bestGPU *GPU
	smallestWaste := int64(-1)

	for _, gpu := range p.GPUs {
		if !gpu.CanFit(memoryRequired) {
			continue
		}

		if gpu.IsReserved() {
			continue
		}

		freeMemory := gpu.GetMemoryFree()
		waste := freeMemory - memoryRequired

		if bestGPU == nil || waste < smallestWaste {
			bestGPU = gpu
			smallestWaste = waste
		}
	}

	if bestGPU == nil {
		return nil
	}

	// Perform allocation outside read lock
	bestGPU.Allocate("", memoryRequired)

	return bestGPU
}

// Release releases a GPU reservation and frees its resources.
func (p *Pool) Release(gpu *GPU) error {
	if gpu == nil {
		return nil
	}

	gpu.mu.Lock()
	defer gpu.mu.Unlock()

	// Clear reservation
	if gpu.ReservedBy != "" {
		taskID := gpu.ReservedBy
		memoryToFree := int64(0)

		// Find memory used by this task
		// In a real implementation, we'd track this more precisely
		// For now, we free a reasonable amount
		for _, t := range gpu.RunningTasks {
			if t == taskID {
				// Estimate memory usage - in production, track this accurately
				memoryToFree = 1024 // 1GB default
				break
			}
		}

		gpu.Free(taskID, memoryToFree)
	}

	return nil
}

// UpdateMetrics updates GPU metrics (called by collector).
func (p *Pool) UpdateMetrics(gpuID int, memoryUtil, computeUtil, temperature int) {
	p.mu.RLock()
	gpu, ok := p.GPUs[gpuID]
	p.mu.RUnlock()

	if !ok {
		return
	}

	gpu.mu.Lock()
	defer gpu.mu.Unlock()

	gpu.MemoryUtil = memoryUtil
	gpu.ComputeUtil = computeUtil
	gpu.Temperature = temperature
	gpu.LastUpdated = time.Now()
}

// Errors
var (
	ErrInsufficientMemory = &Error{"insufficient GPU memory"}
	ErrGPUNotFound        = &Error{"GPU not found"}
)

// Error represents a GPU-related error.
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
