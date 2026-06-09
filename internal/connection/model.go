package connection

import (
	"time"

	"go-database/internal/plugin"
)

// State represents a connection's current status
type State string

const (
	StateConnected    State = "connected"
	StateDisconnected State = "disconnected"
	StateError        State = "error"
	StateConnecting   State = "connecting"
)

// Connection describes a registered database connection
type Connection struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      plugin.DBType     `json:"type"`
	Source    string            `json:"source"` // "external" | "internal" | "docker" | "file"
	Config    plugin.Config     `json:"config"`
	State     State             `json:"state"`
	Latency   time.Duration     `json:"latency_ms"`
	Error     string            `json:"error,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// Summary is a lightweight view of a connection
type Summary struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Type    plugin.DBType `json:"type"`
	Source  string        `json:"source"`
	State   State         `json:"state"`
	Latency time.Duration `json:"latency_ms"`
	Tags    []string      `json:"tags,omitempty"`
}
