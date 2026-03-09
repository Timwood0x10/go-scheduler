"""GPU Scheduler Python SDK

Usage:
    from gpu_scheduler import GPUClient

    client = GPUClient(host="localhost:50051")

    # Submit task
    task_id = client.submit_task(
        user_id="user_123",
        task_type="llm",
        gpu_memory_mb=8192,
        payload={"prompt": "..."}
    )

    # Get status
    status = client.get_status(task_id)
    print(f"Task status: {status}")

    # Wait for result
    result = client.wait(task_id)
    print(f"Result: {result}")
"""

import json
import time
from typing import Dict, Any, Optional, List

import grpc

from . import gpu_scheduler_pb2 as pb2
from . import gpu_scheduler_pb2_grpc as pb2_grpc


class TaskStatus:
    PENDING = pb2.TASK_STATUS_PENDING
    RUNNING = pb2.TASK_STATUS_RUNNING
    COMPLETED = pb2.TASK_STATUS_COMPLETED
    FAILED = pb2.TASK_STATUS_FAILED
    CANCELLED = pb2.TASK_STATUS_CANCELLED
    REJECTED = pb2.TASK_STATUS_REJECTED

    _names = {
        PENDING: "PENDING",
        RUNNING: "RUNNING",
        COMPLETED: "COMPLETED",
        FAILED: "FAILED",
        CANCELLED: "CANCELLED",
        REJECTED: "REJECTED",
    }

    @classmethod
    def name(cls, status: int) -> str:
        return cls._names.get(status, f"UNKNOWN({status})")


class TaskType:
    UNSPECIFIED = pb2.TASK_TYPE_UNSPECIFIED
    EMBEDDING = pb2.TASK_TYPE_EMBEDDING
    LLM = pb2.TASK_TYPE_LLM
    DIFFUSION = pb2.TASK_TYPE_DIFFUSION
    OTHER = pb2.TASK_TYPE_OTHER

    _names = {
        UNSPECIFIED: "EMBEDDING",
        EMBEDDING: "EMBEDDING",
        LLM: "LLM",
        DIFFUSION: "DIFFUSION",
        OTHER: "OTHER",
    }

    @classmethod
    def from_string(cls, s: str) -> int:
        mapping = {
            "embedding": cls.EMBEDDING,
            "llm": cls.LLM,
            "diffusion": cls.DIFFUSION,
            "other": cls.OTHER,
        }
        return mapping.get(s.lower(), cls.UNSPECIFIED)


class GPUClient:
    """GPU Scheduler Client"""

    def __init__(self, host: str = "localhost:50051"):
        self.host = host
        self.channel = grpc.insecure_channel(host)
        self.stub = pb2_grpc.GPUSchedulerStub(self.channel)

    def submit_task(
        self,
        task_id: str,
        user_id: str,
        task_type: str,
        gpu_memory_mb: int,
        payload: Dict[str, Any],
        gpu_compute_required: int = 0,
        estimated_runtime_ms: int = 0,
    ) -> str:
        """Submit a GPU task

        Args:
            task_id: Unique task ID
            user_id: User ID
            task_type: Task type ("embedding", "llm", "diffusion", "other")
            gpu_memory_mb: Required GPU memory in MB
            payload: Task payload (will be serialized to JSON)
            gpu_compute_required: Required GPU compute (optional)
            estimated_runtime_ms: Estimated runtime in milliseconds (optional)

        Returns:
            task_id: The submitted task ID
        """
        payload_bytes = json.dumps(payload).encode("utf-8")

        request = pb2.TaskRequest(
            task_id=task_id,
            user_id=user_id,
            task_type=TaskType.from_string(task_type),
            gpu_memory_required=gpu_memory_mb,
            gpu_compute_required=gpu_compute_required,
            estimated_runtime_ms=estimated_runtime_ms,
            payload=payload_bytes,
        )

        response = self.stub.SubmitTask(request)

        if not response.accepted:
            raise RuntimeError(f"Task rejected: {response.message}")

        return task_id

    def get_status(self, task_id: str) -> Dict[str, Any]:
        """Get task status

        Args:
            task_id: Task ID

        Returns:
            Dict with status information
        """
        request = pb2.TaskStatusRequest(task_id=task_id)
        response = self.stub.GetTaskStatus(request)

        return {
            "task_id": response.task_id,
            "status": TaskStatus.name(response.status),
            "status_code": response.status,
            "message": response.message,
            "created_at": response.created_at,
            "started_at": response.started_at,
            "completed_at": response.completed_at,
        }

    def cancel_task(self, task_id: str) -> bool:
        """Cancel a task

        Args:
            task_id: Task ID

        Returns:
            True if cancelled successfully
        """
        request = pb2.CancelTaskRequest(task_id=task_id)
        response = self.stub.CancelTask(request)
        return response.success

    def get_gpu_status(self) -> List[Dict[str, Any]]:
        """Get GPU status

        Returns:
            List of GPU information
        """
        request = pb2.GetGPUStatusRequest()
        response = self.stub.GetGPUStatus(request)

        gpus = []
        for gpu in response.gpus:
            gpus.append({
                "gpu_id": gpu.gpu_id,
                "name": gpu.name,
                "memory_total": gpu.memory_total,
                "memory_used": gpu.memory_used,
                "memory_free": gpu.memory_free,
                "compute_util": gpu.compute_util,
                "memory_util": gpu.memory_util,
                "temperature": gpu.temperature,
                "running_tasks": list(gpu.running_tasks),
            })

        return gpus

    def wait(
        self, task_id: str, timeout: Optional[float] = None, poll_interval: float = 1.0
    ) -> Dict[str, Any]:
        """Wait for task to complete

        Args:
            task_id: Task ID
            timeout: Maximum time to wait in seconds
            poll_interval: Poll interval in seconds

        Returns:
            Final task status

        Raises:
            TimeoutError: If timeout is reached
        """
        start_time = time.time()

        while True:
            status = self.get_status(task_id)

            if status["status_code"] in (
                TaskStatus.COMPLETED,
                TaskStatus.FAILED,
                TaskStatus.CANCELLED,
                TaskStatus.REJECTED,
            ):
                return status

            if timeout and (time.time() - start_time) >= timeout:
                raise TimeoutError(f"Task {task_id} timed out")

            time.sleep(poll_interval)

    def close(self):
        """Close the connection"""
        self.channel.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
