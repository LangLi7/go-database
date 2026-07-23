# go-database — LLM / AI-Subsystem

## Pakete

| Paket | Verantwortung |
|-------|---------------|
| `internal/llm/` | LLM-Client Interface + Provider (OpenRouter, LM Studio, Ollama) |
| `internal/api/handler/models.go` | REST-Endpoints für Modell-Entdeckung |
| `internal/mcp/server.go` | MCP-Tools (nl2sql u.a.) nutzen `llm.Client` |

## Provider

### OpenRouter (Default, remote)

1. **FREE Modelle** werden zuerst probiert (live von API abgefragt, Fallback auf hartcodierte Liste):
   - `google/gemma-4-31b-it:free`
   - `google/gemma-4-26b-a4b-it:free`
   - `nvidia/nemotron-3-nano-30b-a3b:free`
   - `nvidia/nemotron-nano-9b-v2:free`
   - `openai/gpt-oss-20b:free`
   - `cohere/north-mini-code:free`
   - `poolside/laguna-m.1:free`
   - `openrouter/free`
2. **Paid Fallback** wenn alle FREE fehlschlagen: `deepseek/deepseek-r1` (~$0.14/M input tokens)
3. Config: `mcp.api_key` (oder `GODB_MCP_API_KEY`)

### LM Studio (lokal)

1. Erwartet laufende LM Studio Instanz auf `http://localhost:1234`
2. Modell-Liste via `GET /api/v1/models/local` → Proxy zur LM Studio API
3. Chat-Completion via `POST /v1/chat/completions`
4. Config: `mcp.provider: lmstudio`, `mcp.model: deepseek-r1-distill-qwen-14b`

### Ollama (lokal)

1. Erwartet Ollama auf `http://localhost:11434`
2. Config: `mcp.provider: ollama`, `mcp.model: llama3`

## Modell-Entdeckung

```bash
# Lokale Modelle (LM Studio)
curl http://localhost:8080/api/v1/models/local

# Remote FREE Modelle (OpenRouter)
curl http://localhost:8080/api/v1/models/remote
```

## Fallback-Logik

Für OpenRouter (`mcp.provider: openrouter`):

```
1. Versuche FREE Modelle (der Reihe nach)
2. Falls alle FREE fehlschlagen → Fallback-Modell (paid: deepseek/deepseek-r1)
3. Falls auch das fehlschlägt → Fehler an Client
```

## Konfiguration

`config/config.yaml`:

```yaml
mcp:
  enabled: false
  endpoint: "/api/v1/mcp"
  api_key: ""
  provider: "openrouter"  # "openrouter" | "lmstudio" | "ollama"
  model: "free"
```

Umgebungsvariablen:

| Variable | Beschreibung |
|----------|-------------|
| `GODB_MCP_API_KEY` | OpenRouter API-Key |
| `GODB_MCP_PROVIDER` | Provider (openrouter/lmstudio/ollama) |
| `GODB_MCP_MODEL` | Modellname (z.B. `deepseek-r1-distill-qwen-14b`) |
| `OLLAMA_URL` | Ollama-Basis-URL (Default: `http://localhost:11434`) |
