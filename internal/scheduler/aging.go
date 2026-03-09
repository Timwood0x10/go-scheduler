package scheduler

import (
	"time"

	"algogpu/pkg/types"
)

// AgingFactor represents the aging factor γ
// priority = base_priority + wait_time * γ
type AgingFactor float64

// TaskAging manages task aging to prevent starvation
type TaskAging struct {
	agingFactor float64
}

// NewTaskAging creates a new TaskAging
func NewTaskAging(agingFactor float64) *TaskAging {
	return &TaskAging{
		agingFactor: agingFactor,
	}
}

// CalculateAgingPriority calculates priority with aging
// priority = base_priority + wait_time * γ
func (a *TaskAging) CalculateAgingPriority(basePriority float64, task *types.Task) float64 {
	waitTime := time.Since(task.CreatedAt).Seconds()
	return basePriority + waitTime*a.agingFactor
}

// ApplyAging applies aging adjustment to task priorities
func (a *TaskAging) ApplyAging(tasksWithPriority []TaskWithPriority, taskQueue []*types.Task) []TaskWithPriority {
	// Build a map for quick task lookup
	taskMap := make(map[string]*types.Task)
	for _, twp := range tasksWithPriority {
		taskMap[twp.Task.ID] = twp.Task
	}

	// Recalculate priorities with aging
	result := make([]TaskWithPriority, 0, len(tasksWithPriority))
	for _, twp := range tasksWithPriority {
		agingPriority := a.CalculateAgingPriority(twp.Priority, twp.Task)
		result = append(result, TaskWithPriority{
			Task:     twp.Task,
			Priority: agingPriority,
		})
	}

	// Sort by aging priority (higher first)
	sortByAgingPriority(result)

	return result
}

// sortByAgingPriority sorts by priority descending
func sortByAgingPriority(twp []TaskWithPriority) {
	// Use the sort from cost_aware package or implement here
	for i := 0; i < len(twp)-1; i++ {
		for j := i + 1; j < len(twp); j++ {
			if twp[i].Priority < twp[j].Priority {
				twp[i], twp[j] = twp[j], twp[i]
			}
		}
	}
}

// GetWaitTime returns the wait time for a task
func (a *TaskAging) GetWaitTime(task *types.Task) time.Duration {
	return time.Since(task.CreatedAt)
}

// IsStale checks if a task has been waiting too long
func (a *TaskAging) IsStale(task *types.Task, maxWait time.Duration) bool {
	return a.GetWaitTime(task) > maxWait
}
