package plugin

import (
	"context"
)

// DBType describes what kind of database this plugin handles
type DBType string

const (
	TypePostgres DBType = "postgres"
	TypeMySQL    DBType = "mysql"
	TypeMariaDB  DBType = "mariadb"
	TypeSQLite   DBType = "sqlite"
	TypeMongoDB  DBType = "mongodb"
	TypeRedis    DBType = "redis"
)

// Config holds connection parameters for any DB type
type Config struct {
	Type     DBType `json:"type" yaml:"type"`
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	Database string `json:"database,omitempty" yaml:"database,omitempty"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"-" yaml:"-"` // never serialized to logs
	FilePath string `json:"filepath,omitempty" yaml:"filepath,omitempty"` // SQLite file path
	SSL      bool   `json:"ssl,omitempty" yaml:"ssl,omitempty"`
	Params   map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}

// ColumnInfo describes a single column in a table
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Primary  bool   `json:"primary"`
	Default  string `json:"default,omitempty"`
}

// TableInfo describes a table and its columns
type TableInfo struct {
	Name       string       `json:"name"`
	RowCount   int64        `json:"row_count"`
	Columns    []ColumnInfo `json:"columns"`
}

// Schema describes the full database schema
type Schema struct {
	Tables []TableInfo `json:"tables"`
}

// Result holds query results
type Result struct {
	Columns []string   `json:"columns"`
	Rows    [][]any    `json:"rows"`
	RowsAffected int64 `json:"rows_affected"`
	Duration int64     `json:"duration_ms"`
}

// DBPlugin is the interface every database plugin must implement
type DBPlugin interface {
	// Type returns the database type identifier
	Type() DBType

	// Connect establishes a connection using the given config
	Connect(ctx context.Context, cfg Config) error

	// Ping checks if the connection is alive
	Ping(ctx context.Context) error

	// Close terminates the connection
	Close() error

	// Query executes a read query and returns results
	Query(ctx context.Context, query string) (*Result, error)

	// Execute runs a write query (INSERT, UPDATE, DELETE)
	Execute(ctx context.Context, query string) (*Result, error)

	// Tables lists all tables/collections
	Tables(ctx context.Context) ([]string, error)

	// Schema returns detailed schema information
	Schema(ctx context.Context) (*Schema, error)

	// Databases lists all databases on the server
	Databases(ctx context.Context) ([]string, error)

	// CreateDatabase creates a new database
	CreateDatabase(ctx context.Context, name string) error

	// DropDatabase drops a database
	DropDatabase(ctx context.Context, name string) error
}

// registry holds all registered plugins
var registry = make(map[DBType]func() DBPlugin)

// Register adds a plugin factory to the registry
func Register(dbType DBType, factory func() DBPlugin) {
	registry[dbType] = factory
}

// New creates a new plugin instance by type
func New(dbType DBType) (DBPlugin, bool) {
	factory, ok := registry[dbType]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// List returns all registered database types
func List() []DBType {
	types := make([]DBType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
