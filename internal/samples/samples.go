package samples

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"go-database/internal/connection"
)

//go:embed data/*/sample.json
var samplesFS embed.FS

// Sample describes a database sample/template
type Sample struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Version     string  `json:"version"`
	Tables      []Table `json:"tables"`
}

// Table describes a table in a sample
type Table struct {
	Name    string           `json:"name"`
	Comment string           `json:"comment"`
	Columns []Column         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

// Column describes a table column
type Column struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	PK            bool   `json:"pk"`
	AutoIncrement bool   `json:"autoincrement"`
	NotNull       bool   `json:"notnull"`
	Unique        bool   `json:"unique"`
	Default       any    `json:"default"`
	Ref           *Ref   `json:"ref"`
}

// Ref describes a foreign key reference
type Ref struct {
	Table  string `json:"table"`
	Column string `json:"column"`
}

// List returns all available sample names
func List() ([]string, error) {
	entries, err := samplesFS.ReadDir("data")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// Get loads a sample by name
func Get(name string) (*Sample, error) {
	data, err := samplesFS.ReadFile(fmt.Sprintf("data/%s/sample.json", name))
	if err != nil {
		return nil, fmt.Errorf("sample %q not found", name)
	}
	var s Sample
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("sample %q: parse error: %w", name, err)
	}
	return &s, nil
}

// SQL returns the SQL statements to create the sample in a specific DB type
func (s *Sample) SQL(dbType string) ([]string, error) {
	var stmts []string
	for _, t := range s.Tables {
		ddl, err := t.createSQL(dbType)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, ddl)
	}
	for _, t := range s.Tables {
		for _, row := range t.Rows {
			ins, err := t.insertSQL(dbType, row)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, ins)
		}
	}
	return stmts, nil
}

func (t *Table) createSQL(dbType string) (string, error) {
	var cols []string
	for _, c := range t.Columns {
		col := fmt.Sprintf("  %s %s", quoteIdent(c.Name), sqlType(c.Type, dbType))
		if c.PK {
			switch dbType {
			case "postgres":
				col += " PRIMARY KEY"
			case "mysql", "mariadb":
				col += " PRIMARY KEY"
			default:
				col += " PRIMARY KEY"
			}
		}
		if c.AutoIncrement {
			switch dbType {
			case "postgres":
				col = fmt.Sprintf("  %s SERIAL PRIMARY KEY", quoteIdent(c.Name))
			case "mysql", "mariadb":
				col = fmt.Sprintf("  %s INTEGER PRIMARY KEY AUTO_INCREMENT", quoteIdent(c.Name))
			default:
				col = fmt.Sprintf("  %s INTEGER PRIMARY KEY", quoteIdent(c.Name))
			}
		}
		if c.NotNull && !c.PK {
			col += " NOT NULL"
		}
		if c.Unique {
			col += " UNIQUE"
		}
		if c.Default != nil {
			col += fmt.Sprintf(" DEFAULT %s", quoteVal(c.Default))
		}
		if c.Ref != nil {
			col += fmt.Sprintf(" REFERENCES %s(%s)", quoteIdent(c.Ref.Table), quoteIdent(c.Ref.Column))
		}
		cols = append(cols, col)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", quoteIdent(t.Name), strings.Join(cols, ",\n")), nil
}

func (t *Table) insertSQL(dbType string, row map[string]any) (string, error) {
	var colNames, colVals []string
	for _, c := range t.Columns {
		val, ok := row[c.Name]
		if !ok || val == nil {
			continue
		}
		// skip auto-increment PK for non-Postgres
		if c.PK && c.AutoIncrement && dbType != "postgres" {
			continue
		}
		colNames = append(colNames, quoteIdent(c.Name))
		colVals = append(colVals, quoteVal(val))
	}
	if len(colNames) == 0 {
		return "", nil
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		quoteIdent(t.Name),
		strings.Join(colNames, ", "),
		strings.Join(colVals, ", ")), nil
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteVal(v any) string {
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func sqlType(colType, dbType string) string {
	switch dbType {
	case "postgres":
		switch colType {
		case "integer":
			return "INTEGER"
		case "real":
			return "REAL"
		default:
			return "TEXT"
		}
	case "mysql", "mariadb":
		switch colType {
		case "integer":
			return "INTEGER"
		case "real":
			return "DOUBLE"
		default:
			return "TEXT"
		}
	default:
		switch colType {
		case "integer":
			return "INTEGER"
		case "real":
			return "REAL"
		default:
			return "TEXT"
		}
	}
}

// Load creates all tables and inserts all data into the given connection
func (s *Sample) Load(ctx context.Context, mgr *connection.Manager, connID string) error {
	conn, err := mgr.Get(connID)
	if err != nil {
		return err
	}
	dbType := string(conn.Type)

	stmts, err := s.SQL(dbType)
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		if _, err := mgr.Execute(ctx, connID, stmt); err != nil {
			return fmt.Errorf("execute: %s: %w", stmt[:min(len(stmt), 80)], err)
		}
	}

	slog.Info("sample loaded", "sample", s.Name, "connection", connID, "statements", len(stmts))
	return nil
}
