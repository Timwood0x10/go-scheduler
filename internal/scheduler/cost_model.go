package scheduler

import (
	"sync"
	"time"

	"algogpu/api"
)

// BucketKey represents a cost bucket key
type BucketKey struct {
	TaskType  api.TaskType
	InputSize int64 // token count
}

// BucketStats represents statistics for a cost bucket
type BucketStats struct {
	AvgRuntimeMs int64   // average runtime in milliseconds
	AvgMemoryMB  int64   // average memory usage in MB
	AvgGPUUtil   float64 // average GPU utilization percentage
	SampleCount  int64   // number of samples
	LastUpdated  time.Time
}

// CostModel represents the GPU cost prediction model
type CostModel struct {
	mu           sync.RWMutex
	buckets      map[BucketKey]*BucketStats
	defaultCosts map[api.TaskType]*BucketStats
	bucketRanges []int64 // bucket boundaries: [256, 1024, 4096, ...]
}

// NewCostModel creates a new CostModel
func NewCostModel() *CostModel {
	return &CostModel{
		buckets:      make(map[BucketKey]*BucketStats),
		defaultCosts: make(map[api.TaskType]*BucketStats),
		bucketRanges: []int64{256, 1024, 4096, 16384},
	}
}

// getBucketKey returns the bucket key for a task
func (m *CostModel) getBucketKey(taskType api.TaskType, inputSize int64) BucketKey {
	bucketIndex := 0
	for i, threshold := range m.bucketRanges {
		if inputSize <= threshold {
			bucketIndex = i
			break
		}
		if i == len(m.bucketRanges)-1 {
			// inputSize exceeds max threshold, use last bucket
			bucketIndex = i
			break
		}
	}

	return BucketKey{
		TaskType:  taskType,
		InputSize: m.bucketRanges[bucketIndex],
	}
}

// EstimateCost estimates GPU cost for a task
func (m *CostModel) EstimateCost(taskType api.TaskType, inputSize int64) *BucketStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getBucketKey(taskType, inputSize)

	if stats, ok := m.buckets[key]; ok {
		return stats
	}

	// Return default if no history
	if defaultStats, ok := m.defaultCosts[taskType]; ok {
		return defaultStats
	}

	// Return generic default
	return &BucketStats{
		AvgRuntimeMs: 1000,
		AvgMemoryMB:  2048,
		AvgGPUUtil:   50.0,
		SampleCount:  0,
		LastUpdated:  time.Now(),
	}
}

// RecordCost records actual cost for a task
func (m *CostModel) RecordCost(taskType api.TaskType, inputSize int64, runtimeMs int64, memoryMB int64, gpuUtil float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getBucketKey(taskType, inputSize)

	if existing, ok := m.buckets[key]; ok {
		// Update running average
		n := float64(existing.SampleCount)
		existing.AvgRuntimeMs = int64((float64(existing.AvgRuntimeMs)*n + float64(runtimeMs)) / (n + 1))
		existing.AvgMemoryMB = int64((float64(existing.AvgMemoryMB)*n + float64(memoryMB)) / (n + 1))
		existing.AvgGPUUtil = (existing.AvgGPUUtil*n + gpuUtil) / (n + 1)
		existing.SampleCount++
		existing.LastUpdated = time.Now()
	} else {
		// Create new bucket
		m.buckets[key] = &BucketStats{
			AvgRuntimeMs: runtimeMs,
			AvgMemoryMB:  memoryMB,
			AvgGPUUtil:   gpuUtil,
			SampleCount:  1,
			LastUpdated:  time.Now(),
		}
	}
}

// SetDefaultCost sets default cost for a task type
func (m *CostModel) SetDefaultCost(taskType api.TaskType, stats *BucketStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultCosts[taskType] = stats
}

// GetBucketStats returns statistics for a specific bucket
func (m *CostModel) GetBucketStats(taskType api.TaskType, inputSize int64) (*BucketStats, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getBucketKey(taskType, inputSize)
	stats, ok := m.buckets[key]
	return stats, ok
}

// GetAllBuckets returns all bucket statistics
func (m *CostModel) GetAllBuckets() map[BucketKey]*BucketStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[BucketKey]*BucketStats)
	for k, v := range m.buckets {
		result[k] = v
	}

	return result
}

// Reset clears all bucket statistics
func (m *CostModel) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buckets = make(map[BucketKey]*BucketStats)
}
