// Package main demonstrates integrating AlgoGPU as a plugin in go-agent.
// This example shows how to use the scheduler for GPU task management.
package main

import (
	"context"
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

// Agent represents a simple AI agent that uses GPU tasks.
type Agent struct {
	scheduler plugin.Scheduler
	userID    string
}

// NewAgent creates a new agent with GPU scheduling capabilities.
func NewAgent(scheduler plugin.Scheduler, userID string) *Agent {
	return &Agent{
		scheduler: scheduler,
		userID:    userID,
	}
}

// ProcessEmbedding processes an embedding task.
func (a *Agent) ProcessEmbedding(ctx context.Context, text string) ([]byte, error) {
	task := &types.Task{
		ID:                 generateID("emb"),
		UserID:             a.userID,
		Type:               api.TaskType_TASK_TYPE_EMBEDDING,
		GPUMemoryRequired:  512,
		GPUComputeRequired: 50,
		EstimatedRuntimeMs: 100,
		Payload:            []byte(text),
	}

	err := a.scheduler.SubmitTask(ctx, task)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	return a.waitForTask(ctx, task.ID)
}

// ProcessLLM processes an LLM inference task.
func (a *Agent) ProcessLLM(ctx context.Context, prompt string) (string, error) {
	task := &types.Task{
		ID:                 generateID("llm"),
		UserID:             a.userID,
		Type:               api.TaskType_TASK_TYPE_LLM,
		GPUMemoryRequired:  4096,
		GPUComputeRequired: 200,
		EstimatedRuntimeMs: 5000,
		Payload:            []byte(prompt),
	}

	err := a.scheduler.SubmitTask(ctx, task)
	if err != nil {
		return "", err
	}

	// Wait for task completion
	_, err = a.waitForTask(ctx, task.ID)
	if err != nil {
		return "", err
	}

	return "Generated response from LLM", nil
}

// ProcessDiffusion processes a diffusion model task.
func (a *Agent) ProcessDiffusion(ctx context.Context, prompt string) ([]byte, error) {
	task := &types.Task{
		ID:                 generateID("diff"),
		UserID:             a.userID,
		Type:               api.TaskType_TASK_TYPE_DIFFUSION,
		GPUMemoryRequired:  8192,
		GPUComputeRequired: 500,
		EstimatedRuntimeMs: 10000,
		Payload:            []byte(prompt),
	}

	err := a.scheduler.SubmitTask(ctx, task)
	if err != nil {
		return nil, err
	}

	// Wait for task completion
	return a.waitForTask(ctx, task.ID)
}

// waitForTask waits for a task to complete.
func (a *Agent) waitForTask(ctx context.Context, taskID string) ([]byte, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, syscall.ETIMEDOUT
		case <-ticker.C:
			task, err := a.scheduler.GetStatus(ctx, taskID)
			if err != nil {
				return nil, err
			}

			switch task.Status {
			case api.TaskStatus_TASK_STATUS_COMPLETED:
				log.Printf("Task %s completed successfully", taskID)
				return []byte("task completed"), nil
			case api.TaskStatus_TASK_STATUS_FAILED:
				log.Printf("Task %s failed: %s", taskID, task.Message)
				return nil, err
			case api.TaskStatus_TASK_STATUS_CANCELLED:
				log.Printf("Task %s was cancelled", taskID)
				return nil, err
			}
		}
	}
}

func main() {
	log.Println("Starting go-agent with AlgoGPU plugin...")

	// Setup AlgoGPU scheduler
	gpuPool := gpu.NewPool()
	gpuPool.AddGPU(0, "NVIDIA-A100", 81920)
	gpuPool.AddGPU(1, "NVIDIA-A100", 81920)

	taskQueue := queue.NewTaskQueue()

	cfg := &scheduler.Config{
		MaxQueueSize:       100,
		TokenRefillRate:    100,
		TokenBucketSize:    1000,
		DailyTokenLimit:    1000000,
		GPULoadThreshold:   0.85,
		AgingFactor:        0.1,
		UsageWindowMinutes: 5,
	}

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool)
	pluginScheduler := plugin.NewPluginScheduler(sched, taskQueue, gpuPool)

	sched.Start()
	defer sched.Stop()

	// Create agent
	agent := NewAgent(pluginScheduler, "agent-001")

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run agent workload
	go runAgentWorkload(ctx, agent)

	// Monitor scheduler
	go monitorScheduler(ctx, pluginScheduler)

	// Wait for shutdown
	<-sigCh
	log.Println("Received shutdown signal, stopping agent...")

	cancel()

	time.Sleep(2 * time.Second)

	log.Println("Agent stopped")
}

// runAgentWorkload runs a sample workload.
func runAgentWorkload(ctx context.Context, agent *Agent) {
	taskCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			taskCount++

			switch taskCount % 3 {
			case 0:
				// Embedding task
				_, err := agent.ProcessEmbedding(ctx, "Sample text for embedding")
				if err != nil {
					log.Printf("Embedding task failed: %v", err)
				}
			case 1:
				// LLM task
				_, err := agent.ProcessLLM(ctx, "Generate a response")
				if err != nil {
					log.Printf("LLM task failed: %v", err)
				}
			case 2:
				// Diffusion task
				_, err := agent.ProcessDiffusion(ctx, "Generate an image")
				if err != nil {
					log.Printf("Diffusion task failed: %v", err)
				}
			}

			time.Sleep(2 * time.Second)
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
			log.Printf("Scheduler Status: Queue=%d, Running=%d",
				sched.GetQueueSize(), sched.GetRunningCount())
		}
	}
}

// generateID generates a unique ID.
func generateID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102-150405")
}
