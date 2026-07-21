# go-database — MCP-Server

## Was ist das?

Ein **Model Context Protocol**-Server, der AI-Clients (Claude Desktop,
OpenRouter, oder jeder MCP-fähige Client) direkt mit der Datenbank-Oberfläche
von go-database verbindet. Tools laufen über stdio JSON-RPC oder wahlweise über
einen HTTP-Endpoint (config-gesteuert).

## Voraussetzung

go 1.26+, Abhängigkeit:

    github.com/modelcontextprotocol/go-sdk v1.6.1

## Start

### Variante A: separater MCP-Process (Stdio)

```bash
go run ./cmd/mcp
```

Nutzbar z.B. mit Claude Desktop:

```json
{
  "mcpServers": {
    "go-database": {
      "command": "go",
      "args": ["run", "./cmd/mcp"]
    }
  }
}
```

### Variante B: HTTP-Endpoint (im REST-Server)

In `config/config.yaml` oder per `GODB_MCP_ENABLED=true` aktivieren:

```yaml
mcp:
  enabled: true
  endpoint: "/api/v1/mcp"
  api_key: "sk-mein-openrouter-key"
  provider: "openrouter"   # "openrouter" | "ollama"
  model: "deepseek/deepseek-r1:free"
```

Dann ist der MCP-Endpoint unter `POST /api/v1/mcp` erreichbar:

```bash
curl -X POST http://localhost:8080/api/v1/mcp \
  -H "Authorization: Bearer sk-mein-openrouter-key" \
  -d '{"tool":"list_connections","args":{}}'
```

## Tools

| Tool | Beschreibung |
|------|-------------|
| `list_connections` | Alle Connections auflisten |
| `query` | `{connection_id, sql}` — SELECT |
| `execute` | `{connection_id, sql}` — INSERT/UPDATE/DDL |
| `list_tables` | `{connection_id}` |
| `schema` | `{connection_id}` |
| `list_databases` | `{connection_id}` |
| `nl2sql` | `{connection_id, question, schema_hint?}` — Natural Language → SQL |

## NL→SQL (echt, kein Platzhalter mehr)

`nl2sql` ruft jetzt **echt** ein LLM auf — entweder OpenRouter (Default) oder ein
lokales Ollama-Modell.

### OpenRouter

1. API-Key bei [openrouter.ai/keys](https://openrouter.ai/keys) holen.
2. Config setzen:
   - `mcp.api_key: "[REDACTED]-..."` (oder `GODB_MCP_API_KEY`)
   - `mcp.provider: "openrouter"`
   - `mcp.model: "deepseek/deepseek-r1:free"` (Default)
3. `nl2sql` sendet den Prompt an OpenRouter und gibt das SQL zurück.

### Lokales Modell (Ollama)

1. [Ollama](https://ollama.ai) installieren und starten.
2. Config setzen:
   - `mcp.provider: "ollama"`
   - `mcp.model: "llama3"` (oder ein anderes installiertes Modell)
3. Optional `OLLAMA_URL` setzen (Default: `http://localhost:11434/api/generate`).

### Prompt-Struktur

Das Tool sendet immer den `schema_hint` mit, falls angegeben:

```
You are a SQL expert. Convert natural language to SQL.
Return ONLY the raw SQL query, no markdown, no explanation.
Schema context:
{table_name (col1, col2, ...)}
Question: {question}
```

## Konfiguration (alle Felder)

Umgebungsvariable (`GODB_MCP_...`) | Config-YAML | Default | Beschreibung
--- | --- | --- | ---
`GODB_MCP_ENABLED` | `mcp.enabled` | `false` | HTTP-Endpoint aktivieren
`GODB_MCP_ENDPOINT` | `mcp.endpoint` | `/api/v1/mcp` | HTTP-Pfad für MCP-Tools
`GODB_MCP_API_KEY` | `mcp.api_key` | `""` | Bearer-Token für HTTP; bei OpenRouter = API-Key
`GODB_MCP_PROVIDER` | `mcp.provider` | `openrouter` | `"openrouter"` \| `"ollama"`
`GODB_MCP_MODEL` | `mcp.model` | `deepseek/deepseek-r1:free` | Modellname

## Sicherheit

- **HTTP-Endpoint**: API-Key via `Authorization: Bearer <key>` oder
  `X-API-Key <key>` erforderlich (prüft gegen `mcp.api_key`). Bei leerem Key
  wird der Endpoint nur für lokale Clients empfohlen.
- **Stdio-Endpoint**: kein API-Key nötig (vertrauenswürdiger lokaler Process).
- **Berechtigungen**: Alle MCP-Tools haben vollen DB-Zugriff (nur für
  vertrauenswürdige Clients).
