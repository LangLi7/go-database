package connection

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go-database/internal/plugin"
)

// Manager handles all database connections
type Manager struct {
	mu    sync.RWMutex
	conns map[string]*managedConn
}

type managedConn struct {
	Connection
	plugin plugin.DBPlugin
	cancel context.CancelFunc
}

// NewManager creates an empty connection manager
func NewManager() *Manager {
	return &Manager{
		conns: make(map[string]*managedConn),
	}
}

// Add creates and registers a new connection
func (m *Manager) Add(ctx context.Context, name string, typ plugin.DBType, source string, cfg plugin.Config, tags []string, ownerID string) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := plugin.New(typ)
	if !ok {
		return nil, fmt.Errorf("connection: unsupported type %q", typ)
	}

	id := generateID()
	now := time.Now()
	mc := &managedConn{
		Connection: Connection{
			ID:        id,
			Name:      name,
			Type:      typ,
			Source:    source,
			Config:    cfg,
			State:     StateConnecting,
			Tags:      tags,
			OwnerID:   ownerID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		plugin: p,
	}

	// Connect with timeout
	connCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	if err := p.Connect(connCtx, cfg); err != nil {
		mc.State = StateError
		mc.Error = err.Error()
		mc.UpdatedAt = time.Now()
		m.conns[id] = mc
		slog.Error("connection failed", "id", id, "type", typ, "error", err)
		return &mc.Connection, fmt.Errorf("connection: %q (%s): %w", name, typ, err)
	}

	mc.State = StateConnected
	mc.Latency = time.Since(start)
	mc.UpdatedAt = time.Now()
	m.conns[id] = mc
	slog.Info("connection established", "id", id, "type", typ, "latency", mc.Latency)
	return &mc.Connection, nil
}

// GetConnection returns connection info by ID (public wrapper)
func (m *Manager) GetConnection(id string) (*Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	mc, ok := m.conns[id]
	if !ok {
		return nil, fmt.Errorf("connection: %q not found", id)
	}
	return &mc.Connection, nil
}

// Get returns a single connection by ID
func (m *Manager) Get(id string) (*managedConn, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mc, ok := m.conns[id]
	if !ok {
		return nil, fmt.Errorf("connection: %q not found", id)
	}
	return mc, nil
}

// List returns summaries of all connections (admin / system overview).
func (m *Manager) List() []Summary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]Summary, 0, len(m.conns))
	for _, mc := range m.conns {
		summaries = append(summaries, Summary{
			ID:      mc.ID,
			Name:    mc.Name,
			Type:    mc.Type,
			Source:  mc.Source,
			State:   mc.State,
			Latency: mc.Latency,
			Tags:    mc.Tags,
		})
	}
	return summaries
}

// ListVisible returns only the connections a user may see: their own
// (OwnerID == userID) plus any explicitly shared via dbAccess. Admins see all.
// ponytail: single map scan, O(n); fine for the connection count we handle.
func (m *Manager) ListVisible(userID string, dbAccess []string, isAdmin bool) []Summary {
	if isAdmin {
		return m.List()
	}
	allowed := make(map[string]bool, len(dbAccess))
	for _, id := range dbAccess {
		allowed[id] = true
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]Summary, 0, len(m.conns))
	for _, mc := range m.conns {
		if mc.OwnerID == userID || allowed[mc.ID] {
			summaries = append(summaries, Summary{
				ID:      mc.ID,
				Name:    mc.Name,
				Type:    mc.Type,
				Source:  mc.Source,
				State:   mc.State,
				Latency: mc.Latency,
				Tags:    mc.Tags,
			})
		}
	}
	return summaries
}

// Remove closes and deletes a connection
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mc, ok := m.conns[id]
	if !ok {
		return fmt.Errorf("connection: %q not found", id)
	}

	if err := mc.plugin.Close(); err != nil {
		slog.Error("connection close error", "id", id, "error", err)
	}
	if mc.cancel != nil {
		mc.cancel()
	}

	delete(m.conns, id)
	slog.Info("connection removed", "id", id)
	return nil
}

