package recipe

import (
	"go-database/internal/syscheck"
)

func init() {
	Register(Recipe{
		Name:        "system_check",
		Description: "Pre-flight System-Check: Docker, llama-server, Agent-Modell, Datenbanken",
		Compute: func(in map[string]any) (map[string]any, error) {
			model, _ := in["model"].(string)
			results := syscheck.Check(model)
			out := make(map[string]any, len(results))
			for k, v := range results {
				out[k] = v.Marshal()
			}
			// overall: ok if all required components pass
			allOK := true
			for _, v := range results {
				if !v.OK {
					allOK = false
					break
				}
			}
			out["all_ok"] = allOK
			return out, nil
		},
	})
}
