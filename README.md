# go-database

**Universelle Datenbank-Middleware / Management-Plattform.**  
Eine API für alle Datenbanken — PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis.

```
Anwendungen                         go-database                      Datenbanken
─────────────                       ───────────                      ──────────
Website ─────┐                                         ┌─── PostgreSQL
Discord Bot ─┤                                         ├─── MySQL
Mobile App ──┼──── API (REST/WS/SSE) ────────► go-database ┼─── MariaDB
AI/Code ─────┤                            │            ├─── SQLite
Minecraft ───┘                            │            ├─── MongoDB
                                      Dashboard         ├─── Redis
                                      (Admin-UI)         └─── ... (erweiterbar)
```

---

## Features

- **6 DB-Engine-Unterstützung** — PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis
- **MSSQL (SQL Server)** — Enterprise/Banken-Support ✅ (seit 2026-07-21)
- **DB-Migration** — Daten inkl. Schema zwischen verschiedenen DB-Typen konvertieren
- **Auto-Provisioning** — Docker + embedded DB-Server starten bei Bedarf
- **Einheitliche REST API** — Alle Datenbanken über eine Schnittstelle
- **First-Setup Wizard** — Admin-Passwort beim ersten Start erzwingen
- **Admin-Dashboard** — Webbasierte Verwaltung als **separater Client** (bindet nur über die API an)
- **Inline DB-Explorer** — Tabellen browsen, Zellen editieren, Zeilen hinzufügen/löschen
- **Query Editor** — SQL direkt im Browser ausführen
- **WebSocket & SSE** — Streaming-Queries + Echtzeit-Updates
- **User & Rollen** — RBAC-Permission-System mit 19 Berechtigungstypen
- **API-Keys** — Generate (crypto/rand + SHA256), Revoke
- **Saisonale Themes** — Dark/Light + Christmas/Halloween/Easter
- **Samples & Import** — Vorgefertigte JSON-Beispieldaten laden
- **AI/LLM Integration** — NL→SQL via OpenRouter (FREE + optional Paid-Fallback), LM Studio oder Ollama; MCP-Server (7 Tools) steuerbar per Config
- **LM Studio Modelle** — `GET /api/v1/models/local` zeigt verfügbare GGUF-Modelle an
- **Internal DB** — Wählbar: SQLite (default) oder PostgreSQL
- **API-only** — Single Binary (reines Go-Backend, kein eingebettetes Frontend)
- **OpenAPI-kompatibel** — Alle Fehler als `{success, data, error, meta}`
- **Mehrere Protokolle (geplant)** — REST ✅, WebSocket ✅, SSE ✅, sowie GraphQL/gRPC/OData/JSON-RPC/SOAP/MQTT/Webhooks/FIX als Design-Spec — siehe `docs/PROTOCOLS.md`
- **Graph-Datenbank-Plugin** — Eigenständiger embedded Graph-DB-Typ (`graph`), Nodes/Edges/Traversal (BFS), JSON-File-Persistenz, kein externer Server
- **AI-Database-Engine** — Vektor-Suche + RAG über das Agent-Tool-Set (`vector_search`, `rag`), pluggable Embedder (Ollama / OpenAI / Hash-Fallback)
- **SSH-Style passwordless Admin-Login** — Ed25519 Public-Key Challenge-Response, keine Passwörter im Klartext; Admin-Accounts via `GODB_ADMIN_PUBKEYS` bootstrap
- **API-Key-Isolation** — Keys gehören einem User (`owner_id`) + haben eigene `db_access`-Scopes; Multi-Tenant-fähig
- **Luckperms-Rollen-Vererbung** — Rollen erben Permissions + DB-Access von Parent-Rollen (Permission `*` = Admin, `-db` = Deny-Wins)

---

## Dokumentation

- **`docs/README.md`** — Docs-Index (alle Dokumente verlinkt)
- **`docs/STRUCTURE.md`** — *Wo ist was, wo kommt was her* — Paket-Karte + Request-Lifecycle
- **`docs/PROJEKT.md`** — Vision, Architektur, Permission-Modell
- **`docs/DECISIONS.md`** — ADRs (warum SQLite-Default, kein Frontend im Repo, Rust-Status, Concurrency)
- **`docs/RISKS.md`** — Offene Risiken für parallele externe Nutzer
- **`docs/api.md`** — REST/WS/SSE-Referenz mit Beispielen
- **`docs/PROTOCOLS.md`** — Alle Protokolle (implementiert + geplant)

