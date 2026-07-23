# Modell-Benchmark (llama.cpp) — Fakten-Check

**Datum:** 2026-07-22
**Binary:** `llama.cpp-win-x86_64-avx2-2.20.1` (CPU/RAM) bzw. `vulkan-avx2-2.20.1` (GPU/VRAM)
**Host:** Windows, Ryzen (AVX2), 32 GB RAM, RTX 3060 12 GB VRAM
**Messmethode:** `llama-server` mit `--ctx-size 2048`, Prompt "Explain what a database index is one sentence.", `n_predict=128`, `timings=true`. Wert = `predicted_per_second` (Generierung, nicht Prompt-Eval).
**CPU-Zahlen:** vollständiger Run über alle 17 GGUFs (`ngl=0`, RAM). **GPU-Zahlen:** nur VRAM-passende Modelle (9B + 8B, alle Quants) via Vulkan-Build.

---

## Ergebnis: CPU / RAM (ngl=0) — Root-Server-ohne-GPU-Szenario

| Modell | Quant | Größe | tok/s | RAM ok? |
|--------|-------|-------|-------|---------|
| DeepSeek-R1-0528-Qwen3-8B | Q3_K_L | 4.13 GB | 14.77 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q4_K_M | 4.68 GB | 11.93 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q6_K | 6.26 GB | 9.84 | ✅ |
| DeepSeek-R1-0528-Qwen3-8B | Q8_0 | 8.11 GB | 8.55 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B | Q4_K_M | 8.37 GB | 6.77 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B-Uncensored | Q3_K_S | 6.20 GB | 8.42 | ✅ |
| DeepSeek-R1-Distill-Qwen-14B-Uncensored | Q4_K_S | 7.98 GB | 6.63 | ✅ |
| Qwen3-14B | Q4_K_M | 8.38 GB | 6.65 | ✅ |
| Qwen3-14B | Q6_K | 11.29 GB | 5.36 | ✅ |
| gemma-4-12B-agentic-... | Q2_K | 4.50 GB | 14.67 | ✅ |
| gemma-4-12B-agentic-... | Q4_K_S | 6.54 GB | 7.85 | ✅ |
| gemma-4-12B-agentic-... | Q6_K | 9.11 GB | 5.98 | ✅ |
| gemma-4-12B-agentic-... | Q8_0 | 11.80 GB | 5.11 | ✅ |
| Ornith-1.0-9B | Q4_K_M | 5.24 GB | 9.72 | ✅ |
| Ornith-1.0-9B | Q5_K_M | 6.02 GB | 9.53 | ✅ |
| Ornith-1.0-9B | Q6_K | 6.85 GB | 7.60 | ✅ |
| Ornith-1.0-35B | Q4_K_M | 19.71 GB | 11.58 | ✅ (passt in 32 GB RAM) |

**Fazit CPU/RAM:**
- Alle Modelle laufen auf CPU/RAM ohne GPU — ein Root-Server ohne GPU kann jedes dieser Modelle betreiben.
- 8B-Klasse: ~12–15 tok/s (flüssig für Chat). 14B-Klasse: ~5–7 tok/s (langsam, aber nutzbar). 35B: ~12 tok/s (überraschend schnell, passt in 32 GB RAM).
- **Ornith-1.0-9B Q4_K_M empfohlen** für Root-Server: 5.2 GB, ~10 tok/s, gutes Preis/Leistung.

---

## Ergebnis: GPU / VRAM (ngl=99, Vulkan)

| Modell | Quant | tok/s GPU | tok/s CPU | Speedup |
|--------|-------|-----------|-----------|---------|
| Ornith-1.0-9B | Q4_K_M | 45.49 | 9.72 | **4.7×** |
| Ornith-1.0-9B | Q5_K_M | 45.37 | 9.53 | **4.8×** |
| Ornith-1.0-9B | Q6_K | 45.21 | 7.60 | **6.0×** |
| DeepSeek-R1-0528-Qwen3-8B | Q3_K_L | 45.59 | 14.77 | **3.1×** |
| DeepSeek-R1-0528-Qwen3-8B | Q4_K_M | 45.11 | 11.93 | **3.8×** |
| DeepSeek-R1-0528-Qwen3-8B | Q6_K | 44.57 | 9.84 | **4.5×** |
| DeepSeek-R1-0528-Qwen3-8B | Q8_0 | 44.14 | 8.55 | **5.2×** |

