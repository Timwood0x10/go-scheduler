package queue

import (
	"testing"
	"time"

	"algogpu/api"
	"algogpu/pkg/types"
)

func TestNewTaskQueue(t *testing.T) {
	q := NewTaskQueue()

	if q == nil {
		t.Fatal("NewTaskQueue() returned nil")
	}

	if q.heap == nil {
		t.Error("NewTaskQueue() did not initialize heap")
	}

	if q.running == nil {
		t.Error("NewTaskQueue() did not initialize running map")
	}

	if q.tasks == nil {
		t.Error("NewTaskQueue() did not initialize tasks map")
	}
}

func TestTaskQueue_Enqueue(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := q.Enqueue(task)

	if err != nil {
		t.Errorf("Enqueue() error = %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Queue length = %d, want 1", q.Len())
	}

	// Try to enqueue same task again
	err = q.Enqueue(task)

	if err != ErrTaskExists {
		t.Errorf("Enqueue() duplicate error = %v, want ErrTaskExists", err)
	}
}

func TestTaskQueue_Dequeue(t *testing.T) {
	q := NewTaskQueue()

	// Dequeue from empty queue
	task := q.Dequeue()

	if task != nil {
		t.Error("Dequeue() should return nil for empty queue")
	}

	// Enqueue tasks
	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		err := q.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	// Dequeue should return tasks in priority order
	for i := 0; i < 3; i++ {
		task = q.Dequeue()

		if task == nil {
			t.Error("Dequeue() should not return nil")
		}

		if task.Status != api.TaskStatus_TASK_STATUS_PENDING {
			t.Errorf("Task status = %v, want PENDING", task.Status)
		}
	}

	// Queue should be empty
	if q.Len() != 0 {
		t.Errorf("Queue length = %d, want 0", q.Len())
	}
}

func TestTaskQueue_Get(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := q.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Get existing task
	retrievedTask, exists := q.Get("task-1")

	if !exists {
		t.Error("Get() should find task-1")
	}

	if retrievedTask == nil {
		t.Fatal("Get() returned nil")
	}

	if retrievedTask.ID != "task-1" {
		t.Errorf("Task ID = %s, want task-1", retrievedTask.ID)
	}

	// Get non-existent task
	_, exists = q.Get("non-existent")

	if exists {
		t.Error("Get() should not find non-existent task")
	}
}

func TestTaskQueue_UpdateStatus(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := q.Enqueue(task)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Update to RUNNING
	err = q.UpdateStatus("task-1", api.TaskStatus_TASK_STATUS_RUNNING)

	if err != nil {
		t.Errorf("UpdateStatus() error = %v", err)
	}

	if q.RunningCount() != 1 {
		t.Errorf("Running count = %d, want 1", q.RunningCount())
	}

	retrievedTask, _ := q.Get("task-1")

	if retrievedTask.Status != api.TaskStatus_TASK_STATUS_RUNNING {
		t.Errorf("Task status = %v, want RUNNING", retrievedTask.Status)
	}

	if retrievedTask.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}

	// Update to COMPLETED
	err = q.UpdateStatus("task-1", api.TaskStatus_TASK_STATUS_COMPLETED)

	if err != nil {
		t.Errorf("UpdateStatus() error = %v", err)
	}

	if q.RunningCount() != 0 {
		t.Errorf("Running count = %d, want 0", q.RunningCount())
	}

	retrievedTask, _ = q.Get("task-1")

	if retrievedTask.Status != api.TaskStatus_TASK_STATUS_COMPLETED {
		t.Errorf("Task status = %v, want COMPLETED", retrievedTask.Status)
	}

	if retrievedTask.CompletedAt.IsZero() {
		t.Error("CompletedAt should be set")
	}

	// Try to update non-existent task
	err = q.UpdateStatus("non-existent", api.TaskStatus_TASK_STATUS_RUNNING)

	if err != ErrTaskNotFound {
		t.Errorf("UpdateStatus() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskQueue_Cancel(t *testing.T) {
	q := NewTaskQueue()

	// Cancel pending task
	task1 := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := q.Enqueue(task1)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	err = q.Cancel("task-1")

	if err != nil {
		t.Errorf("Cancel() error = %v", err)
	}

	if q.Len() != 0 {
		t.Errorf("Queue length = %d, want 0", q.Len())
	}

	retrievedTask, _ := q.Get("task-1")

	if retrievedTask.Status != api.TaskStatus_TASK_STATUS_CANCELLED {
		t.Errorf("Task status = %v, want CANCELLED", retrievedTask.Status)
	}

	// Cancel running task
	task2 := &types.Task{
		ID:                 "task-2",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err = q.Enqueue(task2)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	err = q.UpdateStatus("task-2", api.TaskStatus_TASK_STATUS_RUNNING)
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	err = q.Cancel("task-2")

	if err != nil {
		t.Errorf("Cancel() error = %v", err)
	}

	if q.RunningCount() != 0 {
		t.Errorf("Running count = %d, want 0", q.RunningCount())
	}

	// Try to cancel non-existent task
	err = q.Cancel("non-existent")

	if err != ErrTaskNotFound {
		t.Errorf("Cancel() error = %v, want ErrTaskNotFound", err)
	}
}

func TestTaskQueue_GetAllPending(t *testing.T) {
	q := NewTaskQueue()

	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		err := q.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	pending := q.GetAllPending()

	if len(pending) != 3 {
		t.Errorf("GetAllPending() returned %d tasks, want 3", len(pending))
	}

	for _, task := range pending {
		if task.Status != api.TaskStatus_TASK_STATUS_PENDING {
			t.Errorf("Task status = %v, want PENDING", task.Status)
		}
	}
}

func TestTaskQueue_GetAllRunning(t *testing.T) {
	q := NewTaskQueue()

	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		err := q.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}

		err = q.UpdateStatus(task.ID, api.TaskStatus_TASK_STATUS_RUNNING)
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
	}

	running := q.GetAllRunning()

	if len(running) != 3 {
		t.Errorf("GetAllRunning() returned %d tasks, want 3", len(running))
	}

	for _, task := range running {
		if task.Status != api.TaskStatus_TASK_STATUS_RUNNING {
			t.Errorf("Task status = %v, want RUNNING", task.Status)
		}
	}
}

