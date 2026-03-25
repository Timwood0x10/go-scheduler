package gpu

import (
	"testing"
	"time"
)

func TestNewState(t *testing.T) {
	state := NewState(0, "NVIDIA GPU 0", 16384)

	if state == nil {
		t.Error("NewState should not return nil")
	}

	if state.ID != 0 {
		t.Errorf("expected ID 0, got %d", state.ID)
	}

	if state.Name != "NVIDIA GPU 0" {
		t.Errorf("expected name NVIDIA GPU 0, got %s", state.Name)
	}

	if state.MemoryTotal != 16384 {
		t.Errorf("expected memory 16384, got %d", state.MemoryTotal)
	}

	if state.Status != StatusOnline {
		t.Errorf("expected status online, got %s", state.Status)
	}
}

func TestState_UpdateMetrics(t *testing.T) {
	state := NewState(0, "GPU", 16384)

	state.UpdateMetrics(8192, 50, 50, 60)

	if state.MemoryUsed != 8192 {
		t.Errorf("expected 8192, got %d", state.MemoryUsed)
	}

	if state.MemoryFree != 8192 {
		t.Errorf("expected 8192, got %d", state.MemoryFree)
	}

	if state.ComputeUtil != 50 {
		t.Errorf("expected 50, got %d", state.ComputeUtil)
	}
}

func TestState_Heartbeat(t *testing.T) {
	state := NewState(0, "GPU", 16384)

	before := state.LastHeartbeat
	state.Heartbeat()

	if !state.LastHeartbeat.After(before) {
		t.Error("Heartbeat should update LastHeartbeat")
	}
}

func TestState_CheckHealth(t *testing.T) {
	state := NewState(0, "GPU", 16384)

	// Should be healthy
	if !state.CheckHealth() {
		t.Error("New GPU should be healthy")
	}

	// Set temperature to critical
	state.Temperature = 95
	if state.CheckHealth() {
		t.Error("GPU with critical temperature should not be healthy")
	}

	// Reset and test heartbeat timeout
	state.Temperature = 60
	state.LastHeartbeat = time.Now().Add(-10 * time.Second)
	state.HeartbeatTTL = 1 * time.Second

	if state.CheckHealth() {
		t.Error("GPU with expired heartbeat should not be healthy")
	}
}

func TestState_GetStatus(t *testing.T) {
	state := NewState(0, "GPU", 16384)

	// No running tasks should be idle
	if state.GetStatus() != StatusIdle {
		t.Errorf("expected idle, got %s", state.GetStatus())
	}

	// Add running task
	state.RunningTasks = append(state.RunningTasks, "task-1")

	if state.GetStatus() != StatusBusy {
		t.Errorf("expected busy, got %s", state.GetStatus())
	}
}

func TestState_CalculateFragmentation(t *testing.T) {
	state := NewState(0, "GPU", 16384)

	// Use UpdateMetrics to set half memory used
	state.UpdateMetrics(8192, 50, 50, 60)

	frag := state.CalculateFragmentation()

	if frag != 0.5 {
		t.Errorf("expected 0.5, got %f", frag)
	}
}

func TestManager_RegisterGPU(t *testing.T) {
	mgr := NewManager()

	mgr.RegisterGPU(0, "GPU 0", 16384)
	mgr.RegisterGPU(1, "GPU 1", 32768)

	if len(mgr.states) != 2 {
		t.Errorf("expected 2 states, got %d", len(mgr.states))
	}
}

func TestManager_GetState(t *testing.T) {
	mgr := NewManager()
	mgr.RegisterGPU(0, "GPU 0", 16384)

	state, ok := mgr.GetState(0)
	if !ok {
		t.Error("Should find GPU 0")
	}

	if state.Name != "GPU 0" {
		t.Errorf("expected GPU 0, got %s", state.Name)
	}

	// Non-existent GPU
	_, ok = mgr.GetState(999)
	if ok {
		t.Error("Should not find GPU 999")
	}
}

func TestManager_GetOnlineGPUs(t *testing.T) {
	mgr := NewManager()
	mgr.RegisterGPU(0, "GPU 0", 16384)
	mgr.RegisterGPU(1, "GPU 1", 16384)

	// Set one GPU as offline
	mgr.states[1].LastHeartbeat = time.Now().Add(-10 * time.Second)
	mgr.states[1].HeartbeatTTL = 1 * time.Second

	online := mgr.GetOnlineGPUs()

	if len(online) != 1 {
		t.Errorf("expected 1 online GPU, got %d", len(online))
	}

	if online[0].ID != 0 {
		t.Errorf("expected GPU 0 to be online, got %d", online[0].ID)
	}
}

func TestManager_RemoveGPU(t *testing.T) {
	mgr := NewManager()
	mgr.RegisterGPU(0, "GPU 0", 16384)

	mgr.RemoveGPU(0)

	if len(mgr.states) != 0 {
		t.Errorf("expected 0 states, got %d", len(mgr.states))
	}
}
