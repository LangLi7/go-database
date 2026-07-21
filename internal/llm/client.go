package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// OpenRouter FREE models (verified 2026-07-21).
// String replacement in suggest_ai.go, not a stale list.
var FreeModels = []string{
	"google/gemma-4-31b-it:free",
	"google/gemma-4-26b-a4b-it:free",
	"nvidia/nemotron-3-nano-30b-a3b:free",
	"nvidia/nemotron-nano-9b-v2:free",
	"openai/gpt-oss-20b:free",
	"poolside/laguna-m.1:free",
	"cohere/north-mini-code:free",
	"openrouter/free",
}

// FallbackModel is the cheapest reasonable paid model if all free models fail.
var FallbackModel = "deepseek/deepseek-r1"

// Client is the LLM provider interface.
type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
	Name() string
}

// Streamer is an optional interface for clients that support token streaming.
type Streamer interface {
	Stream(ctx context.Context, prompt string) (<-chan string, error)
}

// OpenRouterClient calls the OpenRouter API with optional free→paid fallback.
type OpenRouterClient struct {
	apiKey    string
	model     string // configured model; if "free" it tries FreeModels first
	allowPaid bool   // if true, tries FallbackModel after free models fail
	client    *http.Client
}

func NewOpenRouter(apiKey, model string, allowPaid bool) *OpenRouterClient {
	if model == "" {
		model = "free"
	}
	return &OpenRouterClient{
		apiKey:    apiKey,
		model:     model,
		allowPaid: allowPaid,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenRouterClient) Name() string { return "openrouter" }

func (c *OpenRouterClient) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{RoleUser: prompt},
		},
		"temperature": 0.0,
		"stream":      true,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderContentType, "application/json")
	req.Header.Set(HeaderAuthorization, "Bearer "+c.apiKey)
	req.Header.Set(HeaderReferer, "https://github.com/go-database")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 64)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					return
				}
				var chunk struct {
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					continue
				}
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					select {
					case ch <- chunk.Choices[0].Delta.Content:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func (c *OpenRouterClient) Complete(ctx context.Context, prompt string) (string, error) {
	models := c.resolveModels()
	for i, m := range models {
		result, err := c.tryModel(ctx, m, prompt)
		if err == nil {
			return result, nil
		}
		slog.Warn("openrouter model failed", "model", m, "error", err, "attempt", i+1, "total", len(models))
	}
	return "", fmt.Errorf("all %d openrouter models failed", len(models))
}

func (c *OpenRouterClient) resolveModels() []string {
	if c.model != "free" {
		if c.allowPaid {
			return []string{c.model, FallbackModel}
		}
		return []string{c.model}
	}
	if c.allowPaid {
		return append(FreeModels, FallbackModel)
	}
	return FreeModels
}

func (c *OpenRouterClient) tryModel(ctx context.Context, model, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{RoleUser: prompt},
		},
		"temperature": 0.0,
		"max_tokens":  1024,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", OpenRouterEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set(HeaderContentType, "application/json")
	req.Header.Set(HeaderAuthorization, "Bearer "+c.apiKey)
	req.Header.Set(HeaderReferer, "https://github.com/go-database")
	req.Header.Set("X-Title", "go-database")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("api: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// LMStudioClient calls the local LM Studio server.
type LMStudioClient struct {
	baseURL string
	model   string // optional; if empty LM Studio picks the loaded model
	client  *http.Client
}

func NewLMStudio(baseURL, model string) *LMStudioClient {
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}
	return &LMStudioClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *LMStudioClient) Name() string { return "lmstudio" }

func (c *LMStudioClient) Complete(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":      c.model,
		"messages":   []map[string]string{{RoleUser: prompt}},
		"stream":     false,
		"max_tokens": 2048,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+LMEndpointChat, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set(HeaderContentType, "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// FetchOpenRouterModels queries the OpenRouter API for available models (cached).
func FetchOpenRouterModels(ctx context.Context, apiKey string) ([]RemoteModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", OpenRouterModelsEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(HeaderAuthorization, "Bearer "+apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []RemoteModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Models, nil
}

// FetchLMStudioModels queries the local LM Studio for available models.
func FetchLMStudioModels(ctx context.Context, baseURL string) ([]LocalModel, error) {
	if baseURL == "" {
		baseURL = "http://localhost:1234"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+LMEndpointModels, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []LocalModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// try different JSON shape (LM Studio v1 response)
		var raw struct {
			Models []LocalModel `json:"models"`
		}
		if err2 := json.NewDecoder(resp.Body).Decode(&raw); err2 != nil {
			return nil, err
		}
		result.Models = raw.Models
	}
	return result.Models, nil
}

// BuildPrompt creates a standard SQL NL→SQL prompt.
func BuildPrompt(question, schemaHint string) string {
	var b strings.Builder
	b.WriteString("You are a SQL expert. Convert natural language to SQL.\n")
	b.WriteString("Return ONLY the raw SQL query, no markdown, no explanation.\n")
	if schemaHint != "" {
		b.WriteString("Schema context:\n")
		b.WriteString(schemaHint)
		b.WriteString("\n")
	}
	b.WriteString("Question: ")
	b.WriteString(question)
	return b.String()
}

// OllamaClient calls a local Ollama server.
type OllamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}
	return &OllamaClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *OllamaClient) Name() string { return "ollama" }

func (c *OllamaClient) Complete(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":  c.model,
		"prompt": prompt,
		"stream": false,
	})
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+OllamaEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set(HeaderContentType, "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Response), nil
}

// NewClient creates the correct client based on provider string.
func NewClient(provider, apiKey, model, lmstudioURL string, allowPaid bool) Client {
	switch provider {
	case "ollama":
		return NewOllama("", model)
	case "lmstudio":
		return NewLMStudio(lmstudioURL, model)
	default:
		return NewOpenRouter(apiKey, model, allowPaid)
	}
}
