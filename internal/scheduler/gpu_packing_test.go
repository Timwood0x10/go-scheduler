package scheduler

import (
	"testing"

	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

func TestGPUPackingStrategy_FindBestGPU(t *testing.T) {
	pool := gpu.NewPool()
	pool.AddGPU(0, "GPU 0", 16000) // 16GB
	pool.AddGPU(1, "GPU 1", 32000) // 32GB

	cfg := DefaultConfig()
	strategy := NewGPUPackingStrategy(pool, cfg)

	task := &types.Task{
		ID:                "task-1",
		GPUMemoryRequired: 8000, // 8GB
	}

	// Should fit on both GPUs, GPU 0 is best fit
	gpuInfo, err := strategy.FindBestGPU(task)
	if err != nil {
		t.Errorf("Failed to find GPU: %v", err)
	}

	// Best fit should be GPU 0 (8GB free vs 24GB free)
	if gpuInfo.ID != 0 {
		t.Errorf("Expected GPU 0 (best fit), got GPU %d", gpuInfo.ID)
	}
}

func TestGPUPackingStrategy_NoAvailableGPU(t *testing.T) {
	pool := gpu.NewPool()
	pool.AddGPU(0, "GPU 0", 8000) // 8GB

	cfg := DefaultConfig()
	strategy := NewGPUPackingStrategy(pool, cfg)

	task := &types.Task{
		ID:                "task-1",
		GPUMemoryRequired: 16000, // 16GB - too much
	}

	_, err := strategy.FindBestGPU(task)
	if err != ErrNoAvailableGPU {
		t.Errorf("Expected ErrNoAvailableGPU, got %v", err)
	}
}

func TestGPUPackingStrategy_IsGPUAvailable(t *testing.T) {
	pool := gpu.NewPool()
	pool.AddGPU(0, "GPU 0", 16000)

	cfg := DefaultConfig()
	strategy := NewGPUPackingStrategy(pool, cfg)

	// Should be available
	if !strategy.IsGPUAvailable(8000) {
		t.Error("GPU should be available for 8GB task")
	}

	// Should not be available
	if strategy.IsGPUAvailable(32000) {
		t.Error("GPU should not be available for 32GB task")
	}
}

func TestGPUPackingStrategy_GetAllGPULoads(t *testing.T) {
	pool := gpu.NewPool()
	pool.AddGPU(0, "GPU 0", 16000)
	pool.AddGPU(1, "GPU 1", 16000)

	cfg := DefaultConfig()
	strategy := NewGPUPackingStrategy(pool, cfg)

	loads := strategy.GetAllGPULoads()

	if len(loads) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(loads))
	}
}
