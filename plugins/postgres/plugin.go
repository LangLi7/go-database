package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypePostgres, func() plugin.DBPlugin { return &pgPlugin{} })
}

type pgPlugin struct {
	pool *pgxpool.Pool
	cfg  plugin.Config
}

func (p *pgPlugin) Type() plugin.DBType { return plugin.TypePostgres }

func (p *pgPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	if cfg.SSL {
		dsn += "?sslmode=require"
	} else {
		dsn += "?sslmode=disable"
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("postgres: parse config: %w", err)
	}
	poolCfg.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("postgres: connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("postgres: ping: %w", err)
	}

	p.pool = pool
	p.cfg = cfg
	return nil
}

func (p *pgPlugin) Ping(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	return p.pool.Ping(ctx)
}

func (p *pgPlugin) Close() error {
	if p.pool == nil {
		return nil
	}
	p.pool.Close()
	return nil
}

func (p *pgPlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("postgres: not connected")
	}
	start := time.Now()
	rows, err := p.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: query: %w", err)
	}
	defer rows.Close()

	cols := rows.FieldDescriptions()
	columns := make([]string, len(cols))
	for i, f := range cols {
		columns[i] = string(f.Name)
	}

	var result [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("postgres: scan: %w", err)
		}
		result = append(result, vals)
	}

	return &plugin.Result{
		Columns: columns,
		Rows:    result,
		RowsAffected: int64(len(result)),
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

func (p *pgPlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("postgres: not connected")
	}
	start := time.Now()
	tag, err := p.pool.Exec(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: exec: %w", err)
	}
	return &plugin.Result{
		RowsAffected: tag.RowsAffected(),
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *pgPlugin) Tables(ctx context.Context) ([]string, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("postgres: not connected")
	}
	rows, err := p.pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("postgres: scan table: %w", err)
		}
		tables = append(tables, t)
	}
	return tables, nil
}

func (p *pgPlugin) Databases(ctx context.Context) ([]string, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("postgres: not connected")
	}
	rows, err := p.pool.Query(ctx,
		"SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, fmt.Errorf("postgres: list databases: %w", err)
	}
	defer rows.Close()
	var dbs []string
	for rows.Next() {
		var db string
		if err := rows.Scan(&db); err != nil {
			return nil, fmt.Errorf("postgres: scan db: %w", err)
		}
		dbs = append(dbs, db)
	}
	return dbs, nil
}

func (p *pgPlugin) CreateDatabase(ctx context.Context, name string) error {
	if p.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	_, err := p.pool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, name))
	if err != nil {
		return fmt.Errorf("postgres: create database: %w", err)
	}
	return nil
}

func (p *pgPlugin) DropDatabase(ctx context.Context, name string) error {
	if p.pool == nil {
		return fmt.Errorf("postgres: not connected")
	}
	_, err := p.pool.Exec(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, name))
	if err != nil {
		return fmt.Errorf("postgres: drop database: %w", err)
	}
	return nil
}

func (p *pgPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("postgres: not connected")
	}
	tables, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, tbl := range tables {
		info := plugin.TableInfo{Name: tbl}
		if err := p.pool.QueryRow(ctx,
			"SELECT reltuples::bigint FROM pg_class WHERE relname = $1", tbl).Scan(&info.RowCount); err != nil {
			slog.Warn("failed to get table row count", "table", tbl, "error", err)
		}

		rows, err := p.pool.Query(ctx, `
			SELECT column_name, data_type, is_nullable,
			       COALESCE(character_maximum_length::text, '') as col_len,
			       COALESCE(column_default, '') as col_default
			FROM information_schema.columns
			WHERE table_name = $1 AND table_schema = 'public'
			ORDER BY ordinal_position`, tbl)
		if err != nil {
			continue
		}

		for rows.Next() {
			var name, typ, nullable, def string
			if err := rows.Scan(&name, &typ, &nullable, new(string), &def); err != nil {
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
