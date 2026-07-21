package mcp

import (
	"context"
	"testing"

	"go-database/internal/connection"
	"go-database/internal/plugin"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeGate implements DBGate for tests without a live server.
type fakeGate struct{}

func (f *fakeGate) List() []connection.Summary {
	return []connection.Summary{
		{ID: "conn-1", Name: "Local SQLite", Type: plugin.TypeSQLite, State: connection.StateConnected, Latency: 1},
	}
}
func (f *fakeGate) GetConnection(id string) (*connection.Connection, error) { return nil, nil }
func (f *fakeGate) Query(ctx context.Context, id string, query string) (*plugin.Result, error) {
	return nil, nil
}
func (f *fakeGate) Execute(ctx context.Context, id string, query string) (*plugin.Result, error) {
	return nil, nil
}
func (f *fakeGate) Tables(ctx context.Context, id string) ([]string, error)       { return nil, nil }
func (f *fakeGate) Schema(ctx context.Context, id string) (*plugin.Schema, error) { return nil, nil }
func (f *fakeGate) Databases(ctx context.Context, id string) ([]string, error)    { return nil, nil }

func TestMCPListConnections(t *testing.T) {
	SetDBGate(&fakeGate{})
	srv := NewServer(nil)
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := srv.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	defer serverSession.Wait()

	client := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "list_connections"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Content) == 0 {
		t.Fatal("empty content")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if text == "" {
		t.Fatal("empty text result")
	}
	t.Logf("list_connections => %s", text)
}
