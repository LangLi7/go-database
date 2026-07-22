package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/agent"
	"go-database/internal/api/response"
	"go-database/internal/llm"
	mcp "go-database/internal/mcp"
)

// HandleAgentChat processes a natural-language request via the AI agent.
func HandleAgentChat(logFn func(action, details string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req agent.ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "message required")
			return
		}
		if req.Message == "" {
			response.BadRequest(c, "message is empty")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
		defer cancel()

		slog.Info("agent chat", "message", req.Message[:min(len(req.Message), 100)], "session", req.SessionID)
		dbAccess, isAdmin := dbScopeFromContext(c)
		resp, err := agent.HandleChat(ctx, req.Message, req.SessionID, dbAccess, isAdmin)
		if err != nil {
			if logFn != nil {
				logFn("agent_error", fmt.Sprintf("msg=%q err=%v", req.Message[:min(len(req.Message), 100)], err))
			}
			response.Error(c, 500, "agent_error", err.Error())
			return
		}
		response.Success(c, resp)
	}
}

// HandleAgentStream SSE-streams the agent's response with real token streaming.
func HandleAgentStream(logFn func(action, details string)) gin.HandlerFunc {
	return func(c *gin.Context) {
		msg := c.Query("message")
		sessionID := c.Query("session_id")
		if msg == "" {
			response.BadRequest(c, "?message= required")
			return
		}

		// Check if the LLM client supports real streaming
		if streamer, ok := getLLMStreamer(); ok {
			streamAgentResponse(c, streamer, msg, sessionID, logFn)
			return
		}

		// Fallback: complete then flush
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
		defer cancel()

		c.SSEvent("status", "processing")
		c.Writer.Flush()

		scope, admin := dbScopeFromContext(c)
		resp, err := agent.HandleChat(ctx, msg, sessionID, scope, admin)
		if err != nil {
			c.SSEvent("error", err.Error())
			return
		}
		c.SSEvent("result", resp)
		c.Writer.Flush()
	}
}

func streamAgentResponse(c *gin.Context, streamer llm.Streamer, msg, sessionID string, logFn func(action, details string)) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()

	// Call the agent (which calls LLM once to decide tool)
	scope, admin := dbScopeFromContext(c)
	resp, err := agent.HandleChat(ctx, msg, sessionID, scope, admin)
	if err != nil {
		c.SSEvent("error", err.Error())
		return
	}

	// Then stream the explanation/tool result via LLM
	prompt := fmt.Sprintf("Summarize the following database operation in one sentence: tool=%s result=%s", resp.Tool, resp.Summary)
	ch, err := streamer.Stream(ctx, prompt)
	if err != nil {
		// Fallback: send full result
		c.SSEvent("result", resp)
		c.Writer.Flush()
		return
	}

	c.SSEvent("tool", map[string]any{"tool": resp.Tool, "args": resp.Args})
	for token := range ch {
		c.SSEvent("token", token)
		c.Writer.Flush()
	}
	c.SSEvent("result", resp)
	c.Writer.Flush()
	if logFn != nil {
		logFn("agent_stream", fmt.Sprintf("session=%s tool=%s", sessionID, resp.Tool))
	}
}

func getLLMStreamer() (llm.Streamer, bool) {
	s, ok := interface{}(agentLLM).(llm.Streamer)
	return s, ok
}

// agentLLM is set by InitAgent in cmd/server.
var agentLLM llm.Client

// SetAgentLLM wires the LLM client for stream detection.
func SetAgentLLM(cl llm.Client) { agentLLM = cl }

// HandleAISetup validates + saves AI config to .env.
func HandleAISetup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Provider     string `json:"provider"`
			APIKey       string `json:"api_key"`
			Model        string `json:"model"`
			FallbackPaid bool   `json:"fallback_paid"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request body")
			return
		}

		status := map[string]any{
			"provider":      req.Provider,
			"model":         req.Model,
			"api_key_set":   req.APIKey != "",
			"fallback_paid": req.FallbackPaid,
		}

		switch req.Provider {
		case "openrouter", "lmstudio", "ollama", "":
			status["valid"] = true
		default:
			status["valid"] = false
			status["error"] = "invalid provider; use openrouter, lmstudio, or ollama"
			response.Success(c, status)
			return
		}
		if req.Provider == "openrouter" && req.APIKey == "" {
			status["valid"] = false
			status["warning"] = "api_key required for openrouter"
			response.Success(c, status)
			return
		}

		// Write to .env (loaded at startup, gitignored)
		if err := writeEnvFile(req.Provider, req.APIKey, req.Model, req.FallbackPaid); err != nil {
			status["env_written"] = false
			status["env_error"] = err.Error()
		} else {
			status["env_written"] = true
			status["note"] = "restart server to apply changes"
		}

		response.Success(c, status)
	}
}

func writeEnvFile(provider, apiKey, model string, fallbackPaid bool) error {
	envPath := ".env"
	existing := map[string]string{}
	if data, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				existing[parts[0]] = parts[1]
			}
		}
	}

	if apiKey != "" {
		existing["GODB_MCP_API_KEY"] = apiKey
	}
	if provider != "" {
		existing["GODB_MCP_PROVIDER"] = provider
	}
	if model != "" {
		existing["GODB_MCP_MODEL"] = model
	}
	existing["GODB_MCP_FALLBACK_PAID"] = fmt.Sprintf("%t", fallbackPaid)
	existing["GODB_MCP_ENABLED"] = "true"

	var b strings.Builder
	b.WriteString("# AI/LLM configuration (written by POST /api/v1/setup/ai)\n")
	for k, v := range existing {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}

	dir := filepath.Dir(envPath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(envPath, []byte(b.String()), 0644)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// dbScopeFromContext extracts the per-caller DB-access scope and admin flag
// set by AuthMiddleware. JWT callers get extra_db_access; API-key callers get
// db_access. isAdmin is true only for the built-in admin role. This is what
// feeds Agent/MCP per-user DB isolation.
func dbScopeFromContext(c *gin.Context) (dbAccess []string, isAdmin bool) {
	if role, ok := c.Get("role"); ok {
		if r, ok := role.(string); ok && r == "admin" {
			isAdmin = true
		}
	}
	if v, ok := c.Get("db_access"); ok {
		if s, ok := v.([]string); ok {
			dbAccess = s
		}
	}
	if v, ok := c.Get("extra_db_access"); ok {
		if s, ok := v.([]string); ok {
			dbAccess = append(dbAccess, s...)
		}
	}
	return dbAccess, isAdmin
}

// HandleMCP wraps the MCP HTTP handler with per-caller DB-access scoping.
// Auth is enforced by the surrounding AuthMiddleware; here we only bind the
// caller's db_access scope so MCP tools can only touch allowed connections.
func HandleMCP(mcpHandler http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbAccess, isAdmin := dbScopeFromContext(c)
		prev := mcp.SetScopedGate(mcp.ScopedGate(dbAccess, isAdmin))
		defer mcp.SetScopedGate(prev)
		mcpHandler.ServeHTTP(c.Writer, c.Request)
	}
}
