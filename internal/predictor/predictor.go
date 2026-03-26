// Package predictor provides GPU resource prediction based on historical data.
// It predicts task duration, memory usage, and packing eligibility.
package predictor

import (
	"context"
	"fmt"
	"log"
	"time"

	"algogpu/internal/db"
	"algogpu/pkg/types"
)

// ResourcePredictor predicts GPU resource requirements for tasks.
type ResourcePredictor struct {
	store *db.Store
	cache *StatsCache
}

// NewResourcePredictor creates a new resource predictor.
func NewResourcePredictor(store *db.Store) *ResourcePredictor {
	return &ResourcePredictor{
		store: store,
		cache: NewStatsCache(5 * time.Minute),
	}
}

// Predict predicts resource requirements for a task.
func (p *ResourcePredictor) Predict(ctx context.Context, taskType string) (*types.ResourcePrediction, error) {
	// Try to get from cache
	if stats, ok := p.cache.Get(taskType); ok {
		return p.buildPrediction(stats), nil
	}

	// Query from database
	stats, err := p.store.GetTaskStats(ctx, taskType, 7) // 7 days
	if err != nil {
		log.Printf("Failed to get task stats for %s: %v", taskType, err)
		// Fall back to default predictions
		return p.getDefaultPrediction(taskType), nil
	}

	// Cache the stats
	p.cache.Set(taskType, stats)

	return p.buildPrediction(stats), nil
}

// buildPrediction creates a prediction from task statistics.
func (p *ResourcePredictor) buildPrediction(stats *db.TaskStats) *types.ResourcePrediction {
	if stats.Count == 0 {
		// No data, use default
		return p.getDefaultPrediction(stats.TaskType)
	}

	prediction := &types.ResourcePrediction{
		EstimatedDurationMs: stats.AvgDurationMs,
		EstimatedMemoryMB:   stats.AvgMemoryMB,
		EstimatedGpuUtil:    stats.AvgGPUUtil,
	}

	// Determine if packing is allowed
	// Tasks with small memory footprint and low GPU utilization can be packed
	if stats.AvgMemoryMB < 4000 && stats.AvgGPUUtil < 0.5 {
		prediction.AllowPacking = true
	} else {
		prediction.AllowPacking = false
	}

	return prediction
}

// getDefaultPrediction returns default prediction for cold start.
func (p *ResourcePredictor) getDefaultPrediction(taskType string) *types.ResourcePrediction {
	predictions, ok := DefaultPredictions[taskType]
	if !ok {
		predictions = DefaultPredictions["other"]
	}

	return &predictions
}

// DefaultPredictions provides default predictions for cold start.
var DefaultPredictions = map[string]types.ResourcePrediction{
	"embedding": {
		EstimatedDurationMs: 50,
		EstimatedMemoryMB:   2000,
		EstimatedGpuUtil:    0.3,
		AllowPacking:        true,
	},
	"llm": {
		EstimatedDurationMs: 1000,
		EstimatedMemoryMB:   16000,
		EstimatedGpuUtil:    0.8,
		AllowPacking:        false,
	},
	"llm_inference": {
		EstimatedDurationMs: 1000,
		EstimatedMemoryMB:   16000,
		EstimatedGpuUtil:    0.8,
		AllowPacking:        false,
	},
	"diffusion": {
		EstimatedDurationMs: 2000,
		EstimatedMemoryMB:   12000,
		EstimatedGpuUtil:    0.85,
		AllowPacking:        false,
	},
	"training": {
		EstimatedDurationMs: 10000,
		EstimatedMemoryMB:   24000,
		EstimatedGpuUtil:    0.95,
		AllowPacking:        false,
	},
	"vision": {
		EstimatedDurationMs: 500,
		EstimatedMemoryMB:   4000,
		EstimatedGpuUtil:    0.7,
		AllowPacking:        true,
	},
	"other": {
		EstimatedDurationMs: 500,
		EstimatedMemoryMB:   8000,
		EstimatedGpuUtil:    0.6,
		AllowPacking:        false,
	},
}

// GetDefaultPrediction returns default prediction for cold start.
func (p *ResourcePredictor) GetDefaultPrediction(taskType string) *types.ResourcePrediction {
	return p.getDefaultPrediction(taskType)
}

// RefreshCache refreshes the stats cache.
func (p *ResourcePredictor) RefreshCache(ctx context.Context, taskTypes []string) {
	for _, taskType := range taskTypes {
		stats, err := p.store.GetTaskStats(ctx, taskType, 7)
		if err != nil {
			log.Printf("Failed to refresh cache for %s: %v", taskType, err)
			continue
		}

		p.cache.Set(taskType, stats)
	}
}

// GetTaskTypeName returns the canonical task type name.
func GetTaskTypeName(taskType string) string {
	// Normalize task type names
	standardTypes := map[string]string{
		"TASK_TYPE_EMBEDDING":   "embedding",
		"TASK_TYPE_LLM":         "llm",
		"TASK_TYPE_DIFFUSION":   "diffusion",
		"TASK_TYPE_UNSPECIFIED": "other",
		"TASK_TYPE_OTHER":       "other",
		"llm_inference":         "llm_inference",
		"text_embedding":        "embedding",
	}

	if canonical, ok := standardTypes[taskType]; ok {
		return canonical
	}

	// Try to match by prefix
	for key, canonical := range standardTypes {
		if len(taskType) >= len(key) && taskType[:len(key)] == key {
			return canonical
		}
	}

	// Default to "other" if no match
	return "other"
}

// PredictWithMetadata predicts resources with additional metadata.
func (p *ResourcePredictor) PredictWithMetadata(ctx context.Context, taskType, model string) (*types.ResourcePrediction, error) {
	// For now, use task type only
	// Future enhancement: use model-specific statistics
	canonicalType := GetTaskTypeName(taskType)
	return p.Predict(ctx, canonicalType)
}

// GetPredictedQueueWait predicts queue wait time based on task duration and queue size.
func (p *ResourcePredictor) GetPredictedQueueWait(ctx context.Context, taskType string, queueSize int) (int64, error) {
	prediction, err := p.Predict(ctx, taskType)
	if err != nil {
		return 0, fmt.Errorf("failed to predict queue wait: %w", err)
	}

	// Simple prediction: queue_size * avg_duration
	// This assumes FIFO scheduling
	predictedWait := int64(queueSize) * prediction.EstimatedDurationMs

	return predictedWait, nil
}
