package executor

import (
	"context"
	"fmt"

	"go-database/internal/connection"
	"go-database/internal/mcp"
	"go-database/internal/plugin"
)

// GuardGate wraps a connection manager so that Execute/Query requests from
// non-interactive callers (Agent, MCP) pass through the risk guard. High-risk
// writes (e.g. DELETE without WHERE) are blocked because these callers cannot
// show a confirm dialog — they run with ConfirmHigh=false by design.
//
// ponytail: Agent/MCP skip the human UI, so NeedsConfirm is treated as a hard
// denial. If you later add a confirm callback for MCP clients, pass it here.
type Manager interface {
	List() []connection.Summary
	ListVisible(userID string, dbAccess []string, isAdmin bool) []connection.Summary
	GetConnection(id string) (*connection.Connection, error)
	Query(ctx context.Context, id, sql string) (*plugin.Result, error)
	Execute(ctx context.Context, id, sql string) (*plugin.Result, error)
	Tables(ctx context.Context, id string) ([]string, error)
	Schema(ctx context.Context, id string) (*plugin.Schema, error)
	Databases(ctx context.Context, id string) ([]string, error)
}

type GuardGate struct {
	mgr     Manager
	ex      *Executor
	dbAccess []string // scoped DB IDs this caller may touch (empty = none)
	isAdmin bool
}

// NewGuardGate returns a Gate-compatible wrapper around mgr that enforces the
// same blast-radius / risk checks the REST executor uses.
func NewGuardGate(mgr Manager) *GuardGate {
	return &GuardGate{mgr: mgr, ex: New(mgr)}
}

// WithScope binds a per-caller DB-access scope (from JWT extra_db_access or
// API-key DBAccess). A non-admin caller may only List/Query/Execute
// connections whose ID is in dbAccess. Admins are unrestricted.
// ponytail: global scope = no per-caller isolation; call WithScope per request.
func (g *GuardGate) WithScope(dbAccess []string, isAdmin bool) mcp.DBGate {
	return &GuardGate{mgr: g.mgr, ex: g.ex, dbAccess: dbAccess, isAdmin: isAdmin}
}

// allowed reports whether id is within the caller's scope.
func (g *GuardGate) allowed(id string) bool {
	if g.isAdmin {
		return true
	}
	for _, a := range g.dbAccess {
		if a == id {
			return true
		}
	}
	return false
}

func (g *GuardGate) List() []connection.Summary {
	if g.isAdmin {
		return g.mgr.List()
	}
	return g.mgr.ListVisible("", g.dbAccess, false)
}

// IsAllowed reports whether id is within the caller's scope.
func (g *GuardGate) IsAllowed(id string) bool { return g.allowed(id) }

// ListVisible returns the scoped connections (delegates to the manager's
// visibility filter using this gate's dbAccess).
func (g *GuardGate) ListVisible(userID string, dbAccess []string, isAdmin bool) []connection.Summary {
	if g.isAdmin {
		return g.mgr.List()
	}
	return g.mgr.ListVisible(userID, g.dbAccess, false)
}

func (g *GuardGate) GetConnection(id string) (*connection.Connection, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	return g.mgr.GetConnection(id)
}

func (g *GuardGate) Query(ctx context.Context, id, sql string) (*plugin.Result, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	return g.mgr.Query(ctx, id, sql)
}

// Execute routes through the risk guard. High-risk or large-blast-radius
// operations are rejected (NeedsConfirm with no caller confirmation = deny).
func (g *GuardGate) Execute(ctx context.Context, id, sql string) (*plugin.Result, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	res := g.ex.Execute(ctx, ExecutionRequest{
		ConnectionID: id,
		SQL:          sql,
		ConfirmHigh:  false, // non-interactive: never auto-confirm
	})
	if !res.Success {
		if res.NeedsConfirm {
			return nil, fmt.Errorf("blocked by guard: %s (needs human confirmation)", res.RiskInfo)
		}
		return nil, fmt.Errorf("%s", res.Error)
	}
	return res.Result, nil
}

func (g *GuardGate) Tables(ctx context.Context, id string) ([]string, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	return g.mgr.Tables(ctx, id)
}

func (g *GuardGate) Schema(ctx context.Context, id string) (*plugin.Schema, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	return g.mgr.Schema(ctx, id)
}

func (g *GuardGate) Databases(ctx context.Context, id string) ([]string, error) {
	if !g.allowed(id) {
		return nil, fmt.Errorf("access denied to connection %s", id)
	}
	return g.mgr.Databases(ctx, id)
}
