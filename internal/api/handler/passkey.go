package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"

	"go-database/internal/api/response"
	"go-database/internal/auth"
	"go-database/internal/internaldb"
)

// passkeySessions holds the in-flight WebAuthn ceremony state per user.
// ponytail: single-instance in-memory map; replace with Redis/shared store if
// the server runs replicated.
type passkeySessions struct {
	mu sync.Mutex
	m  map[string]webauthn.SessionData // key: userID (register) or credentialID (login)
}

var passkeySessionStore = &passkeySessions{m: make(map[string]webauthn.SessionData)}

func (p *passkeySessions) put(key string, sd webauthn.SessionData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.m[key] = sd
}

func (p *passkeySessions) take(key string) (webauthn.SessionData, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	sd, ok := p.m[key]
	if ok {
		delete(p.m, key)
	}
	return sd, ok
}

// newWebAuthn builds a WebAuthn instance from server config.
func newWebAuthn(r *http.Request) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPDisplayName: "go-database",
		// Origin must match the frontend's scheme://host:port exactly.
		RPID:      rpIDFromRequest(r),
		RPOrigins: []string{rpOriginFromRequest(r)},
	})
}

// PasskeyRegisterBegin starts credential creation for the authenticated user.
func PasskeyRegisterBegin(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		user, err := store.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			response.NotFound(c, "user not found")
			return
		}
		existing, _ := store.ListPasskeys(c.Request.Context(), uid)
		creds := make([]webauthn.Credential, 0, len(existing))
		for _, pk := range existing {
			creds = append(creds, pk.ToCredential())
		}
		wu := &auth.WebAuthnUser{User: user, Credentials: creds}

		w, err := newWebAuthn(c.Request)
		if err != nil {
			response.InternalError(c, "webauthn init failed")
			return
		}
		opts, sessionData, err := w.BeginRegistration(wu)
		if err != nil {
			response.InternalError(c, "begin registration failed")
			return
		}
		passkeySessionStore.put(uid, *sessionData)
		c.JSON(http.StatusOK, opts)
	}
}

// PasskeyRegisterFinish completes credential creation and stores the passkey.
func PasskeyRegisterFinish(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		user, err := store.GetUserByID(c.Request.Context(), uid)
		if err != nil {
			response.NotFound(c, "user not found")
			return
		}
		sessionData, ok := passkeySessionStore.take(uid)
		if !ok {
			response.BadRequest(c, "no registration in progress")
			return
		}
		w, err := newWebAuthn(c.Request)
		if err != nil {
			response.InternalError(c, "webauthn init failed")
			return
		}
		cred, err := w.FinishRegistration(&auth.WebAuthnUser{User: user}, sessionData, c.Request)
		if err != nil {
			response.BadRequest(c, "registration failed: "+err.Error())
			return
		}
		name := c.Query("name")
		if name == "" {
			name = "Passkey"
		}
		pk := auth.PasskeyFromCredential(uid, name, *cred, time.Now().Unix())
		if err := store.SavePasskey(c.Request.Context(), pk); err != nil {
			response.InternalError(c, "failed to store passkey")
			return
		}
		_ = store.LogAudit(c.Request.Context(), uid, "passkey.register", name)
		// Return safe view (no private key material).
		response.Created(c, gin.H{"id": pk.ID, "name": pk.Name, "aaguid": pk.AAGUID, "created_at": pk.CreatedAt})
	}
}

// PasskeyLoginBegin starts an assertion for any user (public endpoint).
func PasskeyLoginBegin(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		w, err := newWebAuthn(c.Request)
		if err != nil {
			response.InternalError(c, "webauthn init failed")
			return
		}
		// User is resolved at finish time from the credential ID.
		opts, sessionData, err := w.BeginLogin(nil)
		if err != nil {
			response.InternalError(c, "begin login failed")
			return
		}
		// Key login session by a stable random token handed to the client.
		token := randomSessionToken()
		passkeySessionStore.put(token, *sessionData)
		c.JSON(http.StatusOK, gin.H{"session": token, "options": opts})
	}
}

