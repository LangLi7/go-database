package internaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"go-database/internal/auth"
)

// Store wraps the internal database for auth, config, and logging.
// Supports both SQLite and PostgreSQL backends.
type Store struct {
	db     *sql.DB
	driver string // "sqlite" or "postgres"
}

// Open creates or opens the internal database and runs migrations.
// dsn can be a file path (SQLite) or a postgres:// URL (PostgreSQL).
func Open(ctx context.Context, dsn string) (*Store, error) {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return openPostgres(ctx, dsn)
	}
	return openSQLite(ctx, dsn)
}

func openSQLite(ctx context.Context, path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("internaldb: open: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: ping: %w", err)
	}

	s := &Store{db: db, driver: "sqlite"}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: migrate: %w", err)
	}

	if err := s.seed(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: seed: %w", err)
	}

	return s, nil
}

func openPostgres(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("internaldb: open pg: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: ping pg: %w", err)
	}

	s := &Store{db: db, driver: "postgres"}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: migrate pg: %w", err)
	}

	if err := s.seed(ctx); err != nil {
		return nil, fmt.Errorf("internaldb: seed pg: %w", err)
	}

	return s, nil
}

// sql converts SQLite SQL to PostgreSQL syntax when using PG driver.
func (s *Store) sql(query string) string {
	if s.driver != "postgres" {
		return query
	}

	// Convert ? placeholders to $1, $2, ...
	buf := &strings.Builder{}
	count := 0
	for _, c := range query {
		if c == '?' {
			count++
			buf.WriteString(fmt.Sprintf("$%d", count))
		} else {
			buf.WriteRune(c)
		}
	}
	result := buf.String()

	// INSERT OR REPLACE → INSERT ... ON CONFLICT DO UPDATE SET
	if strings.HasPrefix(result, "INSERT OR REPLACE") {
		result = convertInsertOrReplace(result)
	}

	// INTEGER PRIMARY KEY AUTOINCREMENT → SERIAL PRIMARY KEY
	result = strings.ReplaceAll(result, "INTEGER PRIMARY KEY AUTOINCREMENT", "SERIAL PRIMARY KEY")

	// datetime('now') → NOW()
	result = strings.ReplaceAll(result, "datetime('now')", "NOW()")

	// active = 1 → active = TRUE
	result = strings.ReplaceAll(result, "active = 1", "active = TRUE")

	return result
}

// primaryKeys maps table names to their primary key column for PG upsert.
var primaryKeys = map[string]string{
	"users":         "id",
	"roles":         "id",
	"api_keys":      "prefix",
	"design_config": "id",
	"audit_log":     "id",
}

func convertInsertOrReplace(sql string) string {
	// Extract table name: INSERT OR REPLACE INTO <table> (...)
	rest := strings.TrimPrefix(sql, "INSERT OR REPLACE INTO ")
	spaceIdx := strings.Index(rest, " ")
	if spaceIdx < 0 {
		return sql
	}
	table := rest[:spaceIdx]

	// Extract column list between first (...)
	parenStart := strings.Index(rest, "(")
	if parenStart < 0 {
		return sql
	}
	parenEnd := strings.Index(rest[parenStart:], ")")
	if parenEnd < 0 {
		return sql
	}
	parenEnd += parenStart
	columnsStr := rest[parenStart+1 : parenEnd]

	// Build SET clause for ON CONFLICT
	cols := strings.Split(columnsStr, ",")
	setClauses := make([]string, 0, len(cols))
	for _, col := range cols {
		c := strings.TrimSpace(col)
		if c == "" {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = EXCLUDED.%s", c, c))
	}

	// Remove OR REPLACE and append ON CONFLICT
	result := strings.Replace(sql, "INSERT OR REPLACE", "INSERT", 1)
	pk := primaryKeys[table]
	if pk == "" {
		pk = "id"
	}
	result += fmt.Sprintf(" ON CONFLICT (%s) DO UPDATE SET %s", pk, strings.Join(setClauses, ", "))
	return result
}

// Close shuts down the internal database
func (s *Store) Close() error {
	return s.db.Close()
}

// --- Auth: Users ---

// exec wraps ExecContext with SQL conversion for multi-driver support.
func (s *Store) exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, s.sql(query), args...)
}

// query wraps QueryContext with SQL conversion for multi-driver support.
func (s *Store) query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, s.sql(query), args...)
}

// queryRow wraps QueryRowContext with SQL conversion for multi-driver support.
func (s *Store) queryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, s.sql(query), args...)
}

