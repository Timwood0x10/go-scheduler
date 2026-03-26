// Package db provides SQLite storage for GPU task execution metrics.
// It enables data-driven scheduling by recording historical GPU performance data.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages SQLite database for task execution metrics.
type Store struct {
	db *sql.DB
}

// TaskExecutionRecord represents a task execution summary.
type TaskExecutionRecord struct {
	ID               int64
	TaskID           string
	TaskType         string
	UserID           string
	GPUID            int
	GPUModel         string
	Priority         int
	QueueWaitMs      int64
	ExecutionTimeMs  int64
	AvgGPUUtil       float64
	MaxGPUUtil       float64
	AvgMemUtil       float64
	MaxMemUtil       float64
	GPUMemoryUsedMB  int64
	Success          bool
	CreatedAt        int64
}

// NewStore creates a new SQLite store with WAL mode enabled.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if err := configureDB(db); err != nil {
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	// Create tables
	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	store := &Store{db: db}

	// Start cleanup routine
	go store.cleanupRoutine()

	return store, nil
}

// configureDB sets up SQLite with optimal performance settings.
func configureDB(db *sql.DB) error {
	pragmaSettings := []string{
		"PRAGMA journal_mode=WAL;",      // Write-Ahead Logging
		"PRAGMA synchronous=NORMAL;",    // Balance safety and performance
		"PRAGMA temp_store=MEMORY;",     // Store temp tables in memory
		"PRAGMA cache_size=-10000;",     // 10MB cache
		"PRAGMA busy_timeout=5000;",     // 5 second timeout
	}

	for _, pragma := range pragmaSettings {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to set %s: %w", pragma, err)
		}
	}

	return nil
}

// createTables creates the task_execution table and indexes.
func createTables(db *sql.DB) error {
	// Create task_execution table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS task_execution (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id TEXT NOT NULL,
		task_type TEXT NOT NULL,
		user_id TEXT,
		gpu_id INTEGER,
		gpu_model TEXT,
		priority INTEGER,
		queue_wait_ms INTEGER,
		execution_time_ms INTEGER,
		avg_gpu_util REAL,
		max_gpu_util REAL,
		avg_mem_util REAL,
		max_mem_util REAL,
		gpu_memory_used_mb INTEGER,
		success INTEGER,
		created_at INTEGER
	);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_type ON task_execution(task_type);",
		"CREATE INDEX IF NOT EXISTS idx_created_at ON task_execution(created_at);",
		"CREATE INDEX IF NOT EXISTS idx_gpu_model ON task_execution(gpu_model);",
		"CREATE INDEX IF NOT EXISTS idx_user_id ON task_execution(user_id);",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// RecordExecution saves task execution metrics to database.
