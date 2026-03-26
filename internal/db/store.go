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

// SQLiteStore is an alias for Store for backward compatibility.
type SQLiteStore = Store

// TaskExecutionRecord represents a task execution summary.
type TaskExecutionRecord struct {
	ID              int64
	TaskID          string
	TaskType        string
	UserID          string
	GPUID           int
	GPUModel        string
	Priority        int
	QueueWaitMs     int64
	ExecutionTimeMs int64
	AvgGPUUtil      float64
	MaxGPUUtil      float64
	AvgMemUtil      float64
	MaxMemUtil      float64
	GPUMemoryUsedMB int64
	Success         bool
	CreatedAt       int64
}

// TaskStats represents aggregated statistics for a task type.
type TaskStats struct {
	TaskType       string
	Count          int64
	AvgDurationMs  int64
	AvgMemoryMB    int64
	AvgGPUUtil     float64
	AvgMemUtil     float64
	AvgQueueWaitMs int64
}

// QueueWaitStats represents queue wait time statistics.
type QueueWaitStats struct {
	TaskType     string
	AvgWaitMs    int64
	MinWaitMs    int64
	MaxWaitMs    int64
	TotalSamples int64
}

// ExecutionStats represents execution time statistics.
type ExecutionStats struct {
	TaskType       string
	AvgExecutionMs int64
	MinExecutionMs int64
	MaxExecutionMs int64
	TotalSamples   int64
}

// UserStats represents aggregated statistics for a user.
type UserStats struct {
	UserID          string
	TotalTasks      int64
	SuccessfulTasks int64
	AvgDurationMs   int64
	TotalGPUTimeSec float64
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

// NewSQLiteStore creates a new SQLite store (alias for NewStore).
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	return NewStore(dbPath)
}

