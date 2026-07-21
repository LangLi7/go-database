package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/plugin"
	"go-database/internal/suggest"
)

// dbTypeQueryRequest is the body for per-type endpoints (/api/v1/db/{type}/query).
type dbTypeQueryRequest struct {
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	User     string            `json:"user"`
	Password string            `json:"password"`
	FilePath string            `json:"filepath"`
	SSL      bool              `json:"ssl"`
	Params   map[string]string `json:"params"`
	Query    string            `json:"query" binding:"required"`
}

// DBTypeQuery handles POST /api/v1/db/{type}/query
// It creates a throwaway connection of the given type, runs a SELECT, and closes it.
func DBTypeQuery(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbType := plugin.DBType(c.Param("type"))
		if !plugin.IsSupported(dbType) {
			response.Error(c, http.StatusBadRequest, "UNSUPPORTED_TYPE",
				"unsupported database type: "+string(dbType))
			return
		}

		var req dbTypeQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query required")
			return
		}

		// Guard: ensure SQL is a SELECT
		permSlice := effectivePerms(c)
		if cmd, ok := sqlGuard.CheckCommand(req.Query, permSlice); !ok {
			if cmd == suggest.CmdSelect {
				response.Forbidden(c, "insufficient permissions for SELECT")
			} else {
				response.Forbidden(c, "only SELECT queries allowed on query endpoint")
			}
			return
		}

		cfg := plugin.Config{
			Type:     dbType,
			Host:     req.Host,
			Port:     req.Port,
			Database: req.Database,
			User:     req.User,
			Password: req.Password,
			FilePath: req.FilePath,
			SSL:      req.SSL,
			Params:   req.Params,
		}

		result, err := runThrowaway(c.Request.Context(), cfg, func(p plugin.DBPlugin) (*plugin.Result, error) {
			return p.Query(c.Request.Context(), req.Query)
		})
		if err != nil {
			response.Error(c, http.StatusBadGateway, "QUERY_FAILED", err.Error())
			return
		}
		response.Success(c, result)
	}
}

// DBTypeExecute handles POST /api/v1/db/{type}/execute (writes).
func DBTypeExecute(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbType := plugin.DBType(c.Param("type"))
		if !plugin.IsSupported(dbType) {
			response.Error(c, http.StatusBadRequest, "UNSUPPORTED_TYPE",
				"unsupported database type: "+string(dbType))
			return
		}

		var req dbTypeQueryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query required")
			return
		}

		permSlice := effectivePerms(c)
		if cmd, ok := sqlGuard.CheckCommand(req.Query, permSlice); !ok {
			if cmd == suggest.CmdUnknown {
				if !auth.HasResourcePermission(permSlice, "execute", "*") {
					response.Forbidden(c, "insufficient permissions")
					return
				}
			} else {
				response.Forbidden(c, "insufficient permissions for this SQL operation")
				return
			}
		}

		cfg := plugin.Config{
			Type:     dbType,
			Host:     req.Host,
			Port:     req.Port,
			Database: req.Database,
			User:     req.User,
			Password: req.Password,
			FilePath: req.FilePath,
			SSL:      req.SSL,
			Params:   req.Params,
		}

		result, err := runThrowaway(c.Request.Context(), cfg, func(p plugin.DBPlugin) (*plugin.Result, error) {
			return p.Execute(c.Request.Context(), req.Query)
		})
		if err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		response.Success(c, result)
	}
}

// DBTypeTest handles POST /api/v1/db/{type}/test — connectivity check.
func DBTypeTest(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbType := plugin.DBType(c.Param("type"))
		if !plugin.IsSupported(dbType) {
			response.Error(c, http.StatusBadRequest, "UNSUPPORTED_TYPE",
				"unsupported database type: "+string(dbType))
			return
		}

		var req dbTypeQueryRequest
		_ = c.ShouldBindJSON(&req) // optional body; query not required

		cfg := plugin.Config{
			Type:     dbType,
			Host:     req.Host,
			Port:     req.Port,
			Database: req.Database,
			User:     req.User,
			Password: req.Password,
			FilePath: req.FilePath,
			SSL:      req.SSL,
			Params:   req.Params,
		}

		start := time.Now()
		if err := runThrowawayConnect(c.Request.Context(), cfg); err != nil {
			response.Error(c, http.StatusBadGateway, "CONNECTION_FAILED", err.Error())
			return
		}
		response.Success(c, gin.H{
			"success":    true,
			"latency_ms": time.Since(start).Milliseconds(),
			"message":    "connection successful",
		})
	}
}

// runThrowaway connects to a DB, runs fn, then closes — never persisted.
func runThrowaway(ctx context.Context, cfg plugin.Config, fn func(plugin.DBPlugin) (*plugin.Result, error)) (*plugin.Result, error) {
	p, ok := plugin.New(cfg.Type)
	if !ok {
		return nil, context.Canceled
	}
	connCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := p.Connect(connCtx, cfg); err != nil {
		p.Close()
		return nil, err
	}
	defer p.Close()
	return fn(p)
}

// runThrowawayConnect only tests connectivity.
func runThrowawayConnect(ctx context.Context, cfg plugin.Config) error {
	p, ok := plugin.New(cfg.Type)
	if !ok {
		return context.Canceled
	}
	connCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := p.Connect(connCtx, cfg); err != nil {
		p.Close()
		return err
	}
	p.Close()
	return nil
}

// effectivePerms extracts the permission slice stored by auth middleware.
func effectivePerms(c *gin.Context) []string {
	if v, ok := c.Get("effective_perm"); ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	if v, ok := c.Get("extra_perm"); ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}
