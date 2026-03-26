package state

import (
	"testing"

	"algogpu/api"
)

func TestTaskStateMachine_CreateTask(t *testing.T) {
	sm := NewTaskStateMachine()

	state := sm.CreateTask("task-1")

	if state == nil {
		t.Error("Expected state to be created")
		return
	}

	if state.TaskID != "task-1" {
		t.Errorf("Expected task ID task-1, got %s", state.TaskID)
	}

	if state.CurrentState != api.TaskStatus_TASK_STATUS_PENDING {
		t.Errorf("Expected initial state PENDING, got %v", state.CurrentState)
	}
}

func TestTaskStateMachine_Transition_Valid(t *testing.T) {
	sm := NewTaskStateMachine()
	sm.CreateTask("task-1")

	err := sm.Transition("task-1", api.TaskStatus_TASK_STATUS_RUNNING, "scheduler dispatched")

	if err != nil {
		t.Errorf("Transition should succeed: %v", err)
	}

	state, _ := sm.GetState("task-1")
	if state.CurrentState != api.TaskStatus_TASK_STATUS_RUNNING {
		t.Errorf("Expected state RUNNING, got %v", state.CurrentState)
	}
}

func TestTaskStateMachine_Transition_Invalid(t *testing.T) {
	sm := NewTaskStateMachine()
	sm.CreateTask("task-1")

	// Invalid: PENDING -> COMPLETED is not allowed
	err := sm.Transition("task-1", api.TaskStatus_TASK_STATUS_COMPLETED, "")

	if err == nil {
		t.Error("Transition should fail")
	}
}

func TestTaskStateMachine_GetHistory(t *testing.T) {
	sm := NewTaskStateMachine()
	sm.CreateTask("task-1")

	if err := sm.Transition("task-1", api.TaskStatus_TASK_STATUS_RUNNING, "dispatched"); err != nil {
		t.Fatalf("Failed to transition: %v", err)
	}
	if err := sm.Transition("task-1", api.TaskStatus_TASK_STATUS_COMPLETED, "finished"); err != nil {
		t.Fatalf("Failed to transition: %v", err)
	}

	history, ok := sm.GetHistory("task-1")
	if !ok {
		t.Error("Expected to get history")
	}

	if len(history) != 2 {
		t.Errorf("Expected 2 transitions, got %d", len(history))
	}
}

func TestTaskStateMachine_CanTransition(t *testing.T) {
	sm := NewTaskStateMachine()
	sm.CreateTask("task-1")

	// PENDING -> RUNNING is valid
	if !sm.CanTransition("task-1", api.TaskStatus_TASK_STATUS_RUNNING) {
		t.Error("Transition should be valid")
	}

	// PENDING -> COMPLETED is not valid
	if sm.CanTransition("task-1", api.TaskStatus_TASK_STATUS_COMPLETED) {
		t.Error("Transition should not be valid")
	}
}

func TestIsTerminalState(t *testing.T) {
	tests := []struct {
		state    api.TaskStatus
		expected bool
	}{
		{api.TaskStatus_TASK_STATUS_PENDING, false},
		{api.TaskStatus_TASK_STATUS_RUNNING, false},
		{api.TaskStatus_TASK_STATUS_COMPLETED, true},
		{api.TaskStatus_TASK_STATUS_FAILED, true},
		{api.TaskStatus_TASK_STATUS_CANCELLED, true},
		{api.TaskStatus_TASK_STATUS_REJECTED, true},
	}

	for _, tc := range tests {
		result := IsTerminalState(tc.state)
		if result != tc.expected {
			t.Errorf("Expected IsTerminalState(%v) = %v, got %v", tc.state, tc.expected, result)
		}
	}
}
