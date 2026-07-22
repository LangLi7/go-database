// Package ai provides embedding + RAG helpers for the AI-database engine.
// Embeddings turn text into vectors so go-database can do semantic (vector)
// search over Postgres+pgvector tables, then feed retrieved context to the
// LLM (RAG). The Embedder interface keeps the model backend swappable.
package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Embedder turns text into a fixed-size vector.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// OllamaEmbedder calls a local Ollama server (/api/embed). No external
// dependency, runs the nomic-embed-text model locally (free).
type OllamaEmbedder struct {
	BaseURL string // e.g. http://localhost:11434
	Model   string // e.g. nomic-embed-text
	HTTP    *http.Client
}

func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": e.Model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.BaseURL, "/")+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	hc := e.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama embed: %s: %s", resp.Status, b)
	}
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed: empty response")
	}
	return out.Embeddings[0], nil
}

// OpenAIEmbedder calls the OpenAI-compatible /v1/embeddings endpoint. Use for
// OpenRouter/OpenAI-hosted embedding models.
type OpenAIEmbedder struct {
	BaseURL string // e.g. https://api.openai.com/v1
	Model   string
	APIKey  string
	HTTP    *http.Client
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": e.Model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.BaseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)
	hc := e.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embed: %s: %s", resp.Status, b)
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("openai embed: empty response")
	}
	return out.Data[0].Embedding, nil
}

// HashEmbedder is a deterministic, dependency-free embedder. It is NOT
// semantic — it produces a stable bag-of-words hash vector so the pipeline
// (vector_search / rag) works offline and in tests without a model server.
// ponytail: replace with OllamaEmbedder/OpenAIEmbedder for real semantics.
type HashEmbedder struct {
	Dims int
}

func (e *HashEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	dims := e.Dims
	if dims == 0 {
		dims = 256
	}
	v := make([]float32, dims)
	tokens := strings.Fields(strings.ToLower(text))
	for _, tok := range tokens {
		h := sha256.Sum256([]byte(tok))
		idx := int(h[0]) % dims
		v[idx] += 1.0
	}
	return v, nil
}

// PgVectorLiteral formats a float vector as a pgvector literal: [0.1,0.2,...]
func PgVectorLiteral(v []float32) string {
	var b strings.Builder
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%g", x)
	}
	b.WriteByte(']')
	return b.String()
}
