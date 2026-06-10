package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/executor"
	"go-database/internal/guard"
	"go-database/internal/plugin"
	"go-database/internal/suggest"
)

type suggestRequest struct {
	ConnectionID string `json:"connection_id"`
	Input        string `json:"input"`
	CurrentTable string `json:"current_table"`
}

func GetSuggestions(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req suggestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "input required")
			return
		}

		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")
		perms, _ := c.Get("extra_perm")

		permList, _ := perms.([]string)
		roleStr, _ := role.(string)
		userIDStr, _ := userID.(string)

		var schema *plugin.Schema
		if req.ConnectionID != "" {
			s, err := mgr.Schema(c.Request.Context(), req.ConnectionID)
			if err == nil {
				schema = s
			}
		}

		ctx := suggest.Context{
			UserID:       userIDStr,
			Role:         roleStr,
			ConnectionID: req.ConnectionID,
			CurrentTable: req.CurrentTable,
			Input:        req.Input,
			Schema:       schema,
		}

		engine := suggest.NewEngine()
		suggestions := engine.GetSuggestions(ctx, 10)

		g := guard.New()
		suggestions = g.FilterSuggestions(suggestions, permList)

		response.Success(c, suggestions)
	}
}

type executeSafeRequest struct {
	ConnectionID string `json:"connection_id" binding:"required"`
	SQL          string `json:"sql" binding:"required"`
	ConfirmHigh  bool   `json:"confirm_high"`
}

func ExecuteSafe(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req executeSafeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "connection_id and sql required")
			return
		}

		userIDVal, _ := c.Get("user_id")
		userID, _ := userIDVal.(string)
		roleVal, _ := c.Get("role")
		role, _ := roleVal.(string)
		permsVal, exists := c.Get("extra_perm")
		var perms []string
		if exists {
			perms, _ = permsVal.([]string)
		}

		g := guard.New()
		cmd, ok := g.CheckCommand(req.SQL, perms)
		if !ok {
			response.Error(c, http.StatusForbidden, "PERMISSION_DENIED",
				"you do not have permission to execute "+string(cmd)+" statements")
			return
		}

		exe := executor.New(mgr)
		result := exe.Execute(c.Request.Context(), executor.ExecutionRequest{
			ConnectionID: req.ConnectionID,
			SQL:          req.SQL,
			ConfirmHigh:  req.ConfirmHigh,
			UserID:       userID,
			Role:         role,
			Permissions:  perms,
		})

		if result.NeedsConfirm {
			response.Error(c, http.StatusConflict, "CONFIRMATION_REQUIRED", result.RiskInfo)
			return
		}

		if !result.Success {
			response.Error(c, http.StatusBadGateway, "EXECUTION_FAILED", result.Error)
			return
		}

		response.Success(c, result.Result)
	}
}
