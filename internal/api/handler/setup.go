package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/internaldb"
)

type setupStatusResponse struct {
	SetupComplete bool `json:"setup_complete"`
}

type initializeSetupRequest struct {
	Email    string `json:"email" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=8"`
}

func SetupStatus(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		complete, err := store.IsSetupComplete(c.Request.Context())
		if err != nil {
			response.InternalError(c, "failed to check setup status")
			return
		}
		response.Success(c, setupStatusResponse{SetupComplete: complete})
	}
}

func InitializeSetup(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		complete, err := store.IsSetupComplete(c.Request.Context())
		if err != nil {
			response.InternalError(c, "failed to check setup status")
			return
		}
		if complete {
			response.Error(c, http.StatusConflict, "ALREADY_SETUP", "setup already completed")
			return
		}

		var req initializeSetupRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "valid email (min 3 chars) and password (min 8 chars) required")
			return
		}

		if err := store.CompleteSetup(c.Request.Context(), req.Email, req.Password); err != nil {
			response.InternalError(c, "failed to complete setup")
			return
		}

		if err := store.LogAudit(c.Request.Context(), "admin", "setup.complete", "First-time setup completed"); err != nil {
			// non-fatal
		}

		c.Status(http.StatusNoContent)
	}
}
