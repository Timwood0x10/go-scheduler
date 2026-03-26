package server

import (
	"context"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"

	"algogpu/api"
	"algogpu/internal/db"
	"algogpu/internal/gpu"
	"algogpu/internal/queue"
	"algogpu/internal/scheduler"
	"algogpu/pkg/types"
)

// Server implements the GPUScheduler gRPC service
type Server struct {
	api.UnimplementedGPUSchedulerServer
	taskQueue *queue.TaskQueue
	gpuPool   *gpu.Pool
	scheduler *scheduler.Scheduler
	collector *gpu.Collector
	dbStore   *db.SQLiteStore
}

// NewServer creates a new gRPC server with scheduler
func NewServer() *Server {
	cfg := scheduler.DefaultConfig()
	taskQueue := queue.NewTaskQueue()
	gpuPool := gpu.NewPool()

	// Initialize SQLite database
	dbStore, err := db.NewSQLiteStore("algogpu.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbStore.Close()

	// Add some dummy GPUs for testing
	gpuPool.AddGPU(0, "NVIDIA GPU 0", 16384) // 16GB
	gpuPool.AddGPU(1, "NVIDIA GPU 1", 16384)
	gpuPool.AddGPU(2, "NVIDIA GPU 2", 32768) // 32GB
	gpuPool.AddGPU(3, "NVIDIA GPU 3", 32768)

	sched := scheduler.NewScheduler(cfg, taskQueue, gpuPool, dbStore)
	sched.Start()

	// Start GPU metrics collector
	collector := gpu.NewCollector(gpuPool, 5*time.Second)
	collector.Start()

	return &Server{
		taskQueue: taskQueue,
		gpuPool:   gpuPool,
		scheduler: sched,
		collector: collector,
		dbStore:   dbStore,
	}
	}
}

// SubmitTask handles task submission
func (s *Server) SubmitTask(ctx context.Context, req *api.TaskRequest) (*api.TaskResponse, error) {
	task := types.TaskFromProto(req)

	accepted, message, status := s.scheduler.SubmitTask(task)

	log.Printf("Task submitted: id=%s, user=%s, type=%v, accepted=%v",
		task.ID, task.UserID, task.Type, accepted)

	return &api.TaskResponse{
		Accepted: accepted,
		Message:  message,
		Status:   status,
	}, nil
}

// GetTaskStatus returns the status of a task
func (s *Server) GetTaskStatus(ctx context.Context, req *api.TaskStatusRequest) (*api.TaskStatusResponse, error) {
	task, ok := s.taskQueue.Get(req.TaskId)
	if !ok {
		return &api.TaskStatusResponse{
			TaskId:  req.TaskId,
			Status:  api.TaskStatus_TASK_STATUS_FAILED,
			Message: "Task not found",
		}, nil
	}

	return &api.TaskStatusResponse{
		TaskId:      task.ID,
		Status:      task.Status,
		Message:     task.Message,
		CreatedAt:   task.CreatedAt.Unix(),
		StartedAt:   task.StartedAt.Unix(),
		CompletedAt: task.CompletedAt.Unix(),
	}, nil
}

// CancelTask cancels a task
func (s *Server) CancelTask(ctx context.Context, req *api.CancelTaskRequest) (*api.CancelTaskResponse, error) {
	err := s.taskQueue.Cancel(req.TaskId)
	if err != nil {
		return &api.CancelTaskResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	log.Printf("Task cancelled: id=%s", req.TaskId)

	return &api.CancelTaskResponse{
		Success: true,
		Message: "Task cancelled",
	}, nil
}

// GetGPUStatus returns GPU status
func (s *Server) GetGPUStatus(ctx context.Context, req *api.GetGPUStatusRequest) (*api.GPUStatusResponse, error) {
	gpus := s.gpuPool.GetAllGPUs()

	gpuInfos := make([]*api.GPUInfo, len(gpus))
	for i, g := range gpus {
		gpuInfos[i] = g.ToProto()
	}

	return &api.GPUStatusResponse{
		Gpus:      gpuInfos,
		Timestamp: 0,
	}, nil
}

// TaskEvents streams task events (placeholder implementation)
func (s *Server) TaskEvents(req *api.TaskEventsRequest, stream api.GPUScheduler_TaskEventsServer) error {
	// TODO: implement task event streaming
	return nil
}

// Run starts the gRPC server
func Run(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	server := NewServer()
	api.RegisterGPUSchedulerServer(grpcServer, server)

	log.Printf("gRPC server listening on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	// Stop collector when server stops
	server.collector.Stop()
	log.Println("GPU metrics collector stopped")
}
