package handler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/internaldb"
)

// --- Stats ---

// GetStats returns dashboard statistics
func GetStats(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		conns := mgr.List()
		online := 0
		totalLatency := 0.0
		byType := make(map[string]int)
		var connections []gin.H

		for _, conn := range conns {
			if conn.State == connection.StateConnected {
				online++
				totalLatency += float64(conn.Latency)
			}
			byType[string(conn.Type)]++
			connections = append(connections, gin.H{
				"id":         conn.ID,
				"name":       conn.Name,
				"type":       conn.Type,
				"state":      conn.State,
				"latency_ms": int64(conn.Latency),
				"source":     conn.Source,
			})
		}

		avgLatency := 0.0
		if online > 0 {
			avgLatency = totalLatency / float64(online)
		}

		response.Success(c, gin.H{
			"connections_total":   len(conns),
			"connections_online":  online,
			"connections_offline": len(conns) - online,
			"avg_latency_ms":      avgLatency,
			"connections_by_type": byType,
			"connections":         connections,
		})
	}
}

// --- Design Config ---

type designRequest struct {
	ID     string `json:"id"`
	Name   string `json:"name" binding:"required"`
	Config string `json:"config" binding:"required"`
	Active bool   `json:"active"`
}

// GetDesign returns the active design configuration
func GetDesign(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		dc, err := store.GetActiveDesign(c.Request.Context())
		if err != nil {
			// Return default design
			response.Success(c, gin.H{
				"id":     "default",
				"name":   "Default",
				"config": `{"primary_color":"#6366f1","sidebar_width":"260px","font_size":"14px","compact":false,"dark_mode":false}`,
			})
			return
		}
		response.Success(c, dc)
	}
}

// SaveDesign creates or updates a design configuration
func SaveDesign(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req designRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "name and config required")
			return
		}

		dc := internaldb.DesignConfig{
			ID:        req.ID,
			Name:      req.Name,
			Config:    req.Config,
			Active:    req.Active,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		if dc.ID == "" {
			dc.ID = fmt.Sprintf("design-%d", time.Now().Unix())
		}

		if err := store.SaveDesign(c.Request.Context(), dc); err != nil {
			response.InternalError(c, "failed to save design")
			return
		}

		response.Created(c, dc)
	}
}

// --- Activity ---

// GetActivity returns the audit log
func GetActivity(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logs, err := store.ListAuditLog(c.Request.Context(), 100)
		if err != nil {
			response.InternalError(c, "failed to get activity")
			return
		}
		response.Success(c, logs)
	}
}

// --- Users ---

type userRequest struct {
	Username  string   `json:"username" binding:"required"`
	Password  string   `json:"password,omitempty"`
	Role      string   `json:"role" binding:"required"`
	ExtraPerm []string `json:"extra_perm"`
}

// ListUsers returns all users
func ListUsers(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := store.ListUsers(c.Request.Context())
		if err != nil {
			response.InternalError(c, "failed to list users")
			return
		}
		// Mask password hashes
		type safeUser struct {
			ID        string   `json:"id"`
			Username  string   `json:"username"`
			Role      string   `json:"role"`
			ExtraPerm []string `json:"extra_perm,omitempty"`
		}
		result := make([]safeUser, len(users))
		for i, u := range users {
			result[i] = safeUser{ID: u.ID, Username: u.Username, Role: u.Role, ExtraPerm: u.ExtraPerm}
		}
		response.Success(c, result)
	}
}

// CreateUser creates a new user
func CreateUser(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req userRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "username and role required")
			return
		}

		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			response.InternalError(c, "failed to hash password")
			return
		}

		user := auth.User{
			ID:           fmt.Sprintf("user-%d", time.Now().Unix()),
			Username:     req.Username,
			PasswordHash: hash,
			Role:         req.Role,
			ExtraPerm:    req.ExtraPerm,
		}

		if err := store.SaveUser(c.Request.Context(), user); err != nil {
			response.Conflict(c, "username already exists")
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			response.InternalError(c, "user_id not found in context")
			return
		}
		uid, ok := userID.(string)
		if !ok {
			response.InternalError(c, "invalid user_id type")
			return
		}
		if err := store.LogAudit(c.Request.Context(), uid, "user.create", req.Username); err != nil {
			slog.Warn("failed to log user create audit", "user", uid, "error", err)
		}

		response.Created(c, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
	}
}

// UpdateUser modifies a user's role/permissions
func UpdateUser(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		existing, err := store.GetUserByID(c.Request.Context(), id)
		if err != nil {
			response.NotFound(c, "user not found")
			return
		}

		var req userRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request")
			return
		}

		if req.Username != "" {
			existing.Username = req.Username
		}
		if req.Role != "" {
			existing.Role = req.Role
		}
		if req.Password != "" {
			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				response.InternalError(c, "failed to hash password")
				return
			}
			existing.PasswordHash = hash
		}
		if req.ExtraPerm != nil {
			existing.ExtraPerm = req.ExtraPerm
		}

		if err := store.SaveUser(c.Request.Context(), *existing); err != nil {
			response.InternalError(c, "failed to update user")
			return
		}

		c.Status(204)
	}
}