// Ping checks if a connection is alive and updates latency
func (m *Manager) Ping(ctx context.Context, id string) (time.Duration, error) {
	mc, err := m.Get(id)
	if err != nil {
		return 0, err
	}

	if mc.State == StateError || mc.plugin == nil {
		return 0, fmt.Errorf("connection: %q is not connected (%s)", id, mc.Error)
	}

	start := time.Now()
	if err := mc.plugin.Ping(ctx); err != nil {
		m.mu.Lock()
		mc.State = StateError
		mc.Error = err.Error()
		mc.UpdatedAt = time.Now()
		m.mu.Unlock()
		return 0, fmt.Errorf("connection: ping %q: %w", id, err)
	}

	latency := time.Since(start)
	m.mu.Lock()
	mc.State = StateConnected
	mc.Latency = latency
	mc.Error = ""
	mc.UpdatedAt = time.Now()
	m.mu.Unlock()

	return latency, nil
}

// getActive returns a connection that is in a usable state
func (m *Manager) getActive(id string) (*managedConn, error) {
	mc, err := m.Get(id)
	if err != nil {
		return nil, err
	}
	if mc.State == StateError {
		return nil, fmt.Errorf("connection: %q is in error state: %s", id, mc.Error)
	}
	if mc.plugin == nil {
		return nil, fmt.Errorf("connection: %q has no plugin", id)
	}
	return mc, nil
}

// Query executes a read query on a connection
func (m *Manager) Query(ctx context.Context, id string, query string) (*plugin.Result, error) {
	mc, err := m.getActive(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin.Query(ctx, query)
}

// Execute runs a write query on a connection
func (m *Manager) Execute(ctx context.Context, id string, query string) (*plugin.Result, error) {
	mc, err := m.getActive(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin.Execute(ctx, query)
}

// Tables lists tables for a connection
func (m *Manager) Tables(ctx context.Context, id string) ([]string, error) {
	mc, err := m.getActive(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin.Tables(ctx)
}

// Schema returns full schema for a connection
func (m *Manager) Schema(ctx context.Context, id string) (*plugin.Schema, error) {
	mc, err := m.getActive(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin.Schema(ctx)
}

// Databases lists databases for a connection
func (m *Manager) Databases(ctx context.Context, id string) ([]string, error) {
	mc, err := m.getActive(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin.Databases(ctx)
}

// CreateDatabase creates a new database
func (m *Manager) CreateDatabase(ctx context.Context, id string, name string) error {
	mc, err := m.getActive(id)
	if err != nil {
		return err
	}
	return mc.plugin.CreateDatabase(ctx, name)
}

// DropDatabase drops a database
func (m *Manager) DropDatabase(ctx context.Context, id string, name string) error {
	mc, err := m.getActive(id)
	if err != nil {
		return err
	}
	return mc.plugin.DropDatabase(ctx, name)
}

// Plugin returns the raw plugin instance (for transfer engine etc.)
func (m *Manager) Plugin(ctx context.Context, id string) (plugin.DBPlugin, error) {
	mc, err := m.Get(id)
	if err != nil {
		return nil, err
	}
	return mc.plugin, nil
}

// StartHealthChecker runs periodic pings on all connections
func (m *Manager) StartHealthChecker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkAll(ctx)
			}
		}
	}()
	slog.Info("health checker started", "interval", interval)
}

func (m *Manager) checkAll(ctx context.Context) {
	m.mu.RLock()
	ids := make([]string, 0, len(m.conns))
	for id := range m.conns {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := m.Ping(pingCtx, id)
		cancel()

		m.mu.Lock()
		mc, ok := m.conns[id]
		if ok {
			if err != nil {
				mc.State = StateError
				mc.Connection.Latency = 0
				slog.Warn("health check failed", "id", id, "error", err)
			} else {
				mc.State = StateConnected
			}
		}
		m.mu.Unlock()
	}
}

// generateID creates a short unique ID
func generateID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("conn-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
