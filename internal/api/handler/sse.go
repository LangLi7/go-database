package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/connection"
)

// SSEActivityHandler streams audit log events via Server-Sent Events
func SSEActivityHandler(connMgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(gin.ResponseWriter)
		if !ok {
			slog.Error("sse: flush not supported")
			return
		}

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		// initial connection event
		fmt.Fprintf(c.Writer, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
		flusher.Flush()

		slog.Info("sse connected", "user", username)

		for {
			select {
			case <-c.Request.Context().Done():
				slog.Info("sse disconnected", "user", username)
				return

			case <-ticker.C:
				// send heartbeat + activity placeholder
				stats := connMgr.List()
				payload, _ := json.Marshal(map[string]any{
					"timestamp":   time.Now().UTC(),
					"connections": len(stats),
				})
				fmt.Fprintf(c.Writer, "event: heartbeat\ndata: %s\n\n", payload)
				flusher.Flush()
			}
		}
	}
}

// SSEStatsHandler streams live statistics via Server-Sent Events
func SSEStatsHandler(connMgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(gin.ResponseWriter)
		if !ok {
			slog.Error("sse: flush not supported")
			return
		}

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		fmt.Fprintf(c.Writer, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
		flusher.Flush()

		slog.Info("sse stats connected", "user", username)

		for {
			select {
			case <-c.Request.Context().Done():
				slog.Info("sse stats disconnected", "user", username)
				return

			case <-ticker.C:
				conns := connMgr.List()
				connected := 0
				for _, c := range conns {
					if c.State == "connected" {
						connected++
					}
				}
				payload, _ := json.Marshal(map[string]any{
					"timestamp":      time.Now().UTC(),
					"total_connections": len(conns),
					"active_connections": connected,
				})
				fmt.Fprintf(c.Writer, "event: stats\ndata: %s\n\n", payload)
				flusher.Flush()
			}
		}
	}
}
