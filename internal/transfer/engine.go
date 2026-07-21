package transfer

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"go-database/internal/plugin"
)

// connManager is the subset of connection.Manager that the engine needs.
type connManager interface {
	Schema(ctx context.Context, connID string) (*plugin.Schema, error)
	Query(ctx context.Context, connID string, query string) (*plugin.Result, error)
	Execute(ctx context.Context, connID string, query string) (*plugin.Result, error)
}

type engine struct {
	mgr  connManager
	mu   sync.RWMutex
	jobs map[string]*jobState
}

type jobState struct {
	mu     sync.Mutex
	Job    TransferJob
	Log    []string
	Tables []string
	done   chan struct{}
	subs   []chan ProgressEvent
	subMu  sync.Mutex
	cancel context.CancelFunc
}

// NewEngine creates a new transfer engine
func NewEngine(mgr connManager) TransferEngine {
	return &engine{
		mgr:  mgr,
		jobs: make(map[string]*jobState),
	}
}

func (e *engine) Start(ctx context.Context, job *TransferJob) error {
	if job.BatchSize <= 0 {
		job.BatchSize = 100
	}
	if job.ID == "" {
		job.ID = fmt.Sprintf("transfer-%d", time.Now().UnixNano())
	}
	job.CreatedAt = time.Now()
	job.Status = "pending"

	ctx, cancel := context.WithCancel(ctx)
	js := &jobState{
		Job:    *job,
		Log:    []string{},
		done:   make(chan struct{}),
		cancel: cancel,
	}

	e.mu.Lock()
	e.jobs[job.ID] = js
	e.mu.Unlock()

	go e.run(ctx, js)
	return nil
}

func (e *engine) Subscribe(jobID string) (<-chan ProgressEvent, error) {
	e.mu.RLock()
	js, ok := e.jobs[jobID]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("transfer: job %s not found", jobID)
	}
	ch := make(chan ProgressEvent, 128)
	js.subMu.Lock()
	js.subs = append(js.subs, ch)
	js.subMu.Unlock()
	return ch, nil
}

func (e *engine) Unsubscribe(jobID string, ch <-chan ProgressEvent) {
	e.mu.RLock()
	js, ok := e.jobs[jobID]
	e.mu.RUnlock()
	if !ok {
		return
	}
	js.subMu.Lock()
	for i, sub := range js.subs {
		if sub == ch {
			js.subs = append(js.subs[:i], js.subs[i+1:]...)
			close(sub)
			break
		}
	}
	js.subMu.Unlock()
}

