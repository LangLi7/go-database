package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
)

// AuthMiddleware validates JWT tokens from the Authorization header
func AuthMiddleware(jwt *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		if tokenStr == header {
			response.Unauthorized(c, "invalid authorization format, use: Bearer <token>")
			c.Abort()
			return
		}

		claims, err := jwt.ValidateToken(tokenStr)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("extra_perm", claims.ExtraPerm)
		c.Next()
	}
}

// PermissionMiddleware checks that the authenticated user has the required permission
func PermissionMiddleware(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		extraPerm, _ := c.Get("extra_perm")

		roleStr, _ := role.(string)
		extraPermSlice, _ := extraPerm.([]string)

		rolePerms := getRolePermissions(roleStr)
		effective := auth.GetEffectivePerms(rolePerms, extraPermSlice)

		if !auth.HasPermission(effective, requiredPerm) {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

// getRolePermissions returns the base permissions for a role name
func getRolePermissions(roleName string) []string {
	for _, r := range auth.DefaultRoles() {
		if r.ID == roleName {
			return r.Permissions
		}
	}
	return nil
}
