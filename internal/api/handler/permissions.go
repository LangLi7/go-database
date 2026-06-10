package handler

import (
	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/connection"
	"go-database/internal/internaldb"
)

func GetPermissionGroups() gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, auth.PermissionGroups())
	}
}

func GetConnectionPermissions(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := mgr.Get(c.Param("id"))
		if err != nil {
			response.Success(c, []string{})
			return
		}
		perms := []string{
			"database:" + c.Param("id") + ":read",
			"database:" + c.Param("id") + ":write",
		}
		response.Success(c, perms)
	}
}

func GetUserDBAccess(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("id")
		user, err := store.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			response.Success(c, gin.H{"db_access": []string{}, "extra_db_access": []string{}})
			return
		}
		role, err := store.GetRole(c.Request.Context(), user.Role)
		if err != nil {
			response.Success(c, gin.H{"db_access": []string{}, "extra_db_access": user.ExtraDBAccess})
			return
		}
		response.Success(c, gin.H{"db_access": role.DBAccess, "extra_db_access": user.ExtraDBAccess})
	}
}

func SetUserDBAccess(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ExtraDBAccess []string `json:"extra_db_access"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request")
			return
		}
		if err := store.SetUserDBAccess(c.Request.Context(), c.Param("id"), req.ExtraDBAccess); err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.Success(c, gin.H{"status": "ok"})
	}
}
