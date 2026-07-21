package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
)

// RoleByName loads a role definition by name from the database
type RoleByName func(ctx *gin.Context, name string) *auth.Role

type AuthConfig struct {
	JWT    *auth.JWTService
	APIKey *auth.APIKeyService
}

func AuthMiddleware(cfg AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""

		// 1. Check Authorization header (Bearer <token>)
		header := c.GetHeader("Authorization")
		if header != "" {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
			if tokenStr == header {
				tokenStr = ""
			}
		}

		// 2. Fallback to X-API-Key header (for Discord/Minecraft bots)
		if tokenStr == "" {
			tokenStr = c.GetHeader("X-API-Key")
		}

		// 3. Fallback to token query parameter (for WebSocket connections)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		// 3. Try JWT first, fall back to API key (supports Discord/Minecraft bots)
		claims, jwtErr := cfg.JWT.ValidateToken(tokenStr)
		if jwtErr == nil {
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Set("role", claims.Role)
			c.Set("extra_perm", claims.ExtraPerm)
			c.Set("extra_db_access", claims.ExtraDBAccess)
			c.Next()
			return
		}

		// Try API key auth
		if cfg.APIKey != nil {
			key, apiErr := cfg.APIKey.Validate(c.Request.Context(), tokenStr)
			if apiErr == nil {
				c.Set("user_id", "apikey:"+key.Prefix)
				c.Set("username", "apikey:"+key.Name)
				c.Set("role", "apikey")
				c.Set("extra_perm", key.Permissions)
				c.Next()
				return
			}
		}

		response.Unauthorized(c, "invalid or expired token")
		c.Abort()
	}
}

func PermissionMiddleware(requiredPerm string, loadRole RoleByName) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "missing role in context")
			c.Abort()
			return
		}
		extraPerm, _ := c.Get("extra_perm")

		roleStr, ok := role.(string)
		if !ok {
			response.Forbidden(c, "invalid role type")
			c.Abort()
			return
		}
		extraPermSlice, _ := extraPerm.([]string)

		rolePerms := getRolePermissions(roleStr, c, loadRole)
		effective := auth.GetEffectivePerms(rolePerms, extraPermSlice)

		// Store effective perms in context for downstream use (Guard, etc.)
		c.Set("effective_perm", effective)

		if !auth.HasPermission(effective, requiredPerm) {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

func getRolePermissions(roleName string, c *gin.Context, loadRole RoleByName) []string {
	// Check built-in roles first
	for _, r := range auth.DefaultRoles() {
		if r.ID == roleName {
			return r.Permissions
		}
	}
	// Fall back to DB for custom roles
	if loadRole != nil {
		if r := loadRole(c, roleName); r != nil {
			return r.Permissions
		}
	}
	return nil
}
