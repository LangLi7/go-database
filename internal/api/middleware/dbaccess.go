package middleware

import (
	"strconv"
	"strings"

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

		// Resolve effective perms + db_access including parent-role inheritance.
		loader := func(id string) (*auth.Role, bool) {
			if loadRole == nil {
				return nil, false
			}
			r := loadRole(c, id)
			if r == nil {
				return nil, false
			}
			return r, true
		}
		effectivePerms := auth.GetEffectivePerms(roleStr, loader, extraPermSlice)
		effectiveDBAccess := auth.GetEffectiveDBAccess(roleStr, loader, extraDBAccessSlice)

		// Store effective db_access in context for handlers that need it
		c.Set("effective_db_access", effectiveDBAccess)
		// Propagate to downstream http.Handler (MCP scope func reads these
		// headers, since http.Request has no gin context).
		c.Request.Header.Set("X-Effective-DBAccess", strings.Join(effectiveDBAccess, ","))
		c.Request.Header.Set("X-Is-Admin", strconv.FormatBool(roleStr == "admin"))

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
