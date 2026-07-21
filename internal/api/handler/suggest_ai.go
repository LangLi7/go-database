package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
)

type aiSuggestRequest struct {
	ConnectionID string `json:"connection_id"`
	Input        string `json:"input"`
	FullQuery    string `json:"full_query"`
	Schema       string `json:"schema"`
}

type openRouterRequest struct {
	Model     string          `json:"model"`
	Messages  []openRouterMsg `json:"messages"`
	Stream    bool            `json:"stream"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openRouterMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

var openRouterModels = []string{
	"google/gemma-4-31b-it:free",
	"openrouter/free",
	"qwen/qwen3-32b:free",
	"nvidia/llama-3.1-nemotron-ultra-253b-v1:free",
}

func AISuggest(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req aiSuggestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "input required")
			return
		}

		if req.Input == "" && req.FullQuery == "" {
			response.BadRequest(c, "input or full_query required")
			return
		}

		apiKey := c.GetHeader("X-AI-Key")
		if apiKey == "" {
			apiKey = c.Query("api_key")
		}
		if apiKey == "" {
			response.BadRequest(c, "AI API key required (X-AI-Key header or ?api_key=)")
			return
		}

		ctx := req.FullQuery
		if ctx == "" {
			ctx = req.Input
		}

		// Build schema context
		schemaCtx := req.Schema
		if schemaCtx == "" && req.ConnectionID != "" {
			schema, err := mgr.Schema(c.Request.Context(), req.ConnectionID)
			if err == nil && schema != nil {
				var sb strings.Builder
				for _, t := range schema.Tables {
					sb.WriteString(fmt.Sprintf("Table %s:\n", t.Name))
					for _, col := range t.Columns {
						sb.WriteString(fmt.Sprintf("  - %s (%s)%s%s\n",
							col.Name, col.Type,
							iface(col.Primary, " PK", ""),
							iface(!col.Nullable, " NOT NULL", "")))
					}
				}
				schemaCtx = sb.String()
			}
		}

		prompt := fmt.Sprintf(`You are a SQL expert assistant. Given the database schema and the current SQL query context, suggest completions.

Database Schema:
%s

Current SQL:
%s

Respond with a JSON array of suggestions. Each suggestion has:
- "text": the completion text
- "type": "keyword" | "table" | "column" | "function" | "clause"
- "desc": short description

Example: [{"text": "SELECT", "type": "keyword", "desc": "Select rows"}, {"text": "COUNT(*)", "type": "function", "desc": "Count rows"}]
Return ONLY the JSON array, no other text. Max 5 suggestions.`, schemaCtx, ctx)

		suggestions := callOpenRouter(apiKey, prompt, 3)
		if suggestions == nil {
			response.Success(c, []gin.H{
				{"text": "SELECT", "type": "keyword", "desc": "Select rows", "confidence": 0.5},
				{"text": "FROM", "type": "keyword", "desc": "Specify table", "confidence": 0.5},
				{"text": "WHERE", "type": "keyword", "desc": "Filter results", "confidence": 0.5},
			})
			return
		}
		response.Success(c, suggestions)
	}
}

func callOpenRouter(apiKey, prompt string, maxRetries int) []gin.H {
	for attempt := 0; attempt < maxRetries; attempt++ {
		var lastErr string
		for _, model := range openRouterModels {
			result, err := tryModel(apiKey, model, prompt)
			if err != nil {
				lastErr = err.Error()
				slog.Warn("ai suggest: model failed", "model", model, "error", err)
				continue
			}
			if len(result) > 0 {
				return result
			}
		}
		if lastErr != "" && attempt < maxRetries-1 {
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

func tryModel(apiKey, model, prompt string) ([]gin.H, error) {
	body := openRouterRequest{
		Model: model,
		Messages: []openRouterMsg{
			{Role: "system", Content: "You are a SQL expert. Return ONLY valid JSON arrays."},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 300,
	}

	data, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/go-database")
	req.Header.Set("X-Title", "go-database")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var orResp openRouterResponse
	if err := json.Unmarshal(raw, &orResp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if orResp.Error != nil {
		return nil, fmt.Errorf("api: %s", orResp.Error.Message)
	}

	if len(orResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	content := orResp.Choices[0].Message.Content
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var suggestions []gin.H
	if err := json.Unmarshal([]byte(content), &suggestions); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return suggestions, nil
}

func iface(cond bool, t, f string) string {
	if cond {
		return t
	}
	return f
}
