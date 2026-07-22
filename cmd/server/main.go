package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/agent"
	"go-database/internal/api/handler"
	"go-database/internal/api/middleware"
	"go-database/internal/api/router"
	"go-database/internal/auth"
	"go-database/internal/config"
	"go-database/internal/connection"
	"go-database/internal/crypto"
	"go-database/internal/internaldb"
	"go-database/internal/llm"
	mcp "go-database/internal/mcp"
	"go-database/internal/executor"
	"go-database/internal/provisioner"
	"go-database/internal/scheduler"
	"go-database/internal/transfer"
	_ "go-database/plugins/mariadb"
	_ "go-database/plugins/mongodb"
	_ "go-database/plugins/mssql"
	_ "go-database/plugins/mysql"
	_ "go-database/plugins/postgres"
	_ "go-database/plugins/graph"
	_ "go-database/plugins/redis"
	_ "go-database/plugins/sqlite"
)

var startTime = time.Now()

func main() {
	// ---- Config ----
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ---- Logger ----
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))

	slog.Info("starting go-database", "config", cfg.PrintJSON())

	// ---- Internal Database ----
	ctx := context.Background()
	authDSN := cfg.InternalDB.AuthURL
	if authDSN == "" {
		authDSN = cfg.InternalDB.AuthPath
	}
	store, err := internaldb.Open(ctx, authDSN)
	if err != nil {
		slog.Error("failed to open internal database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	if authDSN != cfg.InternalDB.AuthPath {
		slog.Info("internal database ready", "type", "postgresql")
	} else {
		slog.Info("internal database ready", "path", authDSN)
	}

	// ---- Connection Manager ----
	connMgr := connection.NewManager()
	connCtx, connCancel := context.WithCancel(ctx)
	defer connCancel()
	connMgr.StartHealthChecker(connCtx, 30*time.Second)
	slog.Info("connection manager ready")

	// ---- Auto-Provisioning (docker / embedded) ----
	prov := provisioner.New(ctx, connMgr)
	slog.Info("provisioner ready", "connections", prov.ProvisionedIDs())

	// ---- Auth Services ----
	jwtSvc, err := auth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.TokenDuration)
	if err != nil {
		slog.Error("failed to initialize JWT service", "error", err)
		os.Exit(1)
	}
	apikeySvc := auth.NewAPIKeyService(store)
	slog.Info("token service ready (AES-256-GCM, key stored in ~/.config/go-database/secret.key)")

	// ---- Gin Engine ----
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.Server.CORSOrigin))
	r.Use(middleware.SecurityHeaders())
	r.Use(requestLogger())

	// ---- API-only mode ----
	// The frontend is developed separately and connects via the REST/WS API.
	// No dashboard is embedded (see DECISIONS.md ADR-005).
	slog.Info("API-only mode: frontend is a separate client, no embedded dashboard")

	// ---- Transfer Engine & Scheduler ----
	transferEngine := transfer.NewEngine(connMgr)
	schedStore, _ := scheduler.NewFileStore("scheduled_jobs.json")
	sched := scheduler.New(transferEngine, schedStore)
	sched.Start(ctx)

	// ---- Encryption Service ----
	cryptoStore, err := crypto.NewKeyStore("encryption_keys.json", jwtSvc.MasterKey())
	if err != nil {
		slog.Error("crypto keystore failed", "error", err)
		os.Exit(1)
	}
	cryptoSvc := crypto.NewService(cryptoStore)
	slog.Info("encryption service ready", "algorithms", "aes-256-gcm, aes-256-cbc+hmac, chacha20-poly1305, rsa-oaep-4096, x25519-hybrid")

	// ---- Guard wraps connMgr so Agent + MCP pass through the risk guard ----
	guard := executor.NewGuardGate(connMgr)

	// ---- MCP Server (config-gesteuert) ----
	if cfg.MCP.Enabled {
		if err := mcp.ValidateMCPConfig(cfg.MCP.Provider, cfg.MCP.Model, cfg.MCP.APIKey); err != nil {
			slog.Error("invalid mcp config", "error", err)
		} else {
			mcp.SetDBGate(guard)
			mcp.SetNL2SQLConfig(cfg.MCP.Provider, cfg.MCP.Model, cfg.MCP.APIKey, "", cfg.MCP.FallbackPaid)
			mcpHandler := mcp.HTTPHandler(mcp.APIKeyMiddleware(cfg.MCP.APIKey))
			r.Any(cfg.MCP.Endpoint, gin.WrapH(mcpHandler))
			slog.Info("mcp http endpoint ready", "path", cfg.MCP.Endpoint, "provider", cfg.MCP.Provider)
		}
	}

	// ---- AI Agent (uses same LLM client as MCP) ----
	var llamaSrv *llm.LlamaCppServer
	llamaURL := ""
	if cfg.MCP.Provider == "llamacpp" {
		llamaURL = fmt.Sprintf("http://localhost:%d", cfg.MCP.LlamaCpp.Port)
		if cfg.MCP.LlamaCpp.AutoStart {
			llamaSrv = llm.NewLlamaCppServer(llm.LlamaCppConfig{
				ModelPath:  cfg.MCP.Model,
				Port:       cfg.MCP.LlamaCpp.Port,
				NGPULayers: -1,
				Parallel:   cfg.MCP.LlamaCpp.Parallel,
			})
			if err := llamaSrv.Start(ctx); err != nil {
				slog.Warn("llamacpp auto-start failed, continuing without LLM", "error", err)
			} else {
				slog.Info("llamacpp server started", "port", cfg.MCP.LlamaCpp.Port)
				llamaURL = fmt.Sprintf("http://localhost:%d", cfg.MCP.LlamaCpp.Port)
			}
		}
	}
	llmClient := llm.NewClient(cfg.MCP.Provider, cfg.MCP.APIKey, cfg.MCP.Model, llamaURL, cfg.MCP.FallbackPaid)
	auditFn := func(action, details string) { _ = store.LogAudit(ctx, "system", action, details) }
	agent.InitAgent(llmClient, guard, auditFn, nil)
	handler.SetAgentLLM(llmClient)
	slog.Info("ai agent ready", "provider", cfg.MCP.Provider)

	// ---- Documentation ----
	handler.InitDocs()

	// ---- Routes ----
	router.SetupRoutes(r, store, connMgr, jwtSvc, apikeySvc, transferEngine, sched, schedStore, cryptoSvc)
	slog.Info("routes registered")

	// ---- Server ----
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ---- Graceful Shutdown ----
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info(fmt.Sprintf("listening on %s", cfg.Server.Addr()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	prov.Shutdown(shutdownCtx)

	if llamaSrv != nil {
		_ = llamaSrv.Stop()
		slog.Info("llamacpp server stopped")
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("forced shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
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

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client", c.ClientIP(),
		)
	}
}
