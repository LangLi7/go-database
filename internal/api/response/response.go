package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
	Meta    *APIMeta  `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type APIMeta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

func now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Timestamp: now()},
	})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
		Meta:    &APIMeta{Timestamp: now()},
	})
}

func NoContent(c *gin.Context) {
	c.JSON(http.StatusNoContent, APIResponse{
		Success: true,
		Meta:    &APIMeta{Timestamp: now()},
	})
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Meta: &APIMeta{Timestamp: now()},
	})
}

func ErrorDetailed(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{Timestamp: now()},
	})
}

func BadRequest(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, "BAD_REQUEST", msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

func Forbidden(c *gin.Context, msg string) {
	Error(c, http.StatusForbidden, "FORBIDDEN", msg)
}

func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, "NOT_FOUND", msg)
}

func Conflict(c *gin.Context, msg string) {
	Error(c, http.StatusConflict, "CONFLICT", msg)
}

func InternalError(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", msg)
}
