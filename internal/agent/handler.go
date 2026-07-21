package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go-database/internal/connection"
	"go-database/internal/llm"
	"go-database/internal/plugin"
)

// ToolDef describes an MCP tool for the LLM to choose from.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"input_schema"` // JSON string
}

var availableTools = []ToolDef{
	{Name: "list_connections", Description: "List all database connections", InputSchema: `{}`},
	{Name: "query", Description: "Run a SELECT query on a connection", InputSchema: `{"connection_id":"string","sql":"string"}`},
	{Name: "execute", Description: "Run INSERT/UPDATE/DELETE/DDL", InputSchema: `{"connection_id":"string","sql":"string"}`},
	{Name: "list_tables", Description: "List tables in a connection", InputSchema: `{"connection_id":"string"}`},
	{Name: "schema", Description: "Show schema for a connection", InputSchema: `{"connection_id":"string"}`},
	{Name: "list_databases", Description: "List databases on a connection", InputSchema: `{"connection_id":"string"}`},
	{Name: "nl2sql", Description: "Convert natural language to SQL", InputSchema: `{"connection_id":"string","question":"string","schema_hint?":"string"}`},
}

// Agent routes NL input → LLM decides tool → executes → returns result.
type Agent struct {
	llm    llm.Client
	gate   Gate
	logger *slog.Logger
	logFn  func(action, details string) // optional audit log

	mu       sync.RWMutex
	sessions map[string][]chatTurn // session_id → history
}

type chatTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const sessionTTL = 30 * time.Minute
const maxHistory = 20

// Gate is the minimal surface the agent needs (matches connection.Manager).
type Gate interface {
	List() []connection.Summary
	Query(ctx context.Context, id string, sql string) (*plugin.Result, error)
	Execute(ctx context.Context, id string, sql string) (*plugin.Result, error)
	Tables(ctx context.Context, id string) ([]string, error)
	Schema(ctx context.Context, id string) (*plugin.Schema, error)
	Databases(ctx context.Context, id string) ([]string, error)
}

var agent *Agent

// InitAgent sets up the global agent singleton.
func InitAgent(llmClient llm.Client, dbGate Gate, auditLogFn func(action, details string)) {
	agent = &Agent{
		llm:      llmClient,
		gate:     dbGate,
		logger:   slog.Default(),
		logFn:    auditLogFn,
		sessions: make(map[string][]chatTurn),
	}
	go agent.cleanupLoop()
}

// ChatRequest is the incoming user message.
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"` // optional, for multi-turn
}

// ChatResponse is the structured result sent back.
type ChatResponse struct {
	Tool      string `json:"tool"`
	Args      any    `json:"args"`
	Result    any    `json:"result"`
	Summary   string `json:"summary"`
	SessionID string `json:"session_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// HandleChat processes a natural-language request and returns the tool result.
func HandleChat(ctx context.Context, msg, sessionID string) (*ChatResponse, error) {
	if agent == nil {
		return nil, fmt.Errorf("agent not initialized")
	}

	// Session management
	sid := sessionID
	if sid == "" {
		sid = fmt.Sprintf("sess-%d", time.Now().UnixNano())
	}

	agent.mu.Lock()
	history := agent.sessions[sid]
	if len(history) >= maxHistory {
		history = history[len(history)-maxHistory/2:]
	}
	history = append(history, chatTurn{Role: "user", Content: msg})
	agent.sessions[sid] = history
	agent.mu.Unlock()

	toolCall, err := agent.decideTool(ctx, msg)
	if err != nil {
		if agent.logFn != nil {
			agent.logFn("agent_error", fmt.Sprintf("session=%s msg=%q err=%v", sid, truncate(msg, 100), err))
		}
		return nil, fmt.Errorf("decision: %w", err)
	}

	result, err := agent.executeTool(ctx, toolCall)
	if err != nil {
		if agent.logFn != nil {
			agent.logFn("agent_error", fmt.Sprintf("session=%s tool=%s err=%v", sid, toolCall.Name, err))
		}
		return nil, fmt.Errorf("execute: %w", err)
	}

	summary := fmt.Sprintf("%s → %s", toolCall.Name, truncate(fmt.Sprintf("%v", result), 200))

	agent.mu.Lock()
	agent.sessions[sid] = append(agent.sessions[sid], chatTurn{Role: "assistant", Content: summary})
	agent.mu.Unlock()

	if agent.logFn != nil {
		agent.logFn("agent_chat", fmt.Sprintf("session=%s msg=%q tool=%s summary=%s", sid, truncate(msg, 100), toolCall.Name, summary))
	}

	agent.logger.Info("agent done", "session", sid, "tool", toolCall.Name, "summary", summary)

	return &ChatResponse{
		Tool:      toolCall.Name,
		Args:      toolCall.Args,
		Result:    result,
		Summary:   summary,
		SessionID: sid,
	}, nil
}

func (a *Agent) cleanupLoop() {
	for {
		time.Sleep(sessionTTL)
		a.mu.Lock()
		for k := range a.sessions {
			delete(a.sessions, k) // ponytail: delete all, no per-session TTL
		}
		a.sessions = make(map[string][]chatTurn)
		a.mu.Unlock()
	}
}

// toolCall is the internal representation of the LLM's decision.
type toolCall struct {
	Name string         `json:"tool"`
	Args map[string]any `json:"args"`
}

func (a *Agent) decideTool(ctx context.Context, msg string) (*toolCall, error) {
	prompt := a.buildPrompt(msg)
	resp, err := a.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	resp = cleanJSON(resp)
	var tc toolCall
	if err := json.Unmarshal([]byte(resp), &tc); err != nil {
		// LLM didn't return JSON — wrap the text as nl2sql
		return &toolCall{Name: "nl2sql", Args: map[string]any{"question": msg, "connection_id": ""}}, nil
	}
	if tc.Name == "" {
		return nil, fmt.Errorf("LLM returned no tool name: %s", resp)
	}
	return &tc, nil
}

func (a *Agent) executeTool(ctx context.Context, tc *toolCall) (any, error) {
	switch tc.Name {
	case "list_connections":
		return a.gate.List(), nil
	case "query":
		cid, _ := tc.Args["connection_id"].(string)
		sql, _ := tc.Args["sql"].(string)
		return a.gate.Query(ctx, cid, sql)
	case "execute":
		cid, _ := tc.Args["connection_id"].(string)
		sql, _ := tc.Args["sql"].(string)
		return a.gate.Execute(ctx, cid, sql)
	case "list_tables":
		cid, _ := tc.Args["connection_id"].(string)
		return a.gate.Tables(ctx, cid)
	case "schema":
		cid, _ := tc.Args["connection_id"].(string)
		return a.gate.Schema(ctx, cid)
	case "list_databases":
		cid, _ := tc.Args["connection_id"].(string)
		return a.gate.Databases(ctx, cid)
	default:
		return nil, fmt.Errorf("unknown tool: %s", tc.Name)
	}
}

func (a *Agent) buildPrompt(msg string) string {
	var b strings.Builder
	b.WriteString("You are an AI database assistant. You have these tools:\n\n")
	for _, t := range availableTools {
		fmt.Fprintf(&b, "- %s: %s (args: %s)\n", t.Name, t.Description, t.InputSchema)
	}
	b.WriteString("\nGiven the user request, respond with EXACTLY a JSON object:\n")
	b.WriteString(`{"tool":"tool_name","args":{"arg1":"val1",...}}` + "\n")
	b.WriteString("Only one tool call. No explanation. Valid JSON only.\n\n")
	b.WriteString("User: " + msg)
	return b.String()
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Ensure all imports are used
