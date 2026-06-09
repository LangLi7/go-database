# go-database — Architecture Decisions

## Format
Jede Entscheidung hat:
- **Problem**: Was musste entschieden werden
- **Entscheidung**: Was gewählt wurde
- **Begründung**: Warum
- **Alternativen**: Was verworfen wurde und warum

---

## ADR-001: Web Framework → Gin

**Problem**: Welches Go HTTP Framework für die REST API?

**Entscheidung**: Gin

**Begründung**:
- Sehr gute Performance (httprouter-basiert)
- Grosse Community, viel Middleware verfügbar
- Einfaches Middleware-System für Auth, Logging, Security Gate
- Gut dokumentiert

**Alternativen verworfen**:
- Echo: Ähnlich gut, aber Gin hat mehr Community-Momentum
- Chi: Zu minimal, mehr Boilerplate nötig
- net/http pure: Zu viel Boilerplate für diesen Scope

---

## ADR-002: PostgreSQL Treiber → pgx/v5

**Problem**: Welcher PostgreSQL Treiber?

**Entscheidung**: `github.com/jackc/pgx/v5`

**Begründung**:
- Direkter PostgreSQL Treiber (kein database/sql Overhead)
- Unterstützt PostgreSQL-spezifische Features (LISTEN/NOTIFY, COPY, Arrays)
- Besser für Connection Pooling (pgxpool)
- Wird benötigt für CDC via LISTEN/NOTIFY

**Alternativen verworfen**:
- `database/sql` + `lib/pq`: Zu generisch, verliert PG-spezifische Features
- `database/sql` + pgx driver: Kompromiss, nicht optimal

---

## ADR-003: Go + Rust Aufteilung

**Problem**: Was kommt in Go, was in Rust?

**Entscheidung**: Go für Business Logic + API, Rust für Performance-kritische Low-Level Teile

**Go**:
- REST/WS/SSE/gRPC Handler
- Connection Manager, Job System, Event Bus
- Plugin Interface + Loader
- Auth, Config, Dashboard Serving

**Rust**:
- CDC Engine (Binlog/WAL Reading — byte-level Protokoll Parsing)
- Wire Protocol Handler (PG + MySQL Wire — performance-kritisch)
- AES-256-GCM Crypto (Credentials)
- Migration Transform Engine (hoher Datendurchsatz)

**Kommunikation Go ↔ Rust**: localhost gRPC (nicht CGo)

**Begründung**:
- CGo hat Lock-Overhead und macht Cross-Compilation schwieriger
- gRPC ist klarer getrennt, einfacher zu testen
- Rust Teile können auch standalone laufen

**Alternativen verworfen**:
- Alles Go: CDC und Wire Protocol in Go wäre deutlich langsamer
- Alles Rust: Zu viel Aufwand, Go Ecosystem für Web besser

---

## ADR-004: Interne Datenhaltung → JSON + PostgreSQL

**Problem**: Wo werden Config, Jobs, Logs gespeichert?

**Entscheidung**:
- Config (Connections, API Keys, Server Settings) → JSON Files
- Jobs, Audit Log, Metrics → PostgreSQL (interner Container)

**Begründung**:
- Config als JSON: Einfach editierbar, versionierbar (git), kein DB-Start nötig für Konfiguration
- Jobs/Logs in PostgreSQL: Strukturierte Queries, joins, filtering, history nötig
- Kein SQLite: PostgreSQL bereits vorhanden (interner Container), kein zusätzlicher Treiber

**Alternativen verworfen**:
- Alles PostgreSQL: Config in DB macht Bootstrap schwieriger (Henne-Ei Problem)
- Alles SQLite: Schlechter für concurrent writes (Job System)
- Alles JSON: Jobs/Logs als JSON Files nicht querybar

---

## ADR-005: Frontend → React Embedded

**Problem**: Wie wird das Dashboard gebaut und deployed?

**Entscheidung**: React + TypeScript, via `embed.FS` in Go Binary eingebettet

**Begründung**:
- Single Binary Deployment (kein separater Frontend Container nötig)
- React Ecosystem: Monaco Editor, xterm.js, recharts, react-flow alle verfügbar
- TypeScript: Type Safety, besser wartbar

**Build Flow**:
```
npm run build → dist/ → embed.FS → Go Binary
```

**Alternativen verworfen**:
- Separater Container: Mehr Complexity, nginx nötig
- Vue: React hat besseres Ecosystem für die benötigten Libraries
- HTMX/Server-rendered: Zu limitiert für Realtime Features (WebSocket, Charts)

---

## ADR-006: WebSocket Library → gorilla/websocket

**Problem**: Welche WebSocket Library in Go?

**Entscheidung**: `github.com/gorilla/websocket`

**Begründung**:
- De-facto Standard in Go Ecosystem
- Gut dokumentiert, stabil
- Gin Middleware verfügbar

**Alternativen verworfen**:
- `nhooyr.io/websocket`: Moderner aber kleinere Community
- Gin built-in: Nicht vorhanden

---

## ADR-007: Job Queue → Channel-basiert (kein Redis/NATS)

**Problem**: Externe Queue (Redis, NATS) oder intern?

**Entscheidung**: Interne Channel-basierte Queue (Go)

**Begründung**:
- Keine externe Abhängigkeit nötig
- Go Channels sind für diesen Usecase ausreichend
- Einfacher Deployment (kein Redis Container)
- Job State wird in PostgreSQL persistiert (Restart-sicher)

**Alternativen verworfen**:
- Redis Queue: Overkill für single-node, externe Abhängigkeit
- NATS: Zu viel Complexity für lokalen Usecase
- Asynq: Redis-abhängig

---

## ADR-008: CDC Kommunikation → localhost gRPC

**Problem**: Wie kommunizieren Go und Rust CDC Engine?

**Entscheidung**: localhost gRPC (Rust als gRPC Server, Go als Client)

**Begründung**:
- Saubere Trennung, beide Teile unabhängig testbar
- Protobuf typisiert = klares Interface
- Rust gRPC Server (tonic) gut dokumentiert
- Kein CGo Overhead, keine Memory Safety Probleme

**Alternativen verworfen**:
- CGo: Build Complexity, Lock-Overhead, Cross-Compilation schwieriger
- Unix Socket raw: Kein Type Safety, mehr Serialization Code
- Shared Memory: Zu komplex, unsafe

---

## ADR-009: Auth → JWT

**Problem**: Wie wird Auth für Dashboard + API implementiert?

**Entscheidung**: JWT (JSON Web Tokens)

**Begründung**:
- Stateless (kein Session Store nötig)
- Einfach in API Clients verwendbar
- Refresh Token Pattern für längere Sessions

**API Keys**:
- Separates System für programmatischen Zugriff
- AES-256-GCM verschlüsselt in JSON Config gespeichert (Rust Crypto)
- Permissions pro Key (connections, operations)

---

## ADR-010: Wire Protocol → Rust

**Problem**: PG und MySQL Wire Protocol in Go oder Rust?

**Entscheidung**: Rust

**Begründung**:
- Wire Protocols sind byte-level Parsing — performance-kritisch
- Viele concurrent Connections möglich
- Rust hat keine GC Pauses (wichtig für niedrige Latenz)
- Existierende Rust Libraries: `pgwire` Crate als Basis verwendbar

**Alternativen verworfen**:
- Go: Möglich aber GC Pauses bei vielen Connections problematisch
- Externe Proxy (pgbouncer): Kein direkter Zugriff auf Query-Ebene
