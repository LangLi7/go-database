package transfer

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-database/internal/connection"
	"go-database/internal/plugin"
	_ "go-database/plugins/sqlite"
)

// mockManager implements connManager for testing
type mockManager struct {
	schemaFn  func(ctx context.Context, connID string) (*plugin.Schema, error)
	queryFn   func(ctx context.Context, connID string, query string) (*plugin.Result, error)
	executeFn func(ctx context.Context, connID string, query string) (*plugin.Result, error)
}

func (m *mockManager) Schema(ctx context.Context, connID string) (*plugin.Schema, error) {
	return m.schemaFn(ctx, connID)
}

func (m *mockManager) Query(ctx context.Context, connID string, query string) (*plugin.Result, error) {
	return m.queryFn(ctx, connID, query)
}

func (m *mockManager) Execute(ctx context.Context, connID string, query string) (*plugin.Result, error) {
	return m.executeFn(ctx, connID, query)
}

func mockSchema() *plugin.Schema {
	return &plugin.Schema{
		Tables: []plugin.TableInfo{
			{
				Name:     "users",
				RowCount: 3,
				Columns: []plugin.ColumnInfo{
					{Name: "id", Type: "INTEGER", Primary: true, Nullable: false},
					{Name: "name", Type: "VARCHAR(255)", Nullable: false},
					{Name: "email", Type: "VARCHAR(255)", Nullable: true},
				},
			},
			{
				Name:     "orders",
				RowCount: 5,
				Columns: []plugin.ColumnInfo{
					{Name: "id", Type: "SERIAL", Primary: true, Nullable: false},
					{Name: "user_id", Type: "INTEGER", Nullable: false},
					{Name: "total", Type: "DECIMAL(10,2)", Nullable: false},
				},
			},
		},
	}
}

func mockRows(table string) *plugin.Result {
	switch table {
	case "users":
		return &plugin.Result{
			Columns: []string{"id", "name", "email"},
			Rows: [][]any{
				{int64(1), "Alice", "alice@test.com"},
				{int64(2), "Bob", "bob@test.com"},
				{int64(3), "Charlie", nil},
			},
			RowsAffected: 3,
		}
	case "orders":
		return &plugin.Result{
			Columns: []string{"id", "user_id", "total"},
			Rows: [][]any{
				{int64(1), int64(1), 99.99},
				{int64(2), int64(1), 49.50},
				{int64(3), int64(2), 199.99},
				{int64(4), int64(2), 29.99},
				{int64(5), int64(3), 149.00},
			},
			RowsAffected: 5,
		}
	default:
		return nil
	}
}

// ─── Helper function tests ─────────────────────────────────────────────────

func TestFilterTables(t *testing.T) {
	tables := mockSchema().Tables

	result := filterTables(tables, nil)
	if len(result) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(result))
	}

	result = filterTables(tables, []string{"users"})
	if len(result) != 1 || result[0].Name != "users" {
		t.Fatalf("expected only users table, got %v", result)
	}

	result = filterTables(tables, []string{"nonexistent"})
	if len(result) != 0 {
		t.Fatalf("expected 0 tables, got %d", len(result))
	}
}

