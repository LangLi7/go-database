package connection

import (
	"context"
	"testing"

	"go-database/internal/plugin"
)

// fakePlugin always connects successfully (used so Add stores the connection
// without needing a real DB server in tests).
type fakePlugin struct{}

func (fakePlugin) Connect(ctx context.Context, cfg plugin.Config) error { return nil }
func (fakePlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}
func (fakePlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	return &plugin.Result{}, nil
}
func (fakePlugin) Tables(ctx context.Context) ([]string, error)          { return nil, nil }
func (fakePlugin) Schema(ctx context.Context) (*plugin.Schema, error)    { return nil, nil }
func (fakePlugin) Databases(ctx context.Context) ([]string, error)       { return nil, nil }
func (fakePlugin) CreateDatabase(ctx context.Context, name string) error { return nil }
func (fakePlugin) DropDatabase(ctx context.Context, name string) error   { return nil }
func (fakePlugin) Type() plugin.DBType                                   { return "fake" }
func (fakePlugin) Close() error                                          { return nil }
func (fakePlugin) Ping(ctx context.Context) error                        { return nil }

func init() {
	plugin.Register("fake", func() plugin.DBPlugin { return fakePlugin{} })
}

func TestListVisibleIsolation(t *testing.T) {
	m := NewManager()
	// userA owns dbA, userB owns dbB, system owns dbSys
	connA, _ := m.Add(context.Background(), "dbA", "fake", "local", plugin.Config{}, nil, "userA")
	connB, _ := m.Add(context.Background(), "dbB", "fake", "local", plugin.Config{}, nil, "userB")
	m.Add(context.Background(), "dbSys", "fake", "local", plugin.Config{}, nil, "")

	// userA sees only dbA (not dbB, not dbSys)
	a := m.ListVisible("userA", nil, false)
	if len(a) != 1 || a[0].Name != "dbA" {
		t.Fatalf("userA should see only dbA, got %v", names(a))
	}

	// userB sees only dbB
	b := m.ListVisible("userB", nil, false)
	if len(b) != 1 || b[0].Name != "dbB" {
		t.Fatalf("userB should see only dbB, got %v", names(b))
	}

	// shared: userA gets dbB via dbAccess (by connection ID)
	shared := m.ListVisible("userA", []string{connB.ID}, false)
	if len(shared) != 2 {
		t.Fatalf("userA+share should see dbA+dbB, got %v", names(shared))
	}

	// admin sees all three
	admin := m.ListVisible("admin", nil, true)
	if len(admin) != 3 {
		t.Fatalf("admin should see all 3, got %v", names(admin))
	}
	_ = connA
}

func names(s []Summary) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		out = append(out, x.Name)
	}
	return out
}
