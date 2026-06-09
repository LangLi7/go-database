# go-database — TODO

## Phase 0: Struktur & Planung
- [x] **Dokumentation** — PROJEKT.md, AGENT_RULES.md, DECISIONS.md
- [x] **ROADMAP.md** — Meilensteine M1-M6 mit Platform-Support-Matrix
- [x] **TODO.md** — Diese Datei
- [x] **Projektstruktur** — Go-Standard Ordner angelegt
- [x] **database/ Ordner** — external, internal, samples, docker
- [x] **internal/transfer/** — Transfer Engine Interface + README
- [x] **Sample Datenbanken** — Init-Scripts für alle 6 DB-Typen
- [x] **Sample Docker Compose** — Alle 5 DB-Container mit Sample-Daten
- [x] **External Connection Configs** — YAML-Templates pro DB-Typ
- [ ] **Go Module initialisiert** — `go mod init github.com/.../go-database`

## Phase 1: Core Engine (Backend Fundament)
- [ ] **Config-System** — `internal/config/` (YAML/JSON, CLI-Flags, Env-Override, 3 DB-Formen)
- [ ] **Plugin Interface** — `internal/plugin/interface.go` + Registry
- [ ] **DB-Plugins implementieren:**
  - [ ] PostgreSQL (`plugins/postgres`)
  - [ ] MySQL (`plugins/mysql`)
  - [ ] MariaDB (`plugins/mariadb`)
  - [ ] SQLite (`plugins/sqlite`)
  - [ ] MongoDB (`plugins/mongodb`)
  - [ ] Redis (`plugins/redis`)
- [ ] **Connection Manager** — `internal/connection/manager.go` (Pooling, Health-Check, 3 Quellen)
  - [ ] Externe DBs (remote host:port)
  - [ ] Interne DBs (lokale .db files)
  - [ ] Docker DBs (Container-URLs)
- [ ] **Auth-System:**
  - [ ] JWT — Login, Refresh, Change-Password
  - [ ] API-Keys — generate (crypto/rand+SHA256), validate, revoke
  - [ ] Permission-Modell — Rollen, Rechte, DB-Zugriff
- [ ] **Internal DB (Auth Store)** — `internal/internaldb/` (SQLite für User/Rollen/Logs)
- [ ] **Transfer Engine Implementation** — `internal/transfer/` (Source/Target/TypeMapping)

## Phase 2: REST API
- [ ] **API-Framework aufgesetzt** — Gin, Middleware (Auth, CORS, Logging, Request-ID)
- [ ] **Einheitliches Response-Format** — `{"success": true/false, "data": ..., "error": {...}, "meta": {...}}`
- [ ] **Auth-Endpoints:**
  - [ ] `POST /api/v1/auth/login`
  - [ ] `POST /api/v1/auth/refresh`
  - [ ] `POST /api/v1/auth/change-password`
- [ ] **Connection-Endpoints:**
  - [ ] `GET/POST /api/v1/connections`
  - [ ] `GET/DELETE /api/v1/connections/:id`
  - [ ] `GET /api/v1/connections/:id/ping`
  - [ ] `GET /api/v1/connections/:id/tables`
  - [ ] `GET /api/v1/connections/:id/schema`
  - [ ] `POST /api/v1/connections/:id/query`
  - [ ] `POST /api/v1/connections/:id/execute`
- [ ] **Explorer-Endpoints (Daten-CRUD):**
  - [ ] `GET /api/v1/connections/:id/browse/:table`
  - [ ] `POST /api/v1/connections/:id/row/:table`
  - [ ] `PUT /api/v1/connections/:id/row/:table/:pk/:val`
  - [ ] `DELETE /api/v1/connections/:id/row/:table/:pk/:val`
- [ ] **Transfer-Endpoints:**
  - [ ] `POST /api/v1/transfer` — neuen Transfer-Job starten
  - [ ] `GET /api/v1/transfer/:id` — Status abfragen
  - [ ] `DELETE /api/v1/transfer/:id` — Job abbrechen
  - [ ] `GET /api/v1/transfer/:id/log` — Fehler-Log abrufen
- [ ] **Admin-Endpoints:**
  - [ ] User-CRUD
  - [ ] Rollen-CRUD
  - [ ] Permission-Management
- [ ] **Traffic/Monitoring-Endpoints:**
  - [ ] `GET /api/v1/traffic/stats`
  - [ ] `GET /api/v1/traffic/requests`
- [ ] **API-Key-Endpoints:**
  - [ ] `GET/POST /api/v1/apikeys`
  - [ ] `DELETE /api/v1/apikeys/:prefix`
- [ ] **Rate-Limiting** — pro User/API-Key

## Phase 3: Dashboard (Separates Projekt in web/)
- [ ] **React-Projekt aufgesetzt** — Vite + TypeScript + React Router
- [ ] **Build-Pipeline** — `npm run build` => `web/dist/` => `embed.FS`
- [ ] **Login/Logout** — JWT basiert
- [ ] **Dashboard-Startseite** — Stats-Cards, Activity Feed, Connection-Übersicht
- [ ] **Connection-Management (UI)** — Grid-Ansicht, Add/Ping/Delete
- [ ] **Explorer (UI)** — Tabellen-Liste, Daten browsen/paginieren/sortieren/filtern
- [ ] **Inline-Editing** — Zellen editieren, Insert/Delete Rows
- [ ] **Query-Editor** — SQL-Editor, Connection-Selector, Result-Tabelle
- [ ] **Transfer-Wizard (UI)** — Source → Target Auswahl, Dry-Run, Progress-Bar
- [ ] **Traffic-Monitoring (UI)** — Charts, Request-Log, Audit-Log
- [ ] **User/Rollen-Verwaltung** — CRUD + Permission-Matrix
- [ ] **API-Key-Management (UI)** — Generate, View, Revoke
- [ ] **Settings (UI)** — Design-Config (adaptiv)
- [ ] **Dark/Light Mode**

## Phase 4: Advanced Features
- [ ] **Bidirektionale Sync** — zwischen verschiedenen DB-Typen
- [ ] **Backup/Restore** — Snapshot aus UI, lokal/S3, Scheduling
- [ ] **CDC (Change Data Capture)** — PostgreSQL Logical Replication, MySQL Binlog, MongoDB Change Streams
- [ ] **WebSocket** — Live Query Streaming, Realtime Monitoring
- [ ] **SSE** — Job Progress, Log Streaming
- [ ] **gRPC API** — Service-zu-Service Kommunikation
- [ ] **Event System** — Interner Event Bus + Outbound Webhooks
- [ ] **Job System** — Worker-Pool, Cron Scheduling, Retry
- [ ] **Multi-Node / Cluster** — Horizontal skalieren

## Phase 5: Deployment & Qualität
- [ ] **Dockerfile API** — Multi-Stage Go Build (Linux)
- [ ] **Dockerfile UI** — Multi-Stage React Build (Linux)
- [ ] **docker-compose.yml (Root)** — API + interne DBs + Sample DBs
- [ ] **Makefile** — build, test, run, clean (Linux + Windows)
- [ ] **Linux Service** — systemd unit file
- [ ] **Windows Dev** — Direktstart ohne Docker
- [ ] **Integration-Tests** — httptest für API, mock für Plugins
- [ ] **API-Dokumentation** — OpenAPI/Swagger
- [ ] **Security Audit** — SQL Injection, XSS, Rate-Limiting
- [ ] **Weitere DB-Plugins** — MSSQL, Oracle, CockroachDB, Cassandra, ClickHouse, etc.
