package llm

import (
	"context"
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	p := BuildPrompt("show users", "users (id, name, email)")
	if !strings.Contains(p, "show users") {
		t.Fatalf("missing question: %q", p)
	}
	if !strings.Contains(p, "SQL expert") {
		t.Fatalf("missing instruction: %q", p)
	}
}

func TestOpenRouterFallbackOrder(t *testing.T) {
	c := NewOpenRouter("test-key", "free", true) // with paid fallback
	models := c.resolveModels()
	if len(models) < 2 {
		t.Fatalf("expected at least free models + fallback, got %d", len(models))
	}
	// last model is always the paid fallback
	last := models[len(models)-1]
	if last != FallbackModel {
		t.Fatalf("expected fallback %q, got %q", FallbackModel, last)
	}
	// without paid fallback
	c2 := NewOpenRouter("test-key", "free", false)
	models2 := c2.resolveModels()
	for _, m := range models2 {
		if m == FallbackModel {
			t.Fatalf("paid fallback included when allowPaid=false")
		}
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("openrouter", "key", "free", "", false)
	if c.Name() != "openrouter" {
		t.Fatalf("expected openrouter, got %s", c.Name())
	}
	c2 := NewClient("ollama", "", "llama3", "", false)
	if c2.Name() != "ollama" {
		t.Fatalf("expected ollama, got %s", c2.Name())
	}
	c3 := NewClient("lmstudio", "", "deepseek", "http://localhost:1234", false)
	if c3.Name() != "lmstudio" {
		t.Fatalf("expected lmstudio, got %s", c3.Name())
	}
}

func TestRemoteModelIsFree(t *testing.T) {
	m := RemoteModel{ID: "test/free", Pricing: ModelPricing{Prompt: "0", Completion: "0"}}
	if !m.IsFree() {
		t.Fatal("expected free")
	}
	m2 := RemoteModel{ID: "test/paid", Pricing: ModelPricing{Prompt: "0.01", Completion: "0.01"}}
	if m2.IsFree() {
		t.Fatal("expected not free")
	}
}

func TestCompleteUnconfigured(t *testing.T) {
	c := NewOpenRouter("", "nonexistent-model", false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate cancel → no real HTTP call
	_, err := c.Complete(ctx, "test")
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}
