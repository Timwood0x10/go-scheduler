package scheduler

import (
	"errors"
)

// Config holds scheduler configuration
type Config struct {
	// Admission Control
	MaxQueueSize int `json:"max_queue_size"`

	// Token Bucket
	TokenRefillRate    int64 `json:"token_refill_rate"`    // tokens per second
	TokenBucketSize    int64 `json:"token_bucket_size"`    // max tokens
	DailyTokenLimit    int64 `json:"daily_token_limit"`    // daily limit
	TokenCostEmbedding int64 `json:"token_cost_embedding"` // 1
	TokenCostLLM       int64 `json:"token_cost_llm"`       // 5
	TokenCostDiffusion int64 `json:"token_cost_diffusion"` // 10

	// GPU Packing
	GPULoadThreshold float64 `json:"gpu_load_threshold"` // 0.85
	MemoryWeight     float64 `json:"memory_weight"`      // 0.7
	ComputeWeight    float64 `json:"compute_weight"`     // 0.3

	// Task Aging
	AgingFactor float64 `json:"aging_factor"` // γ

	// Cost-aware Scheduling
	UsageWindowMinutes int `json:"usage_window_minutes"` // 5
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		MaxQueueSize:       1000,
		TokenRefillRate:    100,  // 100 tokens/second
		TokenBucketSize:    1000, // max 1000 tokens
		DailyTokenLimit:    1000000,
		TokenCostEmbedding: 1,
		TokenCostLLM:       5,
		TokenCostDiffusion: 10,
		GPULoadThreshold:   0.85,
		MemoryWeight:       0.7,
		ComputeWeight:      0.3,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}
}

// Errors
var (
	ErrQueueFull         = errors.New("queue is full")
	ErrInsufficientToken = errors.New("insufficient token")
	ErrNoAvailableGPU    = errors.New("no available GPU")
)
