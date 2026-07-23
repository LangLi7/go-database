// Package syscheck runs environment pre-flight checks: docker, llama-server,
// agent model, and database availability. Used by the system_check recipe.
package syscheck

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"

	"go-database/internal/llm"
)

// Status is the result of one component check.
type Status struct {
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Check runs all pre-flight checks and returns them by component name.
// modelPath is the configured mcp.model (bare name or .gguf path); empty = skip model check.
func Check(modelPath string) map[string]Status {
	out := map[string]Status{
		"docker":               dockerStatus(),
		"llama_server":         llamaStatus(),
		"database_sqlite":      {OK: true, Detail: "embedded, always available"},
		"database_provisioner": dbProvisionerStatus(),
	}
	if modelPath != "" {
		out["agent_model"] = modelStatus(modelPath)
	}
	return out
}

func dockerStatus() Status {
	if _, err := exec.LookPath("docker"); err != nil {
		return Status{OK: false, Detail: "docker binary not in PATH"}
	}
	// daemon running?
	cmd := exec.Command("docker", "info", "--format", "{{.ServerVersion}}")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return Status{OK: false, Detail: "docker installed but daemon not running: " + firstLine(buf.String())}
	}
	return Status{OK: true, Detail: "docker " + strings.TrimSpace(buf.String()) + " running"}
}

func llamaStatus() Status {
	bin := llm.FindLlamaCPP()
	if bin == "" {
		return Status{OK: false, Detail: "llama-server not found (install llama.cpp or set in PATH)"}
	}
	return Status{OK: true, Detail: filepath.Base(bin)}
}

func modelStatus(modelPath string) Status {
	resolved := llm.ResolveModelPath(modelPath)
	if resolved == "" {
		// maybe it's already an absolute/relative path
		resolved = modelPath
	}
	if strings.HasSuffix(resolved, ".gguf") {
		return Status{OK: true, Detail: resolved}
	}
	return Status{OK: false, Detail: "model not found: " + modelPath}
}

// dbProvisionerStatus checks whether docker-based sample DBs can be provisioned.
func dbProvisionerStatus() Status {
	if _, err := exec.LookPath("docker"); err != nil {
		return Status{OK: false, Detail: "docker missing — sample DBs (postgres/mysql/mongo) cannot auto-provision"}
	}
	return Status{OK: true, Detail: "docker available for sample DB provisioning"}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return strings.TrimSpace(s)
}

// Marshal is a helper for the recipe to return JSON-serializable output.
func (s Status) Marshal() map[string]any {
	b, _ := json.Marshal(s)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}
