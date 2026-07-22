package handler

import (
	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/internaldb"
)

type createAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
	DBAccess    []string `json:"db_access"` // connection IDs this key may access
}

// ListAPIKeys returns all API keys (without hashes)
func ListAPIKeys(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		keys, err := store.ListKeys(c.Request.Context())
		if err != nil {
			response.InternalError(c, "failed to list API keys")
			return
		}

		// Never expose hashes
		type safeKey struct {
			Prefix      string   `json:"prefix"`
			Name        string   `json:"name"`
			Permissions []string `json:"permissions"`
			OwnerID     string   `json:"owner_id,omitempty"`
			DBAccess    []string `json:"db_access,omitempty"`
			CreatedAt   string   `json:"created_at"`
		}
		result := make([]safeKey, len(keys))
		for i, k := range keys {
			result[i] = safeKey{
				Prefix:      k.Prefix,
				Name:        k.Name,
				Permissions: k.Permissions,
				OwnerID:     k.OwnerID,
				DBAccess:    k.DBAccess,
				CreatedAt:   k.CreatedAt,
			}
		}

		response.Success(c, result)
	}
}

// CreateAPIKey generates a new API key
func CreateAPIKey(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createAPIKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "name required")
			return
		}

		// key belongs to the authenticated user (multi-tenant)
		ownerID, _ := c.Get("user_id")
		owner, _ := ownerID.(string)

		svc := auth.NewAPIKeyService(store)
		rawKey, stored, err := svc.Generate(c.Request.Context(), req.Name, req.Permissions, owner, req.DBAccess)
		if err != nil {
			response.InternalError(c, "failed to generate API key")
			return
		}

		response.Created(c, gin.H{
			"raw_key":     rawKey,
			"prefix":      stored.Prefix,
			"name":        stored.Name,
			"permissions": stored.Permissions,
			"owner_id":    stored.OwnerID,
			"db_access":   stored.DBAccess,
			"formatted":   auth.FormatKey(rawKey),
		})
	}
}

// DeleteAPIKey revokes an API key
func DeleteAPIKey(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc := auth.NewAPIKeyService(store)
		if err := svc.Revoke(c.Request.Context(), c.Param("prefix")); err != nil {
			response.NotFound(c, "API key not found")
			return
		}
		c.Status(204)
	}
}
