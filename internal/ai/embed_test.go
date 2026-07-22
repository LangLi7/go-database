package ai

import (
	"context"
	"strings"
	"testing"
)

func TestHashEmbedderDeterministic(t *testing.T) {
	e := &HashEmbedder{Dims: 64}
	v1, err := e.Embed(context.Background(), "login failed for user")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	v2, err := e.Embed(context.Background(), "login failed for user")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(v1) != 64 || len(v2) != 64 {
		t.Fatalf("expected dim 64, got %d/%d", len(v1), len(v2))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("hash embedder not deterministic at dim %d", i)
		}
	}
	// different text → different vector
	v3, _ := e.Embed(context.Background(), "database backup completed")
	same := true
	for i := range v1 {
		if v1[i] != v3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("different texts produced identical vectors")
	}
}

func TestPgVectorLiteral(t *testing.T) {
	got := PgVectorLiteral([]float32{0.1, 0.2, 0.3})
	want := "[0.1,0.2,0.3]"
	if got != want {
		t.Fatalf("PgVectorLiteral = %q, want %q", got, want)
	}
	if strings.Contains(got, " ") {
		t.Fatalf("pgvector literal must not contain spaces: %q", got)
	}
}
