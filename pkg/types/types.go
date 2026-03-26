package types

import (
	"time"

	"algogpu/api"
)

// Task represents a GPU task in the scheduler
type Task struct {
	ID                 string
	UserID             string
	Type               api.TaskType
	GPUMemoryRequired  int64 // MB
	GPUComputeRequired int64 // TFLOPS
	EstimatedRuntimeMs int64
	Priority           int64
	Payload            []byte
	Status             api.TaskStatus
	CreatedAt          time.Time
	StartedAt          time.Time
	CompletedAt        time.Time
	Message            string
}

// ToProto converts Task to protobuf message
func (t *Task) ToProto() *api.TaskRequest {
	return &api.TaskRequest{
		TaskId:             t.ID,
		UserId:             t.UserID,
		TaskType:           t.Type,
		GpuMemoryRequired:  t.GPUMemoryRequired,
		GpuComputeRequired: t.GPUComputeRequired,
		EstimatedRuntimeMs: t.EstimatedRuntimeMs,
		Payload:            t.Payload,
	}
}

// TaskFromProto creates Task from protobuf message
func TaskFromProto(req *api.TaskRequest) *Task {
	return &Task{
		ID:                 req.TaskId,
		UserID:             req.UserId,
		Type:               req.TaskType,
		GPUMemoryRequired:  req.GpuMemoryRequired,
		GPUComputeRequired: req.GpuComputeRequired,
		EstimatedRuntimeMs: req.EstimatedRuntimeMs,
		Payload:            req.Payload,
		Status:             api.TaskStatus_TASK_STATUS_PENDING,
		CreatedAt:          time.Now(),
	}
}

// GetTaskCost returns the GPU cost for the task type
func GetTaskCost(taskType api.TaskType) int64 {
	switch taskType {
	case api.TaskType_TASK_TYPE_EMBEDDING:
		return 1
	case api.TaskType_TASK_TYPE_LLM:
		return 5
	case api.TaskType_TASK_TYPE_DIFFUSION:
		return 10
	default:
		return 1
	}
}

// ExecutionMetrics represents metrics collected during task execution.
type ExecutionMetrics struct {
	TaskID          string
	QueueWaitMs     int64
	ExecutionTimeMs int64
	AvgGPUUtil      float64
	MaxGPUUtil      float64
	AvgMemUtil      float64
	MaxMemUtil      float64
	GPUMemoryUsedMB int64
	Success         bool
}

// ResourcePrediction represents predicted resource requirements for a task.
type ResourcePrediction struct {
	EstimatedDurationMs int64
	EstimatedMemoryMB   int64
	EstimatedGpuUtil    float64
	AllowPacking        bool
}

// PolicyDecision represents scheduling policy decision for a task.
type PolicyDecision struct {
	Priority             int
	EstimatedDuration    int64
	EstimatedMemoryMB    int
	AllowPacking         bool
	EstimatedQueueWaitMs int64
}
