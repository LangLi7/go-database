package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/microsoft/go-mssqldb"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypeMSSQL, func() plugin.DBPlugin { return &msPlugin{} })
}

type msPlugin struct {
	db  *sql.DB
	cfg plugin.Config
}

func (p *msPlugin) Type() plugin.DBType { return plugin.TypeMSSQL }

func (p *msPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	// go-mssqldb connection URL: sqlserver://user:pass@host:port?database=name
	// Build DSN from config.
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d",
		cfg.User, cfg.Password, cfg.Host, portOrDefault(cfg.Port, 1433))
	if cfg.Database != "" {
		dsn += "?database=" + cfg.Database
	}
	if !cfg.SSL {
		dsn += param(dsn, "encrypt", "disable")
	}

	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return fmt.Errorf("mssql: open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("mssql: ping: %w", err)
	}

	p.db = db
	p.cfg = cfg
	return nil
}

func (p *msPlugin) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("mssql: not connected")
	}
	return p.db.PingContext(ctx)
}

func (p *msPlugin) Close() error {
	if p.db == nil {
		return nil
	}
	return p.db.Close()
}

func (p *msPlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mssql: not connected")
	}
	start := time.Now()
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mssql: query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("mssql: columns: %w", err)
	}

	var result [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("mssql: scan: %w", err)
		}
		result = append(result, vals)
	}

	return &plugin.Result{
		Columns:      cols,
		Rows:         result,
		RowsAffected: int64(len(result)),
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *msPlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mssql: not connected")
	}
	start := time.Now()
	res, err := p.db.ExecContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mssql: exec: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		slog.Warn("mssql: failed to get rows affected", "error", err)
	}
	return &plugin.Result{
		RowsAffected: affected,
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *msPlugin) Tables(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mssql: not connected")
	}
	rows, err := p.db.QueryContext(ctx,
		"SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME")
	if err != nil {
		return nil, fmt.Errorf("mssql: list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("mssql: scan table: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (p *msPlugin) Databases(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mssql: not connected")
	}
	rows, err := p.db.QueryContext(ctx,
		"SELECT name FROM sys.databases WHERE database_id > 4 ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("mssql: list databases: %w", err)
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var db string
		if err := rows.Scan(&db); err != nil {
			return nil, fmt.Errorf("mssql: scan db: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

func (p *msPlugin) CreateDatabase(ctx context.Context, name string) error {
	if p.db == nil {
		return fmt.Errorf("mssql: not connected")
	}
	// SQL Server: CREATE DATABASE [name]
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE [%s]", sanitizeIdent(name)))
	if err != nil {
		return fmt.Errorf("mssql: create database: %w", err)
	}
	return nil
}

func (p *msPlugin) DropDatabase(ctx context.Context, name string) error {
	if p.db == nil {
		return fmt.Errorf("mssql: not connected")
	}
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS [%s]", sanitizeIdent(name)))
	if err != nil {
		return fmt.Errorf("mssql: drop database: %w", err)
	}
	return nil
}

func (p *msPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mssql: not connected")
	}
	tables, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, tbl := range tables {
		info := plugin.TableInfo{Name: tbl}
		if err := p.db.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM [%s]", sanitizeIdent(tbl))).Scan(&info.RowCount); err != nil {
			slog.Warn("mssql: failed to get row count", "table", tbl, "error", err)
		}

		rows, err := p.db.QueryContext(ctx, `
			SELECT c.COLUMN_NAME, c.DATA_TYPE, c.IS_NULLABLE, COALESCE(c.COLUMN_DEFAULT, '')
			FROM INFORMATION_SCHEMA.COLUMNS c
			WHERE c.TABLE_NAME = @p1
			ORDER BY c.ORDINAL_POSITION`, tbl)
		if err != nil {
			continue
		}

		for rows.Next() {
			var name, typ, nullable, def string
			if err := rows.Scan(&name, &typ, &nullable, &def); err != nil {
				continue
			}
			info.Columns = append(info.Columns, plugin.ColumnInfo{
				Name:     name,
				Type:     typ,
				Nullable: nullable == "YES",
				Default:  def,
			})
		}
		rows.Close()

		schema.Tables = append(schema.Tables, info)
	}
	return &schema, nil
}

func portOrDefault(port, def int) int {
	if port == 0 {
		return def
	}
	return port
}

func param(dsn, key, val string) string {
	sep := "?"
	if containsRune(dsn, '?') {
		sep = "&"
	}
	return sep + key + "=" + val
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// sanitizeIdent strips characters invalid in SQL Server identifiers.
func sanitizeIdent(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		}
	}
	return string(out)
}
