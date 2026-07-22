package agent

import (
	"context"
	"testing"

	"go-database/internal/ai"
	"go-database/internal/connection"
	"go-database/internal/plugin"
)

// fakeGate records the last SQL passed to Query and satisfies the Gate iface.
type fakeGate struct {
	lastSQL string
}

func (f *fakeGate) List() []connection.Summary { return nil }
func (f *fakeGate) Query(ctx context.Context, id, sql string) (*plugin.Result, error) {
	f.lastSQL = sql
	return &plugin.Result{Columns: []string{"content", "distance"}, Rows: [][]any{{"relevant row", 0.1}}}, nil
}
func (f *fakeGate) Execute(ctx context.Context, id, sql string) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}
func (f *fakeGate) Tables(ctx context.Context, id string) ([]string, error) { return nil, nil }
func (f *fakeGate) Schema(ctx context.Context, id string) (*plugin.Schema, error) {
	return nil, nil
}
func (f *fakeGate) Databases(ctx context.Context, id string) ([]string, error) { return nil, nil }

func TestAgentVectorSearchTool(t *testing.T) {
	g := &fakeGate{}
	a := &Agent{gate: g, emb: &ai.HashEmbedder{}}
	res, err := a.vectorSearch(context.Background(), map[string]any{
		"connection_id":    "c1",
		"table":            "docs",
		"text_column":      "body",
		"embedding_column": "emb",
		"query":            "login problems",
		"k":                float64(3),
	})
	if err != nil {
		t.Fatalf("vectorSearch: %v", err)
	}
	if !contains(g.lastSQL, "<=> '") {
		t.Fatalf("expected pgvector operator in SQL, got %q", g.lastSQL)
	}
	if !contains(g.lastSQL, "LIMIT 3") {
		t.Fatalf("expected LIMIT 3, got %q", g.lastSQL)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
