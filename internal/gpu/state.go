package gpu

import (
	"sync"
	"time"
)

// State represents the detailed state of a GPU
type State struct {
	ID             int
	Name           string
	Status         Status
	MemoryTotal    int64 // MB
	MemoryUsed     int64 // MB
	MemoryFree     int64 // MB
	ComputeUtil    int   // percentage 0-100
	MemoryUtil     int   // percentage 0-100
	Temperature    int   // celsius
	PowerUsage     int   // watts
	FanSpeed       int   // percentage
	RunningTasks   []string
	CompletedTasks int64
	FailedTasks    int64

	// Heartbeat
	LastHeartbeat time.Time
	HeartbeatTTL  time.Duration // time to live

	// Timing
	CreatedAt     time.Time
	LastUpdated   time.Time
	LastScheduled time.Time

	// Fragmentation
	Fragmentation float64 // 0.0-1.0

	mu sync.RWMutex
}

// Status represents GPU operational status
type Status string

// Status constants
const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
	StatusBusy    Status = "busy"
	StatusIdle    Status = "idle"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

// NewState creates a new GPU state
func NewState(id int, name string, memoryTotal int64) *State {
	return &State{
		ID:             id,
		Name:           name,
		Status:         StatusOnline,
		MemoryTotal:    memoryTotal,
		MemoryUsed:     0,
		MemoryFree:     memoryTotal,
		ComputeUtil:    0,
		MemoryUtil:     0,
		Temperature:    0,
		PowerUsage:     0,
		FanSpeed:       0,
		RunningTasks:   []string{},
		CompletedTasks: 0,
		FailedTasks:    0,
		LastHeartbeat:  time.Now(),
		HeartbeatTTL:   5 * time.Second,
		CreatedAt:      time.Now(),
		LastUpdated:    time.Now(),
		LastScheduled:  time.Now(),
		Fragmentation:  0,
	}
}

// UpdateMetrics updates GPU metrics
func (g *State) UpdateMetrics(memoryUsed int64, computeUtil int, memoryUtil int, temperature int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.MemoryUsed = memoryUsed
	g.MemoryFree = g.MemoryTotal - memoryUsed
	g.ComputeUtil = computeUtil
	g.MemoryUtil = memoryUtil
	g.Temperature = temperature
	g.LastUpdated = time.Now()
}

// UpdatePower updates power usage
func (g *State) UpdatePower(powerWatts int, fanSpeed int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.PowerUsage = powerWatts
	g.FanSpeed = fanSpeed
}

// Heartbeat updates the last heartbeat time
func (g *State) Heartbeat() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.LastHeartbeat = time.Now()
	if g.Status != StatusOnline {
		g.Status = StatusOnline
	}
}

// CheckHealth checks if GPU is healthy
func (g *State) CheckHealth() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check heartbeat TTL
	if time.Since(g.LastHeartbeat) > g.HeartbeatTTL {
		return false
	}

	// Check critical temperature
	if g.Temperature > 90 {
		return false
	}

	return true
}

// GetStatus returns the current status
func (g *State) GetStatus() Status {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Update status based on running tasks
	if len(g.RunningTasks) > 0 {
		return StatusBusy
	}

	// Check if online via heartbeat
	if time.Since(g.LastHeartbeat) > g.HeartbeatTTL {
		return StatusOffline
	}

	return StatusIdle
}

// CalculateFragmentation calculates memory fragmentation
func (g *State) CalculateFragmentation() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.MemoryTotal == 0 {
		return 0
	}

	// Simple fragmentation metric: free memory / total memory
	fragmentation := float64(g.MemoryFree) / float64(g.MemoryTotal)
	return fragmentation
}

// Manager manages GPU states
type Manager struct {
	mu     sync.RWMutex
	states map[int]*State
}

// NewManager creates a new state manager
func NewManager() *Manager {
	return &Manager{
		states: make(map[int]*State),
	}
}

// RegisterGPU registers a new GPU
func (m *Manager) RegisterGPU(id int, name string, memoryTotal int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[id] = NewState(id, name, memoryTotal)
}

// GetState returns GPU state
func (m *Manager) GetState(id int) (*State, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.states[id]
	return state, ok
}

// GetAllStates returns all GPU states
func (m *Manager) GetAllStates() []*State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make([]*State, 0, len(m.states))
	for _, s := range m.states {
		states = append(states, s)
	}

	return states
}

// GetOnlineGPUs returns all online GPUs
func (m *Manager) GetOnlineGPUs() []*State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	online := make([]*State, 0)
	for _, s := range m.states {
		if s.CheckHealth() {
			online = append(online, s)
		}
	}

	return online
}

// RemoveGPU removes a GPU from management
func (m *Manager) RemoveGPU(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, id)
}
