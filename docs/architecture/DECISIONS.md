## ADR-005: Frontend → separater Client (nicht im Repo)

**Problem**: Wie wird das Dashboard/UI gebaut und deployed?

**Entscheidung**: **Kein Frontend im Repository.** go-database ist eine
**Backend-API** ("Hafen"). Das Frontend (Dashboard, Admin-UI) wird als
**separater, eigener Client** entwickelt und bindet nur über die REST/WS/SSE-API
an — genau wie jede andere externe App (Discord Bot, Mobile App, Website).

**Begründung**:
- API-first: Alles ist programmatisch nutzbar, das Frontend ist nur *ein* Consumer.
- Klare Trennung: Backend (Go) und Frontend (beliebige Tech) entkoppeln.
- Single-Binary Deployment des Backends ohne UI-Build-Abhängigkeit (kein npm,
  keine 14 GB node_modules im Repo).
- Spätere Frontends können in jeder Sprache/Stack gebaut werden.

**Konsequenzen im Code**:
- `web/` und `internal/dashboard/` wurden entfernt.
- `cmd/server/main.go` embeddet kein Dashboard mehr → "API-only mode".
- `Makefile` baut nur noch Go (`make build`); `Dockerfile` ist Go-only.
- Kein `/assets/*`, kein `index.html`-Serving mehr.

**Status (Stand Code-Audit)**: Umgesetzt. Frontend komplett entfernt.

**Alternativen verworfen**:
- Embedded React SPA (wie ursprünglich geplant): koppelt Frontend an Repo,
  Build-Komplexität, verstößt gegen "API-first / Frontend ist externer Client".

---

## ADR-011: Concurrency-Modell für parallele externe API-Consumer

**Problem**: Mehrere externe Clients nutzen die API gleichzeitig (async/parallel)
— es dürfen keine Korruptions-/Race-/Overload-Probleme entstehen.

**Entscheidung**: go-database ist als **nebenläufige API** gebaut; die kritischen
Strukturen sind abgesichert, aber für hohe parallele *Schreiblast* auf SQLite
sind Gegenmaßnahmen nötig (siehe RISKS.md C-1..C-4).

**Was heute sicher ist:**
- Connection Manager: `sync.RWMutex` schützt die Connection-Map
  (`internal/connection/manager.go`).
- JWT-Validation: stateless, nebenläufig sicher.
- Postgres-Plugin: `pgxpool` mit `MaxConns=10` (gute Parallelität).
- SQL-Guard: Command-Whitelist verhindert unerlaubte parallele Write/DDL.

**Bekannte Risiken (offen, siehe RISKS.md):**
- ⚠️ SQLite-Plugin + interner Auth-Store: `SetMaxOpenConns(1)` → nur 1 Writer.
  Bei vielen parallelen Schreibern = Flaschenhals / `database is locked`.
- ⚠️ Globales Rate-Limiting nur für Login, nicht pro User/Key für Query/Execute.
- ⚠️ Kein explizites Worker-Pool mit Concurrency-Limit für lange Tasks.

**Richtlinie für später:**
- Parallele Last → PostgreSQL-Connections bevorzugen (nicht SQLite).
- Globalen Rate-Limiter + Per-Connection-Concurrency-Limit ergänzen.
- Lange Operationen als async Job (Transfer/Scheduler bereits als Basis da).

**Status (Stand Code-Audit)**: Risiken dokumentiert, nicht alle Gegenmaßnahmen
implementiert. Siehe `RISKS.md`.
