package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"go-database/internal/api/response"
	"go-database/internal/connection"
)

type TemplateInfo struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Compatible  []string `json:"compatible"`
	Path        string   `json:"-"`
}

var templatesDir = "database/templates"

// ListTemplates returns available SQL templates.
func ListTemplates() gin.HandlerFunc {
	return func(c *gin.Context) {
		entries, err := os.ReadDir(templatesDir)
		if err != nil {
			response.Success(c, []TemplateInfo{})
			return
		}
		var templates []TemplateInfo
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			metaPath := filepath.Join(templatesDir, e.Name(), "metadata.json")
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var t TemplateInfo
			if json.Unmarshal(data, &t) == nil {
				templates = append(templates, t)
			}
		}
		sort.Slice(templates, func(i, j int) bool { return templates[i].Name < templates[j].Name })
		response.Success(c, templates)
	}
}

// ApplyTemplate runs a SQL template on a connection.
func ApplyTemplate(mgr *connection.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ConnectionID string `json:"connection_id"`
			TemplateName string `json:"template_name"` // e.g. "ecommerce"
			WithData     bool   `json:"with_data"`     // also run data.sql
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "connection_id and template_name required")
			return
		}
		schemaSQL, err := os.ReadFile(filepath.Join(templatesDir, req.TemplateName, "schema.sql"))
		if err != nil {
			response.Error(c, 404, "template_not_found", fmt.Sprintf("template %q not found", req.TemplateName))
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
		defer cancel()

		// Execute schema
		result, err := mgr.Execute(ctx, req.ConnectionID, string(schemaSQL))
		if err != nil {
			response.Error(c, 500, "schema_error", err.Error())
			return
		}
		slog.Info("template schema applied", "template", req.TemplateName, "rows", result.RowsAffected)

		// Optionally execute sample data
		if req.WithData {
			dataSQL, err := os.ReadFile(filepath.Join(templatesDir, req.TemplateName, "data.sql"))
			if err == nil {
				dataResult, err := mgr.Execute(ctx, req.ConnectionID, string(dataSQL))
				if err != nil {
					slog.Warn("template data insert warning", "error", err)
				} else {
					slog.Info("template data inserted", "rows", dataResult.RowsAffected)
				}
			}
		}
		response.Success(c, gin.H{"template": req.TemplateName, "applied": true})
	}
}

// DownloadModel runs huggingface-cli to download a model.
func DownloadModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Publisher  string `json:"publisher" binding:"required"` // e.g. "lmstudio-community"
			Repo       string `json:"repo" binding:"required"`      // e.g. "DeepSeek-R1-Distill-Qwen-14B-GGUF"
			File       string `json:"file"`                         // optional: specific .gguf file
			TargetName string `json:"target_name"`                  // optional: custom name
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "publisher and repo required")
			return
		}
		if req.File == "" {
			req.File = "*"
		}

		// Determine target dir
		home, _ := os.UserHomeDir()
		targetDir := filepath.Join(home, ".lmstudio", "models", req.Publisher, req.Repo)
		if req.TargetName != "" {
			targetDir = filepath.Join(home, ".lmstudio", "models", req.Publisher, req.TargetName)
		}
		os.MkdirAll(targetDir, 0755)

		// Run huggingface-cli in background
		go func() {
			args := []string{"download", req.Publisher + "/" + req.Repo, req.File, "--local-dir", targetDir}
			cmd := exec.Command("huggingface-cli", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				slog.Error("model download failed", "publisher", req.Publisher, "repo", req.Repo, "error", err)
				return
			}
			slog.Info("model download complete", "path", targetDir)
		}()

		response.Success(c, gin.H{
			"downloading": true,
			"target_dir":  targetDir,
			"note":        "Download läuft im Hintergrund. Prüfe logs mit 'docker logs' oder server Logs.",
		})
	}
}

// StartModel starts llama-server with a model from local path.
func StartModel(findBinary func() string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ModelPath string `json:"model_path" binding:"required"`
			Port      int    `json:"port"`
			GPULayers int    `json:"gpu_layers"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "model_path required")
			return
		}
		if req.Port == 0 {
			req.Port = 8081
		}
		binPath := findBinary()
		if binPath == "" {
			response.Error(c, 404, "llama_not_found", "llama-server not found on this system")
			return
		}
		args := []string{
			"--model", req.ModelPath,
			"--port", fmt.Sprintf("%d", req.Port),
			"--host", "127.0.0.1",
			"--ctx-size", "4096",
			"--n-gpu-layers", fmt.Sprintf("%d", req.GPULayers),
		}
		cmd := exec.Command(binPath, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			response.Error(c, 500, "start_failed", err.Error())
			return
		}
		response.Success(c, gin.H{
			"started": true,
			"binary":  binPath,
			"port":    req.Port,
			"pid":     cmd.Process.Pid,
		})
	}
}
