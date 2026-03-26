# SQLite Database Design

## Overview

The SQLite database stores GPU task execution metrics to enable data-driven scheduling decisions. It uses WAL (Write-Ahead Logging) mode for better concurrency and performance.

## Schema

### task_execution Table

Stores detailed metrics for each task execution.

```sql
CREATE TABLE task_execution (
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
```

### Columns

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `task_id` | TEXT | Unique task identifier |
| `task_type` | TEXT | Task type (embedding, llm, diffusion, etc.) |
| `user_id` | TEXT | User identifier |
| `gpu_id` | INTEGER | GPU ID used for execution |
| `gpu_model` | TEXT | GPU model name |
| `priority` | INTEGER | Task priority at submission |
| `queue_wait_ms` | INTEGER | Time spent in queue (milliseconds) |
| `execution_time_ms` | INTEGER | Actual execution time (milliseconds) |
| `avg_gpu_util` | REAL | Average GPU utilization (0-100) |
| `max_gpu_util` | REAL | Maximum GPU utilization (0-100) |
| `avg_mem_util` | REAL | Average memory utilization (0-100) |
| `max_mem_util` | REAL | Maximum memory utilization (0-100) |
| `gpu_memory_used_mb` | INTEGER | GPU memory used (MB) |
| `success` | INTEGER | Task success (1) or failure (0) |
| `created_at` | INTEGER | Unix timestamp |

### Indexes

```sql
CREATE INDEX idx_task_type ON task_execution(task_type);
CREATE INDEX idx_user_id ON task_execution(user_id);
CREATE INDEX idx_gpu_id ON task_execution(gpu_id);
CREATE INDEX idx_created_at ON task_execution(created_at);
CREATE INDEX idx_success ON task_execution(success);
```

## Performance Characteristics

- **Write Performance**: ~10,000 inserts/second
- **Read Performance**: ~50,000 queries/second (with indexes)
- **File Size**: ~100 bytes per row
- **1M Records**: ~100 MB
- **WAL Mode**: Enabled for better concurrency

## Queries

### Task Type Statistics

```go
stats, err := store.GetTaskTypeStats(ctx, "llm")
// Returns: TaskStats with avg duration, memory, utilization
```

```sql
SELECT 
    COUNT(*) as total_tasks,
    AVG(execution_time_ms) as avg_execution_ms,
    AVG(gpu_memory_used_mb) as avg_memory_mb,
    AVG(avg_gpu_util) as avg_gpu_util,
    AVG(avg_mem_util) as avg_mem_util,
    SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_tasks
FROM task_execution
WHERE task_type = ?
```

### User Statistics

```go
stats, err := store.GetUserStats(ctx, "user-123")
```

```sql
SELECT 
    COUNT(*) as total_tasks,
    AVG(execution_time_ms) as avg_execution_ms,
    AVG(gpu_memory_used_mb) as avg_memory_mb,
    AVG(avg_gpu_util) as avg_gpu_util,
    AVG(avg_mem_util) as avg_mem_util,
    SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_tasks
FROM task_execution
WHERE user_id = ?
```

### Queue Wait Statistics

```go
waitStats, err := store.GetQueueWaitStats(ctx, "llm")
```

```sql
SELECT 
    AVG(queue_wait_ms) as avg_wait_ms,
    MIN(queue_wait_ms) as min_wait_ms,
    MAX(queue_wait_ms) as max_wait_ms,
    COUNT(*) as total_samples
FROM task_execution
WHERE task_type = ? AND queue_wait_ms > 0
```

## Data Retention

### Automatic Cleanup

```go
// Delete records older than 30 days
err := store.CleanupOldRecords(ctx, 30*24*time.Hour)
```

```sql
DELETE FROM task_execution
WHERE created_at < ?
```

### Performance Optimization

- **WAL Mode**: Reduces lock contention
- **Indexes**: Optimize common query patterns
- **Periodic Vacuum**: Reclaim space
- **Connection Pooling**: Reuse connections

## Usage Example

```go
// Initialize database
store, err := db.NewSQLiteStore("algogpu.db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// Record execution metrics
metrics := &types.ExecutionMetrics{
    ExecutionTimeMs: 1500,
    AvgGPUUtil:      85.5,
    MaxGPUUtil:      95.0,
    AvgMemUtil:      70.0,
    MaxMemUtil:      80.0,
    GPUMemoryUsedMB: 4096,
    Success:         true,
}

err = store.RecordExecution(ctx, task, gpu, queueWait, metrics)

// Query statistics
stats, err := store.GetTaskTypeStats(ctx, "llm")
fmt.Printf("Avg duration: %dms\n", stats.AvgExecutionMs)
fmt.Printf("Avg memory: %dMB\n", stats.AvgMemoryMB)
```

## Best Practices

1. **Use Prepared Statements**: Prevent SQL injection
2. **Batch Inserts**: For high-throughput scenarios
3. **Regular Cleanup**: Prevent database bloat
4. **Connection Pooling**: Reduce connection overhead
5. **Index Strategy**: Optimize for query patterns
6. **WAL Mode**: Better concurrency
7. **Vacuum Periodically**: Reclaim space

## Performance Tuning

### Configuration

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### SQLite Pragmas

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB
PRAGMA temp_store = MEMORY;
```

## Monitoring

### Database Size

```go
// Check database size
info, err := os.Stat(dbPath)
fmt.Printf("DB size: %d bytes\n", info.Size())
```

### Query Performance

```go
// Enable query logging
db.SetLogger(log.New(os.Stdout, "DB: ", log.LstdFlags))
```

## Backup

### Simple Backup

```bash
# Copy database file
cp algogpu.db algogpu.backup.db
```

### Programmatic Backup

```go
// SQLite backup API
backup, err := os.Create("algogpu.backup.db")
if err != nil {
    return err
}
defer backup.Close()

err = sqlite3_backup_init(backup, "main", db, "main")
if err != nil {
    return err
}

for {
    done, err := sqlite3_backup_step(backup, -1)
    if err != nil {
        return err
    }
    if done {
        break
    }
}
```

## Migration

### Schema Version

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER
);
```

### Migration Logic

```go
currentVersion := getCurrentSchemaVersion(db)
if currentVersion < 2 {
    applyMigration(db, "v2_add_indexes.sql")
}
```

## Troubleshooting

### Lock Issues

```sql
-- Check for locks
PRAGMA lock_status;
```

### Performance Issues

```sql
-- Analyze query plan
EXPLAIN QUERY PLAN SELECT * FROM task_execution WHERE task_type = 'llm';

-- Check index usage
PRAGMA index_info(idx_task_type);
```

### Corruption

```bash
# Check database integrity
sqlite3 algogpu.db "PRAGMA integrity_check;"

# Recover from corruption
sqlite3 algogpu.db ".recover" | sqlite3 algogpu_recovered.db
```

## Future Enhancements

1. **FTS5**: Full-text search for task metadata
2. **Computed Columns**: Automatically calculated fields
3. **Triggers**: Automatic statistics updates
4. **Partitioning**: By date for large datasets
5. **Sharding**: For distributed deployments

## References

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [WAL Mode](https://www.sqlite.org/wal.html)
- [Performance Guidelines](https://www.sqlite.org/optoverview.html)