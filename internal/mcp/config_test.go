package mcp

import "testing"

func TestValidateMCPConfig(t *testing.T) {
	tests := []struct {
		name, provider, model, apiKey string
		wantErr                       bool
	}{
		{"empty ok", "", "", "", false},              // no validation for empty/ollama
		{"ollama no model", "ollama", "", "", false}, // ollama defaults locally
		{"openrouter ok", "openrouter", "deepseek/deepseek-r1:free", "sk-test", false},
		{"openrouter missing model", "openrouter", "", "sk-test", true},
		{"openrouter missing key", "openrouter", "deepseek/deepseek-r1:free", "", true},
		{"openrouter missing both", "openrouter", "", "", true},
		{"invalid provider", "claude", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateMCPConfig(tc.provider, tc.model, tc.apiKey)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateMCPConfig(%q,%q,%q) err=%v, wantErr=%v",
					tc.provider, tc.model, tc.apiKey, err, tc.wantErr)
			}
		})
	}
}
