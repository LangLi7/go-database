package internaldb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"

	"go-database/internal/auth"
)

// Store wraps the internal SQLite database for auth, config, and logging
type Store struct {
	db *sql.DB
}

// Open creates or opens the internal database and runs migrations
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("internaldb: open: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: ping: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: migrate: %w", err)
	}

	if err := s.seed(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: seed: %w", err)
	}

	return s, nil
}

// Close shuts down the internal database
func (s *Store) Close() error {
	return s.db.Close()
}

// --- Auth: Users ---

func (s *Store) SaveUser(ctx context.Context, u auth.User) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO users (id, username, password_hash, role, extra_perm, extra_db_access, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.Role,
		joinSlice(u.ExtraPerm), joinSlice(u.ExtraDBAccess), time.Now().Unix())
	return err
}

func (s *Store) GetUser(ctx context.Context, username string) (*auth.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,'')
		 FROM users WHERE username = ?`, username)

	var u auth.User
	var extraPerm, extraDBAccess string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess); err != nil {
		return nil, fmt.Errorf("internaldb: user not found: %w", err)
	}

	u.ExtraPerm = splitSlice(extraPerm)
	u.ExtraDBAccess = splitSlice(extraDBAccess)
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,'')
		 FROM users WHERE id = ?`, id)
	var u auth.User
	var extraPerm, extraDBAccess string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess); err != nil {
		return nil, fmt.Errorf("internaldb: user not found: %w", err)
	}
	u.ExtraPerm = splitSlice(extraPerm)
	u.ExtraDBAccess = splitSlice(extraDBAccess)
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,'')
		 FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []auth.User
	for rows.Next() {
		var u auth.User
		var extraPerm, extraDBAccess string
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess); err != nil {
			continue
		}
		u.ExtraPerm = splitSlice(extraPerm)
		u.ExtraDBAccess = splitSlice(extraDBAccess)
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) SetUserDBAccess(ctx context.Context, id string, dbAccess []string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE users SET extra_db_access = ? WHERE id = ?`,
		joinSlice(dbAccess), id)
	return err
}

// --- Auth: Roles ---

func (s *Store) SaveRole(ctx context.Context, r auth.Role) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO roles (id, name, permissions, db_access)
		 VALUES (?, ?, ?, ?)`,
		r.ID, r.Name, joinSlice(r.Permissions), joinSlice(r.DBAccess))
	return err
}

func (s *Store) GetRole(ctx context.Context, id string) (*auth.Role, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(permissions,''), COALESCE(db_access,'') FROM roles WHERE id = ?`, id)
	var r auth.Role
	var perms, dbAccess string
	if err := row.Scan(&r.ID, &r.Name, &perms, &dbAccess); err != nil {
		return nil, fmt.Errorf("internaldb: role not found: %w", err)
	}
	r.Permissions = splitSlice(perms)
	r.DBAccess = splitSlice(dbAccess)
	return &r, nil
}

func (s *Store) ListRoles(ctx context.Context) ([]auth.Role, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, COALESCE(permissions,''), COALESCE(db_access,'') FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []auth.Role
	for rows.Next() {
		var r auth.Role
		var perms, dbAccess string
		if err := rows.Scan(&r.ID, &r.Name, &perms, &dbAccess); err != nil {
			continue
		}
		r.Permissions = splitSlice(perms)
		r.DBAccess = splitSlice(dbAccess)
		roles = append(roles, r)
	}
	return roles, nil
}

func (s *Store) DeleteRole(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM roles WHERE id = ?`, id)
	return err
}

// --- Auth: API Keys (KeyStore interface implementation) ---

func (s *Store) SaveKey(k auth.APIKey) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO api_keys (prefix, hash, name, permissions, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		k.Prefix, k.Hash, k.Name, joinSlice(k.Permissions), k.CreatedAt)
	return err
}

