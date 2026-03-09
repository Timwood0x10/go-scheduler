package gpu

import (
	"testing"
)

func TestPool_AddGPU(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "NVIDIA GPU 0", 16384)

	gpu, ok := pool.GetGPU(0)
	if !ok {
		t.Error("GPU should exist")
	}

	if gpu.Name != "NVIDIA GPU 0" {
		t.Errorf("Expected name NVIDIA GPU 0, got %s", gpu.Name)
	}

	if gpu.MemoryTotal != 16384 {
		t.Errorf("Expected memory 16384, got %d", gpu.MemoryTotal)
	}
}

func TestPool_GetAllGPUs(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU 0", 16000)
	pool.AddGPU(1, "GPU 1", 32000)

	gpus := pool.GetAllGPUs()

	if len(gpus) != 2 {
		t.Errorf("Expected 2 GPUs, got %d", len(gpus))
	}
}

func TestGPU_Allocate(t *testing.T) {
	pool := NewPool()
	pool.AddGPU(0, "GPU 0", 16000)

	gpu, _ := pool.GetGPU(0)

	err := gpu.Allocate("task-1", 8000)
	if err != nil {
		t.Errorf("Allocation should succeed: %v", err)
	}

	if gpu.GetMemoryUsed() != 8000 {
		t.Errorf("Expected 8000 used memory, got %d", gpu.GetMemoryUsed())
	}
}

func TestGPU_Allocate_InsufficientMemory(t *testing.T) {
	pool := NewPool()
	pool.AddGPU(0, "GPU 0", 8000)

	gpu, _ := pool.GetGPU(0)

	err := gpu.Allocate("task-1", 16000)
	if err != ErrInsufficientMemory {
		t.Errorf("Expected ErrInsufficientMemory, got %v", err)
	}
}

func TestGPU_CanFit(t *testing.T) {
	pool := NewPool()
	pool.AddGPU(0, "GPU 0", 16000)

	gpu, _ := pool.GetGPU(0)

	// Should fit
	if !gpu.CanFit(8000) {
		t.Error("Task should fit")
	}

	// Should not fit
	if gpu.CanFit(32000) {
		t.Error("Task should not fit")
	}
}

func TestGPU_Free(t *testing.T) {
	pool := NewPool()
	pool.AddGPU(0, "GPU 0", 16000)

	gpu, _ := pool.GetGPU(0)

	_ = gpu.Allocate("task-1", 8000)
	gpu.Free("task-1", 8000)

	if gpu.GetMemoryUsed() != 0 {
		t.Errorf("Expected 0 used memory, got %d", gpu.GetMemoryUsed())
	}
}

func TestGPU_GetLoad(t *testing.T) {
	pool := NewPool()
	pool.AddGPU(0, "GPU 0", 16000)

	gpu, _ := pool.GetGPU(0)

	// Set utilization directly (simulating metrics)
	gpu.mu.Lock()
	gpu.MemoryUtil = 70
	gpu.ComputeUtil = 30
	gpu.mu.Unlock()

	load := gpu.GetLoad(0.7, 0.3)

	// 0.7 * 0.7 + 0.3 * 0.3 = 0.49 + 0.09 = 0.58
	expected := 0.7*0.7 + 0.3*0.3
	if load < expected-0.01 || load > expected+0.01 {
		t.Errorf("Expected load %f, got %f", expected, load)
	}
}
