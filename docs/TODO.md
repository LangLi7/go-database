# go-database — TODO / Status

> **Stand (Code-Audit):** Der Server baut sauber (`go build ./cmd/server/`), alle
> Kernmodule sind implementiert. Die meiste verbleibende Arbeit liegt in
> **Doku-Angleichung**, **Feinschliff** und **neuen DB-Plugins**, nicht in Grundfunktion.
> Die unten als `[x]` markierten Punkte sind im Code vorhanden (nicht nur geplant).

## Phase 0: Struktur & Planung
- [x] **Dokumentation** — PROJEKT.md, AGENT_RULES.md, DECISIONS.md
- [x] **ROADMAP.md** — Meilensteine (siehe PROJEKT.md Roadmap)
- [x] **TODO.md** — Diese Datei
- [x] **Projektstruktur** — Go-Standard Ordner angelegt
- [x] **database/ Ordner** — external, internal, samples, docker
- [x] **internal/transfer/** — Transfer Engine Interface + README
- [x] **Sample Datenbanken** — Init-Scripts für alle 6 DB-Typen
- [x] **Sample Docker Compose** — DB-Container mit Sample-Daten
- [x] **External Connection Configs** — YAML-Templates pro DB-Typ
- [x] **Go Module initialisiert** — `go mod init`

## Phase 1: Core Engine (Backend Fundament)
- [x] **Config-System** — `internal/config/` (YAML/JSON, Env-Override via GODB_)
- [x] **Plugin Interface** — `internal/plugin/interface.go` + Registry
- [x] **DB-Plugins implementiert:**
  - [x] PostgreSQL (`plugins/postgres`, pgx/v5)
  - [x] MySQL (`plugins/mysql`)
  - [x] MariaDB (`plugins/mariadb`)
  - [x] SQLite (`plugins/sqlite`, modernc)
  - [x] MongoDB (`plugins/mongodb`)
  - [x] Redis (`plugins/redis`)
- [x] **Connection Manager** — `internal/connection/manager.go` (Health-Check, 3 Quellen)
- [x] **Auth-System:**
  - [x] JWT — Login, Refresh, Change-Password
  - [x] API-Keys — generate (crypto/rand+SHA256), validate, revoke
  - [x] Permission-Modell — Rollen, Rechte, DB-Zugriff
- [x] **Internal DB (Auth Store)** — `internal/internaldb/` (SQLite Default, optional PG)
- [x] **Transfer Engine Implementation** — `internal/transfer/` (Source/Target/TypeMapping)

## Phase 2: REST API
- [x] **API-Framework** — Gin, Middleware (Auth, CORS, Logging, Request-ID, Rate-Limit)
- [x] **Einheitliches Response-Format** — `{success, data, error, meta}`
- [x] **Auth-Endpoints:** login, refresh, change-password, verify, setup
- [x] **Connection-Endpoints:** CRUD, ping, tables, schema, query, execute, databases
- [x] **Explorer-Endpoints (Daten-CRUD):** browse, row insert/update/delete
- [x] **Transfer-Endpoints:** start, status, cancel, log (inkl. WS + SSE)
- [x] **Admin-Endpoints:** User-CRUD, Rollen-CRUD, Permission-Management, Design
- [x] **Traffic/Monitoring-Endpoints:** stats, requests
- [x] **API-Key-Endpoints:** CRUD
- [x] **Rate-Limiting** — pro User/API-Key (middleware/ratelimit.go)
- [x] **WebSocket / SSE** — query streaming, transfer progress, activity/stats
- [x] **Scheduler** — cron-basierte Jobs (scheduled_jobs.json FileStore)
- [x] **Crypto** — AES-256-GCM, AES-CBC+HMAC, ChaCha20, RSA-OAEP, X25519
- [x] **Samples & Import** — JSON-Beispieldaten + CSV-Import

## Phase 3: Frontend (separater Client)
- [ ] **Frontend als eigener Client** — beliebige Tech, bindet über die REST/WS/SSE-API an
      (siehe DECISIONS.md ADR-005 — kein Frontend im Backend-Repo)
- [ ] **Vollständige UI-Features** — Explorer-Inline-Edit, Query-Autocomplete, Charts
- [ ] **Dark/Light Mode** — in Design-Config als "geplant" markiert

## Phase 4: Advanced Features (größtenteils offen)
- [ ] **Bidirektionale Sync** — zwischen DB-Typen (Transfer-Engine ist die Basis)
- [ ] **Backup/Restore** — Snapshot, Scheduling
- [ ] **CDC (Change Data Capture)** — PG Logical Replication, MySQL Binlog
      (ADR-003 sah Rust vor, ist aktuell NICHT implementiert)
- [ ] **gRPC API** — siehe ADR-003/008 (aktuell nicht im Code)
- [ ] **Event System** — Event Bus + Webhooks
- [ ] **Multi-Node / Cluster**

## Phase 5: Deployment & Qualität
- [x] **Dockerfile** — Multi-Stage Go Build
- [x] **docker-compose** — API + Sample DBs
- [x] **Makefile** — build, test, run, clean
- [ ] **Integration-Tests** — httptest + mock Plugins ausbauen
- [ ] **API-Dokumentation** — docs/api.md mit Router abgleichen (teilweise veraltet)
- [ ] **Security Audit** — SQL Injection, XSS, Rate-Limiting
- [ ] **Weitere DB-Plugins** — MSSQL, Oracle, CockroachDB, Cassandra, ClickHouse, etc.

## Hinweis: Bekannte Doku-Drift (bereits korrigiert in DECISIONS.md)
- ADR-004: SQLite ist der Default-Auth-Store, nicht PostgreSQL.
- ADR-003: Rust-Komponenten (CDC/Wire) sind geplant, aber NICHT im Repo vorhanden
  (die Implementierung ist aktuell reines Go).
