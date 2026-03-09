package scheduler

import (
	"testing"
	"time"

	"algogpu/api"
)

func TestTokenBucketManager_CheckAndConsume(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenBucketSize = 100
	cfg.TokenRefillRate = 1000 // High refill rate for testing
	cfg.TokenCostLLM = 5

	mgr := NewTokenBucketManager(cfg)

	// First request should succeed
	err := mgr.CheckAndConsume("user-1", api.TaskType_TASK_TYPE_LLM)
	if err != nil {
		t.Errorf("First request should succeed: %v", err)
	}

	// Wait for refill
	time.Sleep(10 * time.Millisecond)

	// Second request should also succeed
	err = mgr.CheckAndConsume("user-1", api.TaskType_TASK_TYPE_LLM)
	if err != nil {
		t.Errorf("Second request should succeed: %v", err)
	}
}

func TestTokenBucketManager_GetTokenBalance(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenBucketSize = 100

	mgr := NewTokenBucketManager(cfg)

	current, dailyUsed, dailyLimit := mgr.GetTokenBalance("user-1")

	if current != 100 {
		t.Errorf("Expected 100 tokens, got %d", current)
	}

	if dailyUsed != 0 {
		t.Errorf("Expected 0 daily used, got %d", dailyUsed)
	}

	if dailyLimit != cfg.DailyTokenLimit {
		t.Errorf("Expected daily limit %d, got %d", cfg.DailyTokenLimit, dailyLimit)
	}
}

func TestTokenBucketManager_ResetDailyUsage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TokenBucketSize = 1000
	cfg.TokenRefillRate = 1000
	cfg.TokenCostLLM = 1

	mgr := NewTokenBucketManager(cfg)

	// Consume some tokens
	_ = mgr.CheckAndConsume("user-1", api.TaskType_TASK_TYPE_LLM)

	// Reset daily usage
	mgr.ResetDailyUsage()

	// Should have full daily limit again
	_, dailyUsed, _ := mgr.GetTokenBalance("user-1")
	if dailyUsed != 0 {
		t.Errorf("Expected 0 daily used after reset, got %d", dailyUsed)
	}
}

func TestUserTokenBucket_Refill(t *testing.T) {
	bucket := &UserTokenBucket{
		Tokens:         0,
		MaxTokens:      100,
		RefillRate:     10, // 10 tokens per second
		LastRefillTime: time.Now().Add(-1 * time.Second), // 1 second ago
	}

	bucket.Refill()

	if bucket.Tokens != 10 {
		t.Errorf("Expected 10 tokens after refill, got %d", bucket.Tokens)
	}
}

func TestUserTokenBucket_TryConsume(t *testing.T) {
	bucket := &UserTokenBucket{
		Tokens:     10,
		MaxTokens:  100,
		DailyLimit: 100,
		CostLLM:    5,
	}

	// Should succeed (10 tokens, need 5)
	if !bucket.TryConsume(api.TaskType_TASK_TYPE_LLM) {
		t.Error("Should be able to consume tokens")
	}

	// Should succeed (5 tokens left, need 5)
	if !bucket.TryConsume(api.TaskType_TASK_TYPE_LLM) {
		t.Error("Should be able to consume tokens")
	}

	// Should fail (0 tokens left, need 5)
	if bucket.TryConsume(api.TaskType_TASK_TYPE_LLM) {
		t.Error("Should not be able to consume tokens")
	}
}