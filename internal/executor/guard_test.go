package executor

import (
	"context"
	"testing"

	"go-database/internal/connection"
	"go-database/internal/plugin"
)

// fakeMgr implements the minimal surface GuardGate needs for testing the guard
// bypass: it records whether Execute was called.
type fakeMgr struct {
	execCalled bool
}

func (f *fakeMgr) List() []connection.Summary { return nil }
func (f *fakeMgr) GetConnection(id string) (*connection.Connection, error) {
	return nil, nil
}
func (f *fakeMgr) Query(ctx context.Context, id, sql string) (*plugin.Result, error) {
	return &plugin.Result{Rows: [][]any{{"ok"}}}, nil
}
func (f *fakeMgr) Execute(ctx context.Context, id, sql string) (*plugin.Result, error) {
	f.execCalled = true
	return &plugin.Result{RowsAffected: 0}, nil
}
func (f *fakeMgr) Tables(ctx context.Context, id string) ([]string, error) { return nil, nil }
func (f *fakeMgr) Schema(ctx context.Context, id string) (*plugin.Schema, error) { return nil, nil }
func (f *fakeMgr) Databases(ctx context.Context, id string) ([]string, error) { return nil, nil }
func (f *fakeMgr) ListVisible(userID string, dbAccess []string, isAdmin bool) []connection.Summary {
	return nil
}

func TestGuardGateBlocksHighRiskDelete(t *testing.T) {
	fm := &fakeMgr{}
	g := NewGuardGate(fm)
	// DELETE without WHERE is high-risk → must be blocked (no exec call).
	_, err := g.Execute(context.Background(), "conn1", "DELETE FROM users")
	if err == nil {
		t.Fatalf("expected guard to block unconfirmed high-risk DELETE, got nil")
	}
	if fm.execCalled {
		t.Fatalf("underlying Execute must NOT be called for blocked high-risk op")
	}
}

func TestGuardGateAllowsSelect(t *testing.T) {
	fm := &fakeMgr{}
	g := NewGuardGate(fm).WithScope([]string{"conn1"}, false)
	res, err := g.Query(context.Background(), "conn1", "SELECT * FROM users")
	if err != nil {
		t.Fatalf("SELECT should pass through guard: %v", err)
	}
	if res == nil {
		t.Fatalf("expected result")
	}
}
