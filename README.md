# go-database

**Universelle Datenbank-Middleware / Management-Plattform.**  
Eine API für alle Datenbanken — PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis.

```
Anwendungen                         go-database                      Datenbanken
─────────────                       ───────────                      ──────────
Website ─────┐                                         ┌─── PostgreSQL
Discord Bot ─┤                                         ├─── MySQL
Mobile App ──┼──── API (REST/WS/SSE/gRPC) ───► go-database ┼─── MariaDB
AI/Code ─────┤                            │            ├─── SQLite
Minecraft ───┘                            │            ├─── MongoDB
                                     Dashboard         ├─── Redis
                                     (Admin-UI)         └─── ... (erweiterbar)
```

---

## Features

- **6 DB-Engine-Unterstützung** — PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis
- **Einheitliche REST API** — Alle Datenbanken über eine Schnittstelle
- **Admin-Dashboard** — Webbasierte Verwaltung (embedded SPA)
- **Query Editor** — SQL direkt im Browser ausführen
- **Explorer** — Tabellen browsen, CRUD, Paginierung, Filter, Sortierung
- **User & Rollen** — Permission-System mit 19 Berechtigungstypen
- **API-Keys** — Generate (crypto/rand + SHA256), Revoke
- **Adaptives Design** — Themes via API konfigurierbar (Netflix-Stil)
- **Data Transfer** — Daten zwischen verschiedenen DB-Typen migrieren
- **Embedded** — Single Binary (Go + React via embed.FS)

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
→ Login: `admin` / `admin`

### Lokal entwickeln

**Backend:**
```bash
# Go API starten
go run ./cmd/server/

# Oder mit Konfiguration
GODB_AUTH_JWT_SECRET=mysecret go run ./cmd/server/
```

**Frontend (separat):**
```bash
cd web
npm install
npm run dev          # → localhost:5173 (proxied an :8080)
```

**Alles bauen:**
```bash
make build           # React + Go → bin/go-database
./bin/go-database
```

---

## API Dokumentation

Vollständige API-Dokumentation: [`docs/api.md`](docs/api.md)

### Basis-URL

```
http://localhost:8080/api/v1
```

### Authentifizierung

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

# Token in allen weiteren Requests
curl http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer <token>"
```

### Response-Format

```json
// Erfolg
{"success": true, "data": {...}, "meta": {"timestamp": "..."}}

// Fehler
{"success": false, "error": {"code": "NOT_FOUND", "message": "..."}, "meta": {...}}
```

---

## Datenbanken: 3 Formen

```
1. Extern  → Verbindung zu entfernten DB-Servern (dev/staging/prod)
2. Intern  → .db SQLite-Files (auth.db, jobs.db, metrics.db)
3. Docker  → Sample-Container zum Testen (postgres, mysql, mongo, redis)
```

Alle Konfigurationen in `database/`:

```
database/
├── external/          # YAML-Configs für externe Verbindungen
│   └── sample/        # Beispiel-Configs pro DB-Typ
├── internal/          # Interne SQLite-DBs (werden automatisch erstellt)
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
│   ├── handler/            # Request-Handler
│   ├── middleware/         # Auth, CORS
│   ├── response/           # Einheitliches Response-Format
│   └── router/             # Routen-Definitionen
├── auth/                   # JWT, API-Keys, bcrypt, Permissions
├── config/                 # YAML/JSON/Env-Konfiguration
├── connection/             # Connection Manager + Pooling
├── dashboard/              # Embedded React SPA (embed.FS)
├── internaldb/             # SQLite-DB für Auth/Config/Logs
├── job/                    # Async Job System (geplant)
├── model/                  # Datenmodelle
├── monitor/                # Traffic-Monitoring (geplant)
├── plugin/                 # DBPlugin Interface + Registry
├── query/                  # Query Engine
└── transfer/               # Daten-Transfer Engine
plugins/
├── postgres/               # PostgreSQL (pgx/v5)
├── mysql/                  # MySQL (go-sql-driver/mysql)
├── mariadb/                # MariaDB (go-sql-driver/mysql)
├── sqlite/                 # SQLite (modernc.org/sqlite)
├── mongodb/                # MongoDB (mongo-driver)
└── redis/                  # Redis (go-redis/v9)
web/                        # React SPA (Vite + TypeScript)
config/config.yaml          # Default-Konfiguration
```

---

## Technologie-Stack

| Komponente | Technologie |
|-----------|------------|
| Sprache | Go 1.26 |
| HTTP-Framework | Gin |
| Dashboard | React 19 + TypeScript + Vite |
| Auth | JWT (golang-jwt) + bcrypt |
| API-Keys | crypto/rand + SHA256 |
| Internal DB | SQLite (modernc.org/sqlite, CGO-frei) |
| DB-Plugins | pgx/v5, go-sql-driver/mysql, mongo-driver, go-redis/v9 |
| Embedding | embed.FS |
| Logging | slog (strukturiertes JSON) |

---

## Entwicklung

### Voraussetzungen

- Go 1.26+
- Node.js 20+
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
make build              # Komplett (React + Go)
make build-server-quick # Nur Go (ohne UI)
```

---