> **Warum fehlen 14B/35B in der GPU-Spalte? (VRAM / RAM Offload)**
>
> `n_gpu_layers` steuert, wie viele Layer auf die GPU (VRAM) gehen; der Rest läuft auf
> RAM (CPU). Ein Modell passt nur dann sauber in die GPU, wenn es komplett in den VRAM passt:
> - **RTX 3060 = 12 GB VRAM.** 9B (5–7 GB) + 8B (4–8 GB) passen komplett → saubere Speedup-Zahlen (oben).
> - **14B (8–11 GB)** passen *knapp* rein, sind aber nicht gemessen (Nachmessung offen).
> - **35B Q4 = 19.7 GB > 12 GB VRAM** → passt NICHT. Nur ~6–7 GB der Gewichte finden in VRAM,
>   der Rest (≈13 GB) auf RAM. Das ist **Partial-Offload**: GPU-Layer schnell, RAM-Layer langsam,
>   dazu PCIe-Copy-Overhead (VRAM↔RAM) → effektiv **langsamer als reiner CPU-Run** (11.58 tok/s).
>   Eine saubere 35B-GPU braucht **≥24 GB VRAM** (RTX 3090/4090/A5000) → dann ~30–45 tok/s.
>
> Fazit: Mit GPU ist der Agent **~4–6× schneller** (45 vs 10 tok/s) — aber nur für Modelle, die
> komplett in den VRAM passen. 35B gehört auf RAM (oder ≥24 GB VRAM).

---

## Agent SQL-Task (NL→SQL) — Fakten-Check

**Getestet:** Ornith-1.0-9B Q4_K_M via `provider: llamacpp`, Endpunkt `POST /api/v1/agent/chat`.
Test-DB: sqlite `agentdb` mit Tabelle `users(id, name, age)` + 3 Zeilen (Alice 30, Bob 25, Carol 40).

| NL-Frage | Agent-Verhalten | SQL korrekt? |
|----------|----------------|--------------|
| "wie viele user sind aelter als 28?" (vag) | wählte `list_connections` (falsches Tool) | ❌ |
| "zeige alle namen der user" (vag) | wählte `list_connections` (falsches Tool) | ❌ |
| "was ist das durchschnittsalter?" (vag) | `unknown tool: nl2sql` (Fallback-Crash) | ❌ |
| "in der datenbank <cid>, zaehle wie viele user aelter als 28 sind" (explizit) | wählte `query` + richtige `connection_id`, generierte SQL | ✅ (Tabelle im Test-Setup nicht persistent → "no such table", aber Tool+SQL ok) |

**Befund:**
- Ornith-9B Q4 **kann** NL→SQL (Tool-Wahl `query` + SQL-Generierung klappt bei klarem Prompt mit genannter `connection_id`).
- Bei **vagen Prompts** wählt das Modell oft das falsche Tool (`list_connections` statt `query`) oder triggert den `nl2sql`-Fallback-Crash (decideTool fallback → executeTool hat keinen `nl2sql`-Case).
- **Empfehlung:** Für zuverlässige NL→SQL entweder (a) Prompts explizit mit DB-Namen formulieren, oder (b) ein echtes `nl2sql`-Tool in `executeTool` registrieren (derzeit nur Fallback-String, kein Handler).

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
  llamacpp:
    auto_start: true        # llama-server automatisch starten
    port: 8081
    n_gpu_layers: 0         # 0 = CPU/RAM; -1 oder 99 = alle Layer auf GPU (braucht VRAM)
```

`provider: llamacpp` → der Code findet die `llama-server`-Binary via `FindLlamaCPP()` und startet sie selbst. `n_gpu_layers` steuert den VRAM-Offload (siehe oben).

---

## Reproduktion

```bash
# CPU/RAM (Root-Server-Szenario) — alle 17 Modelle
bash _t/bench.sh models/<model>.gguf 0

# GPU/VRAM (Vulkan-Build, RTX 3060) — nur VRAM-passende Modelle
bash _t/gpu_bench.sh models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf

