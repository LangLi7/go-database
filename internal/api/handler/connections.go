package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/guard"
	"go-database/internal/plugin"
	"go-database/internal/suggest"
)

func ListConnections(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Multi-tenant isolation: non-admins see only their own connections
		// (OwnerID == userID) plus any explicitly shared via db_access.
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")
		isAdmin := role == "admin"
		dbAccess := append(dbAccessFrom(c, "db_access"), dbAccessFrom(c, "extra_db_access")...)
		uid, _ := userID.(string)
		response.Success(c, mgr.ListVisible(uid, dbAccess, isAdmin))
	}
}

// dbAccessFrom reads a []string db-access list from the gin context (best-effort).
func dbAccessFrom(c *gin.Context, key string) []string {
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	if s, ok := v.([]string); ok {
		return s
	}
	return nil
}

func GetConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		mc, err := mgr.Get(c.Param("id"))
		if err != nil {
			response.NotFound(c, "connection not found")
			return
		}
		response.Success(c, mc.Connection)
	}
}

func ListDatabases(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		databases, err := mgr.Databases(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadGateway, "QUERY_FAILED", err.Error())
			return
		}
		response.Success(c, databases)
	}
}

type createDBRequest struct {
	Name string `json:"name" binding:"required"`
}

func CreateDatabase(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createDBRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "name required")
			return
		}
		if err := mgr.CreateDatabase(c.Request.Context(), c.Param("id"), req.Name); err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		response.Created(c, gin.H{"database": req.Name})
	}
}

func DropDatabase(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := mgr.DropDatabase(c.Request.Context(), c.Param("id"), c.Param("name")); err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type createTableRequest struct {
	Name    string `json:"name" binding:"required"`
	Columns string `json:"columns" binding:"required"`
}

func CreateTable(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createTableRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "name and columns required")
			return
		}
		_, err := mgr.Execute(c.Request.Context(), c.Param("id"), req.Columns)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		response.Created(c, gin.H{"table": req.Name})
	}
}

func DropTable(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := mgr.Execute(c.Request.Context(), c.Param("id"), "DROP TABLE IF EXISTS "+quoteTable(c.Param("name")))
		if err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type createConnectionRequest struct {
	Name   string            `json:"name" binding:"required"`
	Type   plugin.DBType     `json:"type" binding:"required"`
	Source string            `json:"source"`
	Host   string            `json:"host"`
	Port   int               `json:"port"`
	DBName string            `json:"database"`
	User   string            `json:"user"`
	Pass   string            `json:"password"`
	File   string            `json:"filepath"`
	SSL    bool              `json:"ssl"`
	Params map[string]string `json:"params"`
	Tags   []string          `json:"tags"`
}

func CreateConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createConnectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request: name and type required")
			return
		}

		if req.Source == "" {
			req.Source = "external"
		}

		cfg := plugin.Config{
			Type:     req.Type,
			Host:     req.Host,
			Port:     req.Port,
			Database: req.DBName,
			User:     req.User,
			Password: req.Pass,
			FilePath: req.File,
			SSL:      req.SSL,
			Params:   req.Params,
		}

		// Auto-detect the database type if requested
		dbType := req.Type
		if dbType == plugin.TypeAuto {
			detected, ok := plugin.DetectType(cfg)
			if !ok {
				response.BadRequest(c, "could not auto-detect database type; specify 'type' explicitly")
				return
			}
			dbType = detected
			cfg.Type = detected
		}

		conn, err := mgr.Add(c.Request.Context(), req.Name, dbType, req.Source, cfg, req.Tags, userIDFrom(c))
		if err != nil {
			response.Error(c, http.StatusBadRequest, "CONNECTION_FAILED", err.Error())
			return
		}

		response.Created(c, conn)
	}
}

type testConnectionRequest struct {
	Name   string            `json:"name"`
	Type   string            `json:"type" binding:"required"`
	Source string            `json:"source"`
	Host   string            `json:"host"`
	Port   int               `json:"port"`
	DBName string            `json:"db_name"`
	User   string            `json:"user"`
	Pass   string            `json:"password"`
	File   string            `json:"filepath"`
	SSL    bool              `json:"ssl"`
	Tags   []string          `json:"tags"`
	Params map[string]string `json:"params"`
}

func TestConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req testConnectionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "type required")
			return
		}

		cfg := plugin.Config{
			Type:     plugin.DBType(req.Type),
			Host:     req.Host,
			Port:     req.Port,
			Database: req.DBName,
			User:     req.User,
			Password: req.Pass,
			FilePath: req.File,
			SSL:      req.SSL,
			Params:   req.Params,
		}

		// Auto-detect the database type if requested
		if plugin.DBType(req.Type) == plugin.TypeAuto {
			detected, ok := plugin.DetectType(cfg)
			if !ok {
				response.Error(c, http.StatusBadRequest, "DETECT_FAILED",
					"could not auto-detect database type; specify 'type' explicitly")
				return
			}
			cfg.Type = detected
			req.Type = string(detected)
		}

		name := req.Name
		if name == "" {
			b := make([]byte, 8)
			rand.Read(b)
			name = "test-" + hex.EncodeToString(b)
		}

		p, ok := plugin.New(plugin.DBType(req.Type))
		if !ok {
			response.Error(c, http.StatusBadRequest, "UNSUPPORTED_TYPE", "unsupported database type")
			return
		}

		connCtx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		start := time.Now()
		if err := p.Connect(connCtx, cfg); err != nil {
			p.Close()
			response.Error(c, http.StatusBadGateway, "CONNECTION_FAILED", err.Error())
			return
		}
		latency := time.Since(start)
		p.Close()

		response.Success(c, gin.H{
			"success":    true,
			"latency_ms": latency.Milliseconds(),
			"message":    "connection successful",
		})
	}
}

func DeleteConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := mgr.Remove(c.Param("id")); err != nil {
			response.NotFound(c, "connection not found")
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func PingConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		latency, err := mgr.Ping(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadGateway, "PING_FAILED", err.Error())
			return
		}
		response.Success(c, gin.H{
			"latency_ms": latency.Milliseconds(),
			"status":     "ok",
		})
	}
}

func ListTables(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tables, err := mgr.Tables(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadGateway, "QUERY_FAILED", err.Error())
			return
		}
		response.Success(c, tables)
	}
}

func GetSchema(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		schema, err := mgr.Schema(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.Error(c, http.StatusBadGateway, "SCHEMA_FAILED", err.Error())
			return
		}
		response.Success(c, schema)
	}
}

type queryRequest struct {
	Query string `json:"query" binding:"required"`
}

var sqlGuard = guard.New()

func QueryConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req queryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query required")
			return
		}

		// Guard: ensure SQL is a SELECT
		perm, _ := c.Get("effective_perm")
		if perm == nil {
			perm, _ = c.Get("extra_perm")
		}
		permSlice, _ := perm.([]string)
		if cmd, ok := sqlGuard.CheckCommand(req.Query, permSlice); !ok {
			if cmd == suggest.CmdSelect {
				response.Forbidden(c, "insufficient permissions for SELECT")
			} else {
				response.Forbidden(c, "only SELECT queries allowed on query endpoint")
			}
			return
		}

		result, err := mgr.Query(c.Request.Context(), c.Param("id"), req.Query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "QUERY_FAILED", err.Error())
			return
		}
		response.Success(c, result)
	}
}

func ExecuteConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req queryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query required")
			return
		}

		// Guard: check SQL command is allowed
		perm, _ := c.Get("effective_perm")
		if perm == nil {
			perm, _ = c.Get("extra_perm")
		}
		permSlice, _ := perm.([]string)
		if cmd, ok := sqlGuard.CheckCommand(req.Query, permSlice); !ok {
			if cmd == suggest.CmdUnknown {
				// Non-SQL command — allow if user has execute permission
				if !auth.HasResourcePermission(permSlice, "execute", "*") {
					response.Forbidden(c, "insufficient permissions")
					return
				}
			} else {
				response.Forbidden(c, "insufficient permissions for this SQL operation")
				return
			}
		}

		result, err := mgr.Execute(c.Request.Context(), c.Param("id"), req.Query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		response.Success(c, result)
	}
}

// userIDFrom extracts the authenticated user ID from the gin context.
func userIDFrom(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