func (s *Store) SaveUser(ctx context.Context, u auth.User) error {
	_, err := s.exec(ctx,
		`INSERT OR REPLACE INTO users (id, username, password_hash, role, extra_perm, extra_db_access, email, public_key, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Username, u.PasswordHash, u.Role,
		joinSlice(u.ExtraPerm), joinSlice(u.ExtraDBAccess), u.Email, u.PublicKey, time.Now().Unix())
	return err
}

func (s *Store) GetUser(ctx context.Context, username string) (*auth.User, error) {
	row := s.queryRow(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,''), COALESCE(email,''), COALESCE(public_key,'')
		 FROM users WHERE username = ?`, username)

	var u auth.User
	var extraPerm, extraDBAccess, email, pubKey string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess, &email, &pubKey); err != nil {
		return nil, fmt.Errorf("internaldb: user not found: %w", err)
	}

	u.ExtraPerm = splitSlice(extraPerm)
	u.ExtraDBAccess = splitSlice(extraDBAccess)
	u.Email = email
	u.PublicKey = pubKey
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*auth.User, error) {
	row := s.queryRow(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,''), COALESCE(email,''), COALESCE(public_key,'')
		 FROM users WHERE id = ?`, id)
	var u auth.User
	var extraPerm, extraDBAccess, email, pubKey string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess, &email, &pubKey); err != nil {
		return nil, fmt.Errorf("internaldb: user not found: %w", err)
	}
	u.ExtraPerm = splitSlice(extraPerm)
	u.ExtraDBAccess = splitSlice(extraDBAccess)
	u.Email = email
	u.PublicKey = pubKey
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]auth.User, error) {
	rows, err := s.query(ctx,
		`SELECT id, username, password_hash, role, COALESCE(extra_perm,''), COALESCE(extra_db_access,''), COALESCE(email,''), COALESCE(public_key,'')
		 FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []auth.User
	for rows.Next() {
		var u auth.User
		var extraPerm, extraDBAccess, email, pubKey string
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &extraPerm, &extraDBAccess, &email, &pubKey); err != nil {
			return nil, err
		}
		u.ExtraPerm = splitSlice(extraPerm)
		u.ExtraDBAccess = splitSlice(extraDBAccess)
		u.Email = email
		u.PublicKey = pubKey
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) SetUserDBAccess(ctx context.Context, id string, dbAccess []string) error {
	_, err := s.exec(ctx,
		`UPDATE users SET extra_db_access = ? WHERE id = ?`,
		joinSlice(dbAccess), id)
	return err
}

// --- Auth: Passkeys (WebAuthn credentials) ---

// SavePasskey inserts or replaces a WebAuthn credential for a user.
func (s *Store) SavePasskey(ctx context.Context, p *auth.Passkey) error {
	_, err := s.exec(ctx,
		`INSERT OR REPLACE INTO user_passkeys
		 (id, user_id, name, public_key, credential_id, attestation, aaguid, sign_count, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Name, p.PublicKey, p.CredentialID,
		p.Attestation, p.AAGUID, p.SignCount, p.CreatedAt)
	return err
}

// ListPasskeys returns all credentials for a user.
func (s *Store) ListPasskeys(ctx context.Context, userID string) ([]*auth.Passkey, error) {
	rows, err := s.query(ctx,
		`SELECT id, user_id, name, public_key, credential_id, attestation, aaguid, sign_count, created_at
		 FROM user_passkeys WHERE user_id = ? ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*auth.Passkey
	for rows.Next() {
		var p auth.Passkey
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.PublicKey, &p.CredentialID,
			&p.Attestation, &p.AAGUID, &p.SignCount, &p.CreatedAt); err != nil {
			continue
		}
		out = append(out, &p)
	}
	return out, nil
}

// GetPasskeyByCredentialID finds a passkey by its raw credential ID (login flow).
func (s *Store) GetPasskeyByCredentialID(ctx context.Context, credentialID []byte) (*auth.Passkey, error) {
	row := s.queryRow(ctx,
		`SELECT id, user_id, name, public_key, credential_id, attestation, aaguid, sign_count, created_at
		 FROM user_passkeys WHERE credential_id = ?`, credentialID)
	var p auth.Passkey
	if err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.PublicKey, &p.CredentialID,
		&p.Attestation, &p.AAGUID, &p.SignCount, &p.CreatedAt); err != nil {
		return nil, fmt.Errorf("internaldb: passkey not found: %w", err)
	}
	return &p, nil
}

// UpdatePasskeySignCount persists the anti-replay counter after a login.
func (s *Store) UpdatePasskeySignCount(ctx context.Context, id string, signCount uint32) error {
	_, err := s.exec(ctx, `UPDATE user_passkeys SET sign_count = ? WHERE id = ?`, signCount, id)
	return err
}

// DeletePasskey removes a credential.
func (s *Store) DeletePasskey(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM user_passkeys WHERE id = ?`, id)
	return err
}

// --- Auth: Roles ---

func (s *Store) SaveRole(ctx context.Context, r auth.Role) error {
	_, err := s.exec(ctx,
		`INSERT OR REPLACE INTO roles (id, name, permissions, db_access, parent)
		 VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Name, joinSlice(r.Permissions), joinSlice(r.DBAccess), r.Parent)
	return err
}

