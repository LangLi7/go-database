package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/middleware"
	"go-database/internal/api/router"
	"go-database/internal/dashboard"
	"go-database/internal/auth"
	"go-database/internal/config"
	_ "go-database/plugins/postgres"
	_ "go-database/plugins/mysql"
	_ "go-database/plugins/mariadb"
	_ "go-database/plugins/sqlite"
	_ "go-database/plugins/mongodb"
	_ "go-database/plugins/redis"
	"go-database/internal/connection"
	"go-database/internal/internaldb"
	"go-database/internal/provisioner"
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
	store, err := internaldb.Open(ctx, cfg.InternalDB.AuthPath)
	if err != nil {
		slog.Error("failed to open internal database", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	slog.Info("internal database ready", "path", cfg.InternalDB.AuthPath)

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
	r.Use(requestLogger())

	// ---- Embedded Dashboard (SPA) ----
	if dfs, err := dashboard.FS(); err == nil {
		fsrv := http.FileServer(dfs)
		r.GET("/assets/*filepath", gin.WrapH(fsrv))
		r.GET("/favicon.svg", gin.WrapH(fsrv))
		r.GET("/favicon.ico", gin.WrapH(fsrv))
		r.NoRoute(func(c *gin.Context) {
			if c.Request.Method != "GET" {
				c.Next()
				return
			}
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.Next()
				return
			}
			f, err := dfs.Open("index.html")
			if err != nil {
				c.Next()
				return
			}
			defer f.Close()
			stat, err := f.Stat()
			if err != nil {
				slog.Warn("failed to stat index.html", "error", err)
				c.Next()
				return
			}
			http.ServeContent(c.Writer, c.Request, "index.html", stat.ModTime(), f)
		})
		slog.Info("dashboard embedded and serving")
	} else {
		slog.Info("dashboard not embedded, API-only mode (use: cd web && npm run dev)")
	}

	// ---- Routes ----
	router.SetupRoutes(r, store, connMgr, jwtSvc, apikeySvc)
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
