package gpu

import (
	"sync"
	"time"

	"algogpu/api"
)

// GPU represents a single GPU
type GPU struct {
	ID           int
	Name         string
	MemoryTotal  int64 // MB
	MemoryUsed   int64 // MB
	ComputeUtil  int   // percentage 0-100
	MemoryUtil   int   // percentage 0-100
	Temperature  int   // celsius
	RunningTasks []string
	LastUpdated  time.Time
	mu           sync.RWMutex
}

// Pool manages a pool of GPUs
type Pool struct {
	mu   sync.RWMutex
	GPUs map[int]*GPU
}

// NewPool creates a new Pool
func NewPool() *Pool {
	return &Pool{
		GPUs: make(map[int]*GPU),
	}
}

// AddGPU adds a GPU to the pool
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
		RunningTasks: []string{},
		LastUpdated:  time.Now(),
	}
}

// GetGPU returns a GPU by ID
func (p *Pool) GetGPU(id int) (*GPU, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gpu, ok := p.GPUs[id]
	return gpu, ok
}

// GetAllGPUs returns all GPUs
func (p *Pool) GetAllGPUs() []*GPU {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gpus := make([]*GPU, 0, len(p.GPUs))
	for _, gpu := range p.GPUs {
		gpus = append(gpus, gpu)
	}

	return gpus
}

// CanFit checks if a task can fit on a GPU
func (g *GPU) CanFit(memoryRequired int64) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.MemoryTotal-g.MemoryUsed >= memoryRequired
}

// Allocate allocates memory for a task
func (g *GPU) Allocate(taskID string, memory int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.MemoryTotal-g.MemoryUsed < memory {
		return ErrInsufficientMemory
	}

	g.MemoryUsed += memory
	g.RunningTasks = append(g.RunningTasks, taskID)
	g.LastUpdated = time.Now()

	return nil
}

// Free frees memory for a task
func (g *GPU) Free(taskID string, memory int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.MemoryUsed -= memory
	if g.MemoryUsed < 0 {
		g.MemoryUsed = 0
	}

	// Remove task from running tasks
	newTasks := make([]string, 0)
	for _, t := range g.RunningTasks {
		if t != taskID {
			newTasks = append(newTasks, t)
		}
	}
	g.RunningTasks = newTasks
	g.LastUpdated = time.Now()
}

// GetMemoryFree returns free memory
func (g *GPU) GetMemoryFree() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.MemoryTotal - g.MemoryUsed
}

// GetMemoryUsed returns used memory
func (g *GPU) GetMemoryUsed() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.MemoryUsed
}

// GetLoad returns GPU load score (0-1)
func (g *GPU) GetLoad(memoryWeight, computeWeight float64) float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	memoryLoad := float64(g.MemoryUtil) / 100.0
	computeLoad := float64(g.ComputeUtil) / 100.0

	return memoryWeight*memoryLoad + computeWeight*computeLoad
}

// ToProto converts GPU to protobuf message
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

// Errors
var (
	ErrInsufficientMemory = &Error{"insufficient GPU memory"}
	ErrGPUNotFound        = &Error{"GPU not found"}
)

// Error represents a GPU-related error
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
