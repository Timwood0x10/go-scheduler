package state

import (
	"fmt"
	"strings"
	"time"

	"algogpu/api"
)

// TaskState represents the state of a task
type TaskState struct {
	TaskID        string
	CurrentState  api.TaskStatus
	PreviousState api.TaskStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
	// State history for debugging
	History []Record
}

// Record represents a state transition record
type Record struct {
	FromState api.TaskStatus
	ToState   api.TaskStatus
	Timestamp time.Time
	Reason    string
}

// ValidTransitions defines valid state transitions
var ValidTransitions = map[api.TaskStatus][]api.TaskStatus{
	api.TaskStatus_TASK_STATUS_PENDING: {
		api.TaskStatus_TASK_STATUS_RUNNING,
		api.TaskStatus_TASK_STATUS_CANCELLED,
		api.TaskStatus_TASK_STATUS_REJECTED,
	},
	api.TaskStatus_TASK_STATUS_RUNNING: {
		api.TaskStatus_TASK_STATUS_COMPLETED,
		api.TaskStatus_TASK_STATUS_FAILED,
		api.TaskStatus_TASK_STATUS_CANCELLED,
	},
}

// TaskStateMachine manages task state transitions
type TaskStateMachine struct {
	states map[string]*TaskState
}

// NewTaskStateMachine creates a new task state machine
func NewTaskStateMachine() *TaskStateMachine {
	return &TaskStateMachine{
		states: make(map[string]*TaskState),
	}
}

// CreateTask creates a new task state
func (sm *TaskStateMachine) CreateTask(taskID string) *TaskState {
	state := &TaskState{
		TaskID:       taskID,
		CurrentState: api.TaskStatus_TASK_STATUS_PENDING,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		History:      make([]Record, 0),
	}

	sm.states[taskID] = state
	return state
}

// Transition attempts to transition to a new state
func (sm *TaskStateMachine) Transition(taskID string, newState api.TaskStatus, reason string) error {
	state, ok := sm.states[taskID]
	if !ok {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Check if transition is valid
	validStates, ok := ValidTransitions[state.CurrentState]
	if !ok {
		return fmt.Errorf("no valid transitions from state %v", state.CurrentState)
	}

	isValid := false
	for _, s := range validStates {
		if s == newState {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("invalid transition from %v to %v", state.CurrentState, newState)
	}

	// Record transition
	record := Record{
		FromState: state.CurrentState,
		ToState:   newState,
		Timestamp: time.Now(),
		Reason:    reason,
	}

	state.History = append(state.History, record)
	state.PreviousState = state.CurrentState
	state.CurrentState = newState
	state.UpdatedAt = time.Now()

	return nil
}

// GetState returns the current state of a task
func (sm *TaskStateMachine) GetState(taskID string) (*TaskState, bool) {
	state, ok := sm.states[taskID]
	return state, ok
}

// GetHistory returns the state history for a task
func (sm *TaskStateMachine) GetHistory(taskID string) ([]Record, bool) {
	state, ok := sm.states[taskID]
	if !ok {
		return nil, false
	}
	return state.History, true
}

// CanTransition checks if a transition is valid
func (sm *TaskStateMachine) CanTransition(taskID string, newState api.TaskStatus) bool {
	state, ok := sm.states[taskID]
	if !ok {
		return false
	}

	validStates, ok := ValidTransitions[state.CurrentState]
	if !ok {
		return false
	}

	for _, s := range validStates {
		if s == newState {
			return true
		}
	}

	return false
}

// IsTerminalState checks if a state is terminal
func IsTerminalState(state api.TaskStatus) bool {
	switch state {
	case api.TaskStatus_TASK_STATUS_COMPLETED,
		api.TaskStatus_TASK_STATUS_FAILED,
		api.TaskStatus_TASK_STATUS_CANCELLED,
		api.TaskStatus_TASK_STATUS_REJECTED:
		return true
	default:
		return false
	}
}

// String returns a string representation of the state
func (s *TaskState) String() string {
	var b strings.Builder
	b.Grow(128) // Pre-allocate capacity
	b.WriteString("TaskState{taskID=")
	b.WriteString(s.TaskID)
	b.WriteString(", state=")
	b.WriteString(s.CurrentState.String())
	b.WriteString(", createdAt=")
	b.WriteString(s.CreatedAt.Format(time.RFC3339))
	b.WriteString(", updatedAt=")
	b.WriteString(s.UpdatedAt.Format(time.RFC3339))
	b.WriteString("}")
	return b.String()
}
