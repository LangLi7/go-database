package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
)

// GetTrafficStats returns traffic monitoring data
func GetRequests(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple mock traffic data for now
		response.Success(c, gin.H{
			"requests": []gin.H{},
			"period":   "24h",
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
