# go-database — Projektstruktur & Logik

**Ziel:** Auf einen Blick verstehen, *wo was ist* und *wohin eine Anfrage läuft*.
Alle Pfade sind relativ zum Repo-Root. "✅" = implementiert, "📋" = geplant.

---

## 1. Top-Level Layout

```
go-database/
├── README.md              # Landing Page, Features, Docker/Lokal-Quickstart, Docs-Index
├── Makefile               # make build / make build-all / make clean
├── Dockerfile             # Go-only Multi-Stage → alpine Runtime
├── docker-compose.yml     # Service: api (+ profile "samples": postgres/mysql/mariadb/mongo/redis)
├── docker-compose.override.yml  # lokale Dev-Overrides (debug-log, dev-secret)
├── .dockerignore          # was NICHT in den Build-Kontext kommt
├── .gitignore             # build/runtime-artifacts (bin/, *.db, node_modules, logs)
├── config/
│   ├── config.yaml        # Default-Konfiguration (Server, Auth, InternalDB, LogLevel)
│   ├── config.example.yaml
│   └── config.example.json
├── cmd/
│   └── server/
│       └── main.go        # ENTRYPOINT: DI, Server-Setup, Graceful Shutdown (KEINE Logik!)
├── internal/              # alle Business-Logik (private Go-Packages)
├── plugins/               # 6 DB-Treiber (je ein Package)
├── database/              # samples, external-configs, docker-init, storage
└── docs/                  # alle Dokumente (siehe docs/README.md)
```

**Regel (aus AGENT_RULES.md):** Keine Business-Logik in `cmd/`. `main.go` macht
nur: Config laden → Dependencies bauen (Store, Manager, JWT, …) → Router
registrieren → Server starten → Graceful Shutdown.

---

## 2. `internal/` — die Logik-Schichten

```
internal/
├── api/
│   ├── router/routes.go       # ALLE Routen werden HIER registriert (keine Logik)
│   ├── handler/               # Request-Handler = dünne Adapter (JSON ↔ Manager)
│   │   ├── auth.go connections.go explorer.go admin.go apikeys.go
│   │   ├── transfer.go ws.go sse.go suggest.go crypto.go schedule.go samples.go
│   │   └── setup.go traffic.go permissions.go databases.go importcsv.go ...
│   ├── middleware/            # Auth, CORS, RateLimit, RequestID, Security, DB-Access
│   └── response/response.go   # einheitliches {success, data, error, meta}
│
├── connection/                # Connection Manager (Map + Mutex + Health-Checker)
│   ├── manager.go             # Add/Get/Remove/Ping/Query/Execute + HealthChecker
│   └── model.go               # Connection/Summary/State Typen
│
├── plugin/                    # DBPlugin-Interface + Registry (Plugin-Contract)
│   └── interface.go           # DBPlugin, Config, Result, Schema, Registry
│
├── internaldb/                # Auth-Store: Users, Roles, API-Keys, Audit, Design
│   └── store.go               # SQLite-Default, optional Postgres (SQL-Rewriter)
│
├── auth/                      # JWT, API-Keys (crypto/rand+SHA256), bcrypt, Permissions, Roles
├── config/                    # koanf-basiertes Config-Loading (YAML/JSON/Env GODB_*)
├── guard/                     # SQL-Command-Whitelist (SELECT-only auf /query, etc.)
├── executor/                  # Safe-Query-Executor (Guard + Limits)
├── suggest/                   # SQL-Autocomplete (Trie + Levenshtein)
├── transfer/                  # DB→DB Migration Engine (Typ-Mapping, Batch)
├── provisioner/               # Auto-Start von Docker/embedded DB-Servern
├── scheduler/                 # Cron-Jobs (FileStore: scheduled_jobs.json)
├── crypto/                    # AES/ChaCha/RSA/X25519/Argon2/Ed25519 + Hashes
│   ├── crypto.go               # Typen: Algorithm, Crypter-Interface, Request/Result
│   ├── algorithms.go           # AES-GCM, AES-CBC+HMAC, ChaCha20-Poly1305, RSA-OAEP, X25519
│   ├── modern.go               # Argon2id, Ed25519, ECDSA-P256, Hash-Funktionen (sha256/512, blake2b, sha3)
│   ├── engine.go               # Service: Encrypt/Decrypt/Sign/Verify/Hash/ListAlgorithms
│   ├── store.go                # KeyStore (AES-GCM-verschlüsselte Persistenz, pro User)
│   └── service_test.go         # Unit-Tests (Argon2/ed25519/ecdsa/hash)
└── dashboard/                 # ENTFERNT (Frontend ist separater Client, ADR-005)
```

**Abhängigkeits-Richtung (keine Zyklen):**
```
handler → middleware → manager → plugin → treiber-spezifisch
    ↓           ↓
response    auth/internaldb
```
Handler rufen **nie** direkt DB-Treiber auf — immer über `connection.Manager`,
der das passende `plugin.DBPlugin` aus der Registry holt.

---

## 3. `plugins/` — die 6 Datenbank-Treiber

Jeder implementiert `plugin.DBPlugin` und registriert sich per `init()`:

