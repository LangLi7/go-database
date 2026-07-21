package plugin_test

import (
	"testing"

	"go-database/internal/plugin"

	_ "go-database/plugins/mariadb"
	_ "go-database/plugins/mongodb"
	_ "go-database/plugins/mssql"
	_ "go-database/plugins/mysql"
	_ "go-database/plugins/postgres"
	_ "go-database/plugins/redis"
	_ "go-database/plugins/sqlite"
)

func TestDetectType(t *testing.T) {
	tests := []struct {
		name string
		cfg  plugin.Config
		want plugin.DBType
		ok   bool
	}{
		{"postgres dsn", plugin.Config{Params: map[string]string{"dsn": "postgres://localhost/db"}}, plugin.TypePostgres, true},
		{"mongodb dsn", plugin.Config{Params: map[string]string{"dsn": "mongodb+srv://c/cluster"}}, plugin.TypeMongoDB, true},
		{"sqlserver dsn", plugin.Config{Params: map[string]string{"dsn": "sqlserver://host:1433"}}, plugin.TypeMSSQL, true},
		{"mysql host prefix", plugin.Config{Host: "mysql://h"}, plugin.TypeMySQL, true},
		{"port 5432", plugin.Config{Port: 5432}, plugin.TypePostgres, true},
		{"port 1433", plugin.Config{Port: 1433}, plugin.TypeMSSQL, true},
		{"port 27017", plugin.Config{Port: 27017}, plugin.TypeMongoDB, true},
		{"port 6379", plugin.Config{Port: 6379}, plugin.TypeRedis, true},
		{"port 9200", plugin.Config{Port: 9200}, plugin.TypeElastic, true},
		{"port 8123", plugin.Config{Port: 8123}, plugin.TypeClickHouse, true},
		{"sqlite filepath", plugin.Config{FilePath: "/tmp/x.db"}, plugin.TypeSQLite, true},
		{"unknown", plugin.Config{Host: "10.0.0.5", Port: 9999}, plugin.TypeAuto, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := plugin.DetectType(tt.cfg)
			if ok != tt.ok || got != tt.want {
				t.Errorf("DetectType(%+v) = (%q, %v), want (%q, %v)", tt.cfg, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	for _, typ := range []plugin.DBType{
		plugin.TypePostgres, plugin.TypeMySQL, plugin.TypeMariaDB,
		plugin.TypeSQLite, plugin.TypeMongoDB, plugin.TypeRedis, plugin.TypeMSSQL,
	} {
		if !plugin.IsSupported(typ) {
			t.Errorf("expected %q to be a registered (supported) plugin", typ)
		}
	}
	if plugin.IsSupported(plugin.TypeAuto) {
		t.Error("TypeAuto must not resolve to a concrete plugin")
	}
	if plugin.IsSupported(plugin.TypeOracle) {
		t.Error("oracle not implemented yet — should be unsupported")
	}
}
