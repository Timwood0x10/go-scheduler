package scheduler

import (
	"algogpu/internal/queue"
)

// AdmissionControl handles queue admission
type AdmissionControl struct {
	maxQueueSize int
	queue        *queue.TaskQueue
}

// NewAdmissionControl creates a new AdmissionControl
func NewAdmissionControl(maxQueueSize int, q *queue.TaskQueue) *AdmissionControl {
	return &AdmissionControl{
		maxQueueSize: maxQueueSize,
		queue:        q,
	}
}

// Check checks if a task can be admitted
func (a *AdmissionControl) Check() error {
	if a.queue.Len() >= a.maxQueueSize {
		return ErrQueueFull
	}
	return nil
}

// SetMaxQueueSize sets the maximum queue size
func (a *AdmissionControl) SetMaxQueueSize(size int) {
	a.maxQueueSize = size
}

// GetMaxQueueSize returns the maximum queue size
func (a *AdmissionControl) GetMaxQueueSize() int {
	return a.maxQueueSize
}

// GetCurrentQueueSize returns current queue size
func (a *AdmissionControl) GetCurrentQueueSize() int {
	return a.queue.Len()
}

// GetQueueUsage returns queue usage percentage
func (a *AdmissionControl) GetQueueUsage() float64 {
	if a.maxQueueSize == 0 {
		return 0
	}
	return float64(a.queue.Len()) / float64(a.maxQueueSize)
}
