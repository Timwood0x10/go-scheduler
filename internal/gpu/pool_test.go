package gpu

import (
	"testing"
)

func TestNewPool(t *testing.T) {
	pool := NewPool()

	if pool == nil {
		t.Fatal("NewPool() returned nil")
	}

	if pool.GPUs == nil {
		t.Error("NewPool() did not initialize GPUs map")
	}
}

func TestPool_AddGPU(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)
	pool.AddGPU(1, "GPU-1", 16384)

	gpus := pool.GetAllGPUs()

	if len(gpus) != 2 {
		t.Errorf("AddGPU() added %d GPUs, want 2", len(gpus))
	}

	if gpus[0].ID != 0 {
		t.Errorf("GPU 0 ID = %d, want 0", gpus[0].ID)
	}

	if gpus[0].Name != "GPU-0" {
		t.Errorf("GPU 0 name = %s, want GPU-0", gpus[0].Name)
	}

	if gpus[0].MemoryTotal != 8192 {
		t.Errorf("GPU 0 memory = %d, want 8192", gpus[0].MemoryTotal)
	}
}

func TestPool_GetGPU(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)

	gpu, ok := pool.GetGPU(0)

	if !ok {
		t.Error("GetGPU() should find GPU 0")
	}

	if gpu == nil {
		t.Fatal("GetGPU() returned nil")
	}

	if gpu.ID != 0 {
		t.Errorf("GPU ID = %d, want 0", gpu.ID)
	}

	// Try to get non-existent GPU
	_, ok = pool.GetGPU(999)

	if ok {
		t.Error("GetGPU() should not find GPU 999")
	}
}

func TestPool_GetAllGPUs(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)
	pool.AddGPU(1, "GPU-1", 16384)
	pool.AddGPU(2, "GPU-2", 24576)

	gpus := pool.GetAllGPUs()

	if len(gpus) != 3 {
		t.Errorf("GetAllGPUs() returned %d GPUs, want 3", len(gpus))
	}
}

func TestGPU_CanFit(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  2048,
		ReservedBy:  "",
	}

	// Should fit
	if !gpu.CanFit(4096) {
		t.Error("CanFit() should return true for 4096 MB")
	}

	// Should not fit
	if gpu.CanFit(7000) {
		t.Error("CanFit() should return false for 7000 MB")
	}

	// Reserved GPU should not fit
	gpu.ReservedBy = "task-1"
	if gpu.CanFit(1024) {
		t.Error("CanFit() should return false for reserved GPU")
	}
}

func TestGPU_Allocate(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  0,
		ReservedBy:  "",
	}

	err := gpu.Allocate("task-1", 2048)

	if err != nil {
		t.Errorf("Allocate() error = %v", err)
	}

	if gpu.MemoryUsed != 2048 {
		t.Errorf("Memory used = %d, want 2048", gpu.MemoryUsed)
	}

	if len(gpu.RunningTasks) != 1 {
		t.Errorf("Running tasks = %d, want 1", len(gpu.RunningTasks))
	}

	if gpu.RunningTasks[0] != "task-1" {
		t.Errorf("Running task = %s, want task-1", gpu.RunningTasks[0])
	}

	// Try to allocate more than available
	err = gpu.Allocate("task-2", 10000)

	if err != ErrInsufficientMemory {
		t.Errorf("Allocate() error = %v, want ErrInsufficientMemory", err)
	}

	// Try to allocate on reserved GPU
	gpu.ReservedBy = "task-1"
	err = gpu.Allocate("task-2", 1024)

	if err == nil {
		t.Error("Allocate() should fail on reserved GPU")
	}
}

func TestGPU_Reserve(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  0,
		ReservedBy:  "",
	}

	gpu.Reserve("task-1")

	if gpu.ReservedBy != "task-1" {
		t.Errorf("ReservedBy = %s, want task-1", gpu.ReservedBy)
	}

	// Update reservation
	gpu.Reserve("task-2")

	if gpu.ReservedBy != "task-2" {
		t.Errorf("ReservedBy = %s, want task-2", gpu.ReservedBy)
	}
}

func TestGPU_Free(t *testing.T) {
	gpu := &GPU{
		ID:           0,
		MemoryTotal:  8192,
		MemoryUsed:   4096,
		ReservedBy:   "task-1",
		RunningTasks: []string{"task-1", "task-2"},
	}

	gpu.Free("task-1", 2048)

	if gpu.MemoryUsed != 2048 {
		t.Errorf("Memory used = %d, want 2048", gpu.MemoryUsed)
	}

	if gpu.ReservedBy != "" {
		t.Errorf("ReservedBy = %s, want empty", gpu.ReservedBy)
	}

	if len(gpu.RunningTasks) != 1 {
		t.Errorf("Running tasks = %d, want 1", len(gpu.RunningTasks))
	}

	if gpu.RunningTasks[0] != "task-2" {
		t.Errorf("Running task = %s, want task-2", gpu.RunningTasks[0])
	}
}

