package scheduler

import (
	"testing"

	"algogpu/api"
)

func TestCostModel_NewCostModel(t *testing.T) {
	cm := NewCostModel()

	if cm == nil {
		t.Error("NewCostModel should not return nil")
	}

	if cm.buckets == nil {
		t.Error("buckets should be initialized")
	}

	if cm.defaultCosts == nil {
		t.Error("defaultCosts should be initialized")
	}
}

func TestCostModel_getBucketKey(t *testing.T) {
	cm := NewCostModel()

	tests := []struct {
		name      string
		taskType  api.TaskType
		inputSize int64
		expected  int64
	}{
		{"small_embedding", api.TaskType_TASK_TYPE_EMBEDDING, 100, 256},
		{"medium_embedding", api.TaskType_TASK_TYPE_EMBEDDING, 500, 1024},
		{"large_embedding", api.TaskType_TASK_TYPE_EMBEDDING, 2000, 4096},
		{"xl_embedding", api.TaskType_TASK_TYPE_EMBEDDING, 8000, 16384},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cm.getBucketKey(tt.taskType, tt.inputSize)
			if key.InputSize != tt.expected {
				t.Errorf("expected bucket size %d, got %d", tt.expected, key.InputSize)
			}
		})
	}
}

func TestCostModel_EstimateCost(t *testing.T) {
	cm := NewCostModel()

	// Test with no history - should return default
	stats := cm.EstimateCost(api.TaskType_TASK_TYPE_LLM, 1000)

	if stats == nil {
		t.Error("EstimateCost should not return nil")
	}

	if stats.AvgRuntimeMs <= 0 {
		t.Error("AvgRuntimeMs should be positive")
	}
}

func TestCostModel_RecordCost(t *testing.T) {
	cm := NewCostModel()

	// Record some costs
	cm.RecordCost(api.TaskType_TASK_TYPE_LLM, 500, 2000, 8192, 80.0)
	cm.RecordCost(api.TaskType_TASK_TYPE_LLM, 500, 2200, 8400, 85.0)

	// Check bucket was created
	key := cm.getBucketKey(api.TaskType_TASK_TYPE_LLM, 500)
	stats, ok := cm.buckets[key]

	if !ok {
		t.Error("Bucket should be created")
	}

	if stats.SampleCount != 2 {
		t.Errorf("expected 2 samples, got %d", stats.SampleCount)
	}
}

func TestCostModel_EstimateCost_WithHistory(t *testing.T) {
	cm := NewCostModel()

	// Record known costs
	cm.RecordCost(api.TaskType_TASK_TYPE_EMBEDDING, 100, 20, 512, 30.0)

	// Estimate should use recorded data
	stats := cm.EstimateCost(api.TaskType_TASK_TYPE_EMBEDDING, 100)

	if stats.AvgRuntimeMs != 20 {
		t.Errorf("expected 20ms, got %d", stats.AvgRuntimeMs)
	}

	if stats.AvgMemoryMB != 512 {
		t.Errorf("expected 512MB, got %d", stats.AvgMemoryMB)
	}
}

func TestCostModel_SetDefaultCost(t *testing.T) {
	cm := NewCostModel()

	defaultStats := &BucketStats{
		AvgRuntimeMs: 5000,
		AvgMemoryMB:  16384,
		AvgGPUUtil:   90.0,
		SampleCount:  0,
	}

	cm.SetDefaultCost(api.TaskType_TASK_TYPE_DIFFUSION, defaultStats)

	// Estimate without history should use default
	stats := cm.EstimateCost(api.TaskType_TASK_TYPE_DIFFUSION, 100000)

	if stats.AvgMemoryMB != 16384 {
		t.Errorf("expected 16384MB, got %d", stats.AvgMemoryMB)
	}
}

func TestCostModel_GetBucketStats(t *testing.T) {
	cm := NewCostModel()

	cm.RecordCost(api.TaskType_TASK_TYPE_LLM, 1000, 1500, 4096, 60.0)

	stats, ok := cm.GetBucketStats(api.TaskType_TASK_TYPE_LLM, 1000)

	if !ok {
		t.Error("Should find the bucket")
	}

	if stats.AvgRuntimeMs != 1500 {
		t.Errorf("expected 1500, got %d", stats.AvgRuntimeMs)
	}
}

func TestCostModel_GetAllBuckets(t *testing.T) {
	cm := NewCostModel()

	cm.RecordCost(api.TaskType_TASK_TYPE_LLM, 500, 1000, 2048, 50.0)
	cm.RecordCost(api.TaskType_TASK_TYPE_EMBEDDING, 100, 20, 512, 30.0)

	buckets := cm.GetAllBuckets()

	if len(buckets) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(buckets))
	}
}

func TestCostModel_Reset(t *testing.T) {
	cm := NewCostModel()

	cm.RecordCost(api.TaskType_TASK_TYPE_LLM, 500, 1000, 2048, 50.0)

	if len(cm.buckets) != 1 {
		t.Error("Should have 1 bucket before reset")
	}

	cm.Reset()

	if len(cm.buckets) != 0 {
		t.Error("Should have 0 buckets after reset")
	}
}
