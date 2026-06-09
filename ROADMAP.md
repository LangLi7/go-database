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

### M5: Dashboard SPA (Woche 5-7)
```
[████████████████████] 100%
```
- React + TypeScript + Vite als separates Projekt in `web/`
- Login/Logout
- Dashboard-Startseite (Stats, Activity, Connections)
- Connection-Management (UI)
- Explorer (Tabellen browsen, CRUD, Paginierung)
- Query-Editor mit Result-Tabelle
- Transfer-Wizard (Source → Target)
- User/Rollen-Verwaltung
- Traffic-Monitoring
- Settings (adaptives Design)

**Definition of Done**: Dashboard ist voll funktionsfähig, embedded in Go Binary.

### M6: Deployment (Woche 7-8)
```
[████████████████████] 100%
```
- Dockerfile (Multi-Stage Go + React)
- docker-compose.yml (API + interne DBs)
- Sample DBs als Docker-Services
- Linux Production Setup (systemd + docker)
- Windows Dev Setup (direkt + docker)
- Makefile (build, test, run, clean)

**Definition of Done**: `docker-compose up` startet alles, Dashboard ist erreichbar.

---

## Platform Support

| Feature | Linux (Docker) | Linux (native) | Windows (native) | Windows (Docker) |
|---------|---------------|----------------|-----------------|-----------------|
| Go API Server | ✅ | ✅ | ✅ | ✅ |
| Dashboard (embedded) | ✅ | ✅ | ✅ | ✅ |
| Interne SQLite DB | ✅ | ✅ | ✅ | ✅ |
| Sample DBs (Docker) | ✅ | ❌ | ❌ | ✅ |
| Externe DBs | ✅ | ✅ | ✅ | ✅ |
| Systemd Service | ❌ | ✅ | ❌ | ❌ |

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
│     └── database/internal/jobs.db                 │
│     └── database/internal/metrics.db              │
│                                                   │
│  3. Docker DBs (Sample/Test)                      │
│     └── database/docker/docker-compose.yml        │
│     └── Postgres, MySQL, MariaDB, Mongo, Redis    │
│     └── Mit Sample-Daten initialisiert            │
└──────────────────────────────────────────────────┘
```
