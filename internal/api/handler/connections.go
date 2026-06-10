package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/plugin"
)

func ListConnections(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, mgr.List())
	}
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

		conn, err := mgr.Add(c.Request.Context(), req.Name, req.Type, req.Source, cfg, req.Tags)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "CONNECTION_FAILED", err.Error())
			return
		}

		response.Created(c, conn)
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

func QueryConnection(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req queryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "query required")
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
		result, err := mgr.Execute(c.Request.Context(), c.Param("id"), req.Query)
		if err != nil {
			response.Error(c, http.StatusBadGateway, "EXEC_FAILED", err.Error())
			return
		}
		response.Success(c, result)
	}
}
