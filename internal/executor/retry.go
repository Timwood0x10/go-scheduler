package executor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子
}

// DefaultRetryPolicy returns default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// TaskError represents a task execution error
type TaskError struct {
	TaskID     string
	Err        error
	Retriable  bool
	RetryCount int
	Timestamp  time.Time
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("task %s failed after %d retries: %v", e.TaskID, e.RetryCount, e.Err)
}

// IsRetriable returns true if the error is retriable
func (e *TaskError) IsRetriable() bool {
	return e.Retriable
}

// RetryableError creates a retriable error
func RetryableError(taskID string, err error, retryCount int) *TaskError {
	return &TaskError{
		TaskID:     taskID,
		Err:        err,
		Retriable:  true,
		RetryCount: retryCount,
		Timestamp:  time.Now(),
	}
}

// NonRetryableError creates a non-retriable error
func NonRetryableError(taskID string, err error) *TaskError {
	return &TaskError{
		TaskID:     taskID,
		Err:        err,
		Retriable:  false,
		RetryCount: 0,
		Timestamp:  time.Now(),
	}
}

// RetryExecutor wraps a TaskExecutor with retry logic
type RetryExecutor struct {
	executor   TaskExecutor
	policy     *RetryPolicy
	maxRetries int
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(executor TaskExecutor, policy *RetryPolicy) *RetryExecutor {
	return &RetryExecutor{
		executor:   executor,
		policy:     policy,
		maxRetries: policy.MaxRetries,
	}
}

// Type returns the task type
func (r *RetryExecutor) Type() api.TaskType {
	return r.executor.Type()
}

// Execute executes with retry
func (r *RetryExecutor) Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return &TaskResult{
				TaskID: task.ID,
				Status: api.TaskStatus_TASK_STATUS_FAILED,
				Error:  fmt.Sprintf("context cancelled: %v", ctx.Err()),
			}, ctx.Err()
		default:
		}

		result, err := r.executor.Execute(ctx, task, gpuInfo)

		if err == nil {
			return result, nil
		}

		lastErr = err

		// Check if error is retriable
		if taskErr, ok := err.(*TaskError); ok && !taskErr.IsRetriable() {
			return &TaskResult{
				TaskID: task.ID,
				Status: api.TaskStatus_TASK_STATUS_FAILED,
				Error:  err.Error(),
			}, err
		}

		// Log retry attempt
		log.Printf("Task %s failed (attempt %d/%d): %v, retrying...",
			task.ID, attempt+1, r.maxRetries+1, err)

		// Wait before retry with exponential backoff
		if attempt < r.maxRetries {
			delay := r.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return &TaskResult{
					TaskID: task.ID,
					Status: api.TaskStatus_TASK_STATUS_FAILED,
					Error:  fmt.Sprintf("context cancelled: %v", ctx.Err()),
				}, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	// All retries exhausted
	return &TaskResult{
		TaskID: task.ID,
		Status: api.TaskStatus_TASK_STATUS_FAILED,
		Error:  fmt.Sprintf("task failed after %d retries: %v", r.maxRetries+1, lastErr),
	}, lastErr
}

// calculateDelay calculates delay for exponential backoff
func (r *RetryExecutor) calculateDelay(attempt int) time.Duration {
	delay := r.policy.InitialDelay * time.Duration(mathPow(r.policy.BackoffFactor, float64(attempt)))
	if delay > r.policy.MaxDelay {
		delay = r.policy.MaxDelay
	}
	return delay
}

// Close closes the executor
func (r *RetryExecutor) Close() error {
	return r.executor.Close()
}

// mathPow calculates base^exp
func mathPow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// ErrorHandler handles task errors
type ErrorHandler struct {
	mu         sync.RWMutex
	errorLog   []*TaskError
	maxLogSize int
	onError    func(*TaskError)
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(maxLogSize int, onError func(*TaskError)) *ErrorHandler {
	return &ErrorHandler{
		errorLog:   make([]*TaskError, 0),
		maxLogSize: maxLogSize,
		onError:    onError,
	}
}

// HandleError handles an error
func (h *ErrorHandler) HandleError(err *TaskError) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.errorLog = append(h.errorLog, err)

	// Trim log if too large
	if len(h.errorLog) > h.maxLogSize {
		h.errorLog = h.errorLog[len(h.errorLog)-h.maxLogSize:]
	}

	// Call error callback
	if h.onError != nil {
		h.onError(err)
	}
}

// GetErrorLog returns the error log
func (h *ErrorHandler) GetErrorLog() []*TaskError {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*TaskError, len(h.errorLog))
	copy(result, h.errorLog)
	return result
}

// GetErrorCount returns the number of errors
func (h *ErrorHandler) GetErrorCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.errorLog)
}

// Clear clears the error log
func (h *ErrorHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorLog = make([]*TaskError, 0)
}
