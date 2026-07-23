# Local Models Cookbook

## ✅ Verifiziert getestet (mit AI-Agent, NL→Tool→SQL)

Diese Modelle wurden end-to-end durch den AI-Agent getestet (`/api/v1/agent/chat`:
list_connections → list_tables → SELECT via natürlicher Sprache). Alle bestanden.

| Modell | Variante | Pfad (relativ zu Projekt) | Speed | Empfehlung |
|--------|----------|---------------------------|-------|------------|
| 🥇 **Ornith 1.0 9B** | Q4_K_M | `models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf` | **~2-4s** | Default |
| 🥈 **DeepSeek R1 14B Uncensored** | Q4_K_S | `models/DeepSeek-R1-Distill-Qwen-14B-Uncensored-GGUF/DeepSeek-R1-Distill-Qwen-14B-Uncensored.Q4_K_S.gguf` | ~3-5s | Beste SQL |
| DeepSeek R1 8B | Q4_K_M | `models/DeepSeek-R1-0528-Qwen3-8B-GGUF/DeepSeek-R1-0528-Qwen3-8B-Q4_K_M.gguf` | ~3-10s | Alternative |

**Aktivierung** (`config/config.yaml`, Ornith als Default):
```yaml
mcp:
  enabled: true
  provider: "llamacpp"
  model: "models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf"
  llamacpp:
    auto_start: true
    port: 8081
```

`llama-server` wird automatisch aus den installierten LM-Studio-Backends gefunden
(`~/.lmstudio/extensions/backends/`, avx2-CPU-Build wird bevorzugt — CUDA-Builds
brauchen CUDA-DLLs, die standalone fehlen und mit Exit 127 abbrechen).

**Hinweis Nebenläufigkeit:** Die REST- und Agent-API sind voll parallel (Go-Goroutines,
`sync.RWMutex` im ConnectionManager). Der lokale `llama-server` verarbeitet Prompts
aber **seriell** (1 Slot) — für Multi-User-AI `parallel` in der Config erhöhen:

```yaml
mcp:
  llamacpp:
    parallel: 4   # --parallel 4 + --cont-batching: 4 gleichzeitige Slots
```

Bei `parallel > 1` startet go-database den llama-server mit `--parallel N --cont-batching`.
Mehr Slots = mehr RAM (jeder Slot hält Kontext). Für Einzelnutzer bei `1` lassen.

**Paid Provider (Multi-User ohne lokale Limits):** OpenRouter ist bereits integriert.
`fallback_paid: true` + `model: "deepseek/deepseek-r1"` (günstig, stark bei SQL/Reasoning)
oder `model: "anthropic/claude-3.5-haiku"` (bestes Tool-Use, teurer). API-seitig parallel,
kein lokaler Slot-Engpass.

## Installierte Modelle (LM Studio)

| Modell | Variante | Größe | RAM | Publisher | Typ |
|--------|----------|-------|-----|-----------|-----|
| **DeepSeek R1 Distill Qwen 14B** ⭐ | Q4_K_M | 8,9 GB | ~12 GB | lmstudio-community | Code/SQL |
| DeepSeek R1 Distill Qwen 14B Uncensored | Q3_K_S | 6,7 GB | ~10 GB | mradermacher | Unzensiert |
| DeepSeek R1 Distill Qwen 14B Uncensored | Q4_K_S | 8,6 GB | ~12 GB | mradermacher | Unzensiert |
| UncensoredLM DeepSeek R1 Distill Qwen 14B | Q4_K_S | 8,3 GB | ~11 GB | bartowski | Unzensiert |
| **Ornith 1.0 35B** ⭐ | Q4_K_M | 21 GB | ~24 GB | deepreinforce-ai | Universal (tool-use) |
| **Ornith 1.0 9B** ⭐ | Q4_K_M | 5,6 GB | ~8 GB | deepreinforce-ai | Universal |
| Ornith 1.0 9B | Q5_K_M | 6,5 GB | ~9 GB | deepreinforce-ai | Universal |
| Ornith 1.0 9B | Q6_K | 7,5 GB | ~10 GB | deepreinforce-ai | Universal |

⭐ = vom Benutzer bevorzugte Modelle

## Empfehlung: welche Modelle für was?

### Für NL→SQL & DB-Management (go-database)