func TestTableNames(t *testing.T) {
	tables := mockSchema().Tables
	names := tableNames(tables)
	if len(names) != 2 || names[0] != "users" || names[1] != "orders" {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", `"users"`},
		{"table name", `"table name"`},
		{"", `""`},
	}
	for _, tt := range tests {
		got := quoteIdent(tt.input)
		if got != tt.want {
			t.Errorf("quoteIdent(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestEscapeValue(t *testing.T) {
	tests := []struct {
		val  any
		want string
	}{
		{nil, "NULL"},
		{true, "TRUE"},
		{false, "FALSE"},
		{int64(42), "42"},
		{3.14, "3.14"},
		{"hello", "'hello'"},
		{"it's", "'it''s'"},
	}
	for _, tt := range tests {
		got := escapeValue(tt.val, plugin.TypePostgres)
		if got != tt.want {
			t.Errorf("escapeValue(%v) = %q; want %q", tt.val, got, tt.want)
		}
	}
}

// ─── SQL generation tests ──────────────────────────────────────────────────

func TestGenerateCreateSQL(t *testing.T) {
	e := &engine{}
	tables := mockSchema().Tables

	sql := e.generateCreateSQL(tables[0], plugin.TypePostgres)
	if sql == "" {
		t.Fatal("expected non-empty CREATE SQL")
	}
	if !strings.Contains(sql, "CREATE TABLE IF NOT EXISTS") {
		t.Error("missing CREATE TABLE IF NOT EXISTS")
	}
	if !strings.Contains(sql, "PRIMARY KEY") {
		t.Error("missing PRIMARY KEY")
	}

	sqliteSQL := e.generateCreateSQL(tables[0], plugin.TypeSQLite)
	if strings.Contains(sqliteSQL, "DEFAULT") {
		t.Error("SQLite CREATE should not contain DEFAULT clauses")
	}
}

func TestGenerateInsertSQL(t *testing.T) {
	e := &engine{}
	tableInfo := mockSchema().Tables[0]

	rows := []Row{
		{"id": int64(1), "name": "Alice", "email": "alice@test.com"},
		{"id": int64(2), "name": "Bob", "email": nil},
	}

	sql := e.generateInsertSQL(tableInfo, rows, plugin.TypeMySQL)
	if sql == "" {
		t.Fatal("expected non-empty INSERT SQL")
	}
	if !strings.Contains(sql, "INSERT INTO") {
		t.Error("missing INSERT INTO")
	}
	if !strings.Contains(sql, "NULL") {
		t.Error("expected NULL for nil value")
	}

	pgSQL := e.generateInsertSQL(tableInfo, rows, plugin.TypePostgres)
	if !strings.Contains(pgSQL, "ON CONFLICT DO NOTHING") {
		t.Error("PG INSERT should have ON CONFLICT DO NOTHING")
	}

	empty := e.generateInsertSQL(tableInfo, nil, plugin.TypeMySQL)
	if empty != "" {
		t.Error("expected empty string for nil rows")
	}
}

func TestTruncateSQL(t *testing.T) {
	e := &engine{}
	sql := e.truncateSQL("users", plugin.TypePostgres)
	if sql != "TRUNCATE TABLE \"users\"" {
		t.Errorf("unexpected truncate SQL: %s", sql)
	}

	sqliteSQL := e.truncateSQL("users", plugin.TypeSQLite)
	if sqliteSQL != "DELETE FROM \"users\"" {
		t.Errorf("unexpected SQLite truncate: %s", sqliteSQL)
	}
}

// ─── Engine lifecycle tests ────────────────────────────────────────────────

func TestNewEngine(t *testing.T) {
	mgr := &mockManager{
		schemaFn: func(ctx context.Context, connID string) (*plugin.Schema, error) {
			return mockSchema(), nil
		},
		queryFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return mockRows(connID), nil
		},
		executeFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return &plugin.Result{RowsAffected: 1}, nil
		},
	}
	eng := NewEngine(mgr)
	if eng == nil {
		t.Fatal("NewEngine returned nil")
	}
}

func TestDryRunTransfer(t *testing.T) {
	mgr := &mockManager{
		schemaFn: func(ctx context.Context, connID string) (*plugin.Schema, error) {
			return mockSchema(), nil
		},
		queryFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return mockRows(connID), nil
		},
		executeFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return &plugin.Result{RowsAffected: 1}, nil
		},
	}
	eng := NewEngine(mgr)
	job := TransferJob{
		SourceType: "postgres",
		TargetType: "mysql",
		SourceConn: "users",
		TargetConn: "target",
		DryRun:     true,
		BatchSize:  100,
	}

	ctx := context.Background()
	if err := eng.Start(ctx, &job); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	status, err := waitForStatus(eng, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "done" {
		t.Errorf("expected done, got %s (job SourceType=%s TargetType=%s)",
			status.Status, job.SourceType, job.TargetType)
	}
}

func TestCancelTransfer(t *testing.T) {
	mgr := &mockManager{
		schemaFn: func(ctx context.Context, connID string) (*plugin.Schema, error) {
			return mockSchema(), nil
		},
		queryFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return mockRows(connID), nil
		},
		executeFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return &plugin.Result{RowsAffected: 1}, nil
		},
	}
	eng := NewEngine(mgr)
	ctx := context.Background()
	job := TransferJob{
		SourceType: "postgres",
		TargetType: "mysql",
		SourceConn: "users",
		TargetConn: "target",
	}
	if err := eng.Start(ctx, &job); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := eng.Cancel(job.ID); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	status, _ := eng.Status(job.ID)
	if status.Status != "cancelled" {
		t.Errorf("expected cancelled, got %s", status.Status)
	}
}

func TestTransferStatusNotFound(t *testing.T) {
	eng := NewEngine(&mockManager{})
	_, err := eng.Status("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown job")
	}
}

func TestTransferCancelNotFound(t *testing.T) {
	eng := NewEngine(&mockManager{})
	if err := eng.Cancel("nonexistent"); err == nil {
		t.Fatal("expected error for unknown job")
	}
}