func TestTaskQueue_Len(t *testing.T) {
	q := NewTaskQueue()

	if q.Len() != 0 {
		t.Errorf("Len() = %d, want 0", q.Len())
	}

	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		err := q.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}
	}

	if q.Len() != 3 {
		t.Errorf("Len() = %d, want 3", q.Len())
	}
}

func TestTaskQueue_RunningCount(t *testing.T) {
	q := NewTaskQueue()

	if q.RunningCount() != 0 {
		t.Errorf("RunningCount() = %d, want 0", q.RunningCount())
	}

	for i := 0; i < 3; i++ {
		task := &types.Task{
			ID:                 string(rune('0' + i)),
			UserID:             "user-1",
			Type:               api.TaskType_TASK_TYPE_EMBEDDING,
			GPUMemoryRequired:  1024,
			GPUComputeRequired: 100,
			EstimatedRuntimeMs: 1000,
		}

		err := q.Enqueue(task)
		if err != nil {
			t.Fatalf("Enqueue() error = %v", err)
		}

		err = q.UpdateStatus(task.ID, api.TaskStatus_TASK_STATUS_RUNNING)
		if err != nil {
			t.Fatalf("UpdateStatus() error = %v", err)
		}
	}

	if q.RunningCount() != 3 {
		t.Errorf("RunningCount() = %d, want 3", q.RunningCount())
	}
}

func TestTaskQueue_Requeue(t *testing.T) {
	q := NewTaskQueue()

	task := &types.Task{
		ID:                 "task-1",
		UserID:             "user-1",
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  1024,
		GPUComputeRequired: 100,
		EstimatedRuntimeMs: 1000,
	}

	err := q.Requeue(task)

	if err != nil {
		t.Errorf("Requeue() error = %v", err)
	}

	if q.Len() != 1 {
		t.Errorf("Queue length = %d, want 1", q.Len())
	}

	// Try to requeue same task
	err = q.Requeue(task)

	if err != ErrTaskExists {
		t.Errorf("Requeue() duplicate error = %v, want ErrTaskExists", err)
	}
}

func TestTaskHeap_Less(t *testing.T) {
	heap := &TaskHeap{}

	now := time.Now()

	task1 := &types.Task{
		ID:        "task-1",
		Type:      api.TaskType_TASK_TYPE_EMBEDDING,
		CreatedAt: now,
	}

	task2 := &types.Task{
		ID:        "task-2",
		Type:      api.TaskType_TASK_TYPE_LLM,
		CreatedAt: now,
	}

	task3 := &types.Task{
		ID:        "task-3",
		Type:      api.TaskType_TASK_TYPE_EMBEDDING,
		CreatedAt: now.Add(time.Second),
	}

	*heap = append(*heap, task1, task2, task3)

	// EMBEDDING should come before LLM
	if !heap.Less(0, 1) {
		t.Error("EMBEDDING should have higher priority than LLM")
	}

	// Older EMBEDDING should come before newer
	if !heap.Less(0, 2) {
		t.Error("Older task should have higher priority")
	}
}

func TestError(t *testing.T) {
	err := &Error{"test error"}

	if err.Error() != "test error" {
		t.Errorf("Error() = %s, want 'test error'", err.Error())
	}

	if ErrTaskNotFound.Error() != "task not found" {
		t.Errorf("ErrTaskNotFound message incorrect")
	}

	if ErrTaskExists.Error() != "task already exists" {
		t.Errorf("ErrTaskExists message incorrect")
	}
}
