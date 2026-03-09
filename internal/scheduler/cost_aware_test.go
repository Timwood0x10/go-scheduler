package scheduler

import (
	"testing"

	"algogpu/api"
	"algogpu/pkg/types"
)

func TestUsageTracker_AddUsage(t *testing.T) {
	tracker := NewUsageTracker(5) // 5 minutes window

	tracker.AddUsage("user-1", 10)

	usage := tracker.GetRecentUsage("user-1")
	if usage != 10 {
		t.Errorf("Expected 10 usage, got %d", usage)
	}
}

func TestCostAwareScheduler_CalculatePriority(t *testing.T) {
	tracker := NewUsageTracker(5)
	scheduler := NewCostAwareScheduler(tracker)

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM, // cost = 5
	}

	// First task should have high priority
	priority := scheduler.CalculatePriority(task)

	// Add usage for this user
	tracker.AddUsage("user-1", 100)

	// Second task should have lower priority
	priority2 := scheduler.CalculatePriority(task)

	if priority <= priority2 {
		t.Error("Priority should decrease after usage")
	}
}

func TestCostAwareScheduler_SetUserWeight(t *testing.T) {
	tracker := NewUsageTracker(5)
	scheduler := NewCostAwareScheduler(tracker)

	scheduler.SetUserWeight("user-1", 2.0)

	weight := scheduler.GetUserWeight("user-1")
	if weight != 2.0 {
		t.Errorf("Expected weight 2.0, got %f", weight)
	}
}

func TestCostAwareScheduler_DefaultWeight(t *testing.T) {
	tracker := NewUsageTracker(5)
	scheduler := NewCostAwareScheduler(tracker)

	// User without explicit weight should have default 1.0
	weight := scheduler.GetUserWeight("unknown-user")
	if weight != 1.0 {
		t.Errorf("Expected default weight 1.0, got %f", weight)
	}
}

func TestSortByPriority(t *testing.T) {
	tracker := NewUsageTracker(5)
	scheduler := NewCostAwareScheduler(tracker)

	tasks := []*types.Task{
		{ID: "task-1", UserID: "user-1", Type: api.TaskType_TASK_TYPE_EMBEDDING},
		{ID: "task-2", UserID: "user-2", Type: api.TaskType_TASK_TYPE_DIFFUSION},
		{ID: "task-3", UserID: "user-3", Type: api.TaskType_TASK_TYPE_LLM},
	}

	result := SortByPriority(tasks, scheduler)

	// All should have positive priority
	for _, twp := range result {
		if twp.Priority <= 0 {
			t.Errorf("Priority should be positive, got %f", twp.Priority)
		}
	}
}
