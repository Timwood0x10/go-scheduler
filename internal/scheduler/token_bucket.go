package scheduler

import (
	"sync"
	"time"

	"algogpu/api"
)

// UserTokenBucket represents a user's token bucket
type UserTokenBucket struct {
	UserID         string
	mu             sync.RWMutex
	Tokens         int64
	MaxTokens      int64
	RefillRate     int64 // tokens per second
	LastRefillTime time.Time
	DailyUsed      int64
	DailyLimit     int64
	CostEmbedding  int64
	CostLLM        int64
	CostDiffusion  int64
}

// TokenBucketManager manages token buckets for all users
type TokenBucketManager struct {
	mu            sync.RWMutex
	users         map[string]*UserTokenBucket
	refillRate    int64
	maxTokens     int64
	dailyLimit    int64
	costEmbedding int64
	costLLM       int64
	costDiffusion int64
}

// NewTokenBucketManager creates a new TokenBucketManager
func NewTokenBucketManager(cfg *Config) *TokenBucketManager {
	return &TokenBucketManager{
		users:         make(map[string]*UserTokenBucket),
		refillRate:    cfg.TokenRefillRate,
		maxTokens:     cfg.TokenBucketSize,
		dailyLimit:    cfg.DailyTokenLimit,
		costEmbedding: cfg.TokenCostEmbedding,
		costLLM:       cfg.TokenCostLLM,
		costDiffusion: cfg.TokenCostDiffusion,
	}
}

// getOrCreateUser gets or creates a user token bucket
func (m *TokenBucketManager) getOrCreateUser(userID string) *UserTokenBucket {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bucket, exists := m.users[userID]; exists {
		return bucket
	}

	bucket := &UserTokenBucket{
		UserID:         userID,
		Tokens:         m.maxTokens,
		MaxTokens:      m.maxTokens,
		RefillRate:     m.refillRate,
		LastRefillTime: time.Now(),
		DailyUsed:      0,
		DailyLimit:     m.dailyLimit,
		CostEmbedding:  m.costEmbedding,
		CostLLM:        m.costLLM,
		CostDiffusion:  m.costDiffusion,
	}

	m.users[userID] = bucket
	return bucket
}

// Refill refills tokens for a user
func (b *UserTokenBucket) Refill() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.LastRefillTime).Seconds()
	refill := int64(elapsed * float64(b.RefillRate))

	b.Tokens += refill
	if b.Tokens > b.MaxTokens {
		b.Tokens = b.MaxTokens
	}

	b.LastRefillTime = now
}

// TryConsume tries to consume tokens for a task
func (b *UserTokenBucket) TryConsume(taskType api.TaskType) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	cost := b.getCost(taskType)
	if cost <= 0 {
		return true
	}

	// Check daily limit
	if b.DailyUsed+cost > b.DailyLimit {
		return false
	}

	// Check token balance
	if b.Tokens < cost {
		return false
	}

	b.Tokens -= cost
	b.DailyUsed += cost
	return true
}

// getCost returns the cost for a task type
func (b *UserTokenBucket) getCost(taskType api.TaskType) int64 {
	switch taskType {
	case api.TaskType_TASK_TYPE_EMBEDDING:
		return b.CostEmbedding
	case api.TaskType_TASK_TYPE_LLM:
		return b.CostLLM
	case api.TaskType_TASK_TYPE_DIFFUSION:
		return b.CostDiffusion
	default:
		return 1
	}
}

// CheckAndConsume checks token availability and consumes if possible
func (m *TokenBucketManager) CheckAndConsume(userID string, taskType api.TaskType) error {
	bucket := m.getOrCreateUser(userID)
	bucket.Refill()

	if !bucket.TryConsume(taskType) {
		return ErrInsufficientToken
	}

	return nil
}

// GetTokenBalance returns current token balance for a user
func (m *TokenBucketManager) GetTokenBalance(userID string) (current, dailyUsed, dailyLimit int64) {
	bucket := m.getOrCreateUser(userID)
	bucket.Refill()

	bucket.mu.RLock()
	defer bucket.mu.RUnlock()

	return bucket.Tokens, bucket.DailyUsed, bucket.DailyLimit
}

// ResetDailyUsage resets daily usage for all users (call at midnight)
func (m *TokenBucketManager) ResetDailyUsage() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bucket := range m.users {
		bucket.mu.Lock()
		bucket.DailyUsed = 0
		bucket.mu.Unlock()
	}
}
