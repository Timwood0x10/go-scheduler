package monitor

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/internal/scheduler"
)

// Metrics holds system metrics
type Metrics struct {
	Timestamp      int64  `json:"timestamp"`
	QueueSize      int    `json:"queue_size"`
	RunningTasks   int    `json:"running_tasks"`
	CompletedTasks int64  `json:"completed_tasks"`
	FailedTasks    int64  `json:"failed_tasks"`
	RejectedTasks  int64  `json:"rejected_tasks"`
	GPUMetrics     string `json:"gpu_metrics"`
}

// Monitor monitors the scheduler and exposes metrics
type Monitor struct {
	scheduler *scheduler.Scheduler
	gpuPool   *gpu.Pool
	collector *gpu.Collector
	taskQueue *queue.TaskQueue

	// Metrics
	mu             sync.RWMutex
	completedTasks int64
	failedTasks    int64
	rejectedTasks  int64

	// HTTP server
	server *http.Server
}

// NewMonitor creates a new monitor
func NewMonitor(sched *scheduler.Scheduler, gpuPool *gpu.Pool, collector *gpu.Collector, taskQueue *queue.TaskQueue) *Monitor {
	return &Monitor{
		scheduler: sched,
		gpuPool:   gpuPool,
		collector: collector,
		taskQueue: taskQueue,
	}
}

// Start starts the monitor
func (m *Monitor) Start(addr string) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", m.handleHealth)

	// Metrics endpoint
	mux.HandleFunc("/metrics", m.handleMetrics)

	// GPU metrics endpoint
	mux.HandleFunc("/gpu/metrics", m.handleGPUMetrics)

	// Queue status endpoint
	mux.HandleFunc("/queue/status", m.handleQueueStatus)

	m.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Monitor HTTP server starting on %s", addr)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
}

// Stop stops the monitor
func (m *Monitor) Stop() {
	if m.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}
}

// handleHealth handles health check requests
func (m *Monitor) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := `{"status": "healthy", "timestamp": ` + strconv.FormatInt(time.Now().Unix(), 10) + `}`
	if _, err := w.Write([]byte(response)); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}

// handleMetrics handles metrics requests
func (m *Monitor) handleMetrics(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	completed := m.completedTasks
	failed := m.failedTasks
	rejected := m.rejectedTasks
	m.mu.RUnlock()

	gpuMetrics, _ := m.collector.GetGPUMetricsJSON()

	metrics := Metrics{
		Timestamp:      time.Now().Unix(),
		QueueSize:      m.taskQueue.Len(),
		RunningTasks:   m.taskQueue.RunningCount(),
		CompletedTasks: completed,
		FailedTasks:    failed,
		RejectedTasks:  rejected,
		GPUMetrics:     gpuMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode metrics: %v", err)
	}
}

// handleGPUMetrics handles GPU metrics requests
func (m *Monitor) handleGPUMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := m.collector.GetGPUMetricsJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(metrics)); err != nil {
		log.Printf("Failed to write GPU metrics response: %v", err)
	}
}

// handleQueueStatus handles queue status requests
func (m *Monitor) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	pending := m.taskQueue.GetAllPending()
	running := m.taskQueue.GetAllRunning()

	type taskInfo struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
		Type   string `json:"type"`
	}

	pendingTasks := make([]taskInfo, 0, len(pending))
	for _, t := range pending {
		pendingTasks = append(pendingTasks, taskInfo{
			ID:     t.ID,
			UserID: t.UserID,
			Type:   t.Type.String(),
		})
	}

	runningTasks := make([]taskInfo, 0, len(running))
	for _, t := range running {
		runningTasks = append(runningTasks, taskInfo{
			ID:     t.ID,
			UserID: t.UserID,
			Type:   t.Type.String(),
		})
	}

	status := map[string]interface{}{
		"queue_size":    m.taskQueue.Len(),
		"running_count": m.taskQueue.RunningCount(),
		"pending":       pendingTasks,
		"running":       runningTasks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode status: %v", err)
	}
}

// RecordCompleted increments completed tasks counter
func (m *Monitor) RecordCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completedTasks++
}

// RecordFailed increments failed tasks counter
func (m *Monitor) RecordFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failedTasks++
}

// RecordRejected increments rejected tasks counter
func (m *Monitor) RecordRejected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rejectedTasks++
}

// WaitForSignal waits for termination signals
func WaitForSignal() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	log.Printf("Received signal: %v", sig)
	log.Println("Shutting down gracefully...")
}
