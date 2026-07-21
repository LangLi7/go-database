# Konzept: Agent sieht die komplette DB/SQL-API

_Status:_ Brainstorming — noch nicht gebaut. Trigger: User will, dass die AI
nicht nur generische CRUD-Tools, sondern die gesamte API "versteht".

## Problem (Stand jetzt)
Der AI-Agent (`internal/agent/handler.go`) kennt nur 7 **generische** Tools:
`list_connections, query, execute, list_tables, schema, list_databases, nl2sql`.
Diese sind **hartcodiert** (`availableTools`-Slice, Zeile 24).

Was der Agent NICHT sieht:
- **Connection-Parameter** (`plugin.Config`): Host/Port/SSL/Params, DSN-Detection
- **Guard-Regeln** (`internal/guard/rbac.go`): was blockiert wird
  (DROP/DELETE/INSERT ohne `connections:execute`, UNKNOWN/EXPLAIN per Default block)
- **DB-spezifische Ops**: Redis KEYS, Mongo aggregate, MSSQL TOP, Postgres $1-Param
- **Volles REST-Schema** (mehr als die 7 Tools!):
  - `POST /connections` (CreateConnection)
  - `POST /connections/test` (TestConnection)
  - `POST /connections/:id/tables` (CreateTable)
  - `DELETE /connections/:id/tables/:name` (DropTable)
  - `POST /connections/:id/databases` (CreateDatabase)
  - `DELETE /connections/:id/databases/:name` (DropDatabase)
  - `POST /templates/apply`, `GET /templates`
  - `GET /models/local`, `GET /models/remote`, `POST /models/download`, `POST /models/start`

Ergebnis: Agent kann "zeig mir Tabellen" + "SELECT…", aber nicht
"leg eine Tabelle an" / "erstell die DB" / "lade Modell X" per Sprache.

## Zielbild
User sagt: "Erstell eine Tabelle `users` mit id, name, email auf Connection X"
→ Agent wählt `CreateTable` mit korrektem Schema.
Oder: "Lade das Modell Ornith" → Agent ruft `models/download` + `models/start`.
Agent kennt die Guard-Regeln → warnt "DROP braucht execute-Permission",
statt blind zu crashen.

## Design-Skizze (3 Bausteine)

### 1. Single Source of Truth für Tools
`availableTools` nicht mehr hartcodieren. Stattdessen aus dem **registrierten
MCP-Toolset** generieren (der MCP-Server `internal/mcp/server.go` hat bereits
alle 7 Tools + deren JSON-Schemas). Agent importiert dieselbe Tool-Liste.
→ Kein Drift mehr zwischen MCP und Agent.

### 2. Guard + DB-Capabilities in den Prompt
- Guard-Regeln als Textblock in `decideTool()`-Prompt:
  "DROP/DELETE/INSERT blockiert ohne permission connections:execute"
- Pro Connection-Typ Capabilities mitgeben (Redis: KEYS/GET/SET; Mongo:
  aggregate/find; SQL: DDL erlaubt). Agent wählt passendes Tool/Statement.

### 3. DB-spezifische Tools freischalten
- Redis: `redis_get`, `redis_set`, `redis_keys`
- Mongo: `mongo_find`, `mongo_aggregate`
- Generisch erweiterbar über `plugin.DBPlugin`-Interface (jedes Plugin kann
  eigene Tool-Deskriptoren liefern).

## Offene Fragen
- Wie viel Prompt-Context fressen Guard-Regeln + alle Tool-Schemas? (eigene
  `ctx-size` hochsetzen nötig?)
- Soll Agent Schema-Änderungen (DDL) überhaupt dürfen? (Sicherheit: nur mit
  `connections:execute` + Bestätigung?)
- Guard-Awareness: Agent soll blockierte Befehle *erklären*, nicht nur scheitern.

## Nächster Schritt bei Umsetzung
1. `availableTools` aus MCP-Toolset ableiten (Refactor, ~30 Z.)
2. Guard-Regel-Text + DB-Capabilities in `decideTool()`-Prompt
3. DB-spezifische Tools pro Plugin registrieren
4. Tests: Agent lehnt DROP ohne Perm ab + erklärt warum