// DeleteUser removes a user
func DeleteUser(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
			response.NotFound(c, "user not found")
			return
		}
		c.Status(204)
	}
}

// GetUserPermissions returns a user's effective permissions
func GetUserPermissions(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := store.GetUserByID(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.NotFound(c, "user not found")
			return
		}

		role, err := store.GetRole(c.Request.Context(), user.Role)
		if err != nil {
			response.NotFound(c, "role not found")
			return
		}

		// effective perms including parent-role inheritance
		loader := func(id string) (*auth.Role, bool) {
			r, err := store.GetRole(c.Request.Context(), id)
			if err != nil {
				return nil, false
			}
			return r, true
		}
		effective := auth.GetEffectivePerms(user.Role, loader, user.ExtraPerm)
		dbAccess := auth.GetEffectiveDBAccess(user.Role, loader, user.ExtraDBAccess)
		response.Success(c, gin.H{
			"user_id":     user.ID,
			"role":        user.Role,
			"role_perms":  role.Permissions,
			"extra_perms": user.ExtraPerm,
			"effective":   effective,
			"db_access":   append(dbAccess, user.ExtraDBAccess...),
		})
	}
}

// SetUserPermissions updates a user's extra permissions
func SetUserPermissions(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		user, err := store.GetUserByID(c.Request.Context(), id)
		if err != nil {
			response.NotFound(c, "user not found")
			return
		}

		var req struct {
			ExtraPerm     []string `json:"extra_perm"`
			ExtraDBAccess []string `json:"extra_db_access"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request")
			return
		}

		user.ExtraPerm = req.ExtraPerm
		user.ExtraDBAccess = req.ExtraDBAccess

		if err := store.SaveUser(c.Request.Context(), *user); err != nil {
			response.InternalError(c, "failed to update permissions")
			return
		}

		c.Status(204)
	}
}

// --- Roles ---

// ListRoles returns all roles
func ListRoles(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, err := store.ListRoles(c.Request.Context())
		if err != nil {
			response.InternalError(c, "failed to list roles")
			return
		}
		response.Success(c, roles)
	}
}

// CreateRole creates a new role
func CreateRole(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ID          string   `json:"id"`
			Name        string   `json:"name" binding:"required"`
			Permissions []string `json:"permissions"`
			DBAccess    []string `json:"db_access"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "name required")
			return
		}

		role := auth.Role{
			ID:          req.ID,
			Name:        req.Name,
			Permissions: req.Permissions,
			DBAccess:    req.DBAccess,
		}
		if role.ID == "" {
			role.ID = fmt.Sprintf("role-%d", time.Now().Unix())
		}

		if err := store.SaveRole(c.Request.Context(), role); err != nil {
			response.Conflict(c, "role already exists")
			return
		}

		response.Created(c, role)
	}
}

// UpdateRole modifies a role
func UpdateRole(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string   `json:"name"`
			Permissions []string `json:"permissions"`
			DBAccess    []string `json:"db_access"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request")
			return
		}

		role, err := store.GetRole(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.NotFound(c, "role not found")
			return
		}

		if req.Name != "" {
			role.Name = req.Name
		}
		if req.Permissions != nil {
			role.Permissions = req.Permissions
		}
		if req.DBAccess != nil {
			role.DBAccess = req.DBAccess
		}

		if err := store.SaveRole(c.Request.Context(), *role); err != nil {
			response.InternalError(c, "failed to update role")
			return
		}

		c.Status(204)
	}
}

// DeleteRole removes a role
func DeleteRole(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := store.DeleteRole(c.Request.Context(), c.Param("id")); err != nil {
			response.NotFound(c, "role not found")
			return
		}
		c.Status(204)
	}
}

// SetRolePermissions updates a role's permissions
func SetRolePermissions(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, err := store.GetRole(c.Request.Context(), c.Param("id"))
		if err != nil {
			response.NotFound(c, "role not found")
			return
		}

		var req struct {
			Permissions []string `json:"permissions"`
			DBAccess    []string `json:"db_access"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request")
			return
		}

		if req.Permissions != nil {
			role.Permissions = req.Permissions
		}
		if req.DBAccess != nil {
			role.DBAccess = req.DBAccess
		}

		if err := store.SaveRole(c.Request.Context(), *role); err != nil {
			response.InternalError(c, "failed to update role permissions")
			return
		}

		c.Status(204)
	}
}
