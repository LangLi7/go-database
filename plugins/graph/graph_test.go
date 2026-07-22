package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go-database/internal/plugin"
)

func newTestGraph(t *testing.T) (*GraphPlugin, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.graph.json")
	g := &GraphPlugin{}
	if err := g.Connect(context.Background(), plugin.Config{FilePath: path}); err != nil {
		t.Fatalf("connect: %v", err)
	}
	cleanup := func() { _ = g.Close(); _ = os.Remove(path) }
	return g, cleanup
}

func TestGraphCRUDAndTraverse(t *testing.T) {
	g, cleanup := newTestGraph(t)
	defer cleanup()
	ctx := context.Background()

	// CREATE NODEs
	if _, err := g.Execute(ctx, `CREATE NODE Person {"name":"Alice"}`); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if _, err := g.Execute(ctx, `CREATE NODE Person {"name":"Bob"}`); err != nil {
		t.Fatalf("create bob: %v", err)
	}
	if _, err := g.Execute(ctx, `CREATE NODE Company {"name":"Acme"}`); err != nil {
		t.Fatalf("create acme: %v", err)
	}
	// Alice -> Acme (works_at), Bob -> Acme (works_at)
	if _, err := g.Execute(ctx, `CREATE EDGE n_1 n_3 works_at {}`); err != nil {
		t.Fatalf("edge alice: %v", err)
	}
	if _, err := g.Execute(ctx, `CREATE EDGE n_2 n_3 works_at {}`); err != nil {
		t.Fatalf("edge bob: %v", err)
	}

	// MATCH Person WHERE name=Alice
	m, err := g.Query(ctx, "MATCH Person WHERE name=Alice")
	if err != nil {
		t.Fatalf("match: %v", err)
	}
	if len(m.Rows) != 1 {
		t.Fatalf("expected 1 alice, got %d", len(m.Rows))
	}

	// NEIGHBORS of Acme (n_3) — should be Alice + Bob
	n, err := g.Query(ctx, "NEIGHBORS n_3 works_at")
	if err != nil {
		t.Fatalf("neighbors: %v", err)
	}
	if len(n.Rows) != 2 {
		t.Fatalf("expected 2 neighbours of acme, got %d", len(n.Rows))
	}

	// TRAVERSE from Alice (n_1) depth 1 — should reach Acme
	tr, err := g.Query(ctx, "TRAVERSE n_1 1")
	if err != nil {
		t.Fatalf("traverse: %v", err)
	}
	if len(tr.Rows) != 1 {
		t.Fatalf("expected 1 node at distance 1, got %d", len(tr.Rows))
	}

	// Persistence: reopen and verify data survived
	if err := g.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	g2 := &GraphPlugin{}
	if err := g2.Connect(ctx, plugin.Config{FilePath: g.path}); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	m2, err := g2.Query(ctx, "MATCH Person")
	if err != nil {
		t.Fatalf("reopen match: %v", err)
	}
	if len(m2.Rows) != 2 {
		t.Fatalf("expected 2 persisted persons, got %d", len(m2.Rows))
	}
}