func TestListEmpty(t *testing.T) {
	eng := NewEngine(&mockManager{})
	jobs, err := eng.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected empty list, got %d", len(jobs))
	}
}

func TestBatchSizeDefault(t *testing.T) {
	mgr := &mockManager{
		schemaFn: func(ctx context.Context, connID string) (*plugin.Schema, error) {
			return mockSchema(), nil
		},
		queryFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) { return nil, nil },
		executeFn: func(ctx context.Context, connID string, query string) (*plugin.Result, error) {
			return &plugin.Result{}, nil
		},
	}
	eng := NewEngine(mgr)
	job := TransferJob{
		SourceType: "sqlite",
		TargetType: "postgres",
		SourceConn: "src",
		TargetConn: "tgt",
		DryRun:     true,
	}
	ctx := context.Background()
	if err := eng.Start(ctx, &job); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	status, err := eng.Status(job.ID)
	if err != nil {
		t.Fatalf("Status error: %v", err)
	}
	if status.Status != "done" {
		// Poll one more time with longer wait
		time.Sleep(200 * time.Millisecond)
		status, _ = eng.Status(job.ID)
		if status.Status == "done" {
			return
		}
		t.Errorf("expected done, got %s", status.Status)
	}
}

func TestSchemaFailPropagates(t *testing.T) {
	mgr := &mockManager{
		schemaFn: func(ctx context.Context, connID string) (*plugin.Schema, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	eng := NewEngine(mgr)
	job := TransferJob{
		SourceType: "postgres",
		TargetType: "mysql",
		SourceConn: "broken",
		TargetConn: "target",
	}
	ctx := context.Background()
	if err := eng.Start(ctx, &job); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	status, err := waitForStatus(eng, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "failed" {
		t.Errorf("expected failed, got %s", status.Status)
	}
}

// ─── Integration: real SQLite → SQLite transfer ────────────────────────────

func TestIntegrationSQLiteTransfer(t *testing.T) {
	mgr := connection.NewManager()

	src, err := mgr.Add(context.Background(), "source", plugin.TypeSQLite, "test", plugin.Config{FilePath: ":memory:", Database: "src"}, nil)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	defer mgr.Remove(src.ID)

	tgt, err := mgr.Add(context.Background(), "target", plugin.TypeSQLite, "test", plugin.Config{FilePath: ":memory:", Database: "tgt"}, nil)
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	defer mgr.Remove(tgt.ID)

	seedSQL := []string{
		`CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, name TEXT NOT NULL, price REAL)`,
		`INSERT INTO items VALUES (1, 'Widget', 9.99)`,
		`INSERT INTO items VALUES (2, 'Gadget', 24.99)`,
		`INSERT INTO items VALUES (3, 'Doohickey', 4.99)`,
	}
	for _, s := range seedSQL {
		if _, err := mgr.Execute(context.Background(), src.ID, s); err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	eng := NewEngine(mgr)
	job := TransferJob{
		SourceType: "sqlite",
		TargetType: "sqlite",
		SourceConn: src.ID,
		TargetConn: tgt.ID,
		Tables:     []string{"items"},
		BatchSize:  10,
		OnConflict: "overwrite",
	}
	ctx := context.Background()
	if err := eng.Start(ctx, &job); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	status, err := waitForStatus(eng, job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "done" {
		t.Fatalf("expected done, got %s", status.Status)
	}

	tgtResult, err := mgr.Query(context.Background(), tgt.ID, "SELECT COUNT(*) as cnt FROM items")
	if err != nil {
		t.Fatalf("target query failed: %v", err)
	}
	if len(tgtResult.Rows) == 0 || len(tgtResult.Rows[0]) == 0 {
		t.Fatal("no result from target")
	}
	count, ok := tgtResult.Rows[0][0].(int64)
	if !ok || count != 3 {
		t.Fatalf("expected 3 rows, got %v", tgtResult.Rows[0][0])
	}
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func waitForStatus(eng TransferEngine, jobID string) (*TransferJob, error) {
	// Total wait up to 8 seconds: goroutines may be delayed when many tests run
	for i := 0; i < 80; i++ {
		status, err := eng.Status(jobID)
		if err != nil {
			return nil, err
		}
		if status.Status == "done" || status.Status == "failed" || status.Status == "cancelled" {
			return status, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	status, _ := eng.Status(jobID)
	return status, fmt.Errorf("timeout waiting for job %s, last status: %s", jobID, status.Status)
}
