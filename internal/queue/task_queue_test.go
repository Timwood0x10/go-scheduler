package queue

import (
	"testing"

	"algogpu/api"
	"algogpu/pkg/types"
)

func TestTaskQueue_Enqueue(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	err := q.Enqueue(task)
	if err != nil {
		t.Errorf("Failed to enqueue task: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Expected queue size 1, got %d", q.Len())
	}
}

func TestTaskQueue_Enqueue_Duplicate(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	_ = q.Enqueue(task)
	err := q.Enqueue(task)

	if err != ErrTaskExists {
		t.Errorf("Expected ErrTaskExists, got %v", err)
	}
}

func TestTaskQueue_Dequeue(t *testing.T) {
	q := NewTaskQueue()

	task1 := &types.Task{ID: "task-1", UserID: "user-1"}
	task2 := &types.Task{ID: "task-2", UserID: "user-2"}

	_ = q.Enqueue(task1)
	_ = q.Enqueue(task2)

	dequeued := q.Dequeue()
	if dequeued == nil {
		t.Error("Expected to dequeue a task")
	}

	if q.Len() != 1 {
		t.Errorf("Expected queue size 1, got %d", q.Len())
	}
}

func TestTaskQueue_Get(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	_ = q.Enqueue(task)

	found, ok := q.Get("task-1")
	if !ok {
		t.Error("Task not found")
	}

	if found.ID != "task-1" {
		t.Errorf("Expected task ID task-1, got %s", found.ID)
	}
}

func TestTaskQueue_Get_NotFound(t *testing.T) {
	q := NewTaskQueue()

	_, ok := q.Get("nonexistent")
	if ok {
		t.Error("Expected task not found")
	}
}

func TestTaskQueue_Cancel(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	_ = q.Enqueue(task)
	err := q.Cancel("task-1")

	if err != nil {
		t.Errorf("Failed to cancel task: %v", err)
	}

	if q.Len() != 0 {
		t.Errorf("Expected queue size 0, got %d", q.Len())
	}
}

func TestTaskQueue_UpdateStatus(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	_ = q.Enqueue(task)

	err := q.UpdateStatus("task-1", api.TaskStatus_TASK_STATUS_RUNNING)
	if err != nil {
		t.Errorf("Failed to update status: %v", err)
	}

	task, _ = q.Get("task-1")
	if task.Status != api.TaskStatus_TASK_STATUS_RUNNING {
		t.Errorf("Expected status RUNNING, got %v", task.Status)
	}

	if task.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
}

func TestTaskQueue_GetAllPending(t *testing.T) {
	q := NewTaskQueue()

	_ = q.Enqueue(&types.Task{ID: "task-1", UserID: "user-1"})
	_ = q.Enqueue(&types.Task{ID: "task-2", UserID: "user-1"})
	_ = q.Enqueue(&types.Task{ID: "task-3", UserID: "user-2"})

	pending := q.GetAllPending()
	if len(pending) != 3 {
		t.Errorf("Expected 3 pending tasks, got %d", len(pending))
	}
}

func TestTaskQueue_Requeue(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:     "task-1",
		UserID: "user-1",
		Type:   api.TaskType_TASK_TYPE_LLM,
	}

	_ = q.Enqueue(task)
	q.Dequeue()

	err := q.Requeue(task)
	if err != nil {
		t.Errorf("Failed to requeue task: %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Expected queue size 1, got %d", q.Len())
	}
}
