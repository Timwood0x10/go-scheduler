package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

func TestNewSQLiteStore(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	if store == nil {
		t.Fatal("Expected non-nil store")
	}
}

func TestRecordExecution(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Create test task
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	// Create test GPU
	gpu := &gpu.GPU{
		ID:          0,
		Name:        "NVIDIA-A100",
		MemoryTotal: 81920,
		MemoryUsed:  4096,
		ComputeUtil: 85,
		MemoryUtil:  50,
	}

	// Create metrics
	metrics := &types.ExecutionMetrics{
		ExecutionTimeMs: 1500,
		AvgGPUUtil:      85.5,
		MaxGPUUtil:      95.0,
		AvgMemUtil:      70.0,
		MaxMemUtil:      80.0,
		GPUMemoryUsedMB: 4096,
		Success:         true,
	}

	queueWait := 100 * time.Millisecond

	// Record execution
	err = store.RecordExecution(context.Background(), task, gpu, queueWait, metrics)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Verify record exists
	stats, err := store.GetTaskTypeStats(context.Background(), "llm")
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 task, got %d", stats.TotalTasks)
	}

	if stats.AvgExecutionMs != 1500 {
		t.Errorf("Expected avg execution 1500ms, got %d", stats.AvgExecutionMs)
	}
}

func TestGetTaskTypeStats(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Insert test data
	ctx := context.Background()
	task := &types.Task{
		ID:        "task-1",
		UserID:    "user-1",
		Type:      api.TaskType_TASK_TYPE_LLM,
		Priority:  10,
		CreatedAt: time.Now(),
	}

	gpu := &gpu.GPU{
		ID:          0,
		Name:        "NVIDIA-A100",
		MemoryTotal: 81920,
		MemoryUsed:  4096,
		ComputeUtil: 85,
		MemoryUtil:  50,
	}

	metrics := &types.ExecutionMetrics{
		ExecutionTimeMs: 1000,
		AvgGPUUtil:      80.0,
		MaxGPUUtil:      90.0,
		AvgMemUtil:      60.0,
		MaxMemUtil:      70.0,
		GPUMemoryUsedMB: 4096,
		Success:         true,
	}

	err = store.RecordExecution(ctx, task, gpu, 100*time.Millisecond, metrics)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	// Get stats
	stats, err := store.GetTaskTypeStats(ctx, "llm")
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 task, got %d", stats.TotalTasks)
	}

	if stats.AvgExecutionMs != 1000 {
		t.Errorf("Expected avg execution 1000ms, got %d", stats.AvgExecutionMs)
	}

	if stats.AvgMemoryMB != 4096 {
		t.Errorf("Expected avg memory 4096MB, got %d", stats.AvgMemoryMB)
	}
}

func TestGetUserStats(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	task := &types.Task{
		ID:        "task-1",
		UserID:    "user-1",
		Type:      api.TaskType_TASK_TYPE_EMBEDDING,
		Priority:  10,
		CreatedAt: time.Now(),
	}

	gpu := &gpu.GPU{
		ID:          0,
		Name:        "NVIDIA-A100",
		MemoryTotal: 81920,
		MemoryUsed:  1024,
		ComputeUtil: 60,
		MemoryUtil:  30,
	}

	metrics := &types.ExecutionMetrics{
		ExecutionTimeMs: 200,
		AvgGPUUtil:      60.0,
		MaxGPUUtil:      70.0,
		AvgMemUtil:      30.0,
		MaxMemUtil:      40.0,
		GPUMemoryUsedMB: 1024,
		Success:         true,
	}

	err = store.RecordExecution(ctx, task, gpu, 50*time.Millisecond, metrics)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	stats, err := store.GetUserStats(ctx, "user-1")
	if err != nil {
		t.Fatalf("Failed to get user stats: %v", err)
	}

	if stats.TotalTasks != 1 {
		t.Errorf("Expected 1 task, got %d", stats.TotalTasks)
	}
}

