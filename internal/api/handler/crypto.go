package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/crypto"
)

type createKeyRequest struct {
	Algorithm string `json:"algorithm" binding:"required"`
	Purpose   string `json:"purpose" binding:"required"`
}

type encryptRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	Plaintext string `json:"plaintext" binding:"required"`
	AAD       string `json:"aad,omitempty"`
}

type decryptRequest struct {
	KeyID      string `json:"key_id" binding:"required"`
	Ciphertext string `json:"ciphertext" binding:"required"`
	Nonce      string `json:"nonce" binding:"required"`
	Algorithm  string `json:"algorithm" binding:"required"`
	Tag        string `json:"tag,omitempty"`
	AAD        string `json:"aad,omitempty"`
}

type columnCryptoRequest struct {
	KeyID string `json:"key_id" binding:"required"`
	Where string `json:"where,omitempty"`
}

func escapeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ListCryptoKeys returns all encryption keys for the current user
func ListCryptoKeys(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		keys, err := svc.ListKeys(uid)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.Success(c, keys)
	}
}

// CreateCryptoKey generates a new encryption key
func CreateCryptoKey(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		key, err := svc.CreateKey(uid, crypto.Algorithm(req.Algorithm), crypto.KeyPurpose(req.Purpose))
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		key.PrivKey = nil
		response.Created(c, key)
	}
}

// DeleteCryptoKey removes an encryption key
func DeleteCryptoKey(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteKey(c.Param("id")); err != nil {
			response.NotFound(c, "key not found")
			return
		}
		c.Status(204)
	}
}
func HashData(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req crypto.HashRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		result, err := svc.Hash(req.Algorithm, req.Data, req.Encoding)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.Success(c, result)
	}
}

// ListCryptoAlgorithms returns metadata about supported algorithms.
func ListCryptoAlgorithms(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.Success(c, svc.ListAlgorithms())
	}
}

// SignData signs data with an asymmetric key.
func SignData(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req crypto.SignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		result, err := svc.Sign(uid, req.KeyID, req.Data)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.Success(c, result)
	}
}

// VerifyData verifies a signature.
func VerifyData(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req crypto.VerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		result, err := svc.Verify(uid, req.KeyID, req.Data, req.Signature)
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.Success(c, result)
	}
}

// RotateCryptoKey generates a new key version for a user (key rotation).
func RotateCryptoKey(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		key, err := svc.CreateKey(uid, crypto.Algorithm(req.Algorithm), crypto.KeyPurpose(req.Purpose))
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		key.PrivKey = nil
		response.Created(c, key)
	}
}

// EncryptData encrypts plaintext using the specified key
func EncryptData(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req encryptRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		result, err := svc.Encrypt(&crypto.EncryptRequest{
			KeyID:     req.KeyID,
			Plaintext: req.Plaintext,
			AAD:       req.AAD,
		}, userID.(string))
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.Success(c, result)
	}
}

// DecryptData decrypts ciphertext using the specified key
func DecryptData(svc *crypto.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req decryptRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		result, err := svc.Decrypt(&crypto.DecryptRequest{
			KeyID:      req.KeyID,
			Ciphertext: req.Ciphertext,
			Nonce:      req.Nonce,
			Algorithm:  req.Algorithm,
			Tag:        req.Tag,
			AAD:        req.AAD,
		}, userID.(string))
		if err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		response.Success(c, result)
	}
}

// EncryptColumn encrypts all values in a table column
func EncryptColumn(svc *crypto.Service, connMgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")
		column := c.Param("column")
		userID, _ := c.Get("user_id")

		var req columnCryptoRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}

		where := req.Where
		if where == "" {
			where = "1=1"
		}
		query := `SELECT rowid, "` + column + `" FROM "` + table + `" WHERE ` + where
		result, err := connMgr.Query(c.Request.Context(), connID, query)
		if err != nil {
			response.InternalError(c, "query failed: "+err.Error())
			return
		}
		if result == nil || len(result.Rows) == 0 {
			response.Success(c, gin.H{"rows_affected": 0})
			return
		}

		affected := 0
		for _, row := range result.Rows {
			if len(row) < 2 {
				continue
			}
			rowid := row[0]
			val := row[1]
			if val == nil {
				continue
			}
			plaintext := fmt.Sprintf("%v", val)
			encResult, err := svc.Encrypt(&crypto.EncryptRequest{
				KeyID:     req.KeyID,
				Plaintext: plaintext,
			}, userID.(string))
			if err != nil {
				slog.Warn("encrypt failed", "rowid", rowid, "err", err)
				continue
			}
			encoded := fmt.Sprintf(`{"ct":"%s","n":"%s","t":"%s","a":"%s","k":"%s"}`,
				encResult.Ciphertext, encResult.Nonce, encResult.Tag, encResult.Algorithm, encResult.KeyID)
			updateQuery := fmt.Sprintf(`UPDATE "%s" SET "%s" = '%s' WHERE rowid = %v`,
				table, column, escapeSQL(encoded), rowid)
			if _, err := connMgr.Execute(c.Request.Context(), connID, updateQuery); err != nil {
				slog.Warn("update failed", "rowid", rowid, "err", err)
				continue
			}
			affected++
		}

		response.Success(c, gin.H{"rows_affected": affected})
	}
}

// DecryptColumn decrypts all values in a table column
func DecryptColumn(svc *crypto.Service, connMgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		connID := c.Param("id")
		table := c.Param("table")
		column := c.Param("column")
		userID, _ := c.Get("user_id")

		where := c.DefaultQuery("where", "1=1")
		query := fmt.Sprintf(`SELECT rowid, "%s" FROM "%s" WHERE %s`, column, table, where)
		result, err := connMgr.Query(c.Request.Context(), connID, query)
		if err != nil {
			response.InternalError(c, "query failed: "+err.Error())
			return
		}
		if result == nil || len(result.Rows) == 0 {
			response.Success(c, gin.H{"rows_affected": 0})
			return
		}

		affected := 0
		for _, row := range result.Rows {
			if len(row) < 2 {
				continue
			}
			rowid := row[0]
			val := row[1]
			if val == nil {
				continue
			}
			valStr := fmt.Sprintf("%v", val)
			var env struct {
				Ct string `json:"ct"`
				N  string `json:"n"`
				T  string `json:"t"`
				A  string `json:"a"`
				K  string `json:"k"`
			}
			if err := json.Unmarshal([]byte(valStr), &env); err != nil {
				slog.Warn("parse envelope failed", "rowid", rowid)
				continue
			}
			decResult, err := svc.Decrypt(&crypto.DecryptRequest{
				KeyID:      env.K,
				Ciphertext: env.Ct,
				Nonce:      env.N,
				Algorithm:  env.A,
				Tag:        env.T,
			}, userID.(string))
			if err != nil {
				slog.Warn("decrypt failed", "rowid", rowid, "err", err)
				continue
			}
			updateQuery := fmt.Sprintf(`UPDATE "%s" SET "%s" = '%s' WHERE rowid = %v`,
				table, column, escapeSQL(decResult.Plaintext), rowid)
			if _, err := connMgr.Execute(c.Request.Context(), connID, updateQuery); err != nil {
				slog.Warn("update failed", "rowid", rowid, "err", err)
				continue
			}
			affected++
		}

		response.Success(c, gin.H{"rows_affected": affected})
	}
}
