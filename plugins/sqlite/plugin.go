package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypeSQLite, func() plugin.DBPlugin { return &sqlitePlugin{} })
}

type sqlitePlugin struct {
	db  *sql.DB
	cfg plugin.Config
}

func (p *sqlitePlugin) Type() plugin.DBType { return plugin.TypeSQLite }

func (p *sqlitePlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	path := cfg.FilePath
	if path == "" {
		path = fmt.Sprintf("%s.db", cfg.Database)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("sqlite: open: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("sqlite: ping: %w", err)
	}

	p.db = db
	p.cfg = cfg
	return nil
}

func (p *sqlitePlugin) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("sqlite: not connected")
	}
	return p.db.PingContext(ctx)
}

func (p *sqlitePlugin) Close() error {
	if p.db == nil {
		return nil
	}
	return p.db.Close()
}

func (p *sqlitePlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("sqlite: not connected")
	}
	start := time.Now()
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("sqlite: query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("sqlite: columns: %w", err)
	}

	var result [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("sqlite: scan: %w", err)
		}
		result = append(result, vals)
	}
	return &plugin.Result{
		Columns: cols,
		Rows:    result,
		RowsAffected: int64(len(result)),
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

func (p *sqlitePlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("sqlite: not connected")
	}
	start := time.Now()
	res, err := p.db.ExecContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("sqlite: exec: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		slog.Warn("sqlite: failed to get rows affected", "error", err)
	}
	return &plugin.Result{
		RowsAffected: affected,
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *sqlitePlugin) Tables(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("sqlite: not connected")
	}
	rows, err := p.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("sqlite: list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("sqlite: scan table: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (p *sqlitePlugin) Databases(ctx context.Context) ([]string, error) {
	return []string{p.cfg.Database}, nil
}

func (p *sqlitePlugin) CreateDatabase(ctx context.Context, name string) error {
	return fmt.Errorf("sqlite: creating databases is not supported")
}

func (p *sqlitePlugin) DropDatabase(ctx context.Context, name string) error {
	return fmt.Errorf("sqlite: dropping databases is not supported")
}

func (p *sqlitePlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.db == nil {
		return nil, fmt.Errorf("sqlite: not connected")
	}
	tables, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, tbl := range tables {
		info := plugin.TableInfo{Name: tbl}
		if err := p.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM \""+tbl+"\"").Scan(&info.RowCount); err != nil {
			slog.Warn("sqlite: failed to get row count", "table", tbl, "error", err)
		}

		rows, err := p.db.QueryContext(ctx, "PRAGMA table_info(\""+tbl+"\")")
		if err != nil {
			continue
		}

		for rows.Next() {
			var cid int
			var name, typ string
			var notNull int
			var def sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
				continue
			}
			info.Columns = append(info.Columns, plugin.ColumnInfo{
				Name:     name,
				Type:     typ,
				Nullable: notNull == 0,
				Primary:  pk == 1,
				Default:  def.String,
			})
		}
		rows.Close()

		schema.Tables = append(schema.Tables, info)
	}
	return &schema, nil
}
