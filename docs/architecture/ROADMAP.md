# go-database — Roadmap

## Meilensteine

### M1: Core Engine (Woche 1-2)
```
[████████░░░░░░░░░░░░] 40%
```
- Go Module + Projektstruktur
- Plugin Interface (`internal/plugin/interface.go`)
- **6 DB-Plugins**: Postgres, MySQL, MariaDB, SQLite, MongoDB, Redis
- Connection Manager mit Pooling + Health-Check
- Config-System (JSON/YAML + Env + CLI)
- Interne SQLite-DB für Auth-Store

**Definition of Done**: Alle 6 DBs können verbunden werden, Ping funktioniert, Schema wird ausgelesen.

### M2: Auth & Security (Woche 2-3)
```
[████████████░░░░░░░░] 60%
```
- JWT-Auth (Login, Refresh, Change-Password)
- API-Key-Management (crypto/rand + SHA256)
- Permission-Modell (Rollen + User-Overrides)
- Security Gate (SQL Injection Detection)
- Rate-Limiting

**Definition of Done**: User kann sich einloggen, API-Key generieren, Permissions setzen.

### M3: REST API (Woche 3-4)
```
[████████████████░░░░] 80%
```
- Vollständige REST-API (Auth, Connections, Explorer, Admin, Traffic, API-Keys)
- Einheitliches Response-Format
- Request/Response Logging + Audit-Log
- Swagger/OpenAPI Dokumentation

**Definition of Done**: Alle Endpoints aus PROJEKT.md sind implementiert und getestet.

### M4: Transfer Engine (Woche 4-5)
```
[██████████████████░░] 90%
```
- Data Transfer zwischen beliebigen DB-Typen (z.B. SQLite → PostgreSQL)
- Auto Schema Mapping + Type Conversion
- Dry-Run Modus
- Batch Transfer mit Progress
- Import/Export (CSV, JSON, SQL)

**Definition of Done**: Daten können zwischen je 2 beliebigen DBs transferiert werden.

### M5: Frontend als separater Client (Woche 5-7)
```text
[████████████████████] 100% (Backend-Seite: API bereit)
```
- Frontend wird als **eigenständiger Client** entwickelt (eigene Tech-Stack-Wahl)
- Bindet ausschließlich über die REST/WS/SSE-API an (wie jede externe App)
- Mögliche Features: Login, Stats/Activity, Connection-Management, Explorer,
  Query-Editor, Transfer-Wizard, User/Rollen, Traffic-Monitoring, Settings

**Definition of Done**: Ein separates Frontend nutzt die go-database API
vollständig. Es ist NICHT im Backend-Repo eingebettet (siehe ADR-005).

### M6: Deployment (Woche 7-8)
```text
[████████████████████] 100%
```
- Dockerfile (Go-only Multi-Stage)
- docker-compose.yml (API + interne DBs)
- Sample DBs als Docker-Services
- Linux Production Setup (systemd + docker)
- Windows Dev Setup (direkt + docker)
- Makefile (build, test, run, clean)

**Definition of Done**: `docker-compose up` startet die API; Frontend (sofern
vorhanden) separat.

---

## Platform Support

| Feature | Linux (Docker) | Linux (native) | Windows (native) | Windows (Docker) |
|---------|---------------|----------------|-----------------|-----------------|
| Go API Server | ✅ | ✅ | ✅ | ✅ |
| Frontend (separater Client) | ✅* | ✅* | ✅* | ✅* |
| Interne SQLite DB | ✅ | ✅ | ✅ | ✅ |
| Sample DBs (Docker) | ✅ | ❌ | ❌ | ✅ |
| Externe DBs | ✅ | ✅ | ✅ | ✅ |
| Systemd Service | ❌ | ✅ | ❌ | ❌ |

*\* Frontend ist ein eigener Client, unabhängig vom Backend deploybar.

---

## Datenbanken: 3 Formen

```
┌──────────────────────────────────────────────────┐
│                go-database                        │
│                                                   │
│  1. Externe DBs (Benutzer-definiert)              │
│     └── config/database/external/connections.yaml │
│     └── Beliebiger Host:Port (dev/staging/prod)  │
│                                                   │
│  2. Interne DB (.db SQLite Files)                 │
│     └── database/internal/auth.db                 │
│     └── database/internal/jobs.db (DEPRECATED: nicht mehr genutzt) │
│     └── database/internal/metrics.db (DEPRECATED: nicht mehr genutzt) │
│                                                   │
│  3. Docker DBs (Sample/Test)                      │
│     └── database/docker/docker-compose.yml        │
│     └── Postgres, MySQL, MariaDB, Mongo, Redis    │
│     └── Mit Sample-Daten initialisiert            │
└──────────────────────────────────────────────────┘
```
