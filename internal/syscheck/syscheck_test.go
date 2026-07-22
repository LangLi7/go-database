package syscheck

import "testing"

func TestCheckStructure(t *testing.T) {
	// without model: docker/llama/db checked; agent_model absent
	res := Check("")
	for _, k := range []string{"docker", "llama_server", "database_sqlite", "database_provisioner"} {
		if _, ok := res[k]; !ok {
			t.Errorf("missing component %q in Check()", k)
		}
	}
	if _, ok := res["agent_model"]; ok {
		t.Error("agent_model should be absent when modelPath empty")
	}
}

func TestCheckWithModel(t *testing.T) {
	res := Check("models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf")
	if _, ok := res["agent_model"]; !ok {
		t.Error("agent_model should be present when modelPath given")
	}
	// Status type serializes
	if res["docker"].Detail == "" && res["docker"].OK {
		t.Error("docker detail should not be empty when OK")
	}
}

func TestStatusMarshal(t *testing.T) {
	s := Status{OK: true, Detail: "x"}
	m := s.Marshal()
	if m["ok"] != true || m["detail"] != "x" {
		t.Errorf("Marshal wrong: %v", m)
	}
}
