package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/internaldb"
)

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type tokenResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func Login(store *internaldb.Store, jwt *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "username and password required")
			return
		}

		user, err := store.GetUser(c.Request.Context(), req.Username)
		if err != nil {
			response.Unauthorized(c, "invalid credentials")
			return
		}

		if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
			response.Unauthorized(c, "invalid credentials")
			return
		}

		// First-time setup check: if admin uses default password, force setup
		if user.Username == "admin" && auth.IsDefaultPassword(user.PasswordHash) {
			response.Error(c, http.StatusForbidden, "SETUP_REQUIRED", "default admin password must be changed")
			return
		}
		token, err := jwt.GenerateToken(user.ID, user.Username, user.Role, user.ExtraPerm, user.ExtraDBAccess)

		if err != nil {
			response.InternalError(c, "failed to generate token")
			return
		}

		response.Success(c, tokenResponse{
			Token:    token,
			UserID:   user.ID,
			Username: user.Username,
			Role:     user.Role,
		})
	}
}

type refreshRequest struct {
	Token string `json:"token" binding:"required"`
}

func RefreshToken(jwt *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req refreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			tokenStr := c.GetHeader("Authorization")
			if tokenStr == "" {
				response.BadRequest(c, "token required")
				return
			}
			req.Token = tokenStr
		}

		claims, err := jwt.ValidateToken(req.Token)
		if err != nil {
			response.Unauthorized(c, "invalid token")
			return
		}

		newToken, err := jwt.GenerateToken(claims.UserID, claims.Username, claims.Role, claims.ExtraPerm, claims.ExtraDBAccess)
		if err != nil {
			response.InternalError(c, "failed to refresh token")
			return
		}

		response.Success(c, tokenResponse{
			Token:    newToken,
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		})
	}
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func ChangePassword(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists {
			response.Unauthorized(c, "user not found")
			return
		}
		userID, ok := userIDVal.(string)
		if !ok {
			response.InternalError(c, "invalid user id")
			return
		}

		var req changePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "old and new password required")
			return
		}

		user, err := store.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			response.Unauthorized(c, "user not found")
			return
		}

		if err := auth.CheckPassword(req.OldPassword, user.PasswordHash); err != nil {
			response.Unauthorized(c, "invalid current password")
			return
		}

		newHash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			response.InternalError(c, "failed to hash password")
			return
		}

		user.PasswordHash = newHash
		if err := store.SaveUser(c.Request.Context(), *user); err != nil {
			response.InternalError(c, "failed to update password")
			return
		}

		if err := store.LogAudit(c.Request.Context(), userID, "password.change", "Password changed"); err != nil {
			slog.Warn("failed to log password change audit", "user", userID, "error", err)
		}
		c.Status(http.StatusNoContent)
	}
}

func VerifyToken(jwt *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")
		role, _ := c.Get("role")
		extraPerm, ok := c.Get("extra_perm")
		if !ok {
			response.InternalError(c, "missing auth context")
			return
		}
		response.Success(c, gin.H{
			"user_id":    userID,
			"username":   username,
			"role":       role,
			"extra_perm": extraPerm,
		})
	}
}
