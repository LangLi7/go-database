package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypeMySQL, func() plugin.DBPlugin { return &myPlugin{} })
}

type myPlugin struct {
	db  *sql.DB
	cfg plugin.Config
}

func (p *myPlugin) Type() plugin.DBType { return plugin.TypeMySQL }

func (p *myPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql: open: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("mysql: ping: %w", err)
	}

	p.db = db
	p.cfg = cfg
	return nil
}

func (p *myPlugin) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("mysql: not connected")
	}
	return p.db.PingContext(ctx)
}

func (p *myPlugin) Close() error {
	if p.db == nil {
		return nil
	}
	return p.db.Close()
}

func (p *myPlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mysql: not connected")
	}
	start := time.Now()
	rows, err := p.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("mysql: columns: %w", err)
	}

	var result [][]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("mysql: scan: %w", err)
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

func (p *myPlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mysql: not connected")
	}
	start := time.Now()
	res, err := p.db.ExecContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mysql: exec: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		slog.Warn("mysql: failed to get rows affected", "error", err)
	}
	return &plugin.Result{
		RowsAffected: affected,
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *myPlugin) Tables(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mysql: not connected")
	}
	rows, err := p.db.QueryContext(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_name")
	if err != nil {
		return nil, fmt.Errorf("mysql: list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("mysql: scan table: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (p *myPlugin) Databases(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mysql: not connected")
	}
	rows, err := p.db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("mysql: list databases: %w", err)
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var db string
		if err := rows.Scan(&db); err != nil {
			return nil, fmt.Errorf("mysql: scan db: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

func (p *myPlugin) CreateDatabase(ctx context.Context, name string) error {
	if p.db == nil {
		return fmt.Errorf("mysql: not connected")
	}
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", name))
	if err != nil {
		return fmt.Errorf("mysql: create database: %w", err)
	}
	return nil
}

func (p *myPlugin) DropDatabase(ctx context.Context, name string) error {
	if p.db == nil {
		return fmt.Errorf("mysql: not connected")
	}
	_, err := p.db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
	if err != nil {
		return fmt.Errorf("mysql: drop database: %w", err)
	}
	return nil
}

func (p *myPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mysql: not connected")
	}
	tables, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, tbl := range tables {
		info := plugin.TableInfo{Name: tbl}
		if err := p.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM `"+tbl+"`").Scan(&info.RowCount); err != nil {
			slog.Warn("mysql: failed to get row count", "table", tbl, "error", err)
		}

		rows, err := p.db.QueryContext(ctx, `
			SELECT column_name, column_type, is_nullable, COALESCE(column_default, '')
			FROM information_schema.columns
			WHERE table_name = ? AND table_schema = DATABASE()
			ORDER BY ordinal_position`, tbl)
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