func (e *engine) emit(js *jobState, event ProgressEvent) {
	js.subMu.Lock()
	subs := js.subs
	js.subMu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (e *engine) run(ctx context.Context, js *jobState) {
	js.mu.Lock()
	js.Job.Status = "running"
	js.mu.Unlock()
	defer close(js.done)

	job := js.Job
	e.log(js, "Starting transfer from %s to %s", job.SourceType, job.TargetType)

	// 1. Get source schema
	e.log(js, "Fetching source schema...")
	srcSchema, err := e.getSchema(ctx, job.SourceConn)
	if err != nil {
		e.fail(js, "source schema: %v", err)
		return
	}

	// 2. Filter tables
	tables := filterTables(srcSchema, job.Tables)
	e.log(js, "Found %d tables to transfer", len(tables))

	if job.DryRun {
		e.log(js, "DRY RUN — would create %d tables", len(tables))
		for _, t := range tables {
			createSQL := e.generateCreateSQL(t, plugin.DBType(job.TargetType))
			e.log(js, "CREATE TABLE %s:\n%s", t.Name, createSQL)
			e.log(js, "Would transfer ~%d rows", t.RowCount)
		}
		js.mu.Lock()
		js.Job.Status = "done"
		js.mu.Unlock()
		return
	}

	// 3. Get target connection and create tables
	targetType := plugin.DBType(job.TargetType)
	for _, t := range tables {
		select {
		case <-ctx.Done():
			e.fail(js, "cancelled")
			return
		default:
		}

		e.log(js, "[%s] Creating table...", t.Name)

		// Create table on target
		if err := e.execute(ctx, job.TargetConn, e.generateCreateSQL(t, targetType)); err != nil {
			e.log(js, "[%s] Create table error (may already exist): %v", t.Name, err)
		}

		// Handle conflict strategy
		switch job.OnConflict {
		case "overwrite":
			e.log(js, "[%s] Truncating existing data...", t.Name)
			e.execute(ctx, job.TargetConn, e.truncateSQL(t.Name, targetType))
		case "skip":
			count, _ := e.countRows(ctx, job.TargetConn, t.Name)
			if count > 0 {
				e.log(js, "[%s] Skipping — already has %d rows", t.Name, count)
				continue
			}
		}

		// Read and transfer data
		e.emit(js, ProgressEvent{Type: "table_start", Table: t.Name, Total: int64(t.RowCount), Timestamp: time.Now()})
		e.log(js, "[%s] Transferring ~%d rows...", t.Name, t.RowCount)
		totalRows, err := e.transferTable(ctx, js, t, targetType)
		if err != nil {
			e.fail(js, "[%s] transfer failed: %v", t.Name, err)
			return
		}
		e.emit(js, ProgressEvent{Type: "table_done", Table: t.Name, Rows: int64(totalRows), Timestamp: time.Now()})
		e.log(js, "[%s] Done — %d rows transferred", t.Name, totalRows)
	}

	js.mu.Lock()
	js.Job.Status = "done"
	js.Tables = tableNames(tables)
	js.mu.Unlock()
	e.log(js, "Transfer complete — %d tables processed", len(tables))
	e.emit(js, ProgressEvent{Type: "complete", Message: "All tables processed", Timestamp: time.Now()})
}

func (e *engine) transferTable(ctx context.Context, js *jobState, t plugin.TableInfo, targetType plugin.DBType) (int, error) {
	js.mu.Lock()
	batchSize := js.Job.BatchSize
	srcConn := js.Job.SourceConn
	tgtConn := js.Job.TargetConn
	srcType := js.Job.SourceType
	js.mu.Unlock()

	offset := 0
	total := 0
	batchNum := 0

	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		rows, err := e.readPage(ctx, srcConn, t.Name, offset, batchSize)
		if err != nil {
			return total, fmt.Errorf("read: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		// Map values for target
		mappedRows := make([]Row, len(rows))
		for i, row := range rows {
			mapped := make(Row)
			for k, v := range row {
				mapped[k] = ValueMapper(v, plugin.DBType(srcType), targetType)
			}
			mappedRows[i] = mapped
		}

		// Insert into target
		insertSQL := e.generateInsertSQL(t, mappedRows, targetType)
		if insertSQL == "" {
			offset += len(rows)
			total += len(rows)
			continue
		}

		if err := e.execute(ctx, tgtConn, insertSQL); err != nil {
			return total, fmt.Errorf("insert batch: %w", err)
		}

		total += len(rows)
		offset += len(rows)
		batchNum++
		e.emit(js, ProgressEvent{Type: "batch", Table: t.Name, Batch: batchNum, Rows: int64(total), Timestamp: time.Now()})
	}

	return total, nil
}

func (e *engine) getSchema(ctx context.Context, connID string) ([]plugin.TableInfo, error) {
	schema, err := e.mgr.Schema(ctx, connID)
	if err != nil {
		return nil, err
	}
	return schema.Tables, nil
}

func (e *engine) readPage(ctx context.Context, connID, table string, offset, limit int) ([]Row, error) {
	query := fmt.Sprintf(`SELECT * FROM %s LIMIT %d OFFSET %d`, quoteIdent(table), limit, offset)
	result, err := e.mgr.Query(ctx, connID, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	rows := make([]Row, len(result.Rows))
	for i, vals := range result.Rows {
		row := make(Row)
		for j, col := range result.Columns {
			if j < len(vals) {
				row[col] = vals[j]
			}
		}
		rows[i] = row
	}
	return rows, nil
}

func (e *engine) execute(ctx context.Context, connID, query string) error {
	_, err := e.mgr.Execute(ctx, connID, query)
	return err
}

func (e *engine) countRows(ctx context.Context, connID, table string) (int, error) {
	result, err := e.mgr.Query(ctx, connID, fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s", quoteIdent(table)))
	if err != nil {
		return 0, err
	}
	if len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
		if n, ok := result.Rows[0][0].(int64); ok {
			return int(n), nil
		}
	}
	return 0, nil
}

func (e *engine) generateCreateSQL(t plugin.TableInfo, target plugin.DBType) string {
	var cols []string
	pk := ""
	for _, c := range t.Columns {
		targetType := mapTypeToTarget(c.Type, target)
		colDef := fmt.Sprintf("  %s %s", quoteIdent(c.Name), targetType)

		if !c.Nullable {
			colDef += " NOT NULL"
		}
		if c.Default != "" && target != plugin.TypeSQLite {
			colDef += fmt.Sprintf(" DEFAULT %s", c.Default)
		}
		if c.Primary {
			pk = c.Name
		}
		cols = append(cols, colDef)
	}

	if pk != "" {
		cols = append(cols, fmt.Sprintf("  PRIMARY KEY (%s)", quoteIdent(pk)))
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", quoteIdent(t.Name), strings.Join(cols, ",\n"))
}

func (e *engine) generateInsertSQL(t plugin.TableInfo, rows []Row, target plugin.DBType) string {
	if len(rows) == 0 {
		return ""
	}

	// Build column list from first row
	var colNames []string
	for k := range rows[0] {
		colNames = append(colNames, k)
	}
	sort.Strings(colNames)

	quotedCols := make([]string, len(colNames))
	for i, c := range colNames {
		quotedCols[i] = quoteIdent(c)
	}

	var values []string
	for _, row := range rows {
		var vals []string
		for _, c := range colNames {
			v := row[c]
			if v == nil {
				vals = append(vals, "NULL")
			} else {
				vals = append(vals, escapeValue(v, target))
			}
		}
		values = append(values, "("+strings.Join(vals, ",")+")")
	}

	table := quoteIdent(t.Name)
	if target == plugin.TypePostgres {
		// PostgreSQL: INSERT ... ON CONFLICT DO NOTHING for idempotency
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES\n%s\nON CONFLICT DO NOTHING;",
			table, strings.Join(quotedCols, ","), strings.Join(values, ",\n"))
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES\n%s;",
		table, strings.Join(quotedCols, ","), strings.Join(values, ",\n"))
}

func (e *engine) truncateSQL(table string, target plugin.DBType) string {
	if target == plugin.TypeSQLite {
		return fmt.Sprintf("DELETE FROM %s", quoteIdent(table))
	}
	return fmt.Sprintf("TRUNCATE TABLE %s", quoteIdent(table))
}

func (e *engine) log(js *jobState, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Info("transfer", "job", js.Job.ID, "msg", msg)
	js.mu.Lock()
	js.Log = append(js.Log, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), msg))
	js.mu.Unlock()
	e.emit(js, ProgressEvent{Type: "log", Message: msg, Timestamp: time.Now()})
}

func (e *engine) fail(js *jobState, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Error("transfer failed", "job", js.Job.ID, "error", msg)
	js.mu.Lock()
	js.Log = append(js.Log, fmt.Sprintf("[%s] FAILED: %s", time.Now().Format(time.RFC3339), msg))
	js.Job.Status = "failed"
	js.mu.Unlock()
	e.emit(js, ProgressEvent{Type: "error", Error: msg, Timestamp: time.Now()})
}

func (e *engine) Status(jobID string) (*TransferJob, error) {
	e.mu.RLock()
	js, ok := e.jobs[jobID]
	e.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("transfer: job %s not found", jobID)
	}
	js.mu.Lock()
	defer js.mu.Unlock()
	copy := js.Job
	copy.Tables = js.Tables
	copy.Log = append([]string{}, js.Log...)
	return &copy, nil
}

