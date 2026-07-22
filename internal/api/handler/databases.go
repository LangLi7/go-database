package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		if err := c.BindJSON(&req); err != nil {
			response.BadRequest(c, "type and name required: "+err.Error())
			return
		}

		_, ok := plugin.New(plugin.DBType(req.Type))
		if !ok {
			response.BadRequest(c, fmt.Sprintf("unsupported database type: %s", req.Type))
			return
		}

		dbType := strings.ToLower(req.Type)
		storageDir := filepath.Join("database", "storage", dbType)
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			response.InternalError(c, "cannot create storage directory")
			return
		}

		var (
			conn   *connection.Connection
			cfg    plugin.Config
			source string
		)

		switch dbType {
		case "sqlite":
			dbPath := filepath.Join(storageDir, req.Name+".db")
			cfg = plugin.Config{
				Host:     dbPath,
				FilePath: dbPath,
				Database: req.Name,
			}
			source = "local"
			var err error
			conn, err = mgr.Add(c.Request.Context(), req.Name, "sqlite", source, cfg, nil, userIDFrom(c))
			if err != nil {
				response.InternalError(c, "failed to create connection: "+err.Error())
				return
			}

		default:
			host := "localhost"
			port := defaultPort(dbType)
			// For MySQL/MariaDB: first connect to default DB, create the new database, then switch
			defaultDB := defaultDBName(dbType)
			cfg = plugin.Config{
				Host:     host,
				Port:     port,
				Database: defaultDB,
				User:     defaultUser(dbType),
				Password: "",
			}
			// Connect to default DB first to CREATE DATABASE
			tmpConn, err := mgr.Add(c.Request.Context(), req.Name+"-tmp", plugin.DBType(dbType), source, cfg, nil, userIDFrom(c))
			if err != nil {
				response.InternalError(c, "failed to connect: "+err.Error())
				return
			}
			// Try creating the database (ignore "already exists" errors)
			if _, err := mgr.Execute(c.Request.Context(), tmpConn.ID, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteIdentMySQL(req.Name))); err != nil {
				mgr.Remove(tmpConn.ID)
				response.InternalError(c, "failed to create database: "+err.Error())
				return
			}
			mgr.Remove(tmpConn.ID)

			// Now connect to the new database
			cfg.Database = req.Name
			conn, err = mgr.Add(c.Request.Context(), req.Name, plugin.DBType(dbType), source, cfg, nil, userIDFrom(c))
			if err != nil {
				response.InternalError(c, "failed to create connection: "+err.Error())
				return
			}
		}

		// Seed sample schema (only for SQLite, other types need running server)
		if err := seedSampleSchema(c.Request.Context(), mgr, conn.ID, dbType); err != nil {
			response.Success(c, gin.H{
				"connection": conn,
				"warning":    "connection created but sample schema could not be loaded: " + err.Error(),
				"hint":       fmt.Sprintf("Make sure %s is running on localhost:%d, then use the query editor to run your own schema.", dbType, defaultPort(dbType)),
			})
			return
		}

		response.Created(c, gin.H{
			"connection": conn,
			"type":       dbType,
			"path":       fmt.Sprintf("database/storage/%s/%s.db", dbType, req.Name),
		})
	}
}

func defaultPort(dbType string) int {
	switch dbType {
	case "postgres":
		return 5432
	case "mysql":
		return 3306
	case "mariadb":
		return 3307
	case "mongodb":
		return 27017
	case "redis":
		return 6379
	default:
		return 0
	}
}

func defaultUser(dbType string) string {
	switch dbType {
	case "postgres":
		return "postgres"
	case "mysql", "mariadb":
		return "root"
	case "mongodb":
		return ""
	default:
		return ""
	}
}

func seedSampleSchema(ctx context.Context, mgr *connection.Manager, connID, dbType string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := mgr.Ping(ctx, connID); err != nil {
		return fmt.Errorf("server not reachable: %w", err)
	}

	samples := map[string]string{
		"sqlite": `CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, email TEXT UNIQUE NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS projects (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT, user_id INTEGER REFERENCES users(id), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT OR IGNORE INTO users (id, name, email) VALUES (1, 'Alice Johnson', 'alice@example.com'), (2, 'Bob Smith', 'bob@example.com');
INSERT OR IGNORE INTO projects (id, name, description, user_id) VALUES (1, 'Website Redesign', 'Complete redesign of company website', 1), (2, 'Mobile App', 'iOS and Android app development', 2);`,
		"postgres": `CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS projects (id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, description TEXT, user_id INTEGER REFERENCES users(id), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com'), ('Bob Smith', 'bob@example.com') ON CONFLICT DO NOTHING;
INSERT INTO projects (name, description, user_id) VALUES ('Website Redesign', 'Complete redesign of company website', 1), ('Mobile App', 'iOS and Android app development', 2) ON CONFLICT DO NOTHING;`,
		"mysql": `CREATE TABLE IF NOT EXISTS users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS projects (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, description TEXT, user_id INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT IGNORE INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com'), ('Bob Smith', 'bob@example.com');
INSERT IGNORE INTO projects (name, description, user_id) VALUES ('Website Redesign', 'Complete redesign of company website', 1), ('Mobile App', 'iOS and Android app development', 2);`,
		"mariadb": `CREATE TABLE IF NOT EXISTS users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS projects (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, description TEXT, user_id INT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP);
INSERT IGNORE INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com'), ('Bob Smith', 'bob@example.com');
INSERT IGNORE INTO projects (name, description, user_id) VALUES ('Website Redesign', 'Complete redesign of company website', 1), ('Mobile App', 'iOS and Android app development', 2);`,
		"mongodb": `db.users.insertMany([{name:'Alice Johnson',email:'alice@example.com',created_at:new Date()},{name:'Bob Smith',email:'bob@example.com',created_at:new Date()}]);`,
		"redis":   `SET user:1:name "Alice Johnson"\r\nSET user:2:name "Bob Smith"`,
	}

	sql, ok := samples[dbType]
	if !ok {
		return fmt.Errorf("no sample schema for type: %s", dbType)
	}

	stmts := splitSQL(sql)
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := mgr.Execute(ctx, connID, stmt); err != nil {
			return fmt.Errorf("seed error: %w", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

func defaultDBName(dbType string) string {
	switch dbType {
	case "postgres":
		return "postgres"
	case "mysql", "mariadb":
		return "test"
	default:
		return ""
	}
}

func quoteIdentMySQL(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
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
	if strings.TrimSpace(current) != "" {
		stmts = append(stmts, current)
	}
	return stmts
}
