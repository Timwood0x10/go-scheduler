// Package queue provides a thread-safe priority task queue using container/heap.
// This implementation follows the principle of simplicity by using the standard library.
package queue

import (
	"container/heap"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/pkg/types"
)

// TaskQueue is a thread-safe priority queue for GPU tasks.
// It uses container/heap for efficient priority operations.
type TaskQueue struct {
	mu      sync.RWMutex
	heap    *TaskHeap              // pending tasks (priority heap)
	running map[string]*types.Task // running tasks
	tasks   map[string]bool        // task_id -> exists (for O(1) lookup)

	// metrics
	queueSize    int
	runningCount int
}

// NewTaskQueue creates a new task queue.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		heap:    &TaskHeap{},
		running: make(map[string]*types.Task),
		tasks:   make(map[string]bool),
	}
}

// Enqueue adds a task to the priority queue.
func (q *TaskQueue) Enqueue(task *types.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.tasks[task.ID]; exists {
		return ErrTaskExists
	}

	task.Status = api.TaskStatus_TASK_STATUS_PENDING
	task.CreatedAt = time.Now()

	heap.Push(q.heap, task)
	q.tasks[task.ID] = true
	q.queueSize++

	return nil
}

// Dequeue removes and returns the highest priority task.
func (q *TaskQueue) Dequeue() *types.Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.heap.Len() == 0 {
		return nil
	}

	task := heap.Pop(q.heap).(*types.Task)
	delete(q.tasks, task.ID)
	q.queueSize--

	return task
}

// Get returns a task by ID.
func (q *TaskQueue) Get(taskID string) (*types.Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Check running tasks
	if task, ok := q.running[taskID]; ok {
		return task, true
	}

	// Check heap (requires linear search, but this is acceptable for lookups)
	for _, task := range *q.heap {
		if task.ID == taskID {
			return task, true
		}
	}

	return nil, false
}

// UpdateStatus updates task status and manages running/pending state.
func (q *TaskQueue) UpdateStatus(taskID string, status api.TaskStatus) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.running[taskID]
	if !ok {
		// Task might be in the heap
		for _, t := range *q.heap {
			if t.ID == taskID {
				task = t
				break
			}
		}
		if task == nil {
			return ErrTaskNotFound
		}
	}

	oldStatus := task.Status
	task.Status = status

	// Handle status transitions
	switch status {
	case api.TaskStatus_TASK_STATUS_RUNNING:
		if oldStatus == api.TaskStatus_TASK_STATUS_PENDING {
			task.StartedAt = time.Now()
			q.running[taskID] = task
			q.runningCount++
		}
	case api.TaskStatus_TASK_STATUS_COMPLETED,
		api.TaskStatus_TASK_STATUS_FAILED,
		api.TaskStatus_TASK_STATUS_CANCELLED:
		if oldStatus == api.TaskStatus_TASK_STATUS_RUNNING {
			task.CompletedAt = time.Now()
			delete(q.running, taskID)
			q.runningCount--
		}
	}

	return nil
}

// Cancel cancels a pending or running task.
func (q *TaskQueue) Cancel(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Check running tasks
	if task, ok := q.running[taskID]; ok {
		task.Status = api.TaskStatus_TASK_STATUS_CANCELLED
		task.CompletedAt = time.Now()
		delete(q.running, taskID)
		q.runningCount--
		return nil
	}

	// Check heap
	for i, task := range *q.heap {
		if task.ID == taskID {
			task.Status = api.TaskStatus_TASK_STATUS_CANCELLED
			task.CompletedAt = time.Now()
			heap.Remove(q.heap, i)
			delete(q.tasks, taskID)
			q.queueSize--
			return nil
		}
	}

	return ErrTaskNotFound
}

// GetAllPending returns all pending tasks (priority order).
func (q *TaskQueue) GetAllPending() []*types.Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*types.Task, q.heap.Len())
	copy(tasks, *q.heap)

	return tasks
}

// GetAllRunning returns all running tasks.
func (q *TaskQueue) GetAllRunning() []*types.Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*types.Task, 0, len(q.running))
	for _, task := range q.running {
		tasks = append(tasks, task)
	}

	return tasks
}

// Len returns the number of pending tasks.
func (q *TaskQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.queueSize
}

// RunningCount returns the number of running tasks.
func (q *TaskQueue) RunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.runningCount
}

// Requeue moves a task back to pending queue.
func (q *TaskQueue) Requeue(task *types.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.tasks[task.ID]; exists {
		return ErrTaskExists
	}

	task.Status = api.TaskStatus_TASK_STATUS_PENDING
	heap.Push(q.heap, task)
	q.tasks[task.ID] = true
	q.queueSize++

	return nil
}

// TaskHeap implements heap.Interface for priority queue.
type TaskHeap []*types.Task

func (h TaskHeap) Len() int { return len(h) }

func (h TaskHeap) Less(i, j int) bool {
	// Priority: TaskType > CreationTime
	// Higher priority task types should come first
	if h[i].Type != h[j].Type {
		return h[i].Type < h[j].Type
	}
	// For same type, older tasks first
	return h[i].CreatedAt.Before(h[j].CreatedAt)
}

func (h TaskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *TaskHeap) Push(x interface{}) {
	*h = append(*h, x.(*types.Task))
}

func (h *TaskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// Errors
var (
	ErrTaskNotFound = &Error{"task not found"}
	ErrTaskExists   = &Error{"task already exists"}
)

// Error represents a queue-related error.
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
