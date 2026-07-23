package recipe

import (
	"encoding/json"
	"fmt"
)

// Recipe is a named, computable procedure. Input/output are free-form JSON.
type Recipe struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Compute     func(in map[string]any) (map[string]any, error)
}

var registry = map[string]Recipe{}

// Register adds a built-in recipe. Call from init().
func Register(r Recipe) { registry[r.Name] = r }

// Run executes a registered recipe by name with JSON input.
func Run(name string, in map[string]any) (map[string]any, error) {
	r, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("recipe %q not found", name)
	}
	return r.Compute(in)
}

// List returns metadata for all registered recipes (no compute funcs).
func List() []map[string]string {
	out := make([]map[string]string, 0, len(registry))
	for _, r := range registry {
		out = append(out, map[string]string{"name": r.Name, "description": r.Description})
	}
	return out
}

// inputFloat safely reads a float field from recipe input.
func inputFloat(in map[string]any, key string) (float64, bool) {
	switch v := in[key].(type) {
	case float64:
		return v, true
	case json.Number:
		f, _ := v.Float64()
		return f, true
	case int:
		return float64(v), true
	}
	return 0, false
}
