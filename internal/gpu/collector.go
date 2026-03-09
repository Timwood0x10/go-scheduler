package gpu

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Collector collects GPU metrics
type Collector struct {
	interval time.Duration
	pool     *Pool
	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
}

// NewCollector creates a new GPU metrics collector
func NewCollector(pool *Pool, interval time.Duration) *Collector {
	return &Collector{
		interval: interval,
		pool:     pool,
		stopCh:   make(chan struct{}),
	}
}

// Start starts the metrics collection
func (c *Collector) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}

	c.running = true
	go c.collectLoop()
	log.Printf("GPU metrics collector started with interval %v", c.interval)
}

// Stop stops the metrics collection
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	c.running = false
	close(c.stopCh)
	c.stopCh = make(chan struct{})
	log.Println("GPU metrics collector stopped")
}

// collectLoop runs the collection loop
func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Initial collection
	c.collect()

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-c.stopCh:
			return
		}
	}
}

// collect collects metrics from nvidia-smi
func (c *Collector) collect() {
	metrics, err := c.queryNvidiaSMI()
	if err != nil {
		log.Printf("Failed to collect GPU metrics: %v", err)
		return
	}

	c.updateGPUPool(metrics)
}

// Metrics represents GPU metrics from nvidia-smi
type Metrics struct {
	ID          int
	MemoryUsed  int64
	MemoryTotal int64
	ComputeUtil int
	MemoryUtil  int
	Temperature int
}

// queryNvidiaSMI queries nvidia-smi for GPU metrics
func (c *Collector) queryNvidiaSMI() ([]Metrics, error) {
	// Try to get GPU metrics using nvidia-smi
	// Format: gpu_index, memory_used, memory_total, util.gpu, util.memory, temperature.gpu
	cmd := exec.Command("nvidia-smi",
		"--query-gpu=index,memory.used,memory.total,utilization.gpu,utilization.memory,temperature.gpu",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		// If nvidia-smi fails, return empty metrics
		return c.getDummyMetrics(), nil
	}

	metrics := make([]Metrics, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ", ")
		if len(fields) < 6 {
			continue
		}

		id, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		memUsed, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
		memTotal, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		computeUtil, _ := strconv.Atoi(strings.TrimSpace(fields[3]))
		memUtil, _ := strconv.Atoi(strings.TrimSpace(fields[4]))
		temp, _ := strconv.Atoi(strings.TrimSpace(fields[5]))

		metrics = append(metrics, Metrics{
			ID:          id,
			MemoryUsed:  memUsed,
			MemoryTotal: memTotal,
			ComputeUtil: computeUtil,
			MemoryUtil:  memUtil,
			Temperature: temp,
		})
	}

	if len(metrics) == 0 {
		return c.getDummyMetrics(), nil
	}

	return metrics, nil
}

// getDummyMetrics returns dummy metrics when nvidia-smi is not available
func (c *Collector) getDummyMetrics() []Metrics {
	gpus := c.pool.GetAllGPUs()
	metrics := make([]Metrics, len(gpus))

	for i, gpu := range gpus {
		metrics[i] = Metrics{
			ID:          gpu.ID,
			MemoryUsed:  gpu.GetMemoryUsed(),
			MemoryTotal: gpu.MemoryTotal,
			ComputeUtil: 50,
			MemoryUtil:  50,
			Temperature: 60,
		}
	}

	return metrics
}

// updateGPUPool updates the GPU pool with new metrics
func (c *Collector) updateGPUPool(metrics []Metrics) {
	for _, m := range metrics {
		gpu, ok := c.pool.GetGPU(m.ID)
		if !ok {
			continue
		}

		gpu.mu.Lock()
		gpu.MemoryUsed = m.MemoryUsed
		gpu.ComputeUtil = m.ComputeUtil
		gpu.MemoryUtil = m.MemoryUtil
		gpu.Temperature = m.Temperature
		gpu.LastUpdated = time.Now()
		gpu.mu.Unlock()
	}
}

// GetGPUMetricsJSON returns GPU metrics as JSON
func (c *Collector) GetGPUMetricsJSON() (string, error) {
	gpus := c.pool.GetAllGPUs()

	type gpuMetricJSON struct {
		ID          int   `json:"gpu_id"`
		MemoryTotal int64 `json:"memory_total_mb"`
		MemoryUsed  int64 `json:"memory_used_mb"`
		MemoryFree  int64 `json:"memory_free_mb"`
		ComputeUtil int   `json:"compute_util_percent"`
		MemoryUtil  int   `json:"memory_util_percent"`
		Temperature int   `json:"temperature_celsius"`
		LastUpdated int64 `json:"last_updated_unix"`
	}

	metrics := make([]gpuMetricJSON, len(gpus))
	for i, gpu := range gpus {
		gpu.mu.RLock()
		metrics[i] = gpuMetricJSON{
			ID:          gpu.ID,
			MemoryTotal: gpu.MemoryTotal,
			MemoryUsed:  gpu.MemoryUsed,
			MemoryFree:  gpu.MemoryTotal - gpu.MemoryUsed,
			ComputeUtil: gpu.ComputeUtil,
			MemoryUtil:  gpu.MemoryUtil,
			Temperature: gpu.Temperature,
			LastUpdated: gpu.LastUpdated.Unix(),
		}
		gpu.mu.RUnlock()
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return string(data), nil
}
