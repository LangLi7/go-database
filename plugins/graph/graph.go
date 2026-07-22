// Package graph implements an embedded graph database plugin for go-database.
// Nodes and edges are stored in a JSON file (one per graph DB). No external
// server required — it models real graph data (labels, typed edges, traversal)
// rather than faking it with SQL.
//
// Query syntax (passed to Query/Execute):
//
//	CREATE NODE <label> <json>              create a node with label + props
//	CREATE EDGE <from> <to> <type> <json>   create a directed edge
//	MATCH <label> [WHERE k=v]               list nodes of a label (optional filter)
//	NEIGHBORS <nodeId> [edgeType]           direct neighbours
//	TRAVERSE <nodeId> <depth>                BFS up to depth hops
//	NODE <id>                               fetch a single node
package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"go-database/internal/plugin"
)

// Node is a vertex in the graph.
type Node struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Props  map[string]any `json:"props"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From  string         `json:"from"`
	To    string         `json:"to"`
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

type store struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`
}

// GraphPlugin is the embedded graph database.
type GraphPlugin struct {
	cfg   plugin.Config
	path  string
	mu    sync.Mutex
	data  store
}

// compile-time interface check
var _ plugin.DBPlugin = (*GraphPlugin)(nil)

// Register the plugin with the go-database plugin registry.
func init() {
	plugin.Register(plugin.TypeGraph, func() plugin.DBPlugin { return &GraphPlugin{} })
}

func (g *GraphPlugin) Type() plugin.DBType { return plugin.TypeGraph }

func (g *GraphPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	g.cfg = cfg
	g.data = store{Nodes: map[string]Node{}}
	// Persistence path: explicit FilePath, else a default alongside cwd.
	g.path = cfg.FilePath
	if g.path == "" {
		g.path = "graph.db.json"
	}
	if b, err := os.ReadFile(g.path); err == nil {
		_ = json.Unmarshal(b, &g.data)
	}
	if g.data.Nodes == nil {
		g.data.Nodes = map[string]Node{}
	}
	return nil
}

func (g *GraphPlugin) Ping(ctx context.Context) error { return nil }

func (g *GraphPlugin) Close() error {
	return g.save()
}

func (g *GraphPlugin) save() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	b, err := json.MarshalIndent(g.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.path, b, 0644)
}

// Execute handles write operations (CREATE NODE / CREATE EDGE).
func (g *GraphPlugin) Execute(ctx context.Context, query string) (*plugin.Result, error) {
	q := strings.TrimSpace(query)
	switch {
	case strings.HasPrefix(strings.ToUpper(q), "CREATE NODE"):
		return g.createNode(q)
	case strings.HasPrefix(strings.ToUpper(q), "CREATE EDGE"):
		return g.createEdge(q)
	default:
		return nil, fmt.Errorf("graph: unsupported statement %q (use CREATE NODE/EDGE)", q)
	}
}

// Query handles read operations (MATCH / NEIGHBORS / TRAVERSE / NODE).
func (g *GraphPlugin) Query(ctx context.Context, query string) (*plugin.Result, error) {
	q := strings.TrimSpace(query)
	upper := strings.ToUpper(q)
	switch {
	case strings.HasPrefix(upper, "MATCH"):
		return g.match(q)
	case strings.HasPrefix(upper, "NEIGHBORS"):
		return g.neighbors(q)
	case strings.HasPrefix(upper, "TRAVERSE"):
		return g.traverse(q)
	case strings.HasPrefix(upper, "NODE"):
		return g.getNode(q)
	default:
		return nil, fmt.Errorf("graph: unsupported query %q (use MATCH/NEIGHBORS/TRAVERSE/NODE)", q)
	}
}

func (g *GraphPlugin) createNode(q string) (*plugin.Result, error) {
	// CREATE NODE <label> <json>
	rest := strings.TrimSpace(q[len("CREATE NODE"):])
	sp := strings.IndexByte(rest, ' ')
	if sp < 0 {
		return nil, fmt.Errorf("graph: CREATE NODE needs <label> <json>")
	}
	label := rest[:sp]
	propsRaw := strings.TrimSpace(rest[sp+1:])
	if propsRaw == "" {
		propsRaw = "{}"
	}
	var props map[string]any
	if err := json.Unmarshal([]byte(propsRaw), &props); err != nil {
		return nil, fmt.Errorf("graph: invalid node props JSON: %w", err)
	}
	id := fmt.Sprintf("n_%d", len(g.data.Nodes)+1)
	g.mu.Lock()
	g.data.Nodes[id] = Node{ID: id, Label: label, Props: props}
	g.mu.Unlock()
	if err := g.save(); err != nil {
		return nil, err
	}
	return &plugin.Result{
		Columns: []string{"id", "label"},
		Rows:    [][]any{{id, label}},
	}, nil
}

