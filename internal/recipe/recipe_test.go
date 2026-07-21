package recipe

import "testing"

func TestRamSlotsEstimate(t *testing.T) {
	out, err := Run("ram_slots_estimate", map[string]any{
		"model_gb": 5.6, "slots": 2, "ctx_size": 4096, "system_ram_gb": 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out["feasible"] != true {
		t.Errorf("expected feasible with 16GB RAM, got %v", out)
	}
	// 4 slots * 4096/1000*0.5 = 8 + 5.6 model = 13.6GB, still < 16*0.8=12.8? no -> false
	out2, _ := Run("ram_slots_estimate", map[string]any{
		"model_gb": 5.6, "slots": 4, "ctx_size": 4096, "system_ram_gb": 16,
	})
	if out2["feasible"] != false {
		t.Errorf("expected infeasible at 4 slots/16GB, got %v", out2)
	}
}

func TestModelFitAndRecommend(t *testing.T) {
	fit, err := Run("model_fit", map[string]any{"ram_gb": 8})
	if err != nil {
		t.Fatal(err)
	}
	names := fit["fits"].([]string)
	if len(names) == 0 {
		t.Errorf("expected some fits at 8GB, got none")
	}
	rec, err := Run("recommend", map[string]any{"ram_gb": 16})
	if err != nil {
		t.Fatal(err)
	}
	if rec["recommend"] == nil {
		t.Errorf("expected a recommendation at 16GB")
	}
}

func TestUnknownRecipe(t *testing.T) {
	if _, err := Run("nope", nil); err == nil {
		t.Errorf("expected error for unknown recipe")
	}
}
