# go-database Dokumentation

API-only DB-Gateway mit lokaler LLM-Integration (llamacpp/GGUF, Ollama),
MCP-Server, AI-Agent (NL→Tool-Routing) und Kochbuch-Recipes.

## Schnellstart
- **README (Root):** `../README.md` — Projekt-Überblick + Start.
- **Konfiguration:** `config/config.example.yaml` + `config/config.yaml`.

## Verzeichnisstruktur

### `api/` — API-Dokumentation
- `api.md` — REST-API-Referenz (Endpunkte, Auth, Beispiele).
- `openapi.yaml` — OpenAPI 3.0 Spezifikation (maschinenlesbar).
- `api.html` — HTML-Rendering der API (Swagger-UI-Style).
- `index.html` / `dashboard.html` — Web-Oberflächen (statisch).

### `architecture/` — System-Design
- `STRUCTURE.md` — Code-Struktur, Pakete, Datenfluss.
- `PROTOCOLS.md` — Protokolle (Auth-Handshake, DB-Plugin-Interface).
- `DECISIONS.md` — Architektur-Entscheidungen (ADRs).
- `ROADMAP.md` — Geplante Features + Meilensteine.

### `guides/` — Anleitungen
- `LOCAL_MODELS.md` — llama.cpp / Docker / Offload (RAM↔VRAM).
- `MCP.md` — MCP-Server (7 Tools) Setup + Nutzung.
- `AGENT_RULES.md` — AI-Agent Verhaltensregeln + Tool-Routing.
- `LLM.md` — LLM-Client (OpenRouter / LM Studio / Ollama).
- `CRYPTO.md` — Verschlüsselung (AES-GCM, RSA, x25519).

### `project/` — Projekt-Meta
- `PROJEKT.md` — Deutsche Projektbeschreibung + Ziele.
- `CHANGELOG.md` — Änderungshistorie.
- `TODO.md` — Offene Aufgaben.
- `RISKS.md` — Risikobewertung.
- `CODING.md` — Coding-Conventions.

### Root-Level Docs
- `benchmark-models-2026-07-22.md` — Modell-Benchmark (CPU/GPU/Offload) + Docker + Recipes.
- `examples/python/` — Python-Clients (`api_test.py`, `api_test_all_dbs.py`).

## API-Docs generieren
```bash
# OpenAPI-Spec ist in api/openapi.yaml — mit Swagger-UI oder Redoc rendern:
docker run -p 8088:8080 -v $(pwd)/docs/api/openapi.yaml:/spec.yaml swaggerapi/swagger-ui
```

## Recipes (Cookbook)
System-Checks + Benchmarks via `recipe.Run(...)` (siehe `benchmark-models-2026-07-22.md`
→ Abschnitt "Cookbook-Recipes"): `system_check`, `model_download`, `model_benchmark`,
`model_fit`, `recommend`.
