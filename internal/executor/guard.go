package executor

import (
	"context"
	"fmt"

	"go-database/internal/connection"
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
	GetConnection(id string) (*connection.Connection, error)
	Query(ctx context.Context, id, sql string) (*plugin.Result, error)
	Execute(ctx context.Context, id, sql string) (*plugin.Result, error)
	Tables(ctx context.Context, id string) ([]string, error)
	Schema(ctx context.Context, id string) (*plugin.Schema, error)
	Databases(ctx context.Context, id string) ([]string, error)
}

type GuardGate struct {
	mgr Manager
	ex  *Executor
}

// NewGuardGate returns a Gate-compatible wrapper around mgr that enforces the
// same blast-radius / risk checks the REST executor uses.
func NewGuardGate(mgr Manager) *GuardGate {
	return &GuardGate{mgr: mgr, ex: New(mgr)}
}

func (g *GuardGate) List() []connection.Summary { return g.mgr.List() }

func (g *GuardGate) GetConnection(id string) (*connection.Connection, error) {
	return g.mgr.GetConnection(id)
}

func (g *GuardGate) Query(ctx context.Context, id, sql string) (*plugin.Result, error) {
	return g.mgr.Query(ctx, id, sql)
}

// Execute routes through the risk guard. High-risk or large-blast-radius
// operations are rejected (NeedsConfirm with no caller confirmation = deny).
func (g *GuardGate) Execute(ctx context.Context, id, sql string) (*plugin.Result, error) {
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
	return g.mgr.Tables(ctx, id)
}

func (g *GuardGate) Schema(ctx context.Context, id string) (*plugin.Schema, error) {
	return g.mgr.Schema(ctx, id)
}

func (g *GuardGate) Databases(ctx context.Context, id string) ([]string, error) {
	return g.mgr.Databases(ctx, id)
}