func (s *Store) GetRole(ctx context.Context, id string) (*auth.Role, error) {
	row := s.queryRow(ctx,
		`SELECT id, name, COALESCE(permissions,''), COALESCE(db_access,''), COALESCE(parent,'') FROM roles WHERE id = ?`, id)
	var r auth.Role
	var perms, dbAccess, parent string
	if err := row.Scan(&r.ID, &r.Name, &perms, &dbAccess, &parent); err != nil {
		return nil, fmt.Errorf("internaldb: role not found: %w", err)
	}
	r.Permissions = splitSlice(perms)
	r.DBAccess = splitSlice(dbAccess)
	r.Parent = parent
	return &r, nil
}

func (s *Store) ListRoles(ctx context.Context) ([]auth.Role, error) {
	rows, err := s.query(ctx,
		`SELECT id, name, COALESCE(permissions,''), COALESCE(db_access,''), COALESCE(parent,'') FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []auth.Role
	for rows.Next() {
		var r auth.Role
		var perms, dbAccess, parent string
		if err := rows.Scan(&r.ID, &r.Name, &perms, &dbAccess, &parent); err != nil {
			continue
		}
		r.Permissions = splitSlice(perms)
		r.DBAccess = splitSlice(dbAccess)
		r.Parent = parent
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (s *Store) DeleteRole(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM roles WHERE id = ?`, id)
	return err
}

// --- Auth: API Keys (KeyStore interface implementation) ---

