package provisioner

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"go-database/internal/plugin"
)

var dockerImages = map[plugin.DBType]dockerDef{
	plugin.TypePostgres: {Image: "postgres:16-alpine", Port: 5432, Env: []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=", "POSTGRES_DB=postgres"}},
	plugin.TypeMySQL:    {Image: "mysql:8.4-oraclelinux8", Port: 3306, Env: []string{"MYSQL_ALLOW_EMPTY_PASSWORD=yes", "MYSQL_DATABASE=test"}},
	plugin.TypeMariaDB:  {Image: "mariadb:11.4", Port: 3307, Env: []string{"MYSQL_ALLOW_EMPTY_PASSWORD=yes", "MYSQL_DATABASE=test"}},
	plugin.TypeMongoDB:  {Image: "mongo:7", Port: 27017, Env: []string{"MONGO_INITDB_DATABASE=test"}},
	plugin.TypeRedis:    {Image: "redis:7-alpine", Port: 6379, Env: nil},
}

type dockerDef struct {
	Image string
	Port  int
	Env   []string
}

type dockerProvisioner struct {
	started map[plugin.DBType]bool
}

func newDockerProvisioner() *dockerProvisioner {
	return &dockerProvisioner{started: make(map[plugin.DBType]bool)}
}

func checkDocker() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func (d *dockerProvisioner) Start(ctx context.Context, typ plugin.DBType) (*plugin.Config, error) {
	def, ok := dockerImages[typ]
	if !ok {
		return nil, fmt.Errorf("unsupported type %q", typ)
	}

	containerName := "go-db-" + string(typ)

	if d.started[typ] {
		return d.config(typ), nil
	}

	if err := d.ensureImage(ctx, def.Image); err != nil {
		return nil, err
	}

	existing, err := d.containerStatus(ctx, containerName)
	if err == nil {
		switch existing {
		case "running":
			slog.Info("provisioner: container already running", "name", containerName)
			d.started[typ] = true
			return d.config(typ), nil
		case "exited", "paused":
			slog.Info("provisioner: restarting existing container", "name", containerName)
			exec.CommandContext(ctx, "docker", "start", containerName).Run()
			d.started[typ] = true
			_ = d.waitForPort(ctx, typ, def.Port, 30*time.Second)
			return d.config(typ), nil
		}
	}

	args := []string{"run", "-d", "--name", containerName, "--rm"}
	for _, e := range def.Env {
		args = append(args, "-e", e)
	}
	args = append(args, "-p", fmt.Sprintf("%d:%d", def.Port, def.Port), def.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("docker run: %s: %w", strings.TrimSpace(string(out)), err)
	}

	slog.Info("provisioner: container started", "name", containerName, "image", def.Image)
	d.started[typ] = true

	if err := d.waitForPort(ctx, typ, def.Port, 30*time.Second); err != nil {
		return nil, err
	}

	return d.config(typ), nil
}

func (d *dockerProvisioner) ensureImage(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := cmd.Run(); err == nil {
		return nil
	}
	slog.Info("provisioner: pulling image", "image", image)
	pull := exec.CommandContext(ctx, "docker", "pull", image)
	if out, err := pull.CombinedOutput(); err != nil {
		return fmt.Errorf("docker pull %s: %s: %w", image, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (d *dockerProvisioner) containerStatus(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", name)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (d *dockerProvisioner) waitForPort(ctx context.Context, typ plugin.DBType, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		status, err := d.containerStatus(ctx, "go-db-"+string(typ))
		if err == nil && status == "running" {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s container to be ready", typ)
}

func (d *dockerProvisioner) config(typ plugin.DBType) *plugin.Config {
	def := dockerImages[typ]
	return &plugin.Config{
		Host:     "127.0.0.1",
		Port:     def.Port,
		Database: defaultDBName(typ),
		User:     defaultUser(typ),
		Password: "",
	}
}

func defaultDBName(typ plugin.DBType) string {
	switch typ {
	case plugin.TypePostgres:
		return "postgres"
	case plugin.TypeMySQL, plugin.TypeMariaDB:
		return "test"
	case plugin.TypeMongoDB:
		return "test"
	case plugin.TypeRedis:
		return "0"
	default:
		return "test"
	}
}

func defaultUser(typ plugin.DBType) string {
	switch typ {
	case plugin.TypePostgres:
		return "postgres"
	case plugin.TypeMySQL, plugin.TypeMariaDB:
		return "root"
	default:
		return ""
	}
}

func (d *dockerProvisioner) Shutdown(ctx context.Context) {
	for typ := range d.started {
		containerName := "go-db-" + string(typ)
		exec.CommandContext(ctx, "docker", "stop", containerName).Run()
		slog.Info("provisioner: container stopped", "name", containerName)
	}
}