---

## Schnellstart (Docker) 🐳

**1. Klonen & starten (nur API):**
```bash
git clone https://github.com/Langli7/go-database
cd go-database
docker compose up -d            # startet die API auf :8080
```

**2. Mit Sample-Datenbanken (Postgres, MySQL, MariaDB, Mongo, Redis):**
```bash
docker compose --profile samples up -d
```

**3. Erstinstallation abschließen:**
```bash
# Setup-Status prüfen
curl http://localhost:8080/api/v1/setup/status

# Admin-Passwort + E-Mail setzen (NUR beim ersten Start)
curl -X POST http://localhost:8080/api/v1/setup/initialize \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"MeinSicheresPasswort123"}'

# Einloggen → JWT erhalten
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"MeinSicheresPasswort123"}'
```

→ API bereit auf `http://localhost:8080/api/v1`. Dashboard/Admin-UI kommt später
als **separater Client** (siehe `docs/DECISIONS.md` ADR-005).

---

## Schnellstart (Lokal, ohne Docker)

```bash
# API bauen & starten
make build
./bin/go-database
# oder: go run ./cmd/server/
# oder: go run ./cmd/godb       (Launcher – baut + startet selbst, kein make nötig)

# Mit PostgreSQL als internem Auth-Store (optional):
GODB_INTERNAL_DB_AUTH_URL=postgres://user:***@localhost:5432/godb_auth ./bin/go-database
```

---

## Quickstart

### Docker (empfohlen)

```bash
# Mit Sample-Datenbanken
docker-compose --profile samples up -d

# Nur API
docker-compose up -d
```

→ Dashboard: http://localhost:8080  

**Lokal entwickeln**

```bash
# Go API starten (API-only, kein Frontend im Repo)
go run ./cmd/server/

# Oder mit PostgreSQL als Auth-Backend
GODB_INTERNAL_DB_AUTH_URL=postgres://user:***@localhost:5432/godb_auth go run ./cmd/server/
```

**Build:**

```bash
make build              # Nur Go → bin/go-database
./bin/go-database
```

> **Hinweis:** Das Frontend (Dashboard/Admin-UI) wird als **separater Client**
> entwickelt und nutzt ausschließlich die REST/WS/SSE-API — wie jede andere
> externe App auch. Siehe `docs/DECISIONS.md` ADR-005.

---

## API Dokumentation

Vollständige API-Dokumentation: [`docs/api.md`](docs/api.md)

### Basis-URL

```
http://localhost:8080/api/v1
```

### Authentifizierung

```bash
# Setup-Status prüfen (beim ersten Start)
curl http://localhost:8080/api/v1/setup/status

# Setup initialisieren (nur beim ersten Start)
curl -X POST http://localhost:8080/api/v1/setup/initialize \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"meinPasswort123"}'

# Login (nach Setup mit dem neuen Passwort)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"meinPasswort123"}'

# Token in allen weiteren Requests
curl http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer <token>"
```

**Wichtig:** Beim ersten Start ist `admin:admin` voreingestellt.  
Der Login gibt dann `403 SETUP_REQUIRED` zurück — das Dashboard leitet automatisch zum Setup-Wizard weiter.

### Response-Format

```json
// Erfolg
{"success": true, "data": {...}, "meta": {"timestamp": "..."}}

// Fehler
{"success": false, "error": {"code": "NOT_FOUND", "message": "..."}, "meta": {...}}
```

---

## Datenbanken: 4 Quellen

```
1. Extern  → Verbindung zu entfernten DB-Servern (dev/staging/prod)
2. Intern  → SQLite- oder PostgreSQL-DB für Auth/Config/Logs
3. Auto-Provisioning → Docker-Container oder embedded Server (kein Setup nötig)
4. Docker  → Sample-Container zum Testen (postgres, mysql, mongo, redis)
```

Alle Konfigurationen in `database/`:

```
database/
├── external/          # YAML-Configs für externe Verbindungen
│   └── sample/        # Beispiel-Configs pro DB-Typ
├── internal/          # Interne DBs (SQLite auth.db oder PostgreSQL)
├── samples/           # Init-Scripts mit Beispieldaten
│   ├── postgres/      # E-Commerce-Schema
│   ├── mysql/         # Blog-Plattform
│   ├── mariadb/       # Inventory-Management
│   ├── sqlite/        # Task-Management
│   ├── mongodb/       # Movies-Sammlung
│   └── redis/         # Sessions, Cache, Queue
└── docker/            # Docker-Compose für Sample-Container
```

