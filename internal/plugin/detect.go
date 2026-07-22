package plugin

import (
	"net/url"
	"strings"
)

// DetectType heuristically determines the database type from connection
// parameters. It is used when a request specifies TypeAuto.
//
// Precedence:
//  1. Explicit DSN in Params["dsn"] or Host that starts with a known scheme
//     (e.g. "postgres://...", "mongodb://...").
//  2. Well-known default port (e.g. 5432 -> postgres, 1433 -> mssql).
//  3. SQLite when a FilePath is set.
//
// Returns TypeAuto + false if it cannot be determined.
func DetectType(cfg Config) (DBType, bool) {
	// 1. DSN / scheme prefix
	dsn := cfg.Params["dsn"]
	candidates := []string{dsn, cfg.Host}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		lower := strings.ToLower(c)
		for _, p := range dsnPrefixes {
			if strings.HasPrefix(lower, p.prefix) {
				return p.typ, true
			}
		}
	}

	// 2. Well-known port
	if cfg.Port != 0 {
		if t, ok := wellKnownPorts[cfg.Port]; ok {
			return t, true
		}
	}

	// 3. SQLite file
	if cfg.FilePath != "" {
		if strings.HasSuffix(strings.ToLower(cfg.FilePath), ".graph.json") {
			return TypeGraph, true
		}
		return TypeSQLite, true
	}

	// 4. Host that parses as a URL with a scheme
	if u, err := url.Parse(cfg.Host); err == nil && u.Scheme != "" {
		for _, p := range dsnPrefixes {
			if strings.HasPrefix(strings.ToLower(cfg.Host), p.prefix) {
				return p.typ, true
			}
		}
	}

	return TypeAuto, false
}

// IsSupported reports whether a type has a registered plugin factory.
func IsSupported(t DBType) bool {
	regMu.RLock()
	defer regMu.RUnlock()
	_, ok := registry[t]
	return ok
}