func (g *GraphPlugin) createEdge(q string) (*plugin.Result, error) {
	// CREATE EDGE <from> <to> <type> <json>
	rest := strings.TrimSpace(q[len("CREATE EDGE"):])
	parts := strings.Fields(rest)
	if len(parts) < 3 {
		return nil, fmt.Errorf("graph: CREATE EDGE needs <from> <to> <type> [json]")
	}
	from, to, etype := parts[0], parts[1], parts[2]
	props := map[string]any{}
	if len(parts) >= 4 {
		if err := json.Unmarshal([]byte(parts[3]), &props); err != nil {
			return nil, fmt.Errorf("graph: invalid edge props JSON: %w", err)
		}
	}
	g.mu.Lock()
	g.data.Edges = append(g.data.Edges, Edge{From: from, To: to, Type: etype, Props: props})
	g.mu.Unlock()
	if err := g.save(); err != nil {
		return nil, err
	}
	return &plugin.Result{
		Columns:      []string{"from", "to", "type"},
		Rows:         [][]any{{from, to, etype}},
		RowsAffected: 1,
	}, nil
}

func (g *GraphPlugin) match(q string) (*plugin.Result, error) {
	// MATCH <label> [WHERE k=v]
	rest := strings.TrimSpace(q[len("MATCH"):])
	label := rest
	var whereK, whereV string
	if idx := strings.Index(strings.ToUpper(rest), " WHERE "); idx >= 0 {
		label = strings.TrimSpace(rest[:idx])
		w := strings.TrimSpace(rest[idx+len(" WHERE "):])
		if eq := strings.IndexByte(w, '='); eq >= 0 {
			whereK, whereV = w[:eq], w[eq+1:]
		}
	}
	rows := [][]any{}
	for _, n := range g.data.Nodes {
		if n.Label != label {
			continue
		}
		if whereK != "" {
			if fmt.Sprintf("%v", n.Props[whereK]) != whereV {
				continue
			}
		}
		rows = append(rows, []any{n.ID, n.Label, n.Props})
	}
	return &plugin.Result{
		Columns: []string{"id", "label", "props"},
		Rows:    rows,
	}, nil
}

func (g *GraphPlugin) neighbors(q string) (*plugin.Result, error) {
	parts := strings.Fields(q)
	if len(parts) < 2 {
		return nil, fmt.Errorf("graph: NEIGHBORS needs <nodeId> [edgeType]")
	}
	id := parts[1]
	etype := ""
	if len(parts) >= 3 {
		etype = parts[2]
	}
	rows := [][]any{}
	for _, e := range g.data.Edges {
		// neighbour = the other endpoint of an incident edge (in or out)
		var other string
		switch {
		case e.From == id:
			other = e.To
		case e.To == id:
			other = e.From
		default:
			continue
		}
		if etype != "" && e.Type != etype {
			continue
		}
		if n, ok := g.data.Nodes[other]; ok {
			rows = append(rows, []any{n.ID, n.Label, e.Type})
		}
	}
	return &plugin.Result{
		Columns: []string{"id", "label", "edge"},
		Rows:    rows,
	}, nil
}

func (g *GraphPlugin) traverse(q string) (*plugin.Result, error) {
	parts := strings.Fields(q)
	if len(parts) < 3 {
		return nil, fmt.Errorf("graph: TRAVERSE needs <nodeId> <depth>")
	}
	start := parts[1]
	depth := 0
	fmt.Sscanf(parts[2], "%d", &depth)

	visited := map[string]bool{start: true}
	level := []string{start}
	rows := [][]any{}
	for d := 0; d < depth; d++ {
		next := []string{}
		for _, cur := range level {
			for _, e := range g.data.Edges {
				if e.From != cur || visited[e.To] {
					continue
				}
				visited[e.To] = true
				if n, ok := g.data.Nodes[e.To]; ok {
					rows = append(rows, []any{n.ID, n.Label, d + 1})
				}
				next = append(next, e.To)
			}
		}
		level = next
		if len(level) == 0 {
			break
		}
	}
	return &plugin.Result{
		Columns: []string{"id", "label", "distance"},
		Rows:    rows,
	}, nil
}

func (g *GraphPlugin) getNode(q string) (*plugin.Result, error) {
	id := strings.TrimSpace(q[len("NODE"):])
	n, ok := g.data.Nodes[id]
	if !ok {
		return nil, fmt.Errorf("graph: node %q not found", id)
	}
	return &plugin.Result{
		Columns: []string{"id", "label", "props"},
		Rows:    [][]any{{n.ID, n.Label, n.Props}},
	}, nil
}

// Tables returns the distinct node labels (graph analogue of tables).
func (g *GraphPlugin) Tables(ctx context.Context) ([]string, error) {
	seen := map[string]bool{}
	var labels []string
	for _, n := range g.data.Nodes {
		if !seen[n.Label] {
			seen[n.Label] = true
			labels = append(labels, n.Label)
		}
	}
	return labels, nil
}

// Schema returns labels and edge types.
func (g *GraphPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	seen := map[string]bool{}
	s := &plugin.Schema{}
	for _, n := range g.data.Nodes {
		if !seen[n.Label] {
			seen[n.Label] = true
			s.Tables = append(s.Tables, plugin.TableInfo{Name: n.Label, RowCount: 0})
		}
	}
	return s, nil
}

// Databases returns the single graph file as one "database".
func (g *GraphPlugin) Databases(ctx context.Context) ([]string, error) {
	return []string{g.path}, nil
}

func (g *GraphPlugin) CreateDatabase(ctx context.Context, name string) error {
	return nil // single-file graph; CreateDatabase is a no-op
}

func (g *GraphPlugin) DropDatabase(ctx context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.data = store{Nodes: map[string]Node{}}
	return g.save()
}
