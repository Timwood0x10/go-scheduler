package scheduler

import (
	"testing"

	"algogpu/internal/queue"
	"algogpu/pkg/types"
)

func TestAdmissionControl_Check(t *testing.T) {
	q := queue.NewTaskQueue()
	admission := NewAdmissionControl(2, q)

	// Should pass when queue is not full
	if err := admission.Check(); err != nil {
		t.Errorf("Admission check should pass: %v", err)
	}

	// Fill the queue
	_ = q.Enqueue(&types.Task{ID: "task-1"})
	_ = q.Enqueue(&types.Task{ID: "task-2"})

	// Should fail when queue is full
	if err := admission.Check(); err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

func TestAdmissionControl_GetQueueUsage(t *testing.T) {
	q := queue.NewTaskQueue()
	admission := NewAdmissionControl(10, q)

	_ = q.Enqueue(&types.Task{ID: "task-1"})

	usage := admission.GetQueueUsage()
	if usage != 0.1 {
		t.Errorf("Expected usage 0.1, got %f", usage)
	}
}

func TestAdmissionControl_SetMaxQueueSize(t *testing.T) {
	q := queue.NewTaskQueue()
	admission := NewAdmissionControl(5, q)

	admission.SetMaxQueueSize(10)
	if admission.GetMaxQueueSize() != 10 {
		t.Errorf("Expected max queue size 10, got %d", admission.GetMaxQueueSize())
	}
}
