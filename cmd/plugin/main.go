// Command plugin demonstrates using AlgoGPU as an embedded plugin.
// This mode is ideal for integration with agent frameworks like go-agent.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"algogpu/api"
	"algogpu/internal/gpu"
	"algogpu/internal/plugin"
	"algogpu/internal/queue"
	"algogpu/internal/scheduler"
	"algogpu/pkg/types"
)

var (
	gpuCount    = flag.Int("gpus", 4, "Number of GPUs to simulate")
	gpuMemory   = flag.Int64("memory", 8192, "GPU memory in MB")
	queueSize   = flag.Int("queue-size", 1000, "Maximum queue size")
	tokenRate   = flag.Int("token-rate", 100, "Token refill rate per second")
	agingFactor = flag.Float64("aging", 0.1, "Task aging factor")
	sampleTasks = flag.Int("tasks", 10, "Number of sample tasks to submit")
)

func main() {
	flag.Parse()

	log.Println("Starting AlgoGPU in plugin mode...")

	// Create GPU pool
	gpuPool := gpu.NewPool()
	for i := 0; i < *gpuCount; i++ {
		gpuPool.AddGPU(i, "GPU-"+string(rune('0'+i)), *gpuMemory)
		log.Printf("Added GPU %d with %d MB memory", i, *gpuMemory)
	}

	// Create task queue
	taskQueue := queue.NewTaskQueue()

	// Create scheduler configuration
	cfg := &scheduler.Config{
		MaxQueueSize:       *queueSize,
		TokenRefillRate:    int64(*tokenRate),
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        *agingFactor,
		UsageWindowMinutes: 5,
	}

	// Create scheduler
	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)

	// Create GPU metrics collector
	collector := gpu.NewCollector(gpuPool, 5*time.Second)
	collector.Start()
	defer collector.Stop()

	// Create plugin scheduler
	pluginScheduler := plugin.NewPluginScheduler(sched, taskQueue, gpuPool)

	// Start scheduler
	sched.Start()
	defer sched.Stop()

	log.Println("Scheduler started successfully")

	// Wait for context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Submit sample tasks
	go submitSampleTasks(ctx, pluginScheduler, *sampleTasks)

	// Monitor scheduler
	go monitorScheduler(ctx, pluginScheduler)

	// Wait for shutdown signal
	<-sigCh
	log.Println("Received shutdown signal, stopping...")

	cancel()

	// Wait a bit for graceful shutdown
	time.Sleep(2 * time.Second)

	log.Println("Plugin mode stopped")
}

// submitSampleTasks submits sample tasks for demonstration.
func submitSampleTasks(ctx context.Context, sched plugin.Scheduler, count int) {
	taskTypes := []api.TaskType{
		api.TaskType_TASK_TYPE_EMBEDDING,
		api.TaskType_TASK_TYPE_LLM,
		api.TaskType_TASK_TYPE_DIFFUSION,
		api.TaskType_TASK_TYPE_OTHER,
	}

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			taskType := taskTypes[i%len(taskTypes)]
			task := &types.Task{
				ID:                 generateTaskID(i),
				UserID:             "user-" + string(rune('0'+i%3)),
				Type:               taskType,
				GPUMemoryRequired:  1024 + int64(i%4)*1024,
				GPUComputeRequired: 100,
				EstimatedRuntimeMs: 1000 + int64(i%5)*1000,
				Payload:            []byte(`{"data": "sample"}`),
			}

			err := sched.SubmitTask(ctx, task)
			if err != nil {
				log.Printf("Failed to submit task %s: %v", task.ID, err)
			} else {
				log.Printf("Submitted task %s (type: %s)", task.ID, taskType)
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

// monitorScheduler monitors scheduler status.
func monitorScheduler(ctx context.Context, sched plugin.Scheduler) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Status: Queue=%d, Running=%d",
				sched.GetQueueSize(), sched.GetRunningCount())

			// Print GPU status
			gpus := sched.GetGPUStatus(ctx)
			for _, gpu := range gpus {
				log.Printf("  GPU %s: %d/%d MB used, %d%% load",
					gpu.Name, gpu.MemoryUsed, gpu.MemoryTotal, gpu.ComputeUtil)
			}
		}
	}
}

// generateTaskID generates a unique task ID.
func generateTaskID(index int) string {
	return time.Now().Format("20060102-150405") + "-" + string(rune('0'+index))
}