func TestGPU_GetMemoryFree(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  2048,
	}

	free := gpu.GetMemoryFree()

	if free != 6144 {
		t.Errorf("GetMemoryFree() = %d, want 6144", free)
	}
}

func TestGPU_GetMemoryUsed(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  2048,
	}

	used := gpu.GetMemoryUsed()

	if used != 2048 {
		t.Errorf("GetMemoryUsed() = %d, want 2048", used)
	}
}

func TestGPU_GetLoad(t *testing.T) {
	gpu := &GPU{
		ID:          0,
		MemoryTotal: 8192,
		MemoryUsed:  4096,
		MemoryUtil:  50,
		ComputeUtil: 75,
	}

	load := gpu.GetLoad(0.5, 0.5)

	expected := 0.5*0.5 + 0.5*0.75 // 0.625

	if load != expected {
		t.Errorf("GetLoad() = %f, want %f", load, expected)
	}
}

func TestGPU_IsReserved(t *testing.T) {
	gpu := &GPU{
		ID:         0,
		ReservedBy: "",
	}

	if gpu.IsReserved() {
		t.Error("IsReserved() should return false initially")
	}

	gpu.ReservedBy = "task-1"

	if !gpu.IsReserved() {
		t.Error("IsReserved() should return true after reservation")
	}
}

func TestGPU_ToProto(t *testing.T) {
	gpu := &GPU{
		ID:           0,
		Name:         "GPU-0",
		MemoryTotal:  8192,
		MemoryUsed:   2048,
		MemoryUtil:   25,
		ComputeUtil:  50,
		Temperature:  65,
		RunningTasks: []string{"task-1"},
	}

	proto := gpu.ToProto()

	if proto.GpuId != 0 {
		t.Errorf("Proto GPU ID = %d, want 0", proto.GpuId)
	}

	if proto.Name != "GPU-0" {
		t.Errorf("Proto name = %s, want GPU-0", proto.Name)
	}

	if proto.MemoryTotal != 8192 {
		t.Errorf("Proto memory total = %d, want 8192", proto.MemoryTotal)
	}

	if proto.MemoryUsed != 2048 {
		t.Errorf("Proto memory used = %d, want 2048", proto.MemoryUsed)
	}

	if proto.MemoryFree != 6144 {
		t.Errorf("Proto memory free = %d, want 6144", proto.MemoryFree)
	}
}

func TestPool_Allocate(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)
	pool.AddGPU(1, "GPU-1", 16384)

	// Allocate 4096 MB - should use GPU 0 (best fit)
	gpu := pool.Allocate(4096)

	if gpu == nil {
		t.Fatal("Allocate() returned nil")
	}

	if gpu.ID != 0 {
		t.Errorf("Allocated GPU ID = %d, want 0", gpu.ID)
	}

	// Allocate 12000 MB - should use GPU 1
	gpu = pool.Allocate(12000)

	if gpu == nil {
		t.Fatal("Allocate() returned nil")
	}

	if gpu.ID != 1 {
		t.Errorf("Allocated GPU ID = %d, want 1", gpu.ID)
	}

	// Try to allocate more than available
	gpu = pool.Allocate(20000)

	if gpu != nil {
		t.Error("Allocate() should return nil when no GPU available")
	}
}

func TestPool_Release(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)

	gpu, _ := pool.GetGPU(0)

	// Allocate GPU
	gpu.Allocate("task-1", 2048)
	gpu.Reserve("task-1")

	// Release GPU
	err := pool.Release(gpu)

	if err != nil {
		t.Errorf("Release() error = %v", err)
	}

	if gpu.IsReserved() {
		t.Error("GPU should not be reserved after release")
	}

	// Release nil GPU should not error
	err = pool.Release(nil)

	if err != nil {
		t.Errorf("Release(nil) error = %v", err)
	}
}

func TestPool_UpdateMetrics(t *testing.T) {
	pool := NewPool()

	pool.AddGPU(0, "GPU-0", 8192)

	pool.UpdateMetrics(0, 75, 80, 70)

	gpu, _ := pool.GetGPU(0)

	if gpu.MemoryUtil != 75 {
		t.Errorf("Memory util = %d, want 75", gpu.MemoryUtil)
	}

	if gpu.ComputeUtil != 80 {
		t.Errorf("Compute util = %d, want 80", gpu.ComputeUtil)
	}

	if gpu.Temperature != 70 {
		t.Errorf("Temperature = %d, want 70", gpu.Temperature)
	}

	// Update non-existent GPU should not panic
	pool.UpdateMetrics(999, 50, 50, 50)
}

func TestError(t *testing.T) {
	err := &Error{"test error"}

	if err.Error() != "test error" {
		t.Errorf("Error() = %s, want 'test error'", err.Error())
	}

	if ErrInsufficientMemory.Error() != "insufficient GPU memory" {
		t.Errorf("ErrInsufficientMemory message incorrect")
	}

	if ErrGPUNotFound.Error() != "GPU not found" {
		t.Errorf("ErrGPUNotFound message incorrect")
	}
}