func (s *Store) SaveKey(ctx context.Context, k auth.APIKey) error {
	_, err := s.exec(ctx,
		`INSERT OR REPLACE INTO api_keys (prefix, hash, name, permissions, owner_id, db_access, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		k.Prefix, k.Hash, k.Name, joinSlice(k.Permissions), k.OwnerID, joinSlice(k.DBAccess), k.CreatedAt)
	return err
}

func (s *Store) GetKey(ctx context.Context, prefix string) (*auth.APIKey, error) {
	row := s.queryRow(ctx,
		`SELECT prefix, hash, name, COALESCE(permissions,''), COALESCE(owner_id,''), COALESCE(db_access,''), COALESCE(created_at,'') FROM api_keys WHERE prefix = ?`, prefix)
	var k auth.APIKey
	var perms, ownerID, dbAccess, createdAt string
	if err := row.Scan(&k.Prefix, &k.Hash, &k.Name, &perms, &ownerID, &dbAccess, &createdAt); err != nil {
		return nil, fmt.Errorf("internaldb: key not found: %w", err)
	}
	k.Permissions = splitSlice(perms)
	k.OwnerID = ownerID
	k.DBAccess = splitSlice(dbAccess)
	k.CreatedAt = createdAt
	return &k, nil
}

func (s *Store) ListKeys(ctx context.Context) ([]auth.APIKey, error) {
	rows, err := s.query(ctx,
		`SELECT prefix, name, COALESCE(permissions,''), COALESCE(owner_id,''), COALESCE(db_access,''), COALESCE(created_at,'') FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []auth.APIKey
	for rows.Next() {
		var k auth.APIKey
		var perms, ownerID, dbAccess, createdAt string
		if err := rows.Scan(&k.Prefix, &k.Name, &perms, &ownerID, &dbAccess, &createdAt); err != nil {
			continue
		}
		k.Permissions = splitSlice(perms)
		k.OwnerID = ownerID
		k.DBAccess = splitSlice(dbAccess)
		k.CreatedAt = createdAt
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Store) DeleteKey(ctx context.Context, prefix string) error {
	_, err := s.exec(ctx, `DELETE FROM api_keys WHERE prefix = ?`, prefix)
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
	row := s.queryRow(ctx,
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
		if _, err := s.exec(ctx, `UPDATE design_config SET active = 0`); err != nil {
			return fmt.Errorf("deactivate designs: %w", err)
		}
	}
	_, err := s.exec(ctx,
		`INSERT INTO design_config (id, name, config, active, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		dc.ID, dc.Name, dc.Config, boolToInt(dc.Active), dc.CreatedAt)
	return err
}

func (s *Store) ListDesigns(ctx context.Context) ([]DesignConfig, error) {
	rows, err := s.query(ctx,
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
	_, err := s.exec(ctx,
		`INSERT INTO audit_log (user_id, action, details, created_at)
		 VALUES (?, ?, ?, ?)`,
		userID, action, details, time.Now().Unix())
	return err
}

func (s *Store) ListAuditLog(ctx context.Context, limit int) ([]map[string]string, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.query(ctx,
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
			email TEXT DEFAULT '',
			public_key TEXT DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			permissions TEXT DEFAULT '',
			db_access TEXT DEFAULT '',
			parent TEXT DEFAULT ''
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
		`CREATE TABLE IF NOT EXISTS user_passkeys (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			public_key BLOB NOT NULL,
			credential_id BLOB NOT NULL UNIQUE,
			attestation TEXT DEFAULT '',
			aaguid TEXT DEFAULT '',
			sign_count INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
	}

	for _, schema := range schemas {
		if _, err := s.exec(ctx, schema); err != nil {
			return fmt.Errorf("internaldb: migrate: %w", err)
		}
	}

	// Migrate v2: add email + public_key columns if missing (existing databases)
	if s.driver != "postgres" {
		for _, ddl := range []string{
			`ALTER TABLE users ADD COLUMN email TEXT DEFAULT ''`,
			`ALTER TABLE users ADD COLUMN public_key TEXT DEFAULT ''`,
			`ALTER TABLE roles ADD COLUMN parent TEXT DEFAULT ''`,
			`ALTER TABLE api_keys ADD COLUMN owner_id TEXT DEFAULT ''`,
			`ALTER TABLE api_keys ADD COLUMN db_access TEXT DEFAULT ''`,
		} {
			if _, err := s.exec(ctx, ddl); err != nil {
				// Column may already exist — ignore error
			}
		}
	} else {
		// PG: ADD COLUMN IF NOT EXISTS
		for _, ddl := range []string{
			`ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT DEFAULT ''`,
			`ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT DEFAULT ''`,
			`ALTER TABLE roles ADD COLUMN IF NOT EXISTS parent TEXT DEFAULT ''`,
			`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS owner_id TEXT DEFAULT ''`,
			`ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS db_access TEXT DEFAULT ''`,
		} {
			if _, err := s.exec(ctx, ddl); err != nil {
				slog.Warn("pg migration column", "error", err)
			}
		}
	}

	slog.Info("internal database migrated")
	return nil
}

// seed populates default roles and admin user if empty
func (s *Store) seed(ctx context.Context) error {
	// Seed default roles
	count := 0
	if err := s.queryRow(ctx, `SELECT COUNT(*) FROM roles`).Scan(&count); err != nil {
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
	if err := s.queryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		// SSH-style passwordless admin accounts from env (public keys only).
		// Format: GODB_ADMIN_PUBKEYS={"chang":"ssh-ed25519 AAAA...","hermes":"ssh-ed25519 AAAA..."}
		if pubKeys := os.Getenv("GODB_ADMIN_PUBKEYS"); pubKeys != "" {
			var m map[string]string
			if err := json.Unmarshal([]byte(pubKeys), &m); err == nil {
				for username, pub := range m {
					if username == "" || pub == "" {
						continue
					}
					dummy, herr := auth.HashPassword("pubkey-only-" + username)
					if herr != nil {
						slog.Warn("seed pubkey admin: hash dummy", "user", username, "error", herr)
						continue
					}
					adm := auth.User{
						ID:           "admin-" + username,
						Username:     username,
						PasswordHash: dummy, // dummy; pubkey login only
						Role:         "admin",
						PublicKey:    pub,
					}
					if err := s.SaveUser(ctx, adm); err != nil {
						return fmt.Errorf("internaldb: seed pubkey admin %s: %w", username, err)
					}
					slog.Info("seeded passwordless admin user", "username", username)
				}
			}
		}
		// Fallback: default password admin (setup required on first login).
		if _, err := s.GetUser(ctx, "admin"); err != nil {
			hash, err := auth.HashPassword("admin")
			if err != nil {
				return err
			}
			admin := auth.User{
				ID:           "admin-001",
				Username:     "admin",
				PasswordHash: hash,
				Role:         "admin",
			}
			if err := s.SaveUser(ctx, admin); err != nil {
				return fmt.Errorf("internaldb: seed admin: %w", err)
			}
			slog.Info("default admin user created — setup required on first login")
		}
	}

	return nil
}

// IsSetupComplete returns true if the admin user has changed the default password.
func (s *Store) IsSetupComplete(ctx context.Context) (bool, error) {
	u, err := s.GetUser(ctx, "admin")
	if err != nil {
		// No admin user at all — setup needed
		return false, nil
	}
	return !auth.IsDefaultPassword(u.PasswordHash), nil
}

// CompleteSetup sets the admin email and password, completing first-time setup.
func (s *Store) CompleteSetup(ctx context.Context, email, newPassword string) error {
	u, err := s.GetUser(ctx, "admin")
	if err != nil {
		return fmt.Errorf("internaldb: admin user not found: %w", err)
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("internaldb: hash password: %w", err)
	}

	u.PasswordHash = newHash
	u.Email = email

	return s.SaveUser(ctx, *u)
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
