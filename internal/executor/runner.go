package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/pkg/types"
)

// TaskExecutor defines the interface for executing GPU tasks
type TaskExecutor interface {
	// Execute runs a task and returns the result
	Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error)

	// Type returns the task type this executor handles
	Type() api.TaskType

	// Close releases resources
	Close() error
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID    string
	Status    api.TaskStatus
	Output    []byte
	Error     string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	GPUUsed   int64 // MB
}

// TaskRunner manages task execution
type TaskRunner struct {
	executors  map[api.TaskType]TaskExecutor
	gpuPool    *gpu.Pool
	resultChan chan *TaskResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewTaskRunner creates a new TaskRunner
func NewTaskRunner(gpuPool *gpu.Pool) *TaskRunner {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskRunner{
		executors:  make(map[api.TaskType]TaskExecutor),
		gpuPool:    gpuPool,
		resultChan: make(chan *TaskResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// RegisterExecutor registers a task executor
func (r *TaskRunner) RegisterExecutor(executor TaskExecutor) {
	r.executors[executor.Type()] = executor
	log.Printf("Registered executor for task type: %v", executor.Type())
}

// RunTask executes a task on the specified GPU
func (r *TaskRunner) RunTask(task *types.Task, gpuInfo *gpu.GPU) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		result := &TaskResult{
			TaskID:    task.ID,
			StartTime: time.Now(),
			GPUUsed:   task.GPUMemoryRequired,
		}

		executor, ok := r.executors[task.Type]
		if !ok {
			result.Status = api.TaskStatus_TASK_STATUS_FAILED
			result.Error = fmt.Sprintf("no executor for task type: %v", task.Type)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			r.resultChan <- result
			return
		}

		// Execute the task
		taskResult, err := executor.Execute(r.ctx, task, gpuInfo)
		if err != nil {
			result.Status = api.TaskStatus_TASK_STATUS_FAILED
			result.Error = err.Error()
		} else if taskResult != nil {
			result.Output = taskResult.Output
			result.Status = taskResult.Status
		}

		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		// Free GPU memory
		gpuInfo.Free(task.ID, task.GPUMemoryRequired)

		r.resultChan <- result
	}()
}

// ResultChan returns the result channel
func (r *TaskRunner) ResultChan() <-chan *TaskResult {
	return r.resultChan
}

// Stop stops the task runner
func (r *TaskRunner) Stop() {
	r.cancel()
	r.wg.Wait()
	close(r.resultChan)
}

// Close closes all executors
func (r *TaskRunner) Close() error {
	for _, executor := range r.executors {
		if err := executor.Close(); err != nil {
			return err
		}
	}
	return nil
}

// DefaultExecutors returns default set of executors
func DefaultExecutors() []TaskExecutor {
	return []TaskExecutor{
		NewEmbeddingExecutor(),
		NewLLMExecutor(),
		NewDiffusionExecutor(),
		NewGenericExecutor(),
	}
}

// embeddingExecutor executes embedding tasks
type embeddingExecutor struct{}

// NewEmbeddingExecutor creates a new embedding executor
func NewEmbeddingExecutor() TaskExecutor {
	return &embeddingExecutor{}
}

func (e *embeddingExecutor) Type() api.TaskType {
	return api.TaskType_TASK_TYPE_EMBEDDING
}

func (e *embeddingExecutor) Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error) {
	// Simulate embedding task execution
	time.Sleep(20 * time.Millisecond)

	log.Printf("Embedding task %s executed on GPU %d", task.ID, gpuInfo.ID)

	return &TaskResult{
		TaskID:   task.ID,
		Status:   api.TaskStatus_TASK_STATUS_COMPLETED,
		Output:   []byte(`{"embedding": [0.1, 0.2, 0.3]}`),
		Duration: 20 * time.Millisecond,
	}, nil
}

func (e *embeddingExecutor) Close() error {
	return nil
}

// llmExecutor executes LLM inference tasks
type llmExecutor struct{}

// NewLLMExecutor creates a new LLM executor
func NewLLMExecutor() TaskExecutor {
	return &llmExecutor{}
}

func (e *llmExecutor) Type() api.TaskType {
	return api.TaskType_TASK_TYPE_LLM
}

func (e *llmExecutor) Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error) {
	// Simulate LLM inference
	time.Sleep(100 * time.Millisecond)

	log.Printf("LLM task %s executed on GPU %d", task.ID, gpuInfo.ID)

	// Parse payload to get prompt
	var payload map[string]interface{}
	if len(task.Payload) > 0 {
		_ = json.Unmarshal(task.Payload, &payload)
	}

	return &TaskResult{
		TaskID:   task.ID,
		Status:   api.TaskStatus_TASK_STATUS_COMPLETED,
		Output:   []byte(`{"text": "generated response"}`),
		Duration: 100 * time.Millisecond,
	}, nil
}

func (e *llmExecutor) Close() error {
	return nil
}

// diffusionExecutor executes diffusion tasks
type diffusionExecutor struct{}

// NewDiffusionExecutor creates a new diffusion executor
func NewDiffusionExecutor() TaskExecutor {
	return &diffusionExecutor{}
}

func (e *diffusionExecutor) Type() api.TaskType {
	return api.TaskType_TASK_TYPE_DIFFUSION
}

func (e *diffusionExecutor) Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error) {
	// Simulate diffusion task
	time.Sleep(500 * time.Millisecond)

	log.Printf("Diffusion task %s executed on GPU %d", task.ID, gpuInfo.ID)

	return &TaskResult{
		TaskID:   task.ID,
		Status:   api.TaskStatus_TASK_STATUS_COMPLETED,
		Output:   []byte(`{"image": "base64 encoded image"}`),
		Duration: 500 * time.Millisecond,
	}, nil
}

func (e *diffusionExecutor) Close() error {
	return nil
}

// genericExecutor executes generic GPU tasks
type genericExecutor struct{}

// NewGenericExecutor creates a new generic executor
func NewGenericExecutor() TaskExecutor {
	return &genericExecutor{}
}

func (e *genericExecutor) Type() api.TaskType {
	return api.TaskType_TASK_TYPE_OTHER
}

func (e *genericExecutor) Execute(ctx context.Context, task *types.Task, gpuInfo *gpu.GPU) (*TaskResult, error) {
	// Generic task execution
	time.Sleep(50 * time.Millisecond)

	log.Printf("Generic task %s executed on GPU %d", task.ID, gpuInfo.ID)

	return &TaskResult{
		TaskID:   task.ID,
		Status:   api.TaskStatus_TASK_STATUS_COMPLETED,
		Output:   []byte(`{"result": "ok"}`),
		Duration: 50 * time.Millisecond,
	}, nil
}

func (e *genericExecutor) Close() error {
	return nil
}
