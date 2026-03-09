package scheduler

import (
	"sort"
	"sync"
	"time"

	"algogpu/pkg/types"
)

// UsageRecord represents a GPU usage record
type UsageRecord struct {
	UserID    string
	Cost      int64
	Timestamp time.Time
}

// UsageTracker tracks GPU usage per user with sliding window
type UsageTracker struct {
	mu           sync.RWMutex
	windowSize   time.Duration
	usageRecords map[string][]UsageRecord
}

// NewUsageTracker creates a new UsageTracker
func NewUsageTracker(windowMinutes int) *UsageTracker {
	return &UsageTracker{
		windowSize:   time.Duration(windowMinutes) * time.Minute,
		usageRecords: make(map[string][]UsageRecord),
	}
}

// AddUsage records a GPU usage event
func (t *UsageTracker) AddUsage(userID string, cost int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	record := UsageRecord{
		UserID:    userID,
		Cost:      cost,
		Timestamp: time.Now(),
	}

	t.usageRecords[userID] = append(t.usageRecords[userID], record)
	t.prune(userID)
}

// GetRecentUsage returns recent GPU usage for a user
func (t *UsageTracker) GetRecentUsage(userID string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.prune(userID)

	records := t.usageRecords[userID]
	var total int64
	for _, r := range records {
		total += r.Cost
	}

	return total
}

// prune removes old records outside the window
func (t *UsageTracker) prune(userID string) {
	records := t.usageRecords[userID]
	cutoff := time.Now().Add(-t.windowSize)

	i := 0
	for ; i < len(records); i++ {
		if records[i].Timestamp.After(cutoff) {
			break
		}
	}

	if i > 0 {
		t.usageRecords[userID] = records[i:]
	}
}

// CostAwareScheduler calculates priority based on cost and recent usage
type CostAwareScheduler struct {
	usageTracker *UsageTracker
	userWeights  map[string]float64
	mu           sync.RWMutex
}

// NewCostAwareScheduler creates a new CostAwareScheduler
func NewCostAwareScheduler(usageTracker *UsageTracker) *CostAwareScheduler {
	return &CostAwareScheduler{
		usageTracker: usageTracker,
		userWeights:  make(map[string]float64),
	}
}

// SetUserWeight sets a user's weight
func (s *CostAwareScheduler) SetUserWeight(userID string, weight float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userWeights[userID] = weight
}

// GetUserWeight gets a user's weight
func (s *CostAwareScheduler) GetUserWeight(userID string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if weight, ok := s.userWeights[userID]; ok {
		return weight
	}
	return 1.0 // default weight
}

// CalculatePriority calculates task priority
// priority = weight / (recent_gpu_usage + task_cost)
func (s *CostAwareScheduler) CalculatePriority(task *types.Task) float64 {
	weight := s.GetUserWeight(task.UserID)
	recentUsage := s.usageTracker.GetRecentUsage(task.UserID)
	taskCost := types.GetTaskCost(task.Type)

	denominator := float64(recentUsage + taskCost)
	if denominator <= 0 {
		denominator = 1
	}

	return weight / denominator
}

// TaskWithPriority represents a task with its calculated priority
type TaskWithPriority struct {
	Task     *types.Task
	Priority float64
}

// SortByPriority sorts tasks by priority (higher first)
func SortByPriority(tasks []*types.Task, scheduler *CostAwareScheduler) []TaskWithPriority {
	twp := make([]TaskWithPriority, len(tasks))
	for i, task := range tasks {
		twp[i] = TaskWithPriority{
			Task:     task,
			Priority: scheduler.CalculatePriority(task),
		}
	}

	sort.Slice(twp, func(i, j int) bool {
		return twp[i].Priority > twp[j].Priority // higher priority first
	})

	return twp
}
