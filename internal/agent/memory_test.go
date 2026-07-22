package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryStorePersists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mem.json")
	m := NewMemoryStore(p)

	m.Remember("fact", "user prefers sqlite", "s1")
	m.Remember("correction", "that was wrong, use postgres", "s1")

	// reload from disk (simulates restart)
	m2 := NewMemoryStore(p)
	entries := m2.load()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Type != "fact" || entries[1].Type != "correction" {
		t.Fatalf("unexpected types: %+v", entries)
	}
	ctx := m2.ContextPrompt(10)
	if ctx == "" {
		t.Fatal("ContextPrompt returned empty")
	}
	if !memContains(ctx, "user prefers sqlite") || !memContains(ctx, "correction") {
		t.Fatalf("ContextPrompt missing entries: %q", ctx)
	}

	_ = os.Remove(p)
}

func memContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
