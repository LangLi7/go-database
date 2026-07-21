package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/transfer"
)

type migrateRequest struct {
	SourceConn string   `json:"source_conn" binding:"required"`
	TargetConn string   `json:"target_conn" binding:"required"`
	Tables     []string `json:"tables"`
	DryRun     bool     `json:"dry_run"`
	BatchSize  int      `json:"batch_size"`
	OnConflict string   `json:"on_conflict"` // "error" | "skip" | "overwrite"
}

type migrateResponse struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Tables []string `json:"tables,omitempty"`
	Source string   `json:"source_type"`
	Target string   `json:"target_type"`
}

func StartTransfer(mgr *connection.Manager, engine transfer.TransferEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req migrateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "source_conn and target_conn required")
			return
		}

		// Resolve source and target types
		srcConn, err := mgr.GetConnection(req.SourceConn)
		if err != nil {
			response.BadRequest(c, fmt.Sprintf("source: %v", err))
			return
		}
		tgtConn, err := mgr.GetConnection(req.TargetConn)
		if err != nil {
			response.BadRequest(c, fmt.Sprintf("target: %v", err))
			return
		}

		if srcConn.Type == tgtConn.Type && !req.DryRun {
			response.Error(c, http.StatusBadRequest, "SAME_TYPE", "source and target are the same type — use export instead")
			return
		}

		onConflict := req.OnConflict
		if onConflict == "" {
			onConflict = "error"
		}

		job := &transfer.TransferJob{
			SourceType: string(srcConn.Type),
			TargetType: string(tgtConn.Type),
			SourceConn: req.SourceConn,
			TargetConn: req.TargetConn,
			Tables:     req.Tables,
			DryRun:     req.DryRun,
			BatchSize:  req.BatchSize,
			OnConflict: onConflict,
		}

		if err := engine.Start(context.Background(), job); err != nil {
			response.InternalError(c, fmt.Sprintf("failed to start: %v", err))
			return
		}

		response.Created(c, migrateResponse{
			ID:     job.ID,
			Status: "pending",
			Source: string(srcConn.Type),
			Target: string(tgtConn.Type),
		})
	}
}

func GetTransferStatus(engine transfer.TransferEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		job, err := engine.Status(c.Param("id"))
		if err != nil {
			response.NotFound(c, "transfer job not found")
			return
		}
		response.Success(c, migrateResponse{
			ID:     job.ID,
			Status: job.Status,
			Tables: job.Tables,
			Source: job.SourceType,
			Target: job.TargetType,
		})
	}
}

func CancelTransfer(engine transfer.TransferEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := engine.Cancel(c.Param("id")); err != nil {
			response.NotFound(c, "transfer job not found")
			return
		}
		c.Status(http.StatusNoContent)
	}
}
