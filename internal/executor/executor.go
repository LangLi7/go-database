package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-database/internal/connection"
	"go-database/internal/plugin"
	"go-database/internal/suggest"
)

type ExecutionRequest struct {
	ConnectionID string `json:"connection_id"`
	SQL          string `json:"sql"`
	ConfirmHigh  bool   `json:"confirm_high"`
	UserID       string `json:"user_id"`
	Role         string `json:"role"`
	Permissions  []string `json:"permissions"`
}

type ExecutionResult struct {
	Success     bool              `json:"success"`
	Result      *plugin.Result    `json:"result,omitempty"`
	RiskLevel   suggest.RiskLevel `json:"risk_level"`
	RiskInfo    string            `json:"risk_info,omitempty"`
	NeedsConfirm bool             `json:"needs_confirm"`
	Error       string            `json:"error,omitempty"`
}

type Executor struct {
	mgr  *connection.Manager
	risk *suggest.RiskEvaluator
}

func New(mgr *connection.Manager) *Executor {
	return &Executor{mgr: mgr, risk: suggest.NewRiskEvaluator()}
}

func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) *ExecutionResult {
	sql := strings.TrimSpace(req.SQL)
	if sql == "" {
		return &ExecutionResult{Success: false, Error: "empty query"}
	}

	risk, riskInfo := e.risk.Classify(sql)

	// If high risk and not confirmed, require confirmation
	if risk == suggest.RiskHigh && !req.ConfirmHigh {
		affected := e.estimateAffected(ctx, req.ConnectionID, sql)
		return &ExecutionResult{
			Success:      false,
			RiskLevel:    risk,
			RiskInfo:     riskInfo,
			NeedsConfirm: true,
			Result:       affected,
		}
	}

	// If medium risk and not confirmed, still ask for safety
	if risk == suggest.RiskMedium && !req.ConfirmHigh {
		affected := e.estimateAffected(ctx, req.ConnectionID, sql)
		if affected != nil && affected.RowsAffected > 100 {
			return &ExecutionResult{
				Success:      false,
				RiskLevel:    risk,
				RiskInfo:     riskInfo + fmt.Sprintf(" (affects ~%d rows)", affected.RowsAffected),
				NeedsConfirm: true,
				Result:       affected,
			}
		}
	}

	// Execute
	var result *plugin.Result
	var err error

	start := time.Now()
	isQuery := strings.HasPrefix(strings.ToUpper(sql), "SELECT") ||
		strings.HasPrefix(strings.ToUpper(sql), "SHOW") ||
		strings.HasPrefix(strings.ToUpper(sql), "DESCRIBE") ||
		strings.HasPrefix(strings.ToUpper(sql), "EXPLAIN")

	if isQuery {
		result, err = e.mgr.Query(ctx, req.ConnectionID, sql)
	} else {
		result, err = e.mgr.Execute(ctx, req.ConnectionID, sql)
	}

	if err != nil {
		return &ExecutionResult{
			Success:   false,
			Error:     err.Error(),
			RiskLevel: risk,
			RiskInfo:  riskInfo,
		}
	}

	result.Duration = time.Since(start).Milliseconds()

	return &ExecutionResult{
		Success:   true,
		Result:    result,
		RiskLevel: risk,
		RiskInfo:  riskInfo,
	}
}

func (e *Executor) estimateAffected(ctx context.Context, connID, sql string) *plugin.Result {
	upper := strings.ToUpper(strings.TrimSpace(sql))

	if strings.HasPrefix(upper, "DELETE") || strings.HasPrefix(upper, "UPDATE") {
		// Try to run as SELECT COUNT(*) to estimate
		selectSQL := "SELECT COUNT(*) FROM ("
		// Find FROM clause position
		fromIdx := strings.Index(upper, "FROM")
		if fromIdx >= 0 {
			rest := sql[fromIdx:]
			whereIdx := strings.Index(strings.ToUpper(rest), "WHERE")
			if whereIdx >= 0 {
				selectSQL = "SELECT COUNT(*) AS estimate FROM " + rest
			} else {
				selectSQL = "SELECT COUNT(*) AS estimate FROM " + rest
			}
			result, err := e.mgr.Query(ctx, connID, selectSQL)
			if err == nil && len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
				if count, ok := result.Rows[0][0].(int64); ok {
					return &plugin.Result{RowsAffected: count}
				}
			}
		}
	}

	if strings.HasPrefix(upper, "DROP") || strings.HasPrefix(upper, "TRUNCATE") {
		if strings.Contains(upper, "TABLE") {
			return &plugin.Result{RowsAffected: -1}
		}
	}

	return nil
}
