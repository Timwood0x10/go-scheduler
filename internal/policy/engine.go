// Package policy provides data-driven scheduling policy decisions.
// It uses historical GPU execution data to optimize scheduling.
package policy

import (
	"context"
	"log"

	"algogpu/internal/predictor"
	"algogpu/pkg/types"
)

// Engine evaluates tasks and provides scheduling policy decisions.
type Engine struct {
	predictor *predictor.ResourcePredictor
	rules     *Rules
}

// NewEngine creates a new policy engine.
func NewEngine(predictor *predictor.ResourcePredictor) *Engine {
	return &Engine{
		predictor: predictor,
		rules:     NewRules(),
	}
}

// EvaluateTask evaluates a task and returns scheduling policy decision.
func (e *Engine) EvaluateTask(ctx context.Context, task *types.Task, queueSize int) (*types.PolicyDecision, error) {
	// Get canonical task type
	taskType := predictor.GetTaskTypeName(task.Type.String())

	// Predict resource requirements
	prediction, err := e.predictor.Predict(ctx, taskType)
	if err != nil {
		log.Printf("Failed to predict resources for task %s: %v", task.ID, err)
		// Use default prediction
		prediction = e.predictor.GetDefaultPrediction(taskType)
	}

	// Calculate predicted queue wait
	predictedQueueWait, err := e.predictor.GetPredictedQueueWait(ctx, taskType, queueSize)
	if err != nil {
		log.Printf("Failed to predict queue wait for task %s: %v", task.ID, err)
		predictedQueueWait = 0
	}

	// Build decision
	decision := &types.PolicyDecision{
		EstimatedDuration:   prediction.EstimatedDurationMs,
		EstimatedMemoryMB:   prediction.EstimatedMemoryMB,
		AllowPacking:        prediction.AllowPacking,
		EstimatedQueueWaitMs: predictedQueueWait,
	}

	// Apply policy rules to adjust priority
	decision.Priority = e.rules.ApplyRules(task, prediction, queueSize)

	return decision, nil
}

// EvaluateTaskWithMetadata evaluates task with additional metadata.
func (e *Engine) EvaluateTaskWithMetadata(ctx context.Context, task *types.Task, queueSize int, metadata map[string]string) (*types.PolicyDecision, error) {
	// For now, use basic evaluation
	// Future: use metadata for more sophisticated decisions
	return e.EvaluateTask(ctx, task, queueSize)
}

// GetTaskTypeStats retrieves statistics for a specific task type.
func (e *Engine) GetTaskTypeStats(ctx context.Context, taskType string) (*predictor.TaskStats, error) {
	canonicalType := predictor.GetTaskTypeName(taskType)
	return e.predictor.Predict(ctx, canonicalType), nil
}

// RefreshCache refreshes the predictor cache.
func (e *Engine) RefreshCache(ctx context.Context) {
	taskTypes := []string{"embedding", "llm", "llm_inference", "diffusion", "training", "vision", "other"}
	e.predictor.RefreshCache(ctx, taskTypes)
}