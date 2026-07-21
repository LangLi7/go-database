package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	gmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"go-database/internal/config"
	"go-database/internal/connection"
	"go-database/internal/internaldb"
	"go-database/internal/mcp"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)})))

	store, err := internaldb.Open(ctx, cfg.InternalDB.AuthPath)
	if err != nil {
		slog.Error("failed to open internal database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	connMgr := connection.NewManager()
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()
	connMgr.StartHealthChecker(connCtx, 30*time.Second)
	slog.Info("connection manager ready")

	mcpSrv := mcp.NewServer(slog.Default())
	mcp.SetDBGate(connMgr)
	mcp.SetNL2SQLConfig(cfg.MCP.Provider, cfg.MCP.Model, cfg.MCP.APIKey, "", cfg.MCP.FallbackPaid)
	if err := mcp.ValidateMCPConfig(cfg.MCP.Provider, cfg.MCP.Model, cfg.MCP.APIKey); err != nil {
		slog.Warn("invalid mcp config", "error", err)
	}
	slog.Info("MCP server initialized", "tools", []string{
		"list_connections", "query", "execute", "list_tables", "schema", "list_databases", "nl2sql",
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("shutting down...")
		connCancel()
		_ = store.Close()
		os.Exit(0)
	}()

	slog.Info("go-database MCP server starting (stdio JSON-RPC)")
	if _, err := mcpSrv.Connect(ctx, &gmcp.StdioTransport{}, nil); err != nil {
		slog.Error("mcp connect failed", "error", err)
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
