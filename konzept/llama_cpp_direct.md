# Konzept: llama.cpp Direct — Go-native GGUF Inference ohne LM Studio / Ollama

## Ziel

`go-database` soll `.gguf` Modelle direkt laden können — ohne LM Studio (Python),
ohne Ollama, ohne externe Services. Nur Go + `llama-server.exe` (~5 MB).

## Architektur

```
┌─────────────────────────────────────────────────────────────────┐
│  go-database (Go)                                              │
│                                                                 │
│  main.go                                                        │
│   ├── config.MCP.Provider == "llamacpp"                         │
│   │   └── LlamaCppServer.Start()                                │
│   │       └── exec.Command("llama-server",                      │
│   │              "--model", model.gguf,                         │
│   │              "--port", port,                                │
│   │              "--ctx-size", 4096,                            │
│   │              "--n-gpu-layers", 0)                           │
│   │       └── waitReady() → /health                            │
│   │                                                             │
│   ├── llm.NewClient("llamacpp", ...)                            │
│   │   └── LMStudioClient (baseURL = localhost:port)             │
│   │       └── POST /v1/chat/completions ←── OpenAI-kompatibel   │
│   │                                                             │
│   └── LlamaCppServer.Stop()  (via signal.Shutdown)              │
│       └── cmd.Process.Kill()                                    │
└─────────────────────────────────────────────────────────────────┘
```

**Kern-Idee:** `llama-server` (aus llama.cpp) spricht dieselbe OpenAI-kompatible API
wie LM Studio. Der bestehende `LMStudioClient` kann **unverändert** genutzt werden —
nur die `baseURL` zeigt auf den Subprozess statt auf LM Studio.

## Config

```yaml
mcp:
  enabled: true
  provider: llamacpp                    # neu
  model: "C:/pfad/zu/deepseek-r1-8b-q4_k_m.gguf"  # lokaler .gguf Pfad
  llamacpp:
    auto_start: true                    # true = Go startet llama-server mit
    port: 15535                         # Port für den Subprozess
    n_gpu_layers: 0                     # 0 = CPU only
```

### `auto_start` Modi

| `auto_start` | Verhalten | Use-Case |
|---|---|---|
| `true` | `main.go` startet `llama-server` als Subprozess → wartet auf `/health` → killed beim Shutdown | Production, Ein-Klick-Start |
| `false` | Go verbindet sich nur auf `localhost:15535` — kein Subprozess | Dev: `llama-server` läuft manuell, nur Go wird neugestartet |

## Provider-Registrierung

`NewClient("llamacpp", ...)` erzeugt einen `LMStudioClient` mit `baseURL` auf
den konfigurierten Port. Der `LlamaCppServer` muss vorher gestartet sein
(entweder automatisch oder manuell).

## Lifecycle

```
Start:
  main()
    → config.MCP.Provider == "llamacpp"
      → FindLlamaCPP() prüft PATH + LM Studio extensions
      → Wenn auto_start:
          → LlamaCppServer.Start()
          → Warte auf /health (bis zu 60s)
          → sonst: Warnung, fahre ohne LLM fort
    → llm.NewClient("llamacpp", ...)
    → mcp.SetNL2SQLConfig(...)

Shutdown:
  SIGINT/SIGTERM
    → LlamaCppServer.Stop()
    → srv.Shutdown()
```

## Dateien & Änderungen

| Datei | Änderung |
|---|---|
| `konzept/llama_cpp_direct.md` | — |
| `internal/config/config.go` | `MCP.ModelPath` + `MCP.LlamaCpp` Sub-Config |
| `internal/llm/client.go` | `NewClient("llamacpp")` nutzt Port aus Config statt hardcodiert |
| `internal/llm/llamacpp.go` | `FindLlamaCPP()` Verbesserung, `AutoModel()` implementieren |
| `internal/mcp/httphandler.go` | `ValidateMCPConfig` um `"llamacpp"` erweitern |
| `cmd/server/main.go` | `LlamaCppServer` starten/stoppen |
| `config/config.yaml` | `llamacpp` Section ergänzen |

## Offene Punkte

- [ ] `FindLlamaCPP()`: Auch `%APPDATA%` und `%LOCALAPPDATA%` durchsuchen
- [ ] `AutoModel()`: Rekursive Suche nach `.gguf` in `~/.lmstudio/models/`
- [ ] Port-Konflikt: Wenn Port belegt, automatisch naechsten freien Port wählen
- [ ] Graceful Shutdown: Timeout für `llama-server` Prozess
