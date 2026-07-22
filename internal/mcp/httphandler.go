package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	gmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// jsonError writes a structured JSON error response.
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + strings.ReplaceAll(msg, `"`, `\"`) + `"}`))
}

// HTTPHandlerWithScope returns an http.Handler that scopes the DB gate to the
// caller's db_access before each tool call. scope extracts (dbAccess, isAdmin)
// from the request (e.g. from a gin context propagated via header). When scope
// is nil the global gate is used unchanged.
func HTTPHandlerWithScope(requireAPIKey func(r *http.Request) bool, scope func(r *http.Request) (dbAccess []string, isAdmin bool)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if scope != nil {
			dbAccess, isAdmin := scope(r)
			prev := setRequestGate(ScopedGate(dbAccess, isAdmin))
			defer setRequestGate(prev)
		}
		HTTPHandler(requireAPIKey).ServeHTTP(w, r)
	})
}

// scopable is implemented by gates that support per-caller scoping (GuardGate).
type scopable interface {
	WithScope(dbAccess []string, isAdmin bool) DBGate
}

// ScopedGate returns connectionManager scoped to dbAccess/isAdmin, or the
// unscoped gate if the underlying gate does not support scoping.
func ScopedGate(dbAccess []string, isAdmin bool) DBGate {
	if s, ok := connectionManager.(scopable); ok {
		return s.WithScope(dbAccess, isAdmin)
	}
	return connectionManager
}

// SetScopedGate swaps in a per-request scoped gate and returns the previous one.
func SetScopedGate(g DBGate) DBGate { return setRequestGate(g) }
// Each request creates a fresh in-memory session, calls the tool, and returns JSON.
// ponytail: one-shot session per request, no streaming; add persistent session when throughput matters.
func HTTPHandler(requireAPIKey func(r *http.Request) bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		start := time.Now()

		if r.Method != "POST" {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if requireAPIKey != nil && !requireAPIKey(r) {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Tool string         `json:"tool"`
			Args map[string]any `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Tool == "" {
			jsonError(w, "tool name is required", http.StatusBadRequest)
			return
		}

		if connectionManager == nil {
			jsonError(w, "server not initialized", http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		clientT, serverT := gmcp.NewInMemoryTransports()

		srv := gmcp.NewServer(&gmcp.Implementation{Name: "go-database-http", Version: "0.1.0"}, nil)
		RegisterDBTools(srv, slog.Default())
		RegisterNL2SQLTool(srv, slog.Default())

		serverSession, err := srv.Connect(ctx, serverT, nil)
		if err != nil {
			jsonError(w, "session: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer serverSession.Close()

		client := gmcp.NewClient(&gmcp.Implementation{Name: "http-client"}, nil)
		clientSession, err := client.Connect(ctx, clientT, nil)
		if err != nil {
			jsonError(w, "client: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer clientSession.Close()

		res, err := clientSession.CallTool(ctx, &gmcp.CallToolParams{
			Name:      req.Tool,
			Arguments: req.Args,
		})
		if err != nil {
			jsonError(w, "tool: "+err.Error(), http.StatusInternalServerError)
			return
		}
		slog.Info("mcp tool call",
			"tool", req.Tool,
			"latency_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
		_ = json.NewEncoder(w).Encode(res)
	})
}

// APIKeyMiddleware returns a requireAPIKey func for HTTPHandler.
func APIKeyMiddleware(expectedKey string) func(r *http.Request) bool {
	if expectedKey == "" {
		return nil // no auth required for empty key
	}
	return func(r *http.Request) bool {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") && strings.TrimPrefix(auth, "Bearer ") == expectedKey {
			return true
		}
		return r.Header.Get("X-API-Key") == expectedKey
	}
}

// ValidateMCPConfig returns an error if the MCP config has invalid values.
func ValidateMCPConfig(provider, model, apiKey string) error {
	switch provider {
	case "openrouter", "ollama", "lmstudio", "llamacpp", "":
	default:
		return errInvalidProvider
	}
	if provider == "openrouter" && model == "" {
		return errEmptyModel
	}
	if provider == "openrouter" && apiKey == "" {
		return errMissingAPIKey
	}
	return nil
}

var (
	errInvalidProvider = &configError{"invalid mcp provider; use openrouter, ollama, lmstudio, or llamacpp"}
	errEmptyModel      = &configError{"model is required for openrouter provider"}
	errMissingAPIKey   = &configError{"api_key is required for openrouter provider"}
)

type configError struct{ msg string }

func (e *configError) Error() string { return e.msg }

// NewRequestID returns a middleware that injects X-Request-Id into the context.
func NewRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = r.Header.Get("X-Amzn-Trace-Id")
		}
		if id != "" {
			w.Header().Set("X-Request-Id", id)
		}
		next.ServeHTTP(w, r)
	})
}

// Ensure bytes is used (referenced by NewRequestID despite not being used directly in this file)
