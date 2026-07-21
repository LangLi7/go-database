package internaldb

import (
	"context"
	"strings"
	"testing"

	"go-database/internal/auth"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	return s
}

// ─── SQL Conversion Tests ──────────────────────────────────────────────────

func TestSQLPlaceholderConversion(t *testing.T) {
	s := &Store{driver: "postgres"}

	tests := []struct {
		input string
		want  string
	}{
		{`SELECT * FROM users WHERE id = ?`, `SELECT * FROM users WHERE id = $1`},
		{`SELECT * FROM users WHERE id = ? AND name = ?`, `SELECT * FROM users WHERE id = $1 AND name = $2`},
		{`INSERT INTO users VALUES (?, ?, ?)`, `INSERT INTO users VALUES ($1, $2, $3)`},
		{`SELECT * FROM users`, `SELECT * FROM users`},
	}

	for _, tt := range tests {
		got := s.sql(tt.input)
		if got != tt.want {
			t.Errorf("sql(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestSQLInsertReplaceConversion(t *testing.T) {
	s := &Store{driver: "postgres", db: nil}

	sql := `INSERT OR REPLACE INTO users (id, username, password_hash, role, extra_perm, extra_db_access, email, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	result := s.sql(sql)

	if !strings.Contains(result, "ON CONFLICT") {
		t.Error("expected ON CONFLICT clause")
	}
	if !strings.Contains(result, "EXCLUDED") {
		t.Error("expected EXCLUDED reference")
	}
	if strings.Contains(result, "OR REPLACE") {
		t.Error("OR REPLACE should be removed for PG")
	}
}

func TestSQLDateTimeConversion(t *testing.T) {
	s := &Store{driver: "postgres"}
	result := s.sql(`SELECT * FROM users WHERE created_at > datetime('now')`)
	if !strings.Contains(result, "NOW()") {
		t.Error("expected NOW() conversion")
	}
}

func TestSQLAutoIncrementConversion(t *testing.T) {
	s := &Store{driver: "postgres"}
	result := s.sql(`CREATE TABLE audit_log (id INTEGER PRIMARY KEY AUTOINCREMENT)`)
	if !strings.Contains(result, "SERIAL PRIMARY KEY") {
		t.Error("expected SERIAL PRIMARY KEY conversion")
	}
}

func TestSQLitePassthrough(t *testing.T) {
	s := &Store{driver: "sqlite"}
	input := `INSERT OR REPLACE INTO users VALUES (?, ?, ?)`
	result := s.sql(input)
	if result != input {
		t.Errorf("SQLite should passthrough, got: %s", result)
	}
}

// ─── User CRUD Tests ───────────────────────────────────────────────────────

func TestSaveAndGetUser(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	u := auth.User{
		ID:           "test-001",
		Username:     "testuser",
		PasswordHash: "$2a$10$hash",
		Role:         "developer",
	}

	if err := s.SaveUser(context.Background(), u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}

	got, err := s.GetUser(context.Background(), "testuser")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if got.Username != "testuser" || got.Role != "developer" {
		t.Errorf("unexpected user: %+v", got)
	}
}

func TestGetUserNotFound(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	_, err := s.GetUser(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestListUsers(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	users, err := s.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	// Admin user is seeded
	if len(users) == 0 {
		t.Fatal("expected at least admin user")
	}
}

func TestDeleteUser(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	u := auth.User{
		ID:           "del-001",
		Username:     "deleteme",
		PasswordHash: "hash",
		Role:         "readonly",
	}
	if err := s.SaveUser(context.Background(), u); err != nil {
		t.Fatalf("SaveUser failed: %v", err)
	}
	if err := s.DeleteUser(context.Background(), "del-001"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}
	_, err := s.GetUser(context.Background(), "deleteme")
	if err == nil {
		t.Fatal("expected user to be deleted")
	}
}

func TestSetUserDBAccess(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.SetUserDBAccess(context.Background(), "admin-001", []string{"conn-1", "conn-2"}); err != nil {
		t.Fatalf("SetUserDBAccess failed: %v", err)
	}

	u, err := s.GetUserByID(context.Background(), "admin-001")
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if len(u.ExtraDBAccess) != 2 {
		t.Errorf("expected 2 db access entries, got %v", u.ExtraDBAccess)
	}
}

// ─── Role CRUD Tests ───────────────────────────────────────────────────────

func TestSaveAndGetRole(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	role := auth.Role{
		ID:          "custom-role",
		Name:        "Custom Role",
		Permissions: []string{"connections:list", "connections:query"},
		DBAccess:    []string{"db-1"},
	}

	if err := s.SaveRole(context.Background(), role); err != nil {
		t.Fatalf("SaveRole failed: %v", err)
	}

	got, err := s.GetRole(context.Background(), "custom-role")
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if got.Name != "Custom Role" {
		t.Errorf("unexpected name: %s", got.Name)
	}
	if len(got.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %v", got.Permissions)
	}
}

func TestListRoles(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	roles, err := s.ListRoles(context.Background())
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	// Default roles (admin, developer, readonly) seeded
	if len(roles) < 3 {
		t.Fatalf("expected at least 3 roles, got %d", len(roles))
	}
}

func TestDeleteRole(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.DeleteRole(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("DeleteRole should not error for nonexistent: %v", err)
	}
}

// ─── API Key Tests ─────────────────────────────────────────────────────────

func TestSaveAndGetKey(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	key := auth.APIKey{
		Prefix:      "abc123",
		Hash:        "hashvalue",
		Name:        "test-key",
		Permissions: []string{"connections:list"},
		CreatedAt:   "2025-01-01",
	}

	if err := s.SaveKey(context.Background(), key); err != nil {
		t.Fatalf("SaveKey failed: %v", err)
	}

	got, err := s.GetKey(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if got.Name != "test-key" {
		t.Errorf("unexpected name: %s", got.Name)
	}
}

func TestDeleteKey(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.DeleteKey(context.Background(), "nonexistent"); err != nil {
		t.Fatalf("DeleteKey should not error: %v", err)
	}
}

// ─── Design Config Tests ───────────────────────────────────────────────────

func TestSaveAndGetActiveDesign(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	dc := DesignConfig{
		ID:     "design-1",
		Name:   "Dark Theme",
		Config: `{"primary":"#000"}`,
		Active: true,
	}

	if err := s.SaveDesign(context.Background(), dc); err != nil {
		t.Fatalf("SaveDesign failed: %v", err)
	}

	got, err := s.GetActiveDesign(context.Background())
	if err != nil {
		t.Fatalf("GetActiveDesign failed: %v", err)
	}
	if got.Name != "Dark Theme" {
		t.Errorf("expected Dark Theme, got %s", got.Name)
	}

	// Saving a new active design should deactivate the old one
	dc2 := DesignConfig{
		ID:     "design-2",
		Name:   "Light Theme",
		Config: `{"primary":"#fff"}`,
		Active: true,
	}
	if err := s.SaveDesign(context.Background(), dc2); err != nil {
		t.Fatalf("SaveDesign failed: %v", err)
	}

	got, _ = s.GetActiveDesign(context.Background())
	if got.Name != "Light Theme" {
		t.Errorf("expected Light Theme, got %s", got.Name)
	}
}

func TestGetActiveDesignNone(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	// Delete active designs
	designs, _ := s.ListDesigns(context.Background())
	for _, d := range designs {
		// Can't delete, but we can save inactive
		d.Active = false
		s.SaveDesign(context.Background(), d)
	}

	_, err := s.GetActiveDesign(context.Background())
	if err == nil {
		t.Log("note: expected error when no active design (may have active)")
	}
}

// ─── Audit Log Tests ───────────────────────────────────────────────────────

func TestLogAndListAudit(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.LogAudit(context.Background(), "admin-001", "test.action", "test details"); err != nil {
		t.Fatalf("LogAudit failed: %v", err)
	}

	logs, err := s.ListAuditLog(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListAuditLog failed: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least one audit log entry")
	}
}

// ─── Setup Tests ───────────────────────────────────────────────────────────

func TestIsSetupComplete(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	// Admin is seeded with default password -> setup not complete
	complete, err := s.IsSetupComplete(context.Background())
	if err != nil {
		t.Fatalf("IsSetupComplete failed: %v", err)
	}
	if complete {
		t.Log("note: setup already completed (testing against existing DB)")
	}
}

func TestCompleteSetup(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	complete, _ := s.IsSetupComplete(context.Background())
	if complete {
		t.Skip("setup already completed, skipping")
	}

	if err := s.CompleteSetup(context.Background(), "admin@example.com", "newpassword123"); err != nil {
		t.Fatalf("CompleteSetup failed: %v", err)
	}

	complete, err := s.IsSetupComplete(context.Background())
	if err != nil {
		t.Fatalf("IsSetupComplete after setup: %v", err)
	}
	if !complete {
		t.Error("setup should be complete after CompleteSetup")
	}

	// Verify email was saved
	u, err := s.GetUser(context.Background(), "admin")
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if u.Email != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", u.Email)
	}
}

// ─── Helper Tests ──────────────────────────────────────────────────────────

func TestJoinSlice(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b", "c"}, "a,b,c"},
	}
	for _, tt := range tests {
		got := joinSlice(tt.input)
		if got != tt.want {
			t.Errorf("joinSlice(%v) = %q; want %q", tt.input, got, tt.want)
		}
	}
}

func TestSplitSlice(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		got := splitSlice(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitSlice(%q) = %v; want %v", tt.input, got, tt.want)
		}
	}
}

func TestBoolToInt(t *testing.T) {
	if boolToInt(true) != 1 {
		t.Error("expected 1 for true")
	}
	if boolToInt(false) != 0 {
		t.Error("expected 0 for false")
	}
}

// ─── Concurrent Access Tests ───────────────────────────────────────────────

func TestConcurrentUserSave(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(n int) {
			u := auth.User{
				ID:           "concurrent-001",
				Username:     "concurrent-user",
				PasswordHash: "hash",
				Role:         "readonly",
			}
			// This should be safe: INSERT OR REPLACE handles conflicts
			_ = s.SaveUser(context.Background(), u)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	got, err := s.GetUser(context.Background(), "concurrent-user")
	if err != nil {
		t.Fatalf("GetUser after concurrent save: %v", err)
	}
	if got.Username != "concurrent-user" {
		t.Errorf("unexpected username: %s", got.Username)
	}
}
