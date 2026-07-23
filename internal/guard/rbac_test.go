package guard

import (
	"testing"

	"go-database/internal/suggest"
)

func TestDetectCommandMultiStatement(t *testing.T) {
	// SELECT 1; DROP TABLE users; must be classified as DROP (highest risk),
	// not SELECT — otherwise a query-only permission would allow the DROP.
	cmd := detectCommand("SELECT 1; DROP TABLE users;")
	if cmd != suggest.CmdDrop {
		t.Fatalf("expected CmdDrop, got %v", cmd)
	}
}

func TestDetectCommandSingle(t *testing.T) {
	cases := map[string]suggest.CommandType{
		"SELECT * FROM users":           suggest.CmdSelect,
		"INSERT INTO t VALUES (1)":      suggest.CmdInsert,
		"UPDATE t SET x=1":              suggest.CmdUpdate,
		"DELETE FROM t":                 suggest.CmdDelete,
		"CREATE TABLE t (id INT)":       suggest.CmdCreate,
		"DROP TABLE t":                  suggest.CmdDrop,
		"ALTER TABLE t ADD c INT":       suggest.CmdAlter,
		"TRUNCATE TABLE t":              suggest.CmdTruncate,
		"   -- comment\nSELECT 1":        suggest.CmdSelect,
		"SELECT '; DROP TABLE x;' AS x": suggest.CmdSelect, // ';' inside literal must NOT split
	}
	for sql, want := range cases {
		if got := detectCommand(sql); got != want {
			t.Errorf("detectCommand(%q) = %v, want %v", sql, got, want)
		}
	}
}

func TestCheckCommandBlocksDropViaSelectPerm(t *testing.T) {
	g := New()
	// caller has only query permission
	_, ok := g.CheckCommand("SELECT 1; DROP TABLE users;", []string{"connections:query"})
	if ok {
		t.Fatal("query-only permission must NOT allow SELECT;DROP (classified as DROP)")
	}
}
