package llm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LlamaCppConfig holds the settings for a local llama.cpp server.
type LlamaCppConfig struct {
	ModelPath  string // path to .gguf file
	Port       int    // server port (default 8081)
	NGPULayers int    // GPU offload layers (-1 = all, 0 = CPU only)
	Parallel   int    // concurrent request slots (0/1 = serial, see --parallel). ponytail: means one slot for single-user, raise for multi-user
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

	bin := FindLlamaCPP()
	if bin == "" {
		return fmt.Errorf("llamacpp: llama-server not found in PATH or LM Studio extensions; install llama.cpp or set it in PATH")
	}
	// llama-server.exe is a Windows binary and cannot resolve MSYS-style paths
	// (/c/Users/...); convert to a Windows path (C:/Users/...) it can open.
	bin = normalizePath(bin)
	modelPath := normalizePath(s.cfg.ModelPath)

	args := []string{
		"--model", modelPath,
		"--port", fmt.Sprintf("%d", s.cfg.Port),
		"--host", "127.0.0.1",
		"--ctx-size", "4096",
		"--n-gpu-layers", fmt.Sprintf("%d", s.cfg.NGPULayers),
	}
	if s.cfg.Parallel > 1 {
		args = append(args, "--parallel", fmt.Sprintf("%d", s.cfg.Parallel), "--cont-batching", "--batch-size", "512")
	}

	s.cmd = exec.CommandContext(ctx, bin, args...)
	s.cmd.Dir = filepath.Dir(bin) // load the .dll files that sit next to the binary
	// ponytail: io.Discard not nil — nil stdout/stderr can crash llama.cpp on Windows at log time
	s.cmd.Stdout = io.Discard
	s.cmd.Stderr = io.Discard

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("llamacpp: start failed: %w", err)
	}

	slog.Info("llamacpp server starting", "port", s.cfg.Port, "model", s.cfg.ModelPath, "parallel", s.cfg.Parallel)
	// ponytail: load time grows with parallel slots (more context alloc); give headroom
	readyTimeout := 60 * time.Second
	if s.cfg.Parallel > 1 {
		readyTimeout = time.Duration(60+s.cfg.Parallel*30) * time.Second
	}
	if err := s.waitReady(ctx, readyTimeout); err != nil {
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
	// Glob installed backends. Prefer self-contained builds (avx2 CPU, vulkan)
	// over CUDA: the CUDA builds need the CUDA runtime DLLs on PATH, which only
	// exist inside the LM Studio process — standalone they fail with exit 127.
	patterns := []string{
		home + "/.lmstudio/extensions/backends/*avx2*/llama-server*",
		home + "/.lmstudio/extensions/backends/*vulkan*/llama-server*",
		home + "/.lmstudio/extensions/backends/*/llama-server*",
	}
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		// CUDA builds need the CUDA runtime DLLs on PATH (only inside LM Studio),
		// so they fail standalone with exit 127. Always drop cuda from candidates.
		var nonCuda []string
		for _, m := range matches {
			if !strings.Contains(strings.ToLower(m), "cuda") {
				nonCuda = append(nonCuda, m)
			}
		}
		if len(nonCuda) > 0 {
			return nonCuda[len(nonCuda)-1] // newest non-CUDA version
		}
	}
	return ""
}

// normalizePath converts an MSYS/Cygwin-style path (/c/Users/...) to a Windows
// path (C:/Users/...) so the llama-server.exe binary (a native Windows process)
// can resolve model/binary paths. On non-Windows it's a no-op.
// ponytail: only handles the /c/... → C:/... case; WSL /mnt/c left as-is.
func normalizePath(p string) string {
	if len(p) >= 3 && p[0] == '/' && p[2] == '/' && p[1] >= 'a' && p[1] <= 'z' {
		return strings.ToUpper(string(p[1])) + ":" + p[2:]
	}
	return p
}

// ResolveModelPath returns the full path for a model key (bare name or partial).
// Searches ./models/** and ~/.lmstudio/models/** recursively for a matching .gguf.
// Returns "" if nothing found.
func ResolveModelPath(modelKey string) string {
	if strings.HasSuffix(modelKey, ".gguf") && fileExists(modelKey) {
		return modelKey
	}
	homeDir, _ := exec.Command("sh", "-c", "echo $HOME").Output()
	home := strings.TrimSpace(string(homeDir))
	roots := []string{"./models"}
	if home != "" {
		roots = append(roots, filepath.Join(home, ".lmstudio", "models"))
	}
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, "**", modelKey+"*.gguf"))
		if len(matches) > 0 {
			return matches[0]
		}
		// fallback: any .gguf whose path contains the key
		var found string
		filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(p, ".gguf") && strings.Contains(p, modelKey) {
				found = p
				return filepath.SkipAll
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	slog.Warn("llamacpp: model not found, set mcp.model to full .gguf path",
		"searched", roots, "key", modelKey)
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// AutoModel attempts to pick a suitable model from installed models.
// Returns the first .gguf found under ~/.lmstudio/models/<publisher>/*.gguf
// that matches the requested size (9b, 14b, 35b).
func AutoModel(preferSize string) (string, error) {
	// Could scan ~/.lmstudio/models recursively for .gguf, but that requires filepath.Walk.
	// ponytail: user provides explicit path for now.
	return "", fmt.Errorf("auto model selection not implemented; set mcp.model to full .gguf path")
}
