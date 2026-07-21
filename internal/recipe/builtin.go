package recipe

// ModelProfile describes a local GGUF model for fit/recommendation.
type ModelProfile struct {
	Name    string  `json:"name"`
	GB      float64 `json:"gb"`      // file size on disk
	Quality string  `json:"quality"` // "good" | "best" — relative
}

// built-in model catalog (Ornith + DeepSeek verified in this project)
var catalog = []ModelProfile{
	{Name: "Ornith-1.0-9B-Q4_K_M", GB: 5.6, Quality: "good"},
	{Name: "Ornith-1.0-9B-Q5_K_M", GB: 6.5, Quality: "best"},
	{Name: "DeepSeek-R1-Distill-14B-Q4_K_S", GB: 8.6, Quality: "good"},
	{Name: "DeepSeek-R1-Distill-14B-Q4_K_M", GB: 9.0, Quality: "best"},
	{Name: "DeepSeek-R1-8B-Q4_K_M", GB: 4.8, Quality: "good"},
}

func init() {
	Register(Recipe{
		Name:        "ram_slots_estimate",
		Description: "RAM-Bedarf eines lokalen Modells mit N Slots (verhindert Overcommit)",
		Compute: func(in map[string]any) (map[string]any, error) {
			modelGB, _ := inputFloat(in, "model_gb")
			slots, _ := inputFloat(in, "slots")
			ctx, _ := inputFloat(in, "ctx_size")
			systemRAM, _ := inputFloat(in, "system_ram_gb")
			// KV-Cache pro Slot ~ ctx_size*2 Bytes/param-ish; grobe Faust: 0.5GB/4k-ctx-Slot
			ramGB := modelGB + slots*(ctx/1000.0)*0.5
			out := map[string]any{"ram_gb": ramGB, "model_gb": modelGB, "slots": slots}
			if systemRAM > 0 {
				out["feasible"] = ramGB < systemRAM*0.8
				out["system_ram_gb"] = systemRAM
			}
			return out, nil
		},
	})

	Register(Recipe{
		Name:        "model_fit",
		Description: "Welche Modelle aus dem Katalog passen in verfügbares RAM/VRAM?",
		Compute: func(in map[string]any) (map[string]any, error) {
			ram, _ := inputFloat(in, "ram_gb")
			var fit []string
			for _, m := range catalog {
				if m.GB < ram*0.8 {
					fit = append(fit, m.Name)
				}
			}
			return map[string]any{"ram_gb": ram, "fits": fit}, nil
		},
	})

	Register(Recipe{
		Name:        "recommend",
		Description: "Beste Modell-Empfehlung + llama-server Settings für Hardware",
		Compute: func(in map[string]any) (map[string]any, error) {
			ram, _ := inputFloat(in, "ram_gb")
			best := ModelProfile{}
			for _, m := range catalog {
				if m.GB < ram*0.8 && m.GB > best.GB {
					best = m
				}
			}
			if best.Name == "" {
				return map[string]any{"recommend": nil, "settings": nil, "note": "RAM zu klein für Katalog"}, nil
			}
			settings := map[string]any{
				"ctx_size":     4096,
				"n_gpu_layers": -1,
				"parallel":     1,
				"batch_size":   512,
			}
			return map[string]any{
				"recommend": best.Name,
				"model_gb":  best.GB,
				"settings":  settings,
				"note":      "Ornith 9B = schnellster Tool-Use; 14B = beste SQL",
			}, nil
		},
	})
}