// PasskeyLoginFinish validates the assertion and issues a JWT.
func PasskeyLoginFinish(store *internaldb.Store, jwt *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("session")
		sessionData, ok := passkeySessionStore.take(token)
		if !ok {
			response.BadRequest(c, "no login in progress")
			return
		}
		// Parse the credential ID from the request to find the user.
		credID, err := credentialIDFromRequest(c.Request)
		if err != nil {
			response.BadRequest(c, "cannot read credential: "+err.Error())
			return
		}
		pk, err := store.GetPasskeyByCredentialID(c.Request.Context(), credID)
		if err != nil {
			response.Unauthorized(c, "unknown credential")
			return
		}
		user, err := store.GetUserByID(c.Request.Context(), pk.UserID)
		if err != nil {
			response.Unauthorized(c, "user not found")
			return
		}
		w, err := newWebAuthn(c.Request)
		if err != nil {
			response.InternalError(c, "webauthn init failed")
			return
		}
		cred := pk.ToCredential()
		if _, err := w.FinishLogin(&auth.WebAuthnUser{User: user, Credentials: []webauthn.Credential{cred}}, sessionData, c.Request); err != nil {
			response.Unauthorized(c, "login failed: "+err.Error())
			return
		}
		// Persist updated sign count.
		if err := store.UpdatePasskeySignCount(c.Request.Context(), pk.ID, cred.Authenticator.SignCount); err != nil {
			slog.Warn("update passkey sign count", "error", err)
		}
		_ = store.LogAudit(c.Request.Context(), user.ID, "passkey.login", pk.Name)
		tok, err := jwt.GenerateToken(user.ID, user.Username, user.Role, user.ExtraPerm, user.ExtraDBAccess)
		if err != nil {
			response.InternalError(c, "failed to generate token")
			return
		}
		c.JSON(http.StatusOK, tokenResponse{Token: tok, UserID: user.ID, Username: user.Username, Role: user.Role})
	}
}

// PasskeyList returns the authenticated user's passkeys (safe view).
func PasskeyList(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		pks, err := store.ListPasskeys(c.Request.Context(), uid)
		if err != nil {
			response.InternalError(c, "failed to list passkeys")
			return
		}
		out := make([]gin.H, 0, len(pks))
		for _, pk := range pks {
			out = append(out, gin.H{"id": pk.ID, "name": pk.Name, "aaguid": pk.AAGUID, "created_at": pk.CreatedAt})
		}
		response.Success(c, out)
	}
}

// PasskeyDelete removes a passkey.
func PasskeyDelete(store *internaldb.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(string)
		if !ok || uid == "" {
			response.Unauthorized(c, "missing user context")
			return
		}
		if err := store.DeletePasskey(c.Request.Context(), c.Param("id")); err != nil {
			response.NotFound(c, "passkey not found")
			return
		}
		_ = store.LogAudit(c.Request.Context(), uid, "passkey.delete", c.Param("id"))
		c.Status(http.StatusNoContent)
	}
}

// --- helpers ---

func rpIDFromRequest(r *http.Request) string {
	host := r.Host
	if h, _, err := splitHostPort(r.Host); err == nil {
		host = h
	}
	return host
}

func rpOriginFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func randomSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hexEncode(b)
}

func credentialIDFromRequest(r *http.Request) ([]byte, error) {
	// webauthn parses the body; re-read rawId from a shallow copy.
	var body struct {
		RawID string `json:"rawId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	return decodeBase64URL(body.RawID)
}

func splitHostPort(host string) (string, string, error) {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i], host[i+1:], nil
		}
		if host[i] == ']' { // IPv6
			break
		}
	}
	return host, "", nil
}

func decodeBase64URL(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func hexEncode(b []byte) string { return hex.EncodeToString(b) }