func (e *engine) Cancel(jobID string) error {
	e.mu.RLock()
	js, ok := e.jobs[jobID]
	e.mu.RUnlock()
	if !ok {
		return fmt.Errorf("transfer: job %s not found", jobID)
	}
	js.mu.Lock()
	js.Job.Status = "cancelled"
	if js.cancel != nil {
		js.cancel()
	}
	js.mu.Unlock()
	return nil
}

func (e *engine) List() ([]TransferJob, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	jobs := make([]TransferJob, 0, len(e.jobs))
	for _, js := range e.jobs {
		js.mu.Lock()
		jobs = append(jobs, js.Job)
		js.mu.Unlock()
	}
	return jobs, nil
}

// Helpers

func filterTables(tables []plugin.TableInfo, names []string) []plugin.TableInfo {
	if len(names) == 0 {
		return tables
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	var filtered []plugin.TableInfo
	for _, t := range tables {
		if set[t.Name] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func tableNames(tables []plugin.TableInfo) []string {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.Name
	}
	return names
}

func quoteIdent(name string) string {
	// Use double quotes for identifiers (works in PG, MySQL in ANSI mode, SQLite)
	return `"` + name + `"`
}

func escapeValue(v any, target plugin.DBType) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)
	case float32, float64:
		return fmt.Sprintf("%v", val)
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	default:
		s := fmt.Sprintf("%v", val)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}
