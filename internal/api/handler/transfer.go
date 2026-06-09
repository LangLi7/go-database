package handler

import (
	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
)

type transferRequest struct {
	SourceConn string   `json:"source_conn" binding:"required"`
	TargetConn string   `json:"target_conn" binding:"required"`
	Tables     []string `json:"tables"`
	DryRun     bool     `json:"dry_run"`
	BatchSize  int      `json:"batch_size"`
}

// StartTransfer initiates a data transfer between two connections
func StartTransfer() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req transferRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "source_conn and target_conn required")
			return
		}
		// Placeholder: will be implemented with transfer engine
		response.Created(c, gin.H{
			"id":     "transfer-placeholder",
			"status": "pending",
			"message": "Transfer engine not yet fully implemented",
		})
	}
}

// GetTransferStatus returns the current status of a transfer job
func GetTransferStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, gin.H{
			"id":     c.Param("id"),
			"status": "pending",
		})
	}
}

// CancelTransfer stops a running transfer
func CancelTransfer() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Status(204)
	}
}

// GetTransferLog returns the error log for a transfer
func GetTransferLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, gin.H{
			"id":   c.Param("id"),
			"logs": []string{},
		})
	}
}
