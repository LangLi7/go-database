package middleware

import (
	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
)

// DBAccessMiddleware checks if the user has access to the connection specified in the :id param.
// It reads the user's effective permissions and db_access list from the Gin context.
// The permissions and db_access are combined from the user's role and per-user overrides.
func DBAccessMiddleware(loadRole RoleByName) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		if connID == "" {
			c.Next()
			return
		}

		roleName, exists := c.Get("role")
		if !exists {
			response.Forbidden(c, "missing role")
			c.Abort()
			return
		}
		extraPerm, _ := c.Get("extra_perm")
		extraDBAccess, _ := c.Get("extra_db_access")

		roleStr, _ := roleName.(string)
		extraPermSlice, _ := extraPerm.([]string)
		extraDBAccessSlice, _ := extraDBAccess.([]string)

		// Get role permissions and db_access
		rolePerms := getRolePermissions(roleStr, c, loadRole)
		roleDBAccess := getRoleDBAccess(roleStr, c, loadRole)

		// Merge with user overrides
		effectivePerms := auth.GetEffectivePerms(rolePerms, extraPermSlice)
		effectiveDBAccess := auth.GetEffectiveDBAccess(roleDBAccess, extraDBAccessSlice)

		// Store effective db_access in context for handlers that need it
		c.Set("effective_db_access", effectiveDBAccess)

		if !auth.CheckDBAccess(effectivePerms, effectiveDBAccess, connID) {
			response.Forbidden(c, "no access to this connection")
			c.Abort()
			return
		}
		c.Next()
	}
}

// getRoleDBAccess returns the db_access list for a role
func getRoleDBAccess(roleName string, c *gin.Context, loadRole RoleByName) []string {
	for _, r := range auth.DefaultRoles() {
		if r.ID == roleName {
			return r.DBAccess
		}
	}
	if loadRole != nil {
		if r := loadRole(c, roleName); r != nil {
			return r.DBAccess
		}
	}
	return nil
}
