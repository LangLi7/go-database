# Konzept: AI-Database-Engine (Vektor · RAG · Embeddings)

_Status:_ **Phase 1 gebaut** (embeddings + `vector_search`/`rag` Agent-Tools).
Trigger: User will go-database als "Datenbank-Engine für AI" (Vektor-DB, RAG,
Embeddings), Richtung Obsidian-ähnliches semantisches Zettelkasten.

## Was gebaut ist (Phase 1)

- **`internal/ai/embed.go`** — `Embedder` interface + 3 Implementierungen:
  - `OllamaEmbedder` (lokal, `/api/embed`, nomic-embed-text, kostenlos)
  - `OpenAIEmbedder` (OpenAI-kompatibel `/v1/embeddings`, für Cloud-Modelle)
  - `HashEmbedder` (deterministisch, dependency-frei, Offline/Test — _nicht_
    semantisch; siehe Ponytail-Hinweis)
- **Agent-Tools** `vector_search` + `rag` (in `internal/agent/handler.go`):
  - `vector_search(connection_id, table, text_column, embedding_column, query, k)`
    → embed() → pgvector `SELECT col, emb <=> '[...]' AS distance ... LIMIT k`
  - `rag(...)` → vector_search → Kontext in Prompt → `llm.Complete` Antwort
- **`InitAgent`** nimmt jetzt einen `Embedder` (nil → HashEmbedder Fallback).
- Verifiziert: `internal/ai` Unit-Tests (Determinismus + PgVectorLiteral),
  Agent-Test (pgvector-SQL wird korrekt gebaut).

## Was ist "AI-Database"?

...
abgefragbar ist: "Finde alle Chats, in denen es um Login-Probleme ging" → Vektor-Suche
statt `WHERE text LIKE '%login%'`.

### Bausteine
1. **Embeddings** — Text → Vektor (float[]). Über LLM-Provider (OpenAI/Ollama/local
   llama mit Embedding-Modell) oder lokales Modell.
2. **Vektor-Index** — ANN-Suche (cosine/Euclidean). Optionen:
   - **Postgres + pgvector** — go-database hat Postgres-Plugin → naheliegend, kein
     neuer Storage, RBAC/Auth schon da.
   - **SQLite + sqlite-vec** — lokal, leicht, embedded.
   - **Chroma / Pinecone / Qdrant** — externe Vektor-DBs (mehr Aufwand, Cloud-Abhängigkeit).
3. **RAG-Pipeline** — Retrieve (Vektor-Suche) → Augment (Kontext in Prompt) → Generate
   (LLM-Antwort). Der bestehende AI-Agent kann das: `nl2sql` + Vektor-Retrieve kombiniert.

## Warum go-database der richtige "Hafen" dafür ist

- Bereits **Multi-DB-Mittelsmann** (Postgres/MySQL/Mongo/Redis/…).
- **Auth/RBAC + Crypto** schon vorhanden → Vektor-Daten genießen dieselbe Sicherheit.
- **AI-Agent** schon da → RAG = Agent-Tool `vector_search` + `nl2sql`.
- **Obsidian-Richtung:** Notizen/Markdown → Embedding → Vektor-Suche → "semantisches
  Zettelkasten". Genau die Architektur, nur mit Vektor-Index statt Dateisystem.

## Vorschlag (lazy, incremental)

**Phase 1 — pgvector nutzen (kein neuer Storage):**
- `vector` column type im Postgres-Plugin (pgvector ist ein Extension, kein eigener Server).
- Agent-Tool `vector_search(connection_id, table, column, query, k)` → Embedding + ANN.
- Embedding via bestehendem LLM-Client (Ollama/local hat Embedding-Endpoint).

**Phase 2 — RAG-Agent:**
- `rag(connection_id, question)` → vector_search → Kontext → LLM-Generate.
- Nutzt den Guard (Blast-Radius) für Schreib-Operationen.

**Phase 3 — Obsidian-Engine (optional):**
- Markdown-Importer → Chunking → Embedding → Vektor-Index in Postgres/SQLite.
- Query: "Was weiß ich über X?" → semantische Suche.

## YAGNI / Scope

- **Chroma/Pinecone:** erst wenn pgvector/sqlite-vec nicht reicht. pgvector deckt 95%
  der Fälle (bis ~10M Vektoren) ohne neuen Service.
- **Eigene Vektor-Lib:** nicht bauen, pgvector/sqlite-vec nutzen.

## Sicherheit (Claude-Feedback bezogen)

- Vektor-Daten sind **sensible Klartext-Embeddings** → Maskierung vor Cloud-LLM nötig
  (siehe `internal/llm/fallback.go` Local→Cloud-Failover: Prompt nur NL, keine DB-Zeilen).
- Agent-RAG durch **Guard** (dieses Konzept-Commit) → keine unbestätigten DELETE auf
  Vektor-Tabellen.
- Audit-Log für Vektor-Queries (wie REST-Handler) nachrüsten.

## Entscheidung

**pgvector auf Postgres** als Vektor-Backend (kein neuer Service, RBAC/Crypto wiederverwendet).
RAG = Agent-Tool. Obsidian-Engine = Phase 3 (Markdown → Embedding → pgvector).
Nicht gebaut — Konzept. Sag "bau Phase 1", dann scaffold ich das Postgres `vector`-Type
+ `vector_search`-Agent-Tool.