| Priorität | Modell | Begründung |
|-----------|--------|------------|
| **1** | 🥇 **Ornith 1.0 9B (Q4_K_M)** | `trained_for_tool_use: true`, schnell, 8GB RAM reichen |
| **2** | 🥈 **DeepSeek R1 Distill 14B** | Beste SQL-Kenntnisse, 12GB RAM |
| **3** | 🥉 **Ornith 1.0 35B** | Höchste Qualität, braucht aber 24GB RAM |

### Quantisierung: was bedeuten Q4/Q5/Q6?

| Stufe | Bits/Weight | Qualität | Speicher |
|-------|------------|----------|----------|
| Q3_K_S | 3 bit | Reduziert | -30% |
| **Q4_K_M** | 4 bit | ✅ Gut (Default) | Basis |
| Q5_K_M | 5 bit | Besser | +15% |
| Q6_K | 6 bit | Nah an FP16 | +30% |

**Empfehlung:** Q4_K_M nutzen – bester Kompromiss aus Qualität und RAM.

## Download neuer Modelle

### Von HuggingFace (für LM Studio)

```bash
# Ornith 1.0 9B (empfohlen für go-database)
huggingface-cli download deepreinforce-ai/Ornith-1.0-9B-GGUF ornith-1.0-9b-q4_k_m.gguf --local-dir ~/.lmstudio/models/deepreinforce-ai/Ornith-1.0-9B-GGUF/

# DeepSeek R1 Distill Qwen 14B
huggingface-cli download lmstudio-community/DeepSeek-R1-Distill-Qwen-14B-GGUF DeepSeek-R1-Distill-Qwen-14B-Q4_K_M.gguf --local-dir ~/.lmstudio/models/lmstudio-community/DeepSeek-R1-Distill-Qwen-14B-GGUF/
```

### Via Ollama

```bash
# DeepSeek R1 Distill Qwen 14B
ollama pull deepseek-r1:14b

# Ornith 1.0 9B (falls verfügbar)
ollama pull ornith:9b
```

### Direkter Download (Browser)

| Modell | Link |
|--------|------|
| Ornith 1.0 9B Q4_K_M | https://huggingface.co/deepreinforce-ai/Ornith-1.0-9B-GGUF |
| Ornith 1.0 35B Q4_K_M | https://huggingface.co/deepreinforce-ai/Ornith-1.0-35B-GGUF |
| DeepSeek R1 Distill Qwen 14B Q4_K_M | https://huggingface.co/lmstudio-community/DeepSeek-R1-Distill-Qwen-14B-GGUF |

## Llama.cpp (direkt, ohne LM Studio/Ollama)

Nutzt `llama-server` (Bestandteil von [llama.cpp](https://github.com/ggerganov/llama.cpp)).
Erfordert: `llama-server` im PATH (Build-Anleitung siehe llama.cpp README).

### Setup

```bash
# 1. llama-server bauen (einmalig)
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
make llama-server -j

# 2. Modell herunterladen (siehe "Direkter Download" oben)
mkdir -p models
# .gguf in models/ ablegen

# 3. Server starten (Port 8081)
llama-server --model models/mein-model.gguf --port 8081 --ctx-size 4096 --n-gpu-layers -1
```

### go-database Konfiguration

```yaml
mcp:
  provider: llamacpp
  model: models/mein-model.gguf  # Pfad zur .gguf Datei
```

### Provider-Vergleich

| Provider | Setup | Stream | Auto-Start |
|----------|-------|--------|------------|
| **LM Studio** | GUI + Download | ❌ | Manuell |
| **Ollama** | `ollama pull` | ❌ | `ollama serve` |
| **llama.cpp** | `llama-server` CLI | ✅ (OpenAI-kompatibel) | Manuell oder via `cmd/godb` |

## Konfiguration für go-database

### LM Studio (empfohlen für lokalen NL→SQL)

`config/config.yaml`:
```yaml
mcp:
  enabled: true
  provider: lmstudio
  model: ornith-1.0-9b
  fallback_paid: false
```

### Ollama

```yaml
mcp:
  enabled: true
  provider: ollama
  model: deepseek-r1:14b
  fallback_paid: false
```

## RAM/VRAM-Check

| Modell | Min RAM | Empfohlen | Kann laufen auf |
|--------|---------|-----------|-----------------|
| Ornith 1.0 9B (Q4) | 8 GB | 16 GB | Jeder moderne PC |
| DeepSeek R1 14B (Q4) | 12 GB | 16 GB | Mittelklasse |
| Ornith 1.0 35B (Q4) | 24 GB | 32 GB | High-End / Server |
