# Modell-Benchmark (llama.cpp) — Fakten-Check

**Datum:** 2026-07-22
**Binary:** `llama.cpp-win-x86_64-avx2-2.20.1` (CPU/RAM) bzw. `vulkan-avx2-2.20.1` (GPU/VRAM)
**Host:** Windows, Ryzen (AVX2), 32 GB RAM, RTX 3060 12 GB VRAM
**Messmethode:** `llama-server` mit `--ctx-size 2048`, Prompt "Explain what a database index is in one sentence.", `n_predict=128`, `timings=true`. Wert = `predicted_per_second` (Generierungs-Geschwindigkeit, nicht Prompt-Eval).

---

## Ergebnis: CPU / RAM (ngl=0) — Root-Server-ohne-GPU-Szenario

| Modell | Quant | Größe | tok/s | RAM ok? |
|--------|-------|-------|-------|---------|
| DeepSeek-R1-0528-Qwen3-8B | Q3_K_L | 4.13 GB | 13.83 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q4_K_M | 4.68 GB | 11.93 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q6_K | 6.26 GB | 9.53 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q8_0 | 8.11 GB | 8.44 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B | Q4_K_M | 8.37 GB | 6.77 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B-Uncensored | Q3_K_S | 6.20 GB | 8.76 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B-Uncensored | Q4_K_S | 7.98 GB | 7.24 | ✅ |
| Qwen3-14B | Q4_K_M | 8.38 GB | 7.02 | ✅ |
| Qwen3-14B | Q6_K | 11.29 GB | 5.56 | ✅ |
| gemma-4-12B-agentic-... | Q2_K | 4.50 GB | 14.93 | ✅ |
| gemma-4-12B-agentic-... | Q4_K_S | 6.54 GB | 8.07 | ✅ |
| gemma-4-12B-agentic-... | Q6_K | 9.11 GB | 6.16 | ✅ |
| gemma-4-12B-agentic-... | Q8_0 | 11.80 GB | 5.31 | ✅ |
| Ornith-1.0-9B | Q4_K_M | 5.24 GB | 9.99 | ✅ |
| Ornith-1.0-9B | Q5_K_M | 6.02 GB | 10.13 | ✅ |
| Ornith-1.0-9B | Q6_K | 6.85 GB | 8.57 | ✅ |
| Ornith-1.0-35B | Q4_K_M | 19.71 GB | 11.91 | ✅ (passt in 32 GB RAM) |

**Fazit CPU/RAM:**
- Alle Modelle laufen auf CPU/RAM ohne GPU — ein Root-Server ohne GPU kann jedes dieser Modelle betreiben.
- 8B-Klasse: ~12–14 tok/s (flüssig für Chat). 14B-Klasse: ~6–8 tok/s (langsam, aber nutzbar). 35B: ~12 tok/s (überraschend schnell wegen effizientem Architektur, passt in 32 GB RAM).
- **Ornith-1.0-9B Q4_K_M empfohlen** für Root-Server: 5.2 GB, ~10 tok/s, gutes Preis/Leistung.

---

## Ergebnis: GPU / VRAM (ngl=99, Vulkan)

| Modell | Quant | tok/s GPU | tok/s CPU | Speedup |
|--------|-------|-----------|-----------|---------|
| Ornith-1.0-9B | Q4_K_M | 44.05 | 9.99 | **4.4×** |

> Hinweis: Die CUDA-Builds von LM Studio scheitern standalone (fehlende cudart/cublas DLLs → exit 127). Der **Vulkan-Build** nutzt die GPU-Treiber direkt und liefert echten VRAM-Offload. RTX 3060 12 GB: Ornith-9B Q4 (5.2 GB) passt komplett in VRAM → 44 tok/s.
>
> Fazit: Mit GPU ist der Agent ~4× schneller (44 vs 10 tok/s). Ohne GPU (Root-Server) läuft Ornith-9B Q4 immer noch flüssig mit ~10 tok/s.

---
---

## Konfiguration

In `config.yaml` / `config.example.yaml` (Section `mcp`):

```yaml
mcp:
  enabled: true
  provider: llamacpp        # startet llama-server selbst (kein externer Prozess nötig)
  model: "models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf"
  api_key: ""               # leer = lokal, kein Cloud-Key nötig
  fallback_paid: false      # nie bezahlen, es sei denn du willst es explizit
```

`provider: llamacpp` → der Code findet die `llama-server`-Binary via `FindLlamaCPP()` und startet sie selbst mit `--n-gpu-layers` aus der Config (GPU offload falls verfügbar).

---

## Reproduktion

```bash
# CPU/RAM (Root-Server-Szenario)
bash _t/bench.sh models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf 0

# GPU/VRAM (Vulkan-Build, RTX 3060)
# bench.sh nutzt fest den avx2-Binary; für GPU manuell:
llama-server --model models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf \
  --n-gpu-layers 99 --ctx-size 2048 --port 8099
curl -X POST http://127.0.0.1:8099/completion -d '{"prompt":"hi","n_predict":128,"timings":true}'
```