// configureDB sets up SQLite with optimal performance settings.
func configureDB(db *sql.DB) error {
	pragmaSettings := []string{
		"PRAGMA journal_mode=WAL;",   // Write-Ahead Logging
		"PRAGMA synchronous=NORMAL;", // Balance safety and performance
		"PRAGMA temp_store=MEMORY;",  // Store temp tables in memory
		"PRAGMA cache_size=-10000;",  // 10MB cache
		"PRAGMA busy_timeout=5000;",  // 5 second timeout
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

// RecordExecution records a task execution to the database.
func (s *Store) RecordExecution(ctx context.Context, record *TaskExecutionRecord) error {
	query := `
		INSERT INTO task_execution (
			task_id, task_type, user_id, gpu_id, gpu_model,
			priority, queue_wait_ms, execution_time_ms,
			avg_gpu_util, max_gpu_util, avg_mem_util, max_mem_util,
			gpu_memory_used_mb, success, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	success := 0
	if record.Success {
		success = 1
	}

	result, err := s.db.ExecContext(ctx, query,
		record.TaskID, record.TaskType, record.UserID, record.GPUID, record.GPUModel,
		record.Priority, record.QueueWaitMs, record.ExecutionTimeMs,
		record.AvgGPUUtil, record.MaxGPUUtil, record.AvgMemUtil, record.MaxMemUtil,
		record.GPUMemoryUsedMB, success, record.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert execution record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	record.ID = id
	return nil
}

// RecordTaskExecution is a convenience method that converts task/gpu/metrics to a record and stores it.
func (s *Store) RecordTaskExecution(ctx context.Context, taskID, taskType, userID string, gpuID int,
	gpuModel string, priority int, queueWaitMs, executionTimeMs int64,
	avgGPUUtil, maxGPUUtil, avgMemUtil, maxMemUtil float64, gpuMemoryUsedMB int64, success bool) error {

	record := &TaskExecutionRecord{
		TaskID:          taskID,
		TaskType:        taskType,
		UserID:          userID,
		GPUID:           gpuID,
		GPUModel:        gpuModel,
		Priority:        priority,
		QueueWaitMs:     queueWaitMs,
		ExecutionTimeMs: executionTimeMs,
		AvgGPUUtil:      avgGPUUtil,
		MaxGPUUtil:      maxGPUUtil,
		AvgMemUtil:      avgMemUtil,
		MaxMemUtil:      maxMemUtil,
		GPUMemoryUsedMB: gpuMemoryUsedMB,
		Success:         success,
		CreatedAt:       time.Now().Unix(),
	}

	return s.RecordExecution(ctx, record)
}

// GetTaskTypeStats retrieves statistics for a specific task type (alias for GetTaskStats).
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

// GetTaskTypeStats retrieves statistics for a specific task type (alias for GetTaskStats).
func (s *Store) GetTaskTypeStats(ctx context.Context, taskType string) (*TaskStats, error) {
	return s.GetTaskStats(ctx, taskType, 7)
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

// GetUserStats retrieves statistics for a specific user (alias with default days).
func (s *Store) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
	return s.GetUserStats(ctx, userID, 7)
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

return &stats, nil
}

// GetGPUStats retrieves statistics for a specific GPU.
func (s *Store) GetGPUStats(ctx context.Context, gpuID int) (*TaskStats, error) {
	query := `
		SELECT 
			'task_type' as task_type,
			COUNT(*) as count,
			AVG(execution_time_ms) as avg_duration_ms,
			AVG(gpu_memory_used_mb) as avg_memory_mb,
			AVG(avg_gpu_util) as avg_gpu_util,
			AVG(avg_mem_util) as avg_mem_util,
			AVG(queue_wait_ms) as avg_queue_wait_ms
		FROM task_execution
		WHERE gpu_id = ?
	`

	row := s.db.QueryRowContext(ctx, query, gpuID)

	var stats TaskStats
	err := row.Scan(&stats.TaskType, &stats.Count, &stats.AvgDurationMs, &stats.AvgMemoryMB,
		&stats.AvgGPUUtil, &stats.AvgMemUtil, &stats.AvgQueueWaitMs)
	if err != nil {
		return nil, fmt.Errorf("failed to query GPU stats: %w", err)
	}

	return &stats, nil
}

// GetStatsSince retrieves statistics for records since a given time.
func (s *Store) GetStatsSince(ctx context.Context, since time.Time) ([]*TaskStats, error) {
	// Get all task types with recent activity
	query := `
		SELECT task_type, COUNT(*) as count
		FROM task_execution
		WHERE created_at >= ?
		GROUP BY task_type
	`

	rows, err := s.db.QueryContext(ctx, query, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query task types: %w", err)
	}
	defer rows.Close()

	var statsList []*TaskStats
	for rows.Next() {
		var taskType string
		var count int64

		if err := rows.Scan(&taskType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan task type: %w", err)
		}

		stats, err := s.GetTaskTypeStats(ctx, taskType)
		if err != nil {
			return nil, fmt.Errorf("failed to get stats for %s: %w", taskType, err)
		}

		statsList = append(statsList, stats)
	}

	return statsList, nil
}

// GetTotalExecutions returns the total number of executions.
func (s *Store) GetTotalExecutions(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_execution").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query total executions: %w", err)
	}
	return count, nil
}

// GetExecutionsSince returns the number of executions since a given time.
func (s *Store) GetExecutionsSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_execution WHERE created_at >= ?", since.Unix()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to query executions since: %w", err)
	}
	return count, nil
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

// GetQueueWaitStats retrieves queue wait statistics for a task type.
func (s *Store) GetQueueWaitStats(ctx context.Context, taskType string) (*QueueWaitStats, error) {
	query := `
		SELECT 
			AVG(queue_wait_ms) as avg_wait_ms,
			MIN(queue_wait_ms) as min_wait_ms,
			MAX(queue_wait_ms) as max_wait_ms,
			COUNT(*) as total_samples
		FROM task_execution
		WHERE task_type = ? AND queue_wait_ms > 0
	`

	row := s.db.QueryRowContext(ctx, query, taskType)

	var stats QueueWaitStats
	err := row.Scan(&stats.AvgWaitMs, &stats.MinWaitMs, &stats.MaxWaitMs, &stats.TotalSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to query queue wait stats: %w", err)
	}

	stats.TaskType = taskType
	return &stats, nil
}

// GetExecutionStats retrieves execution time statistics for a task type.
func (s *Store) GetExecutionStats(ctx context.Context, taskType string) (*ExecutionStats, error) {
	query := `
		SELECT 
			AVG(execution_time_ms) as avg_execution_ms,
			MIN(execution_time_ms) as min_execution_ms,
			MAX(execution_time_ms) as max_execution_ms,
			COUNT(*) as total_samples
		FROM task_execution
		WHERE task_type = ?
	`

	row := s.db.QueryRowContext(ctx, query, taskType)

	var stats ExecutionStats
	err := row.Scan(&stats.AvgExecutionMs, &stats.MinExecutionMs, &stats.MaxExecutionMs, &stats.TotalSamples)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution stats: %w", err)
	}

	stats.TaskType = taskType
	return &stats, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// SQLiteStore is an alias for Store for backward compatibility.
type SQLiteStore = Store

// TaskStats represents aggregated statistics for a task type.
type TaskStats struct {
	TaskType       string
	Count          int64
	AvgDurationMs  int64
	AvgMemoryMB    int64
	AvgGPUUtil     float64
	AvgMemUtil     float64
	AvgQueueWaitMs int64
}

// QueueWaitStats represents queue wait time statistics.
type QueueWaitStats struct {
	TaskType     string
	AvgWaitMs    int64
	MinWaitMs    int64
	MaxWaitMs    int64
	TotalSamples int64
}

// ExecutionStats represents execution time statistics.
type ExecutionStats struct {
	TaskType       string
	AvgExecutionMs int64
	MinExecutionMs int64
	MaxExecutionMs int64
	TotalSamples   int64
}

// UserStats represents aggregated statistics for a user.
type UserStats struct {
	UserID          string
	TotalTasks      int64
	SuccessfulTasks int64
	AvgDurationMs   int64
	TotalGPUTimeSec float64
}
