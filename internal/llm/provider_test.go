package llm

import (
	"os"
	"strings"
	"testing"
)

func TestFindLlamaCPP(t *testing.T) {
	path := FindLlamaCPP()
	if path == "" {
		t.Skip("llama-server not installed on this host")
	}
	if !strings.Contains(path, "llama-server") {
		t.Fatalf("unexpected path: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("llama-server binary not readable: %v", err)
	}
	t.Logf("llama-server found: %s", path)
}

func TestLlamaCppNewClient(t *testing.T) {
	// model="test" signals NewClient to skip the eager llama-server auto-start
	// (see client.go: model != "test" guard), so this stays a fast unit test.
	c := NewClient("llamacpp", "", "test", "", false)
	if c.Name() != "lmstudio" {
		t.Fatalf("expected lmstudio (reused), got %s", c.Name())
	}
}

func TestProviderList(t *testing.T) {
	providers := []struct {
		provider string
		expects  string
	}{
		{"openrouter", "openrouter"},
		{"lmstudio", "lmstudio"},
		{"ollama", "ollama"},
		{"llamacpp", "lmstudio"}, // reuses LMStudioClient
	}
	for _, p := range providers {
		c := NewClient(p.provider, "key", "", "", false)
		if c.Name() != p.expects {
			t.Errorf("provider %s: expected Name()=%q, got %q", p.provider, p.expects, c.Name())
		}
	}
}
