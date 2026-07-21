package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-database/internal/plugin"
)

func init() {
	plugin.Register(plugin.TypeMongoDB, func() plugin.DBPlugin { return &mongoPlugin{} })
}

type mongoPlugin struct {
	client *mongo.Client
	db     *mongo.Database
	cfg    plugin.Config
}

func (p *mongoPlugin) Type() plugin.DBType { return plugin.TypeMongoDB }

func (p *mongoPlugin) Connect(ctx context.Context, cfg plugin.Config) error {
	uri := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	clientOpts := options.Client().ApplyURI(uri).
		SetMaxPoolSize(10).
		SetConnectTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("mongodb: connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb: ping: %w", err)
	}

	p.client = client
	p.db = client.Database(cfg.Database)
	p.cfg = cfg
	return nil
}

func (p *mongoPlugin) Ping(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("mongodb: not connected")
	}
	return p.client.Ping(ctx, nil)
}

func (p *mongoPlugin) Close() error {
	if p.client == nil {
		return nil
	}
	return p.client.Disconnect(context.Background())
}

func (p *mongoPlugin) Query(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mongodb: not connected")
	}
	start := time.Now()

	var pipeline bson.A
	if err := bson.UnmarshalExtJSON([]byte(q), true, &pipeline); err != nil {
		return nil, fmt.Errorf("mongodb: invalid aggregation JSON: %w", err)
	}

	cursor, err := p.db.RunCommandCursor(ctx, bson.D{
		{Key: "aggregate", Value: ""},
		{Key: "pipeline", Value: pipeline},
		{Key: "cursor", Value: bson.D{}},
	})
	if err != nil {
		return nil, fmt.Errorf("mongodb: query: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("mongodb: decode: %w", err)
	}

	var columns []string
	var result [][]any
	if len(docs) > 0 {
		for k := range docs[0] {
			columns = append(columns, k)
		}
		for _, doc := range docs {
			row := make([]any, len(columns))
			for i, c := range columns {
				row[i] = doc[c]
			}
			result = append(result, row)
		}
	}

	return &plugin.Result{
		Columns:      columns,
		Rows:         result,
		RowsAffected: int64(len(result)),
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *mongoPlugin) Execute(ctx context.Context, q string) (*plugin.Result, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mongodb: not connected")
	}
	start := time.Now()

	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(q), true, &cmd); err != nil {
		return nil, fmt.Errorf("mongodb: invalid command JSON: %w", err)
	}

	res := p.db.RunCommand(ctx, cmd)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("mongodb: exec: %w", err)
	}

	return &plugin.Result{
		RowsAffected: 1,
		Duration:     time.Since(start).Milliseconds(),
	}, nil
}

func (p *mongoPlugin) Tables(ctx context.Context) ([]string, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mongodb: not connected")
	}
	cols, err := p.db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("mongodb: list collections: %w", err)
	}
	return cols, nil
}

func (p *mongoPlugin) Databases(ctx context.Context) ([]string, error) {
	if p.client == nil {
		return nil, fmt.Errorf("mongodb: not connected")
	}
	return p.client.ListDatabaseNames(ctx, bson.M{})
}

func (p *mongoPlugin) CreateDatabase(ctx context.Context, name string) error {
	if p.client == nil {
		return fmt.Errorf("mongodb: not connected")
	}
	db := p.client.Database(name)
	err := db.CreateCollection(ctx, "_init_")
	if err != nil {
		return fmt.Errorf("mongodb: create database: %w", err)
	}
	return db.Collection("_init_").Drop(ctx)
}

func (p *mongoPlugin) DropDatabase(ctx context.Context, name string) error {
	if p.client == nil {
		return fmt.Errorf("mongodb: not connected")
	}
	return p.client.Database(name).Drop(ctx)
}

func (p *mongoPlugin) Schema(ctx context.Context) (*plugin.Schema, error) {
	if p.db == nil {
		return nil, fmt.Errorf("mongodb: not connected")
	}
	cols, err := p.Tables(ctx)
	if err != nil {
		return nil, err
	}

	var schema plugin.Schema
	for _, col := range cols {
		info := plugin.TableInfo{Name: col}

		count, err := p.db.Collection(col).CountDocuments(ctx, bson.M{})
		if err != nil {
			slog.Warn("mongodb: failed to count documents", "collection", col, "error", err)
		} else {
			info.RowCount = count
		}

		// Sample one document to infer schema
		cursor := p.db.Collection(col).FindOne(ctx, bson.M{})
		var doc bson.M
		if err := cursor.Decode(&doc); err == nil {
			for k, v := range doc {
				colInfo := plugin.ColumnInfo{
					Name:     k,
					Type:     fmt.Sprintf("%T", v),
					Nullable: v == nil,
				}
				if k == "_id" {
					colInfo.Primary = true
				}
				info.Columns = append(info.Columns, colInfo)
			}
		}
		schema.Tables = append(schema.Tables, info)
	}
	return &schema, nil
}