func TestGetQueueWaitStats(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	task := &types.Task{
		ID:        "task-1",
		UserID:    "user-1",
		Type:      api.TaskType_TASK_TYPE_LLM,
		Priority:  10,
		CreatedAt: time.Now(),
	}

	gpu := &gpu.GPU{
		ID:          0,
		Name:        "NVIDIA-A100",
		MemoryTotal: 81920,
		MemoryUsed:  4096,
		ComputeUtil: 85,
		MemoryUtil:  50,
	}

	metrics := &types.ExecutionMetrics{
		ExecutionTimeMs: 1500,
		AvgGPUUtil:      85.0,
		MaxGPUUtil:      95.0,
		AvgMemUtil:      70.0,
		MaxMemUtil:      80.0,
		GPUMemoryUsedMB: 4096,
		Success:         true,
	}

	queueWait := 200 * time.Millisecond
	err = store.RecordExecution(ctx, task, gpu, queueWait, metrics)
	if err != nil {
		t.Fatalf("Failed to record execution: %v", err)
	}

	stats, err := store.GetQueueWaitStats(ctx, "llm")
	if err != nil {
		t.Fatalf("Failed to get queue wait stats: %v", err)
	}

	if stats.TotalSamples != 1 {
		t.Errorf("Expected 1 sample, got %d", stats.TotalSamples)
	}

	if stats.AvgWaitMs != 200 {
		t.Errorf("Expected avg wait 200ms, got %d", stats.AvgWaitMs)
	}
}

func TestCleanupOldRecords(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert old record
	oldTask := &types.Task{
		ID:        "old-task",
		UserID:    "user-1",
		Type:      api.TaskType_TASK_TYPE_LLM,
		Priority:  10,
		CreatedAt: time.Now().Add(-48 * time.Hour), // 2 days ago
	}

	gpu := &gpu.GPU{
		ID:          0,
		Name:        "NVIDIA-A100",
		MemoryTotal: 81920,
		MemoryUsed:  4096,
		ComputeUtil: 85,
		MemoryUtil:  50,
	}

	metrics := &types.ExecutionMetrics{
		ExecutionTimeMs: 1500,
		AvgGPUUtil:      85.0,
		MaxGPUUtil:      95.0,
		AvgMemUtil:      70.0,
		MaxMemUtil:      80.0,
		GPUMemoryUsedMB: 4096,
		Success:         true,
	}

	err = store.RecordExecution(ctx, oldTask, gpu, 100*time.Millisecond, metrics)
	if err != nil {
		t.Fatalf("Failed to record old execution: %v", err)
	}

	// Cleanup records older than 24 hours
	err = store.CleanupOldRecords(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old records: %v", err)
	}

	// Verify cleanup
	stats, err := store.GetTaskTypeStats(ctx, "llm")
	if err != nil {
		t.Fatalf("Failed to get task stats: %v", err)
	}

	if stats.TotalTasks != 0 {
		t.Errorf("Expected 0 tasks after cleanup, got %d", stats.TotalTasks)
	}
}

func TestGetStatsSince(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Insert records
	taskTypes := []api.TaskType{
		api.TaskType_TASK_TYPE_LLM,
		api.TaskType_TASK_TYPE_EMBEDDING,
	}

	for i, taskType := range taskTypes {
		task := &types.Task{
			ID:        fmt.Sprintf("task-%d", i),
			UserID:    "user-1",
			Type:      taskType,
			Priority:  10,
			CreatedAt: time.Now(),
		}

		gpu := &gpu.GPU{
			ID:          0,
			Name:        "NVIDIA-A100",
			MemoryTotal: 81920,
			MemoryUsed:  4096,
			ComputeUtil: 85,
			MemoryUtil:  50,
		}

		metrics := &types.ExecutionMetrics{
			ExecutionTimeMs: 1500,
			AvgGPUUtil:      85.0,
			MaxGPUUtil:      95.0,
			AvgMemUtil:      70.0,
			MaxMemUtil:      80.0,
			GPUMemoryUsedMB: 4096,
			Success:         true,
		}

		err = store.RecordExecution(ctx, task, gpu, 100*time.Millisecond, metrics)
		if err != nil {
			t.Fatalf("Failed to record execution: %v", err)
		}
	}

	// Get stats since 1 hour ago
	since := time.Now().Add(-1 * time.Hour)
	statsList, err := store.GetStatsSince(ctx, since)
	if err != nil {
		t.Fatalf("Failed to get stats since: %v", err)
	}

	if len(statsList) != 2 {
		t.Errorf("Expected 2 stats, got %d", len(statsList))
	}
}