```
plugins/
├── postgres/  plugin.go   # pgx/v5 + pgxpool (MaxConns=10)        ✅
├── mysql/     plugin.go   # go-sql-driver/mysql (MaxConns=10)     ✅
├── mariadb/   plugin.go   # go-sql-driver/mysql                  ✅
├── mssql/     plugin.go   # microsoft/go-mssqldb (MaxConns=10)   ✅ (Enterprise/Banken)
├── sqlite/    plugin.go   # modernc.org/sqlite (MaxConns=1)      ✅
├── mongodb/   plugin.go   # mongo-driver                        ✅
└── redis/     plugin.go   # go-redis/v9                         ✅
```

Neuer Treiber = neues Package + `func init() { plugin.Register(...) }` +
Eintrag in `cmd/server/main.go` (`_ "go-database/plugins/xxx"`).

**Geplant (Industriestandard, noch nicht implementiert):**
`oracle` (sijms/go-ora), `clickhouse` (clickhouse-go/v2),
`elasticsearch` (elastic/go-elasticsearch), `cassandra/scylla` (gocql),
`influxdb` (influxdb-client-go), `neo4j` (neo4j/go-ne4j), `duckdb`.
Siehe `docs/PROTOCOLS.md` bzw. `docs/ROADMAP.md`.

---

## 4. Request-Lifecycle (wo kommt was her)

```
Client
  │  HTTP/WS/SSE  (+ Authorization: Bearer <JWT|APIKey>)
  ▼
Gin Engine (cmd/server/main.go)
  ├─ middleware.RequestID()      → Request-ID ins Context/Log
  ├─ middleware.CORS()           → Cross-Origin-Header
  ├─ middleware.SecurityHeaders()
  ├─ requestLogger()             → Access-Log
  └─ AuthMiddleware              → JWT/APIKey prüfen, User/Rolle/Perm ins Context
        │
        ▼
  router.SetupRoutes()           → wählt Handler nach Pfad + Permission
        │  (jede Route: permMW(store, PermX) + ggf. dbAccessMW)
        ▼
  handler.Xxx()                  → dünner Adapter
        ├─ liest JSON (ShouldBindJSON)
        ├─ ruft connection.Manager.Query/Execute/...
        │       │
        │       ▼
        │   managedConn.plugin (aus Registry)  → DB-spezifischer Treiber
        │       │                                  (pgx/mysql/modernc/mongo/redis)
        │       ▼
        │   Ziel-Datenbank
        └─ response.Success/Error → {success, data, error, meta}
```

**Sicherheit entlang des Pfads:**
1. AuthMiddleware → wer bist du? (JWT valid / API-Key-Hash-Check)
2. Permission-Middleware (`permMW`) → darfst du das? (z.B. `connections:query`)
3. DB-Access-Middleware (`dbAccessMW`) → darfst du *diese* Connection?
4. SQL-Guard (`guard`) → ist der Befehl erlaubt? (SELECT auf /query, Write auf /execute)
5. Response nie rohe DB-Errors → generisch + Details nur im Log

---

## 5. Laufzeit-Daten & Config

| Was | Wo | Git-ignoriert? |
|-----|-----|----------------|
| Interne Auth-DB (Users/Roles/Keys/Audit) | `database/internal/auth.db` (SQLite) oder `GODB_INTERNAL_DB_AUTH_URL=postgres://...` | ✅ (`*.db`) |
| Sample-Datenbanken | `database/samples/<typ>/` (Init-Scripts) | Init-Scripts ja, Runtime-DBs (`database/storage/`) nein |
| Externe Verbindungs-Configs | `database/external/sample/` (YAML-Templates) | Templates ja |
| Docker-Init für Samples | `database/docker/` | ja |
| Scheduler-Jobs | `scheduled_jobs.json` (FileStore) | ✅ |
| Crypto-Keys | `encryption_keys.json` | ✅ |

Config-Priorität (koanf): **Env (`GODB_*`) > YAML/JSON-File > Defaults**.
Beispiel: `GODB_SERVER_PORT=8099` überschreibt `config.yaml`.

---

## 6. Build & Deploy

```bash
# Lokal
make build                 # → bin/go-database   (CGO_ENABLED=0)
./bin/go-database

# Docker
docker compose up -d                      # nur API
docker compose --profile samples up -d    # API + Sample-DBs

# Cross-Compile (Makefile build-all)
bin/go-database-linux-amd64 / -arm64 / -windows-amd64 / -darwin-amd64
```

---

## 7. Roadmap der Struktur

- 📋 **Neue Protokolle** (GraphQL/gRPC/OData/JSON-RPC/SOAP/MQTT/Webhooks/FIX)
  werden als *zusätzliche Transportschicht* vor `connection.Manager` gehängt —
  dieselbe Logik, anderer Ein-/Ausgang. Siehe `docs/PROTOCOLS.md`.
- 📋 **Frontend** (phpMyAdmin-ähnlich, Rust/Tauri v2) = eigenes Repo,
  konsumiert nur die API. Kein Code im Backend-Repo (ADR-005).
