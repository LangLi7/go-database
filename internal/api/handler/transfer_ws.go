package handler

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/transfer"
)

// WSTransferProgress creates a Gin handler for streaming transfer job progress via WebSocket
func WSTransferProgress(engine transfer.TransferEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobID := c.Param("id")

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("ws upgrade failed", "err", err)
			return
		}
		defer ws.Close()

		ch, err := engine.Subscribe(jobID)
		if err != nil {
			slog.Error("subscribe failed", "job", jobID, "err", err)
			ws.WriteJSON(wsRespMsg{Type: "error", Error: err.Error()})
			return
		}
		defer engine.Unsubscribe(jobID, ch)

		// Read loop to detect client disconnect
		go func() {
			for {
				if _, _, err := ws.ReadMessage(); err != nil {
					break
				}
			}
		}()

		for evt := range ch {
			if err := ws.WriteJSON(evt); err != nil {
				break
			}
		}
	}
}

// GetTransferLog returns the logs for a transfer job
func GetTransferLog(engine transfer.TransferEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		job, err := engine.Status(c.Param("id"))
		if err != nil {
			response.NotFound(c, "transfer job not found")
			return
		}
		response.Success(c, gin.H{
			"id":     c.Param("id"),
			"status": job.Status,
			"log":    job.Log,
		})
	}
}
