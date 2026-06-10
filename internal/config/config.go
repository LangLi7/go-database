package config

import (
	"encoding/json"
	"fmt"
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

// Server holds the HTTP server configuration
type Server struct {
	Host        string `json:"host" yaml:"host"`
	Port        int    `json:"port" yaml:"port"`
	BaseURL     string `json:"base_url" yaml:"base_url"`
	CORSOrigin  string `json:"cors_origin" yaml:"cors_origin"`
}

// Auth holds authentication configuration
type Auth struct {
	TokenDuration int    `json:"token_duration" yaml:"token_duration"` // minutes
	JWTSecret     string `json:"jwt_secret" yaml:"jwt_secret"`
}

// InternalDB holds paths for internal SQLite databases
type InternalDB struct {
	AuthPath    string `json:"auth_path" yaml:"auth_path"`
	JobsPath    string `json:"jobs_path" yaml:"jobs_path"`
	MetricsPath string `json:"metrics_path" yaml:"metrics_path"`
}

// Config is the root configuration
type Config struct {
	Server     Server     `json:"server" yaml:"server"`
	Auth       Auth       `json:"auth" yaml:"auth"`
	InternalDB InternalDB `json:"internal_db" yaml:"internal_db"`
	LogLevel   string     `json:"log_level" yaml:"log_level"`
	DataDir    string     `json:"data_dir" yaml:"data_dir"` // root for internal DBs
}

// Load reads configuration from file(s) + environment variables
func Load(paths ...string) (*Config, error) {
	k := koanf.New(".")

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

	// 4. Unmarshal into struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
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
	if cfg.InternalDB.JobsPath == "" {
		cfg.InternalDB.JobsPath = filepath.Join(cfg.DataDir, "jobs.db")
	}
	if cfg.InternalDB.MetricsPath == "" {
		cfg.InternalDB.MetricsPath = filepath.Join(cfg.DataDir, "metrics.db")
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
	if c.Auth.TokenDuration == 0 {
		c.Auth.TokenDuration = 60 // 1 hour
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
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
