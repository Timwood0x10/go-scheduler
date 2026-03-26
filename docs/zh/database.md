# SQLite 数据库设计

## 概述

SQLite 数据库存储 GPU 任务执行指标，支持数据驱动的调度决策。使用 WAL（Write-Ahead Logging）模式以获得更好的并发性能。

## 数据库结构

### task_execution 表

存储每个任务执行的详细指标。

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

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | INTEGER | 自增主键 |
| `task_id` | TEXT | 任务唯一标识 |
| `task_type` | TEXT | 任务类型（embedding, llm, diffusion 等） |
| `user_id` | TEXT | 用户标识 |
| `gpu_id` | INTEGER | 使用的 GPU ID |
| `gpu_model` | TEXT | GPU 型号 |
| `priority` | INTEGER | 提交时的任务优先级 |
| `queue_wait_ms` | INTEGER | 队列等待时间（毫秒） |
| `execution_time_ms` | INTEGER | 实际执行时间（毫秒） |
| `avg_gpu_util` | REAL | 平均 GPU 利用率（0-100） |
| `max_gpu_util` | REAL | 最大 GPU 利用率（0-100） |
| `avg_mem_util` | REAL | 平均内存利用率（0-100） |
| `max_mem_util` | REAL | 最大内存利用率（0-100） |
| `gpu_memory_used_mb` | INTEGER | GPU 内存使用量（MB） |
| `success` | INTEGER | 任务成功（1）或失败（0） |
| `created_at` | INTEGER | Unix 时间戳 |

### 索引

```sql
CREATE INDEX idx_task_type ON task_execution(task_type);
CREATE INDEX idx_user_id ON task_execution(user_id);
CREATE INDEX idx_gpu_id ON task_execution(gpu_id);
CREATE INDEX idx_created_at ON task_execution(created_at);
CREATE INDEX idx_success ON task_execution(success);
```

## 性能特性

- **写入性能**: ~10,000 次/秒
- **读取性能**: ~50,000 次/秒（带索引）
- **文件大小**: 每行约 100 字节
- **100 万记录**: 约 100 MB
- **WAL 模式**: 启用以提高并发性

## 查询示例

### 任务类型统计

```go
stats, err := store.GetTaskTypeStats(ctx, "llm")
// 返回: TaskStats 包含平均持续时间、内存、利用率
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

### 用户统计

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

### 队列等待统计

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

## 数据保留

### 自动清理

```go
// 删除 30 天前的记录
err := store.CleanupOldRecords(ctx, 30*24*time.Hour)
```

```sql
DELETE FROM task_execution
WHERE created_at < ?
```

### 性能优化

- **WAL 模式**: 减少锁竞争
- **索引**: 优化常见查询模式
- **定期清理**: 防止数据库膨胀
- **连接池**: 重用连接

## 使用示例

```go
// 初始化数据库
store, err := db.NewSQLiteStore("algogpu.db")
if err != nil {
    log.Fatal(err)
}
defer store.Close()

// 记录执行指标
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

// 查询统计信息
stats, err := store.GetTaskTypeStats(ctx, "llm")
fmt.Printf("平均持续时间: %dms\n", stats.AvgExecutionMs)
fmt.Printf("平均内存: %dMB\n", stats.AvgMemoryMB)
```

## 最佳实践

1. **使用预编译语句**: 防止 SQL 注入
2. **批量插入**: 用于高吞吐量场景
3. **定期清理**: 防止数据库膨胀
4. **连接池**: 减少连接开销
5. **索引策略**: 根据查询模式优化
6. **WAL 模式**: 更好的并发性
7. **定期清理**: 回收空间

## 性能调优

### 配置

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### SQLite 优化

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB
PRAGMA temp_store = MEMORY;
```

## 监控

### 数据库大小

```go
// 检查数据库大小
info, err := os.Stat(dbPath)
fmt.Printf("数据库大小: %d 字节\n", info.Size())
```

### 查询性能

```go
// 启用查询日志
db.SetLogger(log.New(os.Stdout, "DB: ", log.LstdFlags))
```

## 备份

### 简单备份

```bash
# 复制数据库文件
cp algogpu.db algogpu.backup.db
```

### 程序化备份

```go
// SQLite 备份 API
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

## 迁移

### 数据库版本

```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER
);
```

### 迁移逻辑

```go
currentVersion := getCurrentSchemaVersion(db)
if currentVersion < 2 {
    applyMigration(db, "v2_add_indexes.sql")
}
```

## 故障排除

### 锁问题

```sql
-- 检查锁状态
PRAGMA lock_status;
```

### 性能问题

```sql
-- 分析查询计划
EXPLAIN QUERY PLAN SELECT * FROM task_execution WHERE task_type = 'llm';

-- 检查索引使用
PRAGMA index_info(idx_task_type);
```

### 损坏修复

```bash
# 检查数据库完整性
sqlite3 algogpu.db "PRAGMA integrity_check;"

# 从损坏中恢复
sqlite3 algogpu.db ".recover" | sqlite3 algogpu_recovered.db
```

## 未来增强

1. **FTS5**: 任务元数据的全文搜索
2. **计算列**: 自动计算字段
3. **触发器**: 自动统计更新
4. **分区**: 按日期分区大数据集
5. **分片**: 用于分布式部署

## 参考资料

- [SQLite 文档](https://www.sqlite.org/docs.html)
- [WAL 模式](https://www.sqlite.org/wal.html)
- [性能指南](https://www.sqlite.org/optoverview.html)