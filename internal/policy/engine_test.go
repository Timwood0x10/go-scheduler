package policy

import (
	"context"
	"testing"
	"time"

	"algogpu/api"
	"algogpu/internal/db"
	"algogpu/internal/predictor"
	"algogpu/pkg/types"
)

func TestNewEngine(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	if engine == nil {
		t.Fatal("Expected non-nil engine")
	}
}

func TestEvaluateTask(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	ctx := context.Background()
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	decision, err := engine.EvaluateTask(ctx, task, 0)
	if err != nil {
		t.Fatalf("Failed to evaluate task: %v", err)
	}

	if decision == nil {
		t.Fatal("Expected non-nil decision")
	}

	if decision.EstimatedDurationMs <= 0 {
		t.Error("Expected positive estimated duration")
	}

	if decision.EstimatedMemoryMB <= 0 {
		t.Error("Expected positive estimated memory")
	}
}

func TestEvaluateTaskWithQueueSize(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	ctx := context.Background()
	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 50,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	// Test with different queue sizes
	decisionSmall, err := engine.EvaluateTask(ctx, task, 5)
	if err != nil {
		t.Fatalf("Failed to evaluate task with small queue: %v", err)
	}

	decisionLarge, err := engine.EvaluateTask(ctx, task, 50)
	if err != nil {
		t.Fatalf("Failed to evaluate task with large queue: %v", err)
	}

	// Larger queue should result in higher priority
	if decisionLarge.Priority <= decisionSmall.Priority {
		t.Error("Expected higher priority with larger queue")
	}
}

func TestEvaluateTaskWithWaitTime(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	ctx := context.Background()

	// Task that has been waiting longer
	oldTask := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		Priority:           10,
		CreatedAt:          time.Now().Add(-10 * time.Minute),
	}

	// Task that just arrived
	newTask := &types.Task{
		ID:                 "task-2",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	decisionOld, err := engine.EvaluateTask(ctx, oldTask, 10)
	if err != nil {
		t.Fatalf("Failed to evaluate old task: %v", err)
	}

	decisionNew, err := engine.EvaluateTask(ctx, newTask, 10)
	if err != nil {
		t.Fatalf("Failed to evaluate new task: %v", err)
	}

	// Old task should have higher priority due to aging
	if decisionOld.Priority <= decisionNew.Priority {
		t.Error("Expected higher priority for older task")
	}
}

func TestRefreshCache(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	ctx := context.Background()

	// Refresh cache (should not panic)
	engine.RefreshCache(ctx)
}

func TestGetTaskTypeStats(t *testing.T) {
	tmpFile, tErr := testing.TempFile(t, "test-*.db")
	if tErr != nil {
		t.Fatalf("Failed to create temp file: %v", tErr)
	}
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := predictor.NewStatsCache(5 * time.Second)
	resPredictor := predictor.NewResourcePredictor(store, cache)
	engine := NewEngine(resPredictor)

	ctx := context.Background()

	// Get stats (should not panic)
	stats := engine.GetTaskTypeStats(ctx, "llm")
	if stats == nil {
		t.Fatal("Expected non-nil stats")
	}
}

func TestApplyRules(t *testing.T) {
	rules := NewRules()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 50,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	prediction := &predictor.ResourcePrediction{
		EstimatedDurationMs: 500,
		EstimatedMemoryMB:   1024,
		AllowPacking:        true,
	}

	priority := rules.ApplyRules(task, prediction, 10)

	if priority <= 0 {
		t.Error("Expected positive priority")
	}
}

func TestShouldReject(t *testing.T) {
	rules := NewRules()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 3600000, // 1 hour
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	// Should reject due to long estimated duration
	if !rules.ShouldReject(task, 10) {
		t.Error("Expected task with 1 hour duration to be rejected")
	}

	// Short task should not be rejected
	task.EstimatedRuntimeMs = 5000
	if rules.ShouldReject(task, 10) {
		t.Error("Expected short task to not be rejected")
	}

	// Large queue should reject
	task.EstimatedRuntimeMs = 5000
	if !rules.ShouldReject(task, 2000) {
		t.Error("Expected task with large queue to be rejected")
	}
}

func TestShouldPreempt(t *testing.T) {
	rules := NewRules()

	newTask := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 5000,
		Priority:           10,
		CreatedAt:          time.Now(),
	}

	runningTask := &types.Task{
		ID:                 "task-2",
		UserID:             "user-2",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 50,
		EstimatedRuntimeMs: 500,
		Priority:           5,
		CreatedAt:          time.Now(),
	}

	// Preemption should be disabled for simplicity
	if rules.ShouldPreempt(newTask, runningTask) {
		t.Error("Expected preemption to be disabled")
	}
}
