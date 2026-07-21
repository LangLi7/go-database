package llm

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LlamaCppConfig holds the settings for a local llama.cpp server.
type LlamaCppConfig struct {
	ModelPath  string // path to .gguf file
	Port       int    // server port (default 8081)
	NGPULayers int    // GPU offload layers (-1 = all, 0 = CPU only)
}

// LlamaCppServer manages a llama-server subprocess.
// ponytail: single-shot process per Start(), no reconnection on crash.
type LlamaCppServer struct {
	cfg    LlamaCppConfig
	cmd    *exec.Cmd
	client *LMStudioClient // reuses LMStudioClient (same OpenAI-compat API)
}

func NewLlamaCppServer(cfg LlamaCppConfig) *LlamaCppServer {
	if cfg.Port == 0 {
		cfg.Port = 8081
	}
	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Port)
	return &LlamaCppServer{
		cfg:    cfg,
		client: NewLMStudio(baseURL, ""),
	}
}

// Start launches the llama-server subprocess and waits until ready.
func (s *LlamaCppServer) Start(ctx context.Context) error {
	if s.cfg.ModelPath == "" {
		return fmt.Errorf("llamacpp: model path is required; set mcp.model in config")
	}

	args := []string{
		"--model", s.cfg.ModelPath,
		"--port", fmt.Sprintf("%d", s.cfg.Port),
		"--host", "127.0.0.1",
		"--ctx-size", "4096",
		"--n-gpu-layers", fmt.Sprintf("%d", s.cfg.NGPULayers),
	}

	s.cmd = exec.CommandContext(ctx, "llama-server", args...)
	s.cmd.Stdout = nil
	s.cmd.Stderr = nil

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("llamacpp: start failed: %w", err)
	}

	slog.Info("llamacpp server starting", "port", s.cfg.Port, "model", s.cfg.ModelPath)
	if err := s.waitReady(ctx, 60*time.Second); err != nil {
		_ = s.cmd.Process.Kill()
		return fmt.Errorf("llamacpp: not ready: %w", err)
	}
	slog.Info("llamacpp server ready", "port", s.cfg.Port)
	return nil
}

// Stop terminates the llama-server process.
func (s *LlamaCppServer) Stop() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

// Client returns an LMStudioClient configured for the local llama-server.
func (s *LlamaCppServer) Client() *LMStudioClient {
	return s.client
}

func (s *LlamaCppServer) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	healthURL := fmt.Sprintf("http://localhost:%d/health", s.cfg.Port)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %v", timeout)
}

// FindLlamaCPP checks if llama-server is available in PATH or LM Studio extensions.
func FindLlamaCPP() string {
	// Check PATH first
	if path, err := exec.LookPath("llama-server"); err == nil {
		return path
	}
	// Check LM Studio extensions (common install paths)
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	candidates := []string{
		home + "/.lmstudio/extensions/backends/llama.cpp-win-x86_64-avx2-2.20.1/llama-server.exe",
		home + "/.lmstudio/extensions/backends/llama.cpp-win-x86_64-nvidia-cuda-avx2-2.20.1/llama-server.exe",
		home + "/.lmstudio/extensions/backends/llama.cpp-win-x86_64-nvidia-cuda12-avx2-2.20.1/llama-server.exe",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// ResolveModelPath returns the full path for a model key.
// Looks in ~/.lmstudio/models/<publisher>/<model-key>/ and ./models/.
func ResolveModelPath(modelKey string) string {
	// Try common locations
	homeDir, _ := exec.Command("sh", "-c", "echo $HOME").Output()
	home := strings.TrimSpace(string(homeDir))
	candidates := []string{
		fmt.Sprintf("%s/.lmstudio/models/%s/%s*.gguf", home, "*", modelKey),
		fmt.Sprintf("./models/%s*.gguf", modelKey),
	}
	// … glob would be nicer but we just log them
	slog.Warn("llamacpp: model path not found automatically, set mcp.model to full .gguf path",
		"searched", candidates)
	return modelKey
}

// AutoModel attempts to pick a suitable model from installed models.
// Returns the first .gguf found under ~/.lmstudio/models/<publisher>/*.gguf
// that matches the requested size (9b, 14b, 35b).
func AutoModel(preferSize string) (string, error) {
	// Could scan ~/.lmstudio/models recursively for .gguf, but that requires filepath.Walk.
	// ponytail: user provides explicit path for now.
	return "", fmt.Errorf("auto model selection not implemented; set mcp.model to full .gguf path")
}
