package provisioner

import (
	"context"
	dbsql "database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"

	"go-database/internal/plugin"
)

type embeddedPostgres struct {
	instance *embeddedpostgres.EmbeddedPostgres
	started  bool
}

func newEmbeddedPostgres() *embeddedPostgres {
	return &embeddedPostgres{}
}

func (e *embeddedPostgres) Start(ctx context.Context) (*plugin.Config, error) {
	if e.started {
		return &plugin.Config{Host: "127.0.0.1", Port: 5432, Database: "postgres", User: "postgres", Password: "postgres"}, nil
	}

	if cfg, err := tryExistingPG(ctx); err == nil {
		slog.Info("provisioner: using existing postgres on :5432")
		e.started = true
		return cfg, nil
	}

	port := uint32(5432)
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(port).
			Database("postgres").
			Username("postgres").
			Password("postgres").
			StartTimeout(120*time.Second),
	)

	if err := pg.Start(); err != nil {
		return nil, fmt.Errorf("embedded postgres: %w", err)
	}

	e.instance = pg
	e.started = true
	slog.Info("provisioner: embedded postgres started on :5432")
	return &plugin.Config{Host: "127.0.0.1", Port: 5432, Database: "postgres", User: "postgres", Password: "postgres"}, nil
}

func tryExistingPG(ctx context.Context) (*plugin.Config, error) {
	addr := "127.0.0.1:5432"
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return nil, err
	}
	conn.Close()

	candidates := []struct{ user, password string }{
		{"postgres", "postgres"},
		{"postgres", ""},
		{"postgres", "admin"},
	}
	for _, c := range candidates {
		dsn := fmt.Sprintf("postgres://%s:%s@127.0.0.1:5432/postgres?sslmode=disable&connect_timeout=3", c.user, c.password)
		db, err := dbsql.Open("pgx", dsn)
		if err != nil {
			continue
		}
		ctxPing, cancel := context.WithTimeout(ctx, 3*time.Second)
		pingErr := db.PingContext(ctxPing)
		cancel()
		db.Close()
		if pingErr == nil {
			return &plugin.Config{Host: "127.0.0.1", Port: 5432, Database: "postgres", User: c.user, Password: c.password}, nil
		}
	}

	return nil, fmt.Errorf("port 5432 in use but cannot authenticate")
}

func (e *embeddedPostgres) Shutdown(ctx context.Context) {
	if e.instance != nil {
		if err := e.instance.Stop(); err != nil {
			slog.Warn("provisioner: embedded postgres stop error", "error", err)
		}
		e.started = false
		e.instance = nil
		slog.Info("provisioner: embedded postgres stopped")
	}
}

// ------------------------------------------------------------

type embeddedMySQL struct {
	instances map[int]*mysqlInstance
}

type mysqlInstance struct {
	server *server.Server
	engine *sqle.Engine
}

func newEmbeddedMySQL() *embeddedMySQL {
	return &embeddedMySQL{instances: make(map[int]*mysqlInstance)}
}

func (e *embeddedMySQL) Start(ctx context.Context, port int, dbName string) (*plugin.Config, error) {
	if _, ok := e.instances[port]; ok {
		return &plugin.Config{Host: "127.0.0.1", Port: port, Database: dbName, User: "root", Password: ""}, nil
	}

	db := memory.NewDatabase(dbName)
	db.BaseDatabase.EnablePrimaryKeyIndexes()
	pro := memory.NewDBProvider(db)

	engine := sqle.NewDefault(pro)
	address := fmt.Sprintf("127.0.0.1:%d", port)

	srv, err := server.NewServer(
		server.Config{
			Protocol: "tcp",
			Address:  address,
		},
		engine,
		sql.NewContext,
		memory.NewSessionBuilder(pro),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("embedded mysql: create server: %w", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			slog.Warn("provisioner: embedded mysql server exited", "port", port, "error", err)
		}
	}()

	if err := waitForPort(ctx, "127.0.0.1", port, 10*time.Second); err != nil {
		srv.Close()
		engine.Close()
		return nil, err
	}

	e.instances[port] = &mysqlInstance{server: srv, engine: engine}
	slog.Info(fmt.Sprintf("provisioner: embedded mysql started on :%d", port))
	return &plugin.Config{Host: "127.0.0.1", Port: port, Database: dbName, User: "root", Password: ""}, nil
}

func (e *embeddedMySQL) Shutdown(ctx context.Context) {
	for port, inst := range e.instances {
		inst.server.Close()
		inst.server.SessionManager().WaitForClosedConnections()
		inst.engine.Close()
		delete(e.instances, port)
		slog.Info(fmt.Sprintf("provisioner: embedded mysql on :%d stopped", port))
	}
}

func waitForPort(ctx context.Context, host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}
