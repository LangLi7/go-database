package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"go-database/internal/connection"
	"go-database/internal/llm"
	"go-database/internal/plugin"
)

// DBGate is the minimal surface the MCP tools need.
type DBGate interface {
	List() []connection.Summary
	GetConnection(id string) (*connection.Connection, error)
	Query(ctx context.Context, id string, query string) (*plugin.Result, error)
	Execute(ctx context.Context, id string, query string) (*plugin.Result, error)
	Tables(ctx context.Context, id string) ([]string, error)
	Schema(ctx context.Context, id string) (*plugin.Schema, error)
	Databases(ctx context.Context, id string) ([]string, error)
}

// Server wraps the MCP server for go-database.
type Server struct {
	mcpServer *mcp.Server
}

// NewServer creates and registers all go-database MCP tools.
func NewServer(log *slog.Logger) *Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "go-database", Version: "0.1.0"}, nil)
	RegisterDBTools(s, log)
	RegisterNL2SQLTool(s, log)
	return &Server{mcpServer: s}
}

// Connect starts the server on the given transport.
func (s *Server) Connect(ctx context.Context, transport mcp.Transport, opts *mcp.ServerSessionOptions) (*mcp.ServerSession, error) {
	return s.mcpServer.Connect(ctx, transport, opts)
}

// --- DB tools ---

var connectionManager DBGate

func SetDBGate(g DBGate) { connectionManager = g }

func RegisterDBTools(s *mcp.Server, log *slog.Logger) {
	gate := connectionManager

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_connections",
		Description: "List all configured database connections.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		conns := gate.List()
		out := make([]map[string]any, 0, len(conns))
		for _, c := range conns {
			row := map[string]any{
				"id":    c.ID,
				"name":  c.Name,
				"type":  string(c.Type),
				"state": string(c.State),
			}
			if c.Tags != nil {
				row["tags"] = c.Tags
			}
			out = append(out, row)
		}
		return textResult(out), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "query",
		Description: "Run a read-only SELECT query on a connection.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
				"sql":           map[string]any{"type": "string"},
			},
			"required": []string{"connection_id", "sql"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ConnectionID string `json:"connection_id"`
		SQL          string `json:"sql"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := gate.Query(ctx, args.ConnectionID, args.SQL)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(map[string]any{
			"columns": res.Columns,
			"rows":    res.Rows,
			"count":   len(res.Rows),
		}), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "execute",
		Description: "Run a write query (INSERT/UPDATE/DELETE/DDL) on a connection.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
				"sql":           map[string]any{"type": "string"},
			},
			"required": []string{"connection_id", "sql"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ConnectionID string `json:"connection_id"`
		SQL          string `json:"sql"`
	}) (*mcp.CallToolResult, any, error) {
		res, err := gate.Execute(ctx, args.ConnectionID, args.SQL)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(map[string]any{
			"rows_affected": res.RowsAffected,
			"duration_ms":   res.Duration,
		}), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tables",
		Description: "List tables in a connection/database.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
			},
			"required": []string{"connection_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{ ConnectionID string }) (*mcp.CallToolResult, any, error) {
		tables, err := gate.Tables(ctx, args.ConnectionID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(tables), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "schema",
		Description: "Show schema metadata for a connection.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
			},
			"required": []string{"connection_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{ ConnectionID string }) (*mcp.CallToolResult, any, error) {
		schema, err := gate.Schema(ctx, args.ConnectionID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(schema.Tables), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_databases",
		Description: "List available databases on a connection.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string"},
			},
			"required": []string{"connection_id"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{ ConnectionID string }) (*mcp.CallToolResult, any, error) {
		dbs, err := gate.Databases(ctx, args.ConnectionID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return textResult(dbs), nil, nil
	})
}

// --- NL2SQL — uses llm.Client (OpenRouter / LM Studio / Ollama) ---

var nl2sqlClient llm.Client

// SetNL2SQLConfig wires API key, provider & model into the nl2sql tool.
func SetNL2SQLConfig(provider, model, apiKey, lmstudioURL string, allowPaid bool) {
	nl2sqlClient = llm.NewClient(provider, apiKey, model, lmstudioURL, allowPaid)
}

func RegisterNL2SQLTool(s *mcp.Server, log *slog.Logger) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "nl2sql",
		Description: "Convert natural language to SQL via OpenRouter, LM Studio, or Ollama.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"connection_id": map[string]any{"type": "string", "description": "target connection for dialect-aware SQL"},
				"question":      map[string]any{"type": "string"},
				"schema_hint":   map[string]any{"type": "string"},
			},
			"required": []string{"connection_id", "question"},
		},
	}, func(ctx context.Context, req *mcp.CallToolRequest, args struct {
		ConnectionID string `json:"connection_id"`
		Question     string `json:"question"`
		SchemaHint   string `json:"schema_hint"`
	}) (*mcp.CallToolResult, any, error) {
		if nl2sqlClient == nil {
			return errorResult(errors.New("NL2SQL: not configured; set mcp.api_key / llm.provider in config")), nil, nil
		}
		prompt := llm.BuildPrompt(args.Question, args.SchemaHint)
		sql, err := nl2sqlClient.Complete(ctx, prompt)
		if err != nil {
			return errorResult(fmt.Errorf("LLM call failed: %w", err)), nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sql}},
		}, nil, nil
	})
}

// --- helpers ---

func textResult(v any) *mcp.CallToolResult {
	b, _ := json.Marshal(v)
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "error: " + err.Error()}},
		IsError: true,
	}
}
