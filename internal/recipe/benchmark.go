package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go-database/internal/llm"
)

// model_download fetches a GGUF (or any) file from a URL into dest (default models/).
// Input:  {"url": "...", "dest": "models/foo.gguf"}  (dest optional)
// Output: {"path": "...", "bytes": N}
func init() {
	Register(Recipe{
		Name:        "model_download",
		Description: "Lädt eine GGUF-Datei von einer URL herunter (z.B. HuggingFace)",
		Compute: func(in map[string]any) (map[string]any, error) {
			url, _ := in["url"].(string)
			if url == "" {
				return nil, fmt.Errorf("model_download: url required")
			}
			dest, _ := in["dest"].(string)
			if dest == "" {
				dest = filepath.Join("models", filepath.Base(url))
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return nil, err
			}
			resp, err := http.Get(url) // ponytail: no resume/retry; add if large files flake
			if err != nil {
				return nil, fmt.Errorf("model_download: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("model_download: http %d", resp.StatusCode)
			}
			f, err := os.Create(dest)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			n, err := io.Copy(f, resp.Body)
			if err != nil {
				return nil, err
			}
			return map[string]any{"path": dest, "bytes": float64(n)}, nil
		},
	})

	// model_benchmark starts a local llama-server for the given model and measures
	// generation tok/s (CPU ngl=0 or GPU ngl via input). Reuses llm.LlamaCppServer.
	// Input:  {"model": "models/foo.gguf", "ngl": 0}
	// Output: {"model": "...", "ngl": N, "tok_s": X}
	Register(Recipe{
		Name:        "model_benchmark",
		Description: "Misst Generierungs-Geschwindigkeit (tok/s) eines lokalen GGUF-Modells via llama-server",
		Compute: func(in map[string]any) (map[string]any, error) {
			model, _ := in["model"].(string)
			if model == "" {
				return nil, fmt.Errorf("model_benchmark: model path required")
			}
			ngl := 0
			if v, ok := in["ngl"].(float64); ok {
				ngl = int(v)
			}
			srv := llm.NewLlamaCppServer(llm.LlamaCppConfig{
				ModelPath:  model,
				Port:       8090,
				NGPULayers: ngl,
				Parallel:   1,
			})
			ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
			defer cancel()
			if err := srv.Start(ctx); err != nil {
				return nil, err
			}
			defer srv.Stop()

			// timed generation run
			body, _ := json.Marshal(map[string]any{
				"prompt":      "Explain in one sentence what a database index is.",
				"n_predict":   128,
				"temperature": 0,
				"timings":     true,
			})
			t := time.Now()
			resp, err := http.Post("http://127.0.0.1:8090/completion", "application/json", bytes.NewReader(body))
			if err != nil {
				return nil, fmt.Errorf("model_benchmark: completion: %w", err)
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)
			_ = time.Since(t)
			var out struct {
				Timings struct {
					PredictedPerSecond float64 `json:"predicted_per_second"`
				} `json:"timings"`
			}
			json.Unmarshal(raw, &out)
			return map[string]any{
				"model": model,
				"ngl":   float64(ngl),
				"tok_s": out.Timings.PredictedPerSecond,
			}, nil
		},
	})
}
