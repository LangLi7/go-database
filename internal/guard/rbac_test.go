package guard

import (
	"testing"

	"go-database/internal/suggest"
)

func TestCheckCommand_SelectAllowed(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("SELECT * FROM users", []string{"connections:query"})
	if !ok {
		t.Fatalf("SELECT should be allowed with query perm, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestCheckCommand_SelectBlocked(t *testing.T) {
	g := &Guard{}
	_, ok := g.CheckCommand("SELECT * FROM users", []string{})
	if ok {
		t.Fatal("SELECT should be blocked without permissions")
	}
}

func TestCheckCommand_DropBlocked(t *testing.T) {
	g := &Guard{}
	_, ok := g.CheckCommand("DROP TABLE users", []string{})
	if ok {
		t.Fatal("DROP should be blocked without permissions")
	}
}

func TestCheckCommand_DropWithPerm(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("DROP TABLE users", []string{"connections:execute"})
	if !ok {
		t.Fatalf("DROP should be allowed with execute perm, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestCheckCommand_InsertBlocked(t *testing.T) {
	g := &Guard{}
	_, ok := g.CheckCommand("INSERT INTO users (id) VALUES (1)", []string{})
	if ok {
		t.Fatal("INSERT should be blocked without permissions")
	}
}

func TestCheckCommand_InsertWithPerm(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("INSERT INTO users (id) VALUES (1)", []string{"connections:execute"})
	if !ok {
		t.Fatalf("INSERT should be allowed with execute perm, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestCheckCommand_DeleteBlocked(t *testing.T) {
	g := &Guard{}
	_, ok := g.CheckCommand("DELETE FROM users", []string{})
	if ok {
		t.Fatal("DELETE should be blocked without permissions")
	}
}

func TestCheckCommand_UnknownBlocked(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("EXPLAIN SELECT * FROM users", []string{})
	if ok {
		t.Fatalf("UNKNOWN/EXPLAIN should be blocked by default, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestCheckCommand_CommentPrefix(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("-- comment\nDROP TABLE users", []string{})
	if ok {
		t.Fatalf("DROP with comment prefix should still be blocked, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestCheckCommand_BlockComment(t *testing.T) {
	g := &Guard{}
	cmd, ok := g.CheckCommand("/* block */DROP TABLE users", []string{})
	if ok {
		t.Fatalf("DROP with block comment should still be blocked, got cmd=%s", cmd)
	}
	_ = cmd
}

func TestDetectCommand(t *testing.T) {
	tests := []struct {
		sql string
		cmd suggest.CommandType
	}{
		{"select * from users", suggest.CmdSelect},
		{"INSERT INTO t VALUES (1)", suggest.CmdInsert},
		{"update t set x=1", suggest.CmdUpdate},
		{"delete from t", suggest.CmdDelete},
		{"create table t (id int)", suggest.CmdCreate},
		{"drop table t", suggest.CmdDrop},
		{"alter table t add x int", suggest.CmdAlter},
		{"truncate table t", suggest.CmdTruncate},
		{"   select 1", suggest.CmdSelect},
		{"-- comment\nselect 1", suggest.CmdSelect},
		{"/* block */select 1", suggest.CmdSelect},
		{"vacuum", suggest.CmdUnknown},
	}
	for _, tc := range tests {
		cmd := detectCommand(tc.sql)
		if cmd != tc.cmd {
			t.Errorf("detectCommand(%q) = %s, want %s", tc.sql, cmd, tc.cmd)
		}
	}
}
