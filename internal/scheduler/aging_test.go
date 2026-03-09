package scheduler

import (
	"testing"
	"time"

	"algogpu/pkg/types"
)

func TestTaskAging_CalculateAgingPriority(t *testing.T) {
	aging := NewTaskAging(0.1)

	task := &types.Task{
		ID:        "task-1",
		UserID:    "user-1",
		CreatedAt: time.Now().Add(-10 * time.Second), // 10 seconds ago
	}

	// basePriority + wait_time * γ
	priority := aging.CalculateAgingPriority(1.0, task)

	// 1.0 + 10 * 0.1 = 2.0
	if priority < 1.0 {
		t.Errorf("Priority should increase with wait time, got %f", priority)
	}
}

func TestTaskAging_GetWaitTime(t *testing.T) {
	aging := NewTaskAging(0.1)

	task := &types.Task{
		ID:        "task-1",
		CreatedAt: time.Now().Add(-5 * time.Second),
	}

	waitTime := aging.GetWaitTime(task)

	if waitTime < 4*time.Second || waitTime > 6*time.Second {
		t.Errorf("Expected wait time around 5 seconds, got %v", waitTime)
	}
}

func TestTaskAging_IsStale(t *testing.T) {
	aging := NewTaskAging(0.1)

	// Task waiting less than max wait
	task1 := &types.Task{
		ID:        "task-1",
		CreatedAt: time.Now().Add(-30 * time.Second),
	}

	if aging.IsStale(task1, 1*time.Minute) {
		t.Error("Task should not be stale")
	}

	// Task waiting more than max wait
	task2 := &types.Task{
		ID:        "task-2",
		CreatedAt: time.Now().Add(-2 * time.Minute),
	}

	if !aging.IsStale(task2, 1*time.Minute) {
		t.Error("Task should be stale")
	}
}