---

## Projektstruktur

```
cmd/server/main.go          # Entrypoint
internal/
├── api/                    # REST API
│   ├── handler/            # Request-Handler (auth, connections, explorer, samples, transfer, setup)
│   ├── middleware/         # Auth, CORS, Rate-Limit, Request-ID
│   ├── response/           # Einheitliches {success, data, error, meta}
│   └── router/             # Routen-Definitionen
├── agent/                 # AI-Agent (NL→Tool-Routing, vector_search/rag Tools)
├── ai/                    # Embedder (Ollama/OpenAI/Hash) für Vektor-Suche + RAG
├── auth/                   # JWT, API-Keys, bcrypt, Permissions, Roles
├── config/                 # YAML/JSON/Env-Konfiguration (koanf)
├── connection/             # Connection Manager + Health-Checker
├── internaldb/              # SQLite- oder PostgreSQL-Backend für Auth/Config/Logs
├── plugin/                 # DBPlugin Interface + Registry (6 DB-Typen)
├── provisioner/            # Docker + embedded DB-Auto-Start (Postgres, MySQL, MariaDB)
├── samples/                # JSON-Beispieldaten + Import-Engine (Company HR, E-Commerce, Library, University)
├── transfer/               # Daten-Transfer Engine (Source → Typemap → Target)
├── guard/                  # Query-Guard (SQL-Injection-Prävention)
├── executor/               # Safe-Query-Executor mit Checks
├── suggest/                # SQL-Auto-Vervollständigung
└── transfer/               # DB-Migration zwischen allen Typen
plugins/
├── postgres/               # PostgreSQL (pgx/v5)
├── mysql/                  # MySQL (go-sql-driver/mysql)
├── mariadb/                # MariaDB (go-sql-driver/mysql)
├── sqlite/                 # SQLite (modernc.org/sqlite)
├── mongodb/                # MongoDB (mongo-driver)
└── redis/                  # Redis (go-redis/v9)
└── graph/                  # Graph-DB (embedded, JSON-File-Persistenz)
config/config.example.yaml  # Beispiel-Konfiguration (KEINE Secrets — nutze .env!)
```

---

## Technologie-Stack

| Komponente | Technologie |
|-----------|------------|
| Sprache | Go 1.25+ |
| HTTP-Framework | Gin |
| Dashboard | **separater Client** (eigene Tech, bindet nur über die API) |
| Auth | JWT (golang-jwt) + bcrypt |
| API-Keys | crypto/rand + SHA256 |
| Internal DB | SQLite (modernc.org/sqlite, CGO-frei) **oder** PostgreSQL (pgx/v5) |
| DB-Plugins | pgx/v5, go-sql-driver/mysql, mongo-driver, go-redis/v9 |
| Provisioner | Docker + embedded-postgres + go-mysql-server |
| Embedding | — (kein Frontend embedded; API-only) |
| Logging | slog (strukturiertes JSON) |
| Streaming | WebSocket (gorilla/websocket) + Server-Sent Events |

---

## Security & Konfiguration

- **Secrets gehören NICHT in `config/config.yaml`** — die Datei ist gitignored. Setze API-Keys / JWT-Secret über `.env` (gitignored) oder Env-Variablen:
  - `GODB_MCP_API_KEY` — OpenRouter/LM-Studio API-Key für den AI-Agent / MCP
  - `GODB_AUTH_JWT_SECRET` — JWT-Signing-Secret (persistent über Restarts)
  - `GODB_ADMIN_PUBKEYS` — JSON-Map `{username: "ssh-ed25519 ..."}` für passwordless Admin-Login
- **Beispiel:** `cp config.example.yaml config/config.yaml` und ergänze Secrets nur per Env.
- GitHub **Push Protection** blockiert Commits mit Secrets — rotiere betroffene Keys (OpenRouter etc.) bei Leak.

---

## Entwicklung

### Voraussetzungen

- Go 1.26+
- Docker (optional, für Sample-DBs)

### Tests

```bash
# Alle Tests
make test

# Oder direkt
go test -v -count=1 ./internal/...
```

### Build

```bash
make build              # Nur Go (bin/go-database)
```

---