func (s *Store) GetKey(prefix string) (*auth.APIKey, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT prefix, hash, name, COALESCE(permissions,''), COALESCE(created_at,'') FROM api_keys WHERE prefix = ?`, prefix)
	var k auth.APIKey
	var perms, createdAt string
	if err := row.Scan(&k.Prefix, &k.Hash, &k.Name, &perms, &createdAt); err != nil {
		return nil, fmt.Errorf("internaldb: key not found: %w", err)
	}
	k.Permissions = splitSlice(perms)
	k.CreatedAt = createdAt
	return &k, nil
}

func (s *Store) ListKeys() ([]auth.APIKey, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT prefix, name, COALESCE(permissions,''), COALESCE(created_at,'') FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []auth.APIKey
	for rows.Next() {
		var k auth.APIKey
		var perms, createdAt string
		if err := rows.Scan(&k.Prefix, &k.Name, &perms, &createdAt); err != nil {
			continue
		}
		k.Permissions = splitSlice(perms)
		k.CreatedAt = createdAt
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) DeleteKey(prefix string) error {
	ctx := context.Background()
	_, err := s.db.ExecContext(ctx, `DELETE FROM api_keys WHERE prefix = ?`, prefix)
	return err
}

// --- Design Config (adaptives Design wie Netflix) ---

// DesignConfig holds theme/layout settings (stored as JSON)
type DesignConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Config    string `json:"config"` // JSON blob
	Active    bool   `json:"active"`
	CreatedAt string `json:"created_at"`
}

func (s *Store) GetActiveDesign(ctx context.Context) (*DesignConfig, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, config, created_at FROM design_config WHERE active = 1 LIMIT 1`)
	var dc DesignConfig
	dc.Active = true
	if err := row.Scan(&dc.ID, &dc.Name, &dc.Config, &dc.CreatedAt); err != nil {
		return nil, fmt.Errorf("internaldb: no active design")
	}
	return &dc, nil
}

func (s *Store) SaveDesign(ctx context.Context, dc DesignConfig) error {
	if dc.Active {
		// Deactivate all others
		_, _ = s.db.ExecContext(ctx, `UPDATE design_config SET active = 0`)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO design_config (id, name, config, active, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		dc.ID, dc.Name, dc.Config, boolToInt(dc.Active), dc.CreatedAt)
	return err
}

func (s *Store) ListDesigns(ctx context.Context) ([]DesignConfig, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, config, active, created_at FROM design_config ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designs []DesignConfig
	for rows.Next() {
		var dc DesignConfig
		var active int
		if err := rows.Scan(&dc.ID, &dc.Name, &dc.Config, &active, &dc.CreatedAt); err != nil {
			continue
		}
		dc.Active = active == 1
		designs = append(designs, dc)
	}
	return designs, nil
}

// --- Audit Log ---

func (s *Store) LogAudit(ctx context.Context, userID, action, details string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO audit_log (user_id, action, details, created_at)
		 VALUES (?, ?, ?, ?)`,
		userID, action, details, time.Now().Unix())
	return err
}

func (s *Store) ListAuditLog(ctx context.Context, limit int) ([]map[string]string, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT user_id, action, details, created_at FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]string
	for rows.Next() {
		var userID, action, details string
		var createdAt int64
		if err := rows.Scan(&userID, &action, &details, &createdAt); err != nil {
			continue
		}
		logs = append(logs, map[string]string{
			"user_id":    userID,
			"action":     action,
			"details":    details,
			"created_at": time.Unix(createdAt, 0).Format(time.RFC3339),
		})
	}
	return logs, nil
}

// --- Migrations ---

func (s *Store) migrate(ctx context.Context) error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'readonly',
			extra_perm TEXT DEFAULT '',
			extra_db_access TEXT DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			permissions TEXT DEFAULT '',
			db_access TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			prefix TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			name TEXT NOT NULL,
			permissions TEXT DEFAULT '',
			last_used_at TEXT DEFAULT '',
			expires_at TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS design_config (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			config TEXT NOT NULL DEFAULT '{}',
			active INTEGER DEFAULT 0,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			action TEXT NOT NULL,
			details TEXT DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
	}

	for _, schema := range schemas {
		if _, err := s.db.ExecContext(ctx, schema); err != nil {
			return fmt.Errorf("internaldb: migrate: %w", err)
		}
	}

	slog.Info("internal database migrated")
	return nil
}

// seed populates default roles and admin user if empty
func (s *Store) seed(ctx context.Context) error {
	// Seed default roles
	count := 0
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM roles`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		for _, role := range auth.DefaultRoles() {
			if err := s.SaveRole(ctx, role); err != nil {
				return fmt.Errorf("internaldb: seed role: %w", err)
			}
		}
		slog.Info("seeded default roles")
	}

	// Seed admin user if no users exist
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		hash, err := auth.HashPassword("admin")
		if err != nil {
			return err
		}
		admin := auth.User{
			ID:       "admin-001",
			Username: "admin",
			PasswordHash: hash,
			Role:     "admin",
		}
		if err := s.SaveUser(ctx, admin); err != nil {
			return fmt.Errorf("internaldb: seed admin: %w", err)
		}
		slog.Info("seeded default admin user (password: admin)")
	}

	return nil
}

// --- Helpers ---

func joinSlice(s []string) string {
	if len(s) == 0 {
		return ""
	}
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}

func splitSlice(s string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	current := ""
	for _, ch := range s {
		if ch == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
