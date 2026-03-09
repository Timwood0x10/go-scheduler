package queue

import (
	"container/list"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/pkg/types"
)

// TaskQueue is a thread-safe task queue with priority support
type TaskQueue struct {
	mu      sync.RWMutex
	pending *list.List               // pending tasks
	running map[string]*types.Task   // running tasks
	tasks   map[string]*list.Element // task_id -> element

	// metrics
	queueSize    int
	runningCount int
}

// NewTaskQueue creates a new TaskQueue
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		pending: list.New(),
		running: make(map[string]*types.Task),
		tasks:   make(map[string]*list.Element),
	}
}

// Enqueue adds a task to the queue
func (q *TaskQueue) Enqueue(task *types.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.tasks[task.ID]; exists {
		return ErrTaskExists
	}

	elem := q.pending.PushBack(task)
	q.tasks[task.ID] = elem
	task.Status = api.TaskStatus_TASK_STATUS_PENDING
	task.CreatedAt = time.Now()
	q.queueSize++

	return nil
}

// Dequeue removes and returns the highest priority task
func (q *TaskQueue) Dequeue() *types.Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.pending.Len() == 0 {
		return nil
	}

	elem := q.pending.Front()
	task := elem.Value.(*types.Task)
	q.pending.Remove(elem)
	delete(q.tasks, task.ID)
	q.queueSize--

	return task
}

// Get returns a task by ID
func (q *TaskQueue) Get(taskID string) (*types.Task, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// check running
	if task, ok := q.running[taskID]; ok {
		return task, true
	}

	// check pending
	if elem, ok := q.tasks[taskID]; ok {
		return elem.Value.(*types.Task), true
	}

	return nil, false
}

// UpdateStatus updates task status
func (q *TaskQueue) UpdateStatus(taskID string, status api.TaskStatus) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.running[taskID]
	if !ok {
		if elem, ok := q.tasks[taskID]; ok {
			task = elem.Value.(*types.Task)
		} else {
			return ErrTaskNotFound
		}
	}

	oldStatus := task.Status
	task.Status = status

	// handle status transitions
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

// Cancel cancels a pending or running task
func (q *TaskQueue) Cancel(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// check running
	if task, ok := q.running[taskID]; ok {
		task.Status = api.TaskStatus_TASK_STATUS_CANCELLED
		task.CompletedAt = time.Now()
		delete(q.running, taskID)
		q.runningCount--
		return nil
	}

	// check pending
	if elem, ok := q.tasks[taskID]; ok {
		task := elem.Value.(*types.Task)
		task.Status = api.TaskStatus_TASK_STATUS_CANCELLED
		task.CompletedAt = time.Now()
		q.pending.Remove(elem)
		delete(q.tasks, taskID)
		q.queueSize--
		return nil
	}

	return ErrTaskNotFound
}

// GetAllPending returns all pending tasks
func (q *TaskQueue) GetAllPending() []*types.Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*types.Task, 0, q.pending.Len())
	for elem := q.pending.Front(); elem != nil; elem = elem.Next() {
		tasks = append(tasks, elem.Value.(*types.Task))
	}

	return tasks
}

// GetAllRunning returns all running tasks
func (q *TaskQueue) GetAllRunning() []*types.Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*types.Task, 0, len(q.running))
	for _, task := range q.running {
		tasks = append(tasks, task)
	}

	return tasks
}

// Len returns the number of pending tasks
func (q *TaskQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.queueSize
}

// RunningCount returns the number of running tasks
func (q *TaskQueue) RunningCount() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.runningCount
}

// Requeue moves a task back to pending queue
func (q *TaskQueue) Requeue(task *types.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.tasks[task.ID]; exists {
		return ErrTaskExists
	}

	task.Status = api.TaskStatus_TASK_STATUS_PENDING
	elem := q.pending.PushBack(task)
	q.tasks[task.ID] = elem
	q.queueSize++

	return nil
}

// Errors
var (
	ErrTaskNotFound = &Error{"task not found"}
	ErrTaskExists   = &Error{"task already exists"}
)

// Error represents a queue-related error
type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}