func (s *Store) RecordExecution(ctx context.Context, record *TaskExecutionRecord) error {
	success := 0
	if record.Success {
		success = 1
	}

	query := `
	INSERT INTO task_execution 
	(task_id, task_type, user_id, gpu_id, gpu_model, priority, 
	 queue_wait_ms, execution_time_ms, avg_gpu_util, max_gpu_util,
	 avg_mem_util, max_mem_util, gpu_memory_used_mb, success, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		record.TaskID,
		record.TaskType,
		record.UserID,
		record.GPUID,
		record.GPUModel,
		record.Priority,
		record.QueueWaitMs,
		record.ExecutionTimeMs,
		record.AvgGPUUtil,
		record.MaxGPUUtil,
		record.AvgMemUtil,
		record.MaxMemUtil,
		record.GPUMemoryUsedMB,
		success,
		record.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}

	return nil
}

// GetTaskStats retrieves statistics for a specific task type.
func (s *Store) GetTaskStats(ctx context.Context, taskType string, days int) (*TaskStats, error) {
	stats := &TaskStats{
		TaskType: taskType,
	}

	// Calculate timestamp for N days ago
	cutoffTime := time.Now().AddDate(0, 0, -days).Unix()

	query := `
	SELECT 
		COUNT(*) as count,
		COALESCE(AVG(execution_time_ms), 0) as avg_duration_ms,
		COALESCE(AVG(gpu_memory_used_mb), 0) as avg_memory_mb,
		COALESCE(AVG(avg_gpu_util), 0) as avg_gpu_util,
		COALESCE(AVG(avg_mem_util), 0) as avg_mem_util,
		COALESCE(AVG(queue_wait_ms), 0) as avg_queue_wait_ms
	FROM task_execution
	WHERE task_type = ? AND created_at > ? AND success = 1
	`

	err := s.db.QueryRowContext(ctx, query, taskType, cutoffTime).Scan(
		&stats.Count,
		&stats.AvgDurationMs,
		&stats.AvgMemoryMB,
		&stats.AvgGPUUtil,
		&stats.AvgMemUtil,
		&stats.AvgQueueWaitMs,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get task stats: %w", err)
	}

	return stats, nil
}

// GetUserStats retrieves statistics for a specific user.
func (s *Store) GetUserStats(ctx context.Context, userID string, days int) (*UserStats, error) {
	stats := &UserStats{
		UserID: userID,
	}

	cutoffTime := time.Now().AddDate(0, 0, -days).Unix()

	query := `
	SELECT 
		COUNT(*) as total_tasks,
		SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_tasks,
		COALESCE(AVG(execution_time_ms), 0) as avg_duration_ms,
		COALESCE(SUM(execution_time_ms) / 1000.0, 0) as total_gpu_time_sec
	FROM task_execution
	WHERE user_id = ? AND created_at > ?
	`

	err := s.db.QueryRowContext(ctx, query, userID, cutoffTime).Scan(
		&stats.TotalTasks,
		&stats.SuccessfulTasks,
		&stats.AvgDurationMs,
		&stats.TotalGPUTimeSec,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get user stats: %w", err)
	}

	return stats, nil
}

// GetRecentExecutions retrieves recent task executions.
func (s *Store) GetRecentExecutions(ctx context.Context, limit int) ([]TaskExecutionRecord, error) {
	query := `
	SELECT id, task_id, task_type, user_id, gpu_id, gpu_model, priority,
		queue_wait_ms, execution_time_ms, avg_gpu_util, max_gpu_util,
		avg_mem_util, max_mem_util, gpu_memory_used_mb, success, created_at
	FROM task_execution
	ORDER BY created_at DESC
	LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent executions: %w", err)
	}
	defer rows.Close()

	var records []TaskExecutionRecord
	for rows.Next() {
		var record TaskExecutionRecord
		var success int

		err := rows.Scan(
			&record.ID,
			&record.TaskID,
			&record.TaskType,
			&record.UserID,
			&record.GPUID,
			&record.GPUModel,
			&record.Priority,
			&record.QueueWaitMs,
			&record.ExecutionTimeMs,
			&record.AvgGPUUtil,
			&record.MaxGPUUtil,
			&record.AvgMemUtil,
			&record.MaxMemUtil,
			&record.GPUMemoryUsedMB,
			&success,
			&record.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}

		record.Success = success == 1
		records = append(records, record)
	}

	return records, nil
}

// cleanupRoutine periodically cleans old data.
func (s *Store) cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.cleanupOldData(); err != nil {
			log.Printf("Failed to cleanup old data: %v", err)
		}
	}
}

// cleanupOldData removes records older than 30 days.
func (s *Store) cleanupOldData() error {
	ctx := context.Background()
	cutoffTime := time.Now().AddDate(0, 0, -30).Unix()

	result, err := s.db.ExecContext(ctx,
		"DELETE FROM task_execution WHERE created_at < ?",
		cutoffTime,
	)

	if err != nil {
		return fmt.Errorf("failed to cleanup old data: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old task execution records", rowsAffected)
	}

	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// TaskStats represents aggregated statistics for a task type.
type TaskStats struct {
	TaskType         string
	Count            int64
	AvgDurationMs    int64
	AvgMemoryMB      int64
	AvgGPUUtil       float64
	AvgMemUtil       float64
	AvgQueueWaitMs   int64
}

// UserStats represents aggregated statistics for a user.
type UserStats struct {
	UserID            string
	TotalTasks        int64
	SuccessfulTasks   int64
	AvgDurationMs     int64
	TotalGPUTimeSec   float64
}