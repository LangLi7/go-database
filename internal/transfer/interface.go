package transfer

import (
	"context"
	"time"
)

// Source defines a readable data source (any DB type)
type Source interface {
	// Tables lists all available tables/collections
	Tables(ctx context.Context) ([]string, error)

	// Schema returns the schema for a specific table
	Schema(ctx context.Context, table string) (*TableSchema, error)

	// Read streams rows from a table
	Read(ctx context.Context, table string, batchSize int, send func([]Row) error) error
}

// Target defines a writable data destination (any DB type)
type Target interface {
	// CreateTable creates a table matching the given schema
	CreateTable(ctx context.Context, schema TableSchema) error

	// Write inserts a batch of rows
	Write(ctx context.Context, table string, rows []Row) error

	// Truncate clears a table before transfer
	Truncate(ctx context.Context, table string) error
}

// TableSchema describes a table structure
type TableSchema struct {
	Table   string
	Columns []ColumnSchema
	PK      string // Primary key column name
}

// ColumnSchema describes a single column
type ColumnSchema struct {
	Name     string
	Type     string // Normalized type name
	Nullable bool
	Default  *string
}

// Row is a single row of data (column name → value)
type Row map[string]any

// TransferJob represents a single data transfer operation
type TransferJob struct {
	ID         string
	SourceType string   // e.g. "postgres", "sqlite"
	TargetType string   // e.g. "mysql", "mongodb"
	SourceConn string   // Connection ID
	TargetConn string   // Connection ID
	Tables     []string // Empty = all tables
	DryRun     bool
	BatchSize  int
	OnConflict string // "error" | "skip" | "overwrite"
	CreatedAt  time.Time
	Status     string // "pending" | "running" | "done" | "failed"
	Log        []string
}

// TypeMapper converts types between DB systems
type TypeMapper interface {
	// MapType converts a source type to the target system type
	MapType(sourceType string, targetSystem string) string
}

// ProgressTracker reports transfer progress
type ProgressTracker interface {
	OnTableStart(table string, totalRows int)
	OnBatchComplete(table string, batchIndex int, rowsProcessed int)
	OnTableComplete(table string, totalRows int)
	OnError(table string, rowIndex int, err error)
	OnComplete(job TransferJob)
}

// ProgressEvent is sent to WebSocket subscribers
type ProgressEvent struct {
	Type      string    `json:"type"` // "log" | "batch" | "table_start" | "table_done" | "error" | "complete"
	Table     string    `json:"table,omitempty"`
	Batch     int       `json:"batch,omitempty"`
	Rows      int64     `json:"rows,omitempty"`
	Total     int64     `json:"total,omitempty"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TransferEngine orchestrates data transfer between any two DBs
type TransferEngine interface {
	// Start begins a transfer job (async)
	// On success, job.ID is populated with the generated identifier.
	Start(ctx context.Context, job *TransferJob) error

	// Status returns current job status
	Status(jobID string) (*TransferJob, error)

	// Cancel stops a running job
	Cancel(jobID string) error

	// List returns all jobs
	List() ([]TransferJob, error)

	// Subscribe returns a channel that receives progress events for a job
	Subscribe(jobID string) (<-chan ProgressEvent, error)

	// Unsubscribe removes a channel from receiving progress events
	Unsubscribe(jobID string, ch <-chan ProgressEvent)
}
