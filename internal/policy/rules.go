// Package policy provides scheduling rules for GPU tasks.
package policy

import (
	"time"

	"algogpu/api"
	"algogpu/pkg/types"
)

// Rules defines scheduling policy rules based on historical data.
type Rules struct {
	// Shorter tasks get higher priority
	// Base priority = max_priority - (estimated_duration / priority_factor)
	shortTaskBonus     float64
	shortTaskThreshold time.Duration

	// Longer queue wait increases priority
	// Queue bonus = queue_wait * queue_factor
	queueWaitFactor float64

	// GPU packing preference
	packingPreference float64
}

// NewRules creates default scheduling rules.
func NewRules() *Rules {
	return &Rules{
		shortTaskBonus:     10.0,
		shortTaskThreshold: 30 * time.Second,
		queueWaitFactor:    0.1,
		packingPreference:  1.0,
	}
}

// ApplyRules applies policy rules to determine task priority.
func (r *Rules) ApplyRules(task *types.Task, prediction *types.ResourcePrediction, queueSize int) float64 {
	priority := 0.0

	// Rule 1: Short task bonus
	estimatedDuration := time.Duration(prediction.EstimatedDurationMs) * time.Millisecond
	if estimatedDuration < r.shortTaskThreshold {
		bonus := r.shortTaskBonus * (1.0 - float64(estimatedDuration)/float64(r.shortTaskThreshold))
		priority += bonus
	}

	// Rule 2: Queue wait penalty
	// Tasks that have been waiting longer get higher priority
	queueWait := time.Since(task.CreatedAt)
	if queueWait > 0 {
		priority += float64(queueWait.Milliseconds()) * r.queueWaitFactor
	}

	// Rule 3: Task type weighting
	taskTypeWeight := r.getTaskTypeWeight(task.Type)
	priority += taskTypeWeight

	// Rule 4: GPU packing preference
	if prediction.AllowPacking {
		priority += r.packingPreference
	}

	// Rule 5: Queue size adjustment
	// Higher queue size increases priority for all tasks
	if queueSize > 0 {
		queuePenalty := float64(queueSize) * 0.5
		priority += queuePenalty
	}

	return priority
}

// getTaskTypeWeight returns weight for task type.
func (r *Rules) getTaskTypeWeight(taskType api.TaskType) float64 {
	switch taskType {
	case api.TaskType_TASK_TYPE_EMBEDDING:
		return 1.0
	case api.TaskType_TASK_TYPE_LLM:
		return 2.0
	case api.TaskType_TASK_TYPE_DIFFUSION:
		return 1.5
	default:
		return 1.0
	}
}

// ShouldPreempt determines if a new task should preempt a running task.
// For now, we don't support preemption to keep it simple.
func (r *Rules) ShouldPreempt(newTask *types.Task, runningTask *types.Task) bool {
	// No preemption for simplicity
	return false
}

// ShouldReject determines if a task should be rejected based on policy.
func (r *Rules) ShouldReject(task *types.Task, queueSize int) bool {
	// Reject if queue is too large
	if queueSize > 1000 {
		return true
	}

	// Reject if estimated duration is too long
	if task.EstimatedRuntimeMs > 3600000 { // 1 hour
		return true
	}

	return false
}

// GetPriority returns the calculated priority for a task.
func (r *Rules) GetPriority(task *types.Task, prediction *predictor.ResourcePrediction, queueSize int) float64 {
	return r.ApplyRules(task, prediction, queueSize)
}