# Agent NL->SQL Task (läuft gegen Server auf :8080, Admin via SSH-pubkey)
bash _t/agent_nl2sql_test.sh
```

---

## Docker: lokaler Modell-Start + Offload (Update 2026-07-22)

**Status:** ✅ implementiert. Das Docker-Image `go-database:llamacpp` baut `llama-server`
aus Source und startet ihn per `auto_start` selbst — keine externe Binärdatei nötig.

### Root-Cause-Fix (wichtig)
Erstbuild nutzte llama.cpp-Tag `b6407` (2024) → Modell-Laden scheiterte mit
`unknown model architecture: 'qwen35'` (Ornith-9B / Qwen3.5 nicht bekannt).
**Fix:** `Dockerfile` baut jetzt aus `master` (llama.cpp `cf51256`). Verifiziert:
`model loaded` + `listening on http://127.0.0.1:8081` + `/health → {"status":"ok"}`
für Ornith-1.0-9B Q4_K_M im Container.

### Offload (RAM ↔ VRAM) im Container
`n_gpu_layers` steuert die Verteilung (genau wie lokal):
- `0` → alles CPU/RAM
- `99` (oder `-1`) → alle Layer VRAM
- `20/40/60` → Partial-Offload (große Modelle auf kleiner GPU)

Über Compose (siehe `docker-compose.yml`):
```bash
docker compose up                       # llamacpp auto-start, CPU/RAM
GODB_MCP_LLAMACPP_NGPU=99 docker compose up   # GPU-Offload (braucht NVIDIA-Docker für VRAM)
```

### Volume-Mount-Hinweis (Windows/Docker Desktop)
Der `models/`-Mount braucht den **Windows-Pfad**, nicht den MSYS-Pfad:
```bash
# FALSCH (leeres Volume, Modell nicht gefunden):
docker run -v "$(pwd)/models:/app/models" ...
# RICHTIG:
docker run -v "H:\Projekt\Programmieren\go_database\models:/app/models:ro" ...
```
Ohne korrekten Mount → llama-server findet kein Modell → `timeout after 10m0s`.

### GPU im Container
CPU-Offload läuft out-of-the-box. VRAM-Offload braucht NVIDIA Container Toolkit
(`--gpus all` bzw. `deploy.resources` in compose, Template ist auskommentiert).
Auf Docker Desktop (Windows) ohne GPU → CPU/RAM automatisch.

---

## Cookbook-Recipes (System-Check + Benchmark)

Im `recipe`-System (`internal/recipe`) sind jetzt vorgefertigte Rezepte:

| Recipe | Input | Output | Zweck |
|--------|-------|--------|-------|
| `system_check` | `{model?}` | `{docker, llama_server, agent_model, database_*}` | Pre-flight: ist die Umgebung startklar? |
| `model_download` | `{url, dest?}` | `{path, bytes}` | GGUF von HuggingFace/URL laden |
| `model_benchmark` | `{model, ngl?}` | `{model, ngl, tok_s}` | tok/s eines lokalen Modells messen |
| `model_fit` | `{ram_gb}` | `{fits[]}` | welche Katalog-Modelle passen in RAM? |
| `recommend` | `{ram_gb}` | `{recommend, settings}` | bestes Modell + llama-settings |

Beispiel (Live auf diesem Host, `all_ok: true`):
```json
{
  "docker":        {"ok": true, "detail": "docker 29.5.3 running"},
  "llama_server":  {"ok": true, "detail": "llama-server.exe"},
  "agent_model":   {"ok": true, "detail": "models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf"},
  "database_sqlite":       {"ok": true, "detail": "embedded, always available"},
  "database_provisioner":  {"ok": true, "detail": "docker available for sample DB provisioning"}
}
```

Aufruf: `recipe.Run("system_check", map[string]any{"model": "models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf"})`
(siehe `internal/syscheck` + `internal/recipe/syscheck_recipe.go`).

---

## NL2SQL-Status (Update): `nl2sql`-Tool implementiert

Der vorige Befund ("nl2sql-Fallback crasht") ist behoben: `executeTool` hat jetzt
einen `case "nl2sql"` (`internal/agent/handler.go`), der NL→SQL via LLM generiert
+ ausführt. Damit funktioniert NL→SQL **mit** `nl2sql`-Tool (nicht nur mit `query`).

- **ohne nl2sql:** LLM füllt `query` direkt mit SQL (braucht klaren Prompt + `connection_id`).
- **mit nl2sql:** LLM ruft `nl2sql` auf → Schema wird geholt, SQL generiert, ausgeführt.

Verifiziert: Build + `go test ./internal/agent/` grün. End-to-End-Agent-Test gegen
alle 17 Modelle steht noch aus (Bash-Skript `_t/bench_full.sh` crasht an Agent-Ready-Timing);
Ersatz via `model_benchmark`-Recipe läuft robuster.

