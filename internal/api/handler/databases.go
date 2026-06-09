package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
	"go-database/internal/plugin"
)

func CreateStandaloneDatabase(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Type string `json:"type" binding:"required"`
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "type and name required")
			return
		}

		_, ok := plugin.New(plugin.DBType(req.Type))
		if !ok {
			response.BadRequest(c, fmt.Sprintf("unsupported database type: %s", req.Type))
			return
		}

		storageDir := filepath.Join("database", "storage", req.Type)
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			response.InternalError(c, "failed to create storage directory")
			return
		}

		switch req.Type {
		case "sqlite":
			dbPath := filepath.Join(storageDir, req.Name+".db")
			conn, err := mgr.Add(c.Request.Context(), req.Name, "sqlite", "local", plugin.Config{
				Host:     dbPath,
				FilePath: dbPath,
				Database: req.Name,
			}, nil)
			if err != nil {
				response.InternalError(c, "failed to create connection: "+err.Error())
				return
			}

			if err := seedSampleSchema(mgr, conn.ID, req.Name, req.Type); err != nil {
				response.Success(c, gin.H{
					"connection": conn,
					"warning":    "database created but sample schema failed: " + err.Error(),
				})
				return
			}

			response.Created(c, gin.H{
				"connection": conn,
				"path":       dbPath,
				"type":       req.Type,
			})

		default:
			response.BadRequest(c, fmt.Sprintf("standalone creation not supported for type: %s", req.Type))
		}
	}
}

func seedSampleSchema(mgr *connection.Manager, connID, dbName, dbType string) error {
	samples := map[string]string{
		"sqlite": `CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS projects (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT,
	user_id INTEGER REFERENCES users(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO users (id, name, email) VALUES
(1, 'Alice Johnson', 'alice@example.com'),
(2, 'Bob Smith', 'bob@example.com');
INSERT OR IGNORE INTO projects (id, name, description, user_id) VALUES
(1, 'Website Redesign', 'Complete redesign of company website', 1),
(2, 'Mobile App', 'iOS and Android app development', 2);`,
		"postgres": `CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS projects (
	id SERIAL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	description TEXT,
	user_id INTEGER REFERENCES users(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO users (name, email) VALUES
('Alice Johnson', 'alice@example.com'),
('Bob Smith', 'bob@example.com')
ON CONFLICT DO NOTHING;
INSERT INTO projects (name, description, user_id) VALUES
('Website Redesign', 'Complete redesign of company website', 1),
('Mobile App', 'iOS and Android app development', 2)
ON CONFLICT DO NOTHING;`,
		"mysql": `CREATE TABLE IF NOT EXISTS users (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS projects (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	description TEXT,
	user_id INT REFERENCES users(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT IGNORE INTO users (name, email) VALUES
('Alice Johnson', 'alice@example.com'),
('Bob Smith', 'bob@example.com');
INSERT IGNORE INTO projects (name, description, user_id) VALUES
('Website Redesign', 'Complete redesign of company website', 1),
('Mobile App', 'iOS and Android app development', 2);`,
		"mariadb": `CREATE TABLE IF NOT EXISTS users (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	email VARCHAR(255) UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS projects (
	id INT AUTO_INCREMENT PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	description TEXT,
	user_id INT REFERENCES users(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
INSERT IGNORE INTO users (name, email) VALUES
('Alice Johnson', 'alice@example.com'),
('Bob Smith', 'bob@example.com');
INSERT IGNORE INTO projects (name, description, user_id) VALUES
('Website Redesign', 'Complete redesign of company website', 1),
('Mobile App', 'iOS and Android app development', 2);`,
	}

	sql, ok := samples[dbType]
	if !ok {
		return fmt.Errorf("no sample schema for type: %s", dbType)
	}

	// Execute seed SQL in batches per statement
	stmts := splitSQL(sql)
	ctx := context.Background()
	for _, stmt := range stmts {
		if stmt == "" {
			continue
		}
		if _, err := mgr.Execute(ctx, connID, stmt); err != nil {
			return fmt.Errorf("seed error: %w", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

func splitSQL(sql string) []string {
	var stmts []string
	current := ""
	for _, ch := range sql {
		current += string(ch)
		if ch == ';' {
			stmts = append(stmts, current)
			current = ""
		}
	}
	if current != "" {
		stmts = append(stmts, current)
	}
	return stmts
}
