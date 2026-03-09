package scheduler

import (
	"sort"

	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

// GPUPackingStrategy implements GPU packing with Best Fit + Load Threshold
type GPUPackingStrategy struct {
	gpuPool       *gpu.Pool
	loadThreshold float64
	memoryWeight  float64
	computeWeight float64
}

// NewGPUPackingStrategy creates a new GPUPackingStrategy
func NewGPUPackingStrategy(gpuPool *gpu.Pool, cfg *Config) *GPUPackingStrategy {
	return &GPUPackingStrategy{
		gpuPool:       gpuPool,
		loadThreshold: cfg.GPULoadThreshold,
		memoryWeight:  cfg.MemoryWeight,
		computeWeight: cfg.ComputeWeight,
	}
}

// FindBestGPU finds the best GPU for a task using Best Fit strategy
func (s *GPUPackingStrategy) FindBestGPU(task *types.Task) (*gpu.GPU, error) {
	gpus := s.gpuPool.GetAllGPUs()

	if len(gpus) == 0 {
		return nil, ErrNoAvailableGPU
	}

	// Filter candidates: can fit + load below threshold
	candidates := make([]*gpu.GPU, 0)
	for _, g := range gpus {
		if g.CanFit(task.GPUMemoryRequired) && g.GetLoad(s.memoryWeight, s.computeWeight) < s.loadThreshold {
			candidates = append(candidates, g)
		}
	}

	if len(candidates) == 0 {
		// No candidate below threshold, try to find any GPU that can fit
		for _, g := range gpus {
			if g.CanFit(task.GPUMemoryRequired) {
				candidates = append(candidates, g)
			}
		}

		if len(candidates) == 0 {
			return nil, ErrNoAvailableGPU
		}
	}

	// Best Fit: choose GPU with minimum remaining memory after allocation
	return s.bestFit(candidates, task.GPUMemoryRequired), nil
}

// bestFit finds the GPU with the smallest remaining memory that can fit the task
func (s *GPUPackingStrategy) bestFit(candidates []*gpu.GPU, memoryRequired int64) *gpu.GPU {
	// Sort by free memory (ascending) - Best Fit
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].GetMemoryFree() < candidates[j].GetMemoryFree()
	})

	// Return the first one (minimum fit)
	return candidates[0]
}

// GetGPULoad returns the load of a specific GPU
func (s *GPUPackingStrategy) GetGPULoad(gpuID int) float64 {
	g, ok := s.gpuPool.GetGPU(gpuID)
	if !ok {
		return 0
	}
	return g.GetLoad(s.memoryWeight, s.computeWeight)
}

// GetAllGPULoads returns loads of all GPUs
func (s *GPUPackingStrategy) GetAllGPULoads() map[int]float64 {
	gpus := s.gpuPool.GetAllGPUs()
	loads := make(map[int]float64)

	for _, g := range gpus {
		loads[g.ID] = g.GetLoad(s.memoryWeight, s.computeWeight)
	}

	return loads
}

// IsGPUAvailable checks if any GPU can fit the task
func (s *GPUPackingStrategy) IsGPUAvailable(memoryRequired int64) bool {
	gpus := s.gpuPool.GetAllGPUs()

	for _, g := range gpus {
		if g.CanFit(memoryRequired) {
			return true
		}
	}

	return false
}

// GetAvailableGPUCount returns the number of GPUs that can fit the task
func (s *GPUPackingStrategy) GetAvailableGPUCount(memoryRequired int64) int {
	gpus := s.gpuPool.GetAllGPUs()
	count := 0

	for _, g := range gpus {
		if g.CanFit(memoryRequired) {
			count++
		}
	}

	return count
}
