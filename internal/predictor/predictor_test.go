package predictor

import (
	"context"
	"os"
	"testing"
	"time"

	"algogpu/internal/db"
)

func TestNewResourcePredictor(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	if predictor == nil {
		t.Fatal("Expected non-nil predictor")
	}
}

func TestPredict(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	// Test prediction with no data
	ctx := context.Background()
	prediction := predictor.Predict(ctx, "llm")

	if prediction == nil {
		t.Fatal("Expected non-nil prediction")
	}

	if prediction.EstimatedDurationMs <= 0 {
		t.Error("Expected positive duration")
	}

	if prediction.EstimatedMemoryMB <= 0 {
		t.Error("Expected positive memory")
	}
}

func TestGetTaskTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TASK_TYPE_LLM", "llm"},
		{"TASK_TYPE_EMBEDDING", "embedding"},
		{"TASK_TYPE_DIFFUSION", "diffusion"},
		{"llm", "llm"},
		{"embedding", "embedding"},
		{"unknown", "other"},
	}

	for _, tt := range tests {
		result := GetTaskTypeName(tt.input)
		if result != tt.expected {
			t.Errorf("GetTaskTypeName(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestPredictWithHistoricalData(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	// Insert historical data
	ctx := context.Background()
	taskType := "llm"

	// Simulate inserting execution data
	// Note: This would require actual Task, GPU, and Metrics objects
	// For now, we'll test the default prediction behavior

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	prediction := predictor.Predict(ctx, taskType)

	if prediction == nil {
		t.Fatal("Expected non-nil prediction")
	}

	if !prediction.AllowPacking {
		t.Error("Expected AllowPacking to be true for LLM")
	}
}

func TestRefreshCache(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	ctx := context.Background()
	taskTypes := []string{"llm", "embedding", "diffusion"}

	// Refresh cache (should not panic)
	predictor.RefreshCache(ctx, taskTypes)

	// Verify cache is populated
	if cache.Size() == 0 {
		t.Error("Expected cache to be populated after refresh")
	}
}

func TestGetDefaultPrediction(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	tests := []struct {
		taskType       string
		expectedMinMem int64
		expectedMinDur int64
		allowPacking   bool
	}{
		{"llm", 8192, 5000, true},
		{"embedding", 1024, 500, true},
		{"diffusion", 16384, 30000, false},
		{"unknown", 4096, 2000, true},
	}

	for _, tt := range tests {
		prediction := predictor.GetDefaultPrediction(tt.taskType)

		if prediction.EstimatedMemoryMB < tt.expectedMinMem {
			t.Errorf("Task type %s: expected memory >= %d, got %d",
				tt.taskType, tt.expectedMinMem, prediction.EstimatedMemoryMB)
		}

		if prediction.EstimatedDurationMs < tt.expectedMinDur {
			t.Errorf("Task type %s: expected duration >= %d, got %d",
				tt.taskType, tt.expectedMinDur, prediction.EstimatedDurationMs)
		}

		if prediction.AllowPacking != tt.allowPacking {
			t.Errorf("Task type %s: expected AllowPacking = %v, got %v",
				tt.taskType, tt.allowPacking, prediction.AllowPacking)
		}
	}
}

func TestPredictQueueWait(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := db.NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite store: %v", err)
	}
	defer store.Close()

	cache := NewStatsCache(5 * time.Second)
	predictor := NewResourcePredictor(store, cache)

	ctx := context.Background()

	// Test with no queue
	wait := predictor.GetPredictedQueueWait(ctx, "llm", 0)
	if wait < 0 {
		t.Error("Expected non-negative queue wait")
	}

	// Test with queue
	wait = predictor.GetPredictedQueueWait(ctx, "llm", 10)
	if wait < 0 {
		t.Error("Expected non-negative queue wait with queue size 10")
	}

	// Queue wait should increase with queue size
	waitSmall := predictor.GetPredictedQueueWait(ctx, "llm", 5)
	waitLarge := predictor.GetPredictedQueueWait(ctx, "llm", 20)

	if waitLarge < waitSmall {
		t.Error("Expected larger queue wait for larger queue size")
	}
}
