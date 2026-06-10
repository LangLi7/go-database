package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypeRedis, func() plugin.DBPlugin { return &redisPlugin{} })
}

type redisPlugin struct {
	client *redis.Client
	cfg    plugin.Config
}

func (p *redisPlugin) Type() plugin.DBType { return plugin.TypeRedis }

func (p *redisPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	db := 0
	if d, ok := cfg.Params["db"]; ok {
		if _, err := fmt.Sscanf(d, "%d", &db); err != nil {
			slog.Warn("redis: invalid db parameter, using default", "value", d, "error", err)
		}
	}
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           db,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolSize:     10,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: ping: %w", err)
	}

	p.client = client
	p.cfg = cfg
	return nil
}

func (p *redisPlugin) Ping(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("redis: not connected")
	}
	return p.client.Ping(ctx).Err()
}

func (p *redisPlugin) Close() error {
	if p.client == nil {
		return nil
	}
	return p.client.Close()
}

func (p *redisPlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	start := time.Now()

	// Redis: first word is the command, rest are args
	args := splitArgs(q)
	if len(args) == 0 {
		return nil, fmt.Errorf("redis: empty command")
	}

	cmd := p.client.Do(ctx, args...)
	if cmd.Err() != nil {
		return nil, fmt.Errorf("redis: %w", cmd.Err())
	}

	val := cmd.Val()
	rows := flattenResult(val)

	return &plugin.Result{
		Columns: []string{"result"},
		Rows:    rows,
		RowsAffected: int64(len(rows)),
		Duration: time.Since(start).Milliseconds(),
	}, nil
}

func (p *redisPlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	return p.Query(ctx, q)
}

func (p *redisPlugin) Tables(ctx context.Context) ([]string, error) {
	if p.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	keys, err := p.client.Keys(ctx, "*").Result()
	if err != nil {
		return nil, fmt.Errorf("redis: keys: %w", err)
	}
	if keys == nil {
		return []string{}, nil
	}
	return keys, nil
}

func (p *redisPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.client == nil {
		return nil, fmt.Errorf("redis: not connected")
	}
	keys, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, key := range keys {
		info := plugin.TableInfo{Name: key}

		ttl := p.client.TTL(ctx, key).Val()
		_type := p.client.Type(ctx, key).Val()

		info.Columns = append(info.Columns, plugin.ColumnInfo{
			Name:     "key",
			Type:     _type,
			Nullable: false,
		})
		info.Columns = append(info.Columns, plugin.ColumnInfo{
			Name:     "ttl_seconds",
			Type:     "duration",
			Nullable: false,
			Default:  fmt.Sprintf("%.0f", ttl.Seconds()),
		})

		schema.Tables = append(schema.Tables, info)
	}
	return &schema, nil
}

func (p *redisPlugin) Databases(ctx context.Context) ([]string, error) {
	return []string{"db0"}, nil
}

func (p *redisPlugin) CreateDatabase(ctx context.Context, name string) error {
	return fmt.Errorf("redis: creating databases is not supported")
}

func (p *redisPlugin) DropDatabase(ctx context.Context, name string) error {
	return fmt.Errorf("redis: dropping databases is not supported")
}

func (p *redisPlugin) DB() int {
	db := 0
	if d, ok := p.cfg.Params["db"]; ok {
		if _, err := fmt.Sscanf(d, "%d", &db); err != nil {
			slog.Warn("redis: invalid db parameter in DB()", "value", d, "error", err)
		}
	}
	return db
}

// splitArgs splits a Redis command string into args
func splitArgs(cmd string) []any {
	args := make([]any, 0)
	current := ""
	inQuote := false
	for _, ch := range cmd {
		if ch == '"' || ch == '\'' {
			inQuote = !inQuote
			continue
		}
		if ch == ' ' && !inQuote {
			if current != "" {
				args = append(args, current)
				current = ""
			}
			continue
		}
		current += string(ch)
	}
	if current != "" {
		args = append(args, current)
	}
	return args
}

// flattenResult converts any Redis result to rows
func flattenResult(val any) [][]any {
	switch v := val.(type) {
	case nil:
		return [][]any{{nil}}
	case string:
		return [][]any{{v}}
	case int64:
		return [][]any{{v}}
	case float64:
		return [][]any{{v}}
	case bool:
		return [][]any{{v}}
	case []any:
		rows := make([][]any, len(v))
		for i, item := range v {
			rows[i] = []any{item}
		}
		return rows
	case map[any]any:
		rows := make([][]any, 0, len(v))
		for k, val := range v {
			rows = append(rows, []any{k, val})
		}
		return rows
	default:
		return [][]any{{fmt.Sprintf("%v", val)}}
	}
}
