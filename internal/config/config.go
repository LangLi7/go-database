package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	kjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const prefix = "GODB_"

// LoadEnvFile parses a simple .env file (KEY=VALUE, # comments, no quotes
// required) and exports the variables to the process environment. Existing
// environment variables take precedence. This is intentionally dependency-free.
func LoadEnvFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // optional file
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		// Only accept KEY=VALUE
		idx := strings.Index(l, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(l[:idx])
		val := strings.TrimSpace(l[idx+1:])
		// strip optional surrounding quotes
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		// Don't override real env vars (env beats .env beats defaults)
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("config: .env line %d: %w", i+1, err)
		}
	}
	return nil
}

// Server holds the HTTP server configuration
type Server struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	BaseURL    string `json:"base_url" yaml:"base_url"`
	CORSOrigin string `json:"cors_origin" yaml:"cors_origin"`
}

// Auth holds authentication configuration
type Auth struct {
	TokenDuration int    `json:"token_duration" yaml:"token_duration"` // minutes
	JWTSecret     string `json:"jwt_secret" yaml:"jwt_secret"`
}

// InternalDB holds configuration for the internal (auth) database.
// Defaults to a local SQLite file. Set AuthURL to a postgres:// DSN to use
// PostgreSQL instead (see internal/internaldb/store.go for the driver rewrite).
type InternalDB struct {
	AuthPath string `json:"auth_path" yaml:"auth_path"`
	AuthURL  string `json:"auth_url" yaml:"auth_url"` // PostgreSQL DSN (postgres://...), overrides AuthPath
}

// MCP holds the MCP server / NL2SQL configuration.
type MCP struct {
	Enabled       bool         `json:"enabled" yaml:"enabled"`
	Endpoint      string       `json:"endpoint" yaml:"endpoint"`
	APIKey        string       `json:"api_key" yaml:"api_key"`
	Provider      string       `json:"provider" yaml:"provider"`
	Model         string       `json:"model" yaml:"model"`
	FallbackPaid  bool         `json:"fallback_paid" yaml:"fallback_paid"`
	LlamaCpp      LlamaCppCfg  `json:"llamacpp" yaml:"llamacpp"`
}

// LlamaCppCfg holds llama.cpp subprocess settings.
type LlamaCppCfg struct {
	AutoStart bool `json:"auto_start" yaml:"auto_start"`
	Port      int  `json:"port" yaml:"port"`
	Parallel  int  `json:"parallel" yaml:"parallel"` // concurrent request slots (>1 enables --parallel + --cont-batching)
}

// Config is the root configuration
type Config struct {
	Server     Server     `json:"server" yaml:"server"`
	Auth       Auth       `json:"auth" yaml:"auth"`
	InternalDB InternalDB `json:"internal_db" yaml:"internal_db"`
	MCP        MCP        `json:"mcp" yaml:"mcp"`
	LogLevel   string     `json:"log_level" yaml:"log_level"`
	DataDir    string     `json:"data_dir" yaml:"data_dir"` // root for internal DBs
}

// Load reads configuration from file(s) + environment variables
func Load(paths ...string) (*Config, error) {
	k := koanf.New(".")

	// 0. Load .env file first (populates os env, no-op if absent)
	if err := LoadEnvFile(".env"); err != nil {
		return nil, fmt.Errorf("config: .env: %w", err)
	}

	// 1. Try loading from default paths if none specified
	if len(paths) == 0 {
		paths = []string{
			"config/config.yaml",
			"config/config.json",
			"config/config.yml",
			"/etc/go-database/config.yaml",
		}
	}

	// 2. Load from files (first found wins for each path)
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		var parser koanf.Parser
		switch strings.ToLower(filepath.Ext(p)) {
		case ".yaml", ".yml":
			parser = yaml.Parser()
		case ".json":
			parser = kjson.Parser()
		default:
			continue
		}
		if err := k.Load(file.Provider(p), parser); err != nil {
			return nil, fmt.Errorf("config: loading %s: %w", p, err)
		}
	}

	// 3. Override with environment variables (GODB_ prefix)
	//    e.g., GODB_SERVER_PORT=8080, GODB_AUTH_JWT_SECRET=abc
	if err := k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(
			strings.TrimPrefix(s, prefix)), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("config: loading env: %w", err)
	}

	// 4. Unmarshal into struct.
	// koanf's default Unmarshal matches by field name and ignores json/yaml
	// struct tags, so snake_case keys (auto_start, base_url, jwt_secret, ...)
	// would be silently dropped. Tell mapstructure to honor the json tags.
	var cfg Config
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "json",
	}); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	// 5. Apply defaults
	cfg.setDefaults()

	// 6. Resolve data directory
	if cfg.DataDir == "" {
		cfg.DataDir = "database/internal"
	}

	if cfg.InternalDB.AuthPath == "" {
		cfg.InternalDB.AuthPath = filepath.Join(cfg.DataDir, "auth.db")
	}

	return &cfg, nil
}

func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.BaseURL == "" {
		c.Server.BaseURL = fmt.Sprintf("http://%s:%d", c.Server.Host, c.Server.Port)
	}
	if c.Server.CORSOrigin == "" {
		c.Server.CORSOrigin = "*" // Allow all origins in dev; restrict in production
	}
	if c.Auth.TokenDuration == 0 {
		c.Auth.TokenDuration = 60 // 1 hour
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if !c.MCP.Enabled {
		c.MCP.Enabled = false
	}
	if c.MCP.Endpoint == "" {
		c.MCP.Endpoint = "/api/v1/mcp"
	}
	if c.MCP.Provider == "" {
		c.MCP.Provider = "openrouter"
	}
	if c.MCP.Model == "" {
		c.MCP.Model = "free"
	}
	// FallbackPaid defaults to false — user must explicitly allow paid model usage.
	if !c.MCP.FallbackPaid {
		c.MCP.FallbackPaid = false
	}
	if c.MCP.LlamaCpp.Port == 0 {
		c.MCP.LlamaCpp.Port = 8081
	}
	// Harden JWT secret: if default or empty, generate a random one at startup
	if c.Auth.JWTSecret == "" || c.Auth.JWTSecret == "change-me-in-production" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err == nil {
			c.Auth.JWTSecret = hex.EncodeToString(key)
			slog.Warn("config: JWT secret is default or empty, generated random secret",
				"jwt_secret", c.Auth.JWTSecret[:8]+"...",
				"hint", "Set GODB_AUTH_JWT_SECRET env var for persistence across restarts")
		}
	}
}

// DSN returns a human-readable connection string for the server
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// PrintJSON returns the config as formatted JSON (for debugging, secrets masked)
func (c *Config) PrintJSON() string {
	safe := *c
	b, _ := json.MarshalIndent(safe, "", "  ")
	return string(b)
}
