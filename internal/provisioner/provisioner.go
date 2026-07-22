package provisioner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"go-database/internal/connection"
	"go-database/internal/plugin"
	"time"
)

var defaultDBs = []struct {
	Type   plugin.DBType
	Name   string
	Source string
}{
	{plugin.TypePostgres, "postgres-dev", "docker"},
	{plugin.TypeMySQL, "mysql-dev", "docker"},
	{plugin.TypeMariaDB, "mariadb-dev", "docker"},
	{plugin.TypeMongoDB, "mongodb-dev", "docker"},
	{plugin.TypeRedis, "redis-dev", "docker"},
}

// Provisioner auto-starts databases for development/testing
type Provisioner struct {
	mu          sync.Mutex
	mgr         *connection.Manager
	docker      *dockerProvisioner
	embeddedPG  *embeddedPostgres
	embeddedMY  *embeddedMySQL
	provisioned []string
}

// New creates a provisioner and auto-starts missing databases
func New(ctx context.Context, mgr *connection.Manager) *Provisioner {
	p := &Provisioner{
		mgr: mgr,
	}

	dockerAvailable := checkDocker()
	if dockerAvailable {
		p.docker = newDockerProvisioner()
		slog.Info("provisioner: docker available, will auto-start containers")
	} else {
		slog.Info("provisioner: docker not available, trying embedded servers")
		p.embeddedPG = newEmbeddedPostgres()
		p.embeddedMY = newEmbeddedMySQL()
	}

	provCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, db := range defaultDBs {
		wg.Add(1)
		db := db // capture
		go func() {
			defer wg.Done()
			if err := p.provision(provCtx, db.Type, db.Name, db.Source); err != nil {
				slog.Warn("provisioner: could not start "+string(db.Type), "error", err)
			}
		}()
	}
	wg.Wait()

	return p
}

func (p *Provisioner) provision(ctx context.Context, typ plugin.DBType, name, source string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cfg, err := p.startContainer(ctx, typ)
	if err != nil {
		return fmt.Errorf("%s: %w", typ, err)
	}

	conn, err := p.mgr.Add(ctx, name, typ, source, *cfg, []string{"auto-provisioned"}, "")
	if err != nil {
		return fmt.Errorf("register %s: %w", typ, err)
	}

	// Seed sample table if database is empty
	if err := p.seedSampleTable(ctx, conn.ID, typ); err != nil {
		slog.Warn("provisioner: seed failed for "+string(typ), "error", err)
	}

	p.provisioned = append(p.provisioned, conn.ID)
	slog.Info("provisioner: "+string(typ)+" ready", "id", conn.ID, "name", name)
	return nil
}

func (p *Provisioner) startContainer(ctx context.Context, typ plugin.DBType) (*plugin.Config, error) {
	if p.docker != nil {
		cfg, err := p.docker.Start(ctx, typ)
		if err == nil {
			return cfg, nil
		}
		slog.Warn("provisioner: docker failed for "+string(typ), "error", err)
	}

	switch typ {
	case plugin.TypePostgres:
		if p.embeddedPG != nil {
			return p.embeddedPG.Start(ctx)
		}
	case plugin.TypeMySQL:
		if p.embeddedMY != nil {
			return p.embeddedMY.Start(ctx, 3306, "test")
		}
	case plugin.TypeMariaDB:
		if p.embeddedMY != nil {
			return p.embeddedMY.Start(ctx, 3307, "test")
		}
	}

	return nil, fmt.Errorf("no provisioner available")
}

// Shutdown stops all provisioned databases
func (p *Provisioner) Shutdown(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, id := range p.provisioned {
		if err := p.mgr.Remove(id); err != nil {
			slog.Warn("provisioner: remove failed", "id", id, "error", err)
		}
	}

	if p.docker != nil {
		p.docker.Shutdown(ctx)
	}
	if p.embeddedPG != nil {
		p.embeddedPG.Shutdown(ctx)
	}
	if p.embeddedMY != nil {
		p.embeddedMY.Shutdown(ctx)
	}
	slog.Info("provisioner: all databases stopped")
}

// ProvisionedIDs returns the IDs of auto-provisioned connections
func (p *Provisioner) ProvisionedIDs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := make([]string, len(p.provisioned))
	copy(r, p.provisioned)
	return r
}

func (p *Provisioner) seedSampleTable(ctx context.Context, connID string, typ plugin.DBType) error {
	if typ == plugin.TypeRedis || typ == plugin.TypeMongoDB {
		return nil
	}

	schema, err := p.mgr.Schema(ctx, connID)
	if err != nil {
		return err
	}
	if len(schema.Tables) > 0 {
		return nil
	}

	var stmts []string
	switch typ {
	case plugin.TypePostgres:
		stmts = []string{
			`CREATE TABLE IF NOT EXISTS sample_data (
				id SERIAL PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(200),
				status VARCHAR(20) DEFAULT 'active',
				score INTEGER DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`INSERT INTO sample_data (name, email, status, score) VALUES
				('Alice Johnson', 'alice@example.com', 'active', 95),
				('Bob Smith', 'bob@example.com', 'active', 82),
				('Charlie Brown', 'charlie@example.com', 'inactive', 47)`,
		}
	default:
		stmts = []string{
			`CREATE TABLE IF NOT EXISTS sample_data (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(100) NOT NULL,
				email VARCHAR(200),
				status VARCHAR(20) DEFAULT 'active',
				score INT DEFAULT 0,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`INSERT INTO sample_data (name, email, status, score) VALUES
				('Alice Johnson', 'alice@example.com', 'active', 95),
				('Bob Smith', 'bob@example.com', 'active', 82),
				('Charlie Brown', 'charlie@example.com', 'inactive', 47)`,
		}
	}

	for _, stmt := range stmts {
		if _, err := p.mgr.Execute(ctx, connID, stmt); err != nil {
			slog.Warn("provisioner: seed query failed", "conn", connID, "error", err)
			return err
		}
	}
	slog.Info("provisioner: seeded sample table for "+string(typ), "conn", connID)
	return nil
}
