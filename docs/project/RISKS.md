# go-database — Risiken & Concurrency (Offene Themen)

> **Zweck:** Laufende Liste späterer Risiken, die erst relevant werden, wenn
> go-database produktiv von mehreren externen Clients parallel genutzt wird
> (Multi-Tenant / High-Concurrency API-Hafen). Aktuell (Stand Code-Audit) ist
> die API funktional, aber NICHT für hohe parallele Last optimiert.

---

## C-1: SQLite als Write-Flaschenhals (HOCH, bei parallelen Nutzern)

**Problem:**
SQLite erlaubt nur einen Writer zur Zeit. Sowohl der interne Auth-Store
(`internal/internaldb/store.go`) als auch das SQLite-DB-Plugin
(`plugins/sqlite/plugin.go`) setzen `SetMaxOpenConns(1)`.

Wenn viele externe Clients parallel schreiben (INSERT/UPDATE/DELETE) über
dieselbe SQLite-Connection, stauen sich die Writer → Latenz-Spikes, evtl.
`database is locked`-Fehler bei längeren Transaktionen.

**Status heute:**
- ✅ Connection Manager ist mutex-geschützt (`sync.RWMutex`) → die Connection-Map
  selbst ist nebenläufig sicher.
- ⚠️ Aber: das *darunterliegende* SQLite-Handle ist auf 1 offene Connection
  begrenzt → parallele Writes serialisiert auf DB-Ebene.

**Empfohlene Gegenmaßnahmen (später):**
- Für parallele Last **PostgreSQL** als Connection-Typ nutzen (pgx-Pool,
  `MaxConns=10` bereits gesetzt in `plugins/postgres`).
- Internen Auth-Store bei Bedarf auf `GODB_INTERNAL_DB_AUTH_URL=postgres://...`
  umstellen.
- Writes über SQLite mit kurzem Retry + busy_timeout (PRAGMA) absichern.
- Oder: Write-Queue im Connection Manager (serialisiert Writes pro DB,
  parallele Reads erlaubt).

---

## C-2: Globales Rate-Limiting fehlt (MITTEL)

**Problem:**
Es existiert nur ein Rate-Limiter für den Login (`middleware/ratelimit.go`
→ `LoginRateLimit`). Es gibt **kein** globales Limit pro User/API-Key für
Query/Execute/SSE. Ein einzelner Client kann die API mit vielen parallelen
Requests fluten → Ressourcen-Erschöpfung (Connections, Memory, Downstream-DB).

**Empfohlene Gegenmaßnahmen (später):**
- Globaler Token-Bucket-Limiter pro API-Key / JWT-Subject in der Auth-Middleware.
- Per-Connection-Concurrency-Limit (max. N parallele Queries pro Connection).
- Optional: WebSocket-/SSE-Verbindungs-Limit pro Client.

---

## C-3: Async / parallele Tasks (geplant, nicht implementiert)

**Problem:**
Lange Tasks (Transfer/Migration, große Queries, Backup) laufen heute
teilweise blocking im Request. Bei vielen parallelen Nutzern blockieren
Worker den HTTP-Server-Thread.

**Status heute:**
- ✅ Transfer Engine + Scheduler (`internal/scheduler`, `scheduled_jobs.json`)
  existieren bereits als async-Basis.
- ⚠️ Explizites Worker-Pool mit Concurrency-Limit + Backpressure fehlt noch.

**Empfohlene Gegenmaßnahmen (später):**
- Worker-Pool mit Semaphore (max. N parallele Jobs) + Queue.
- Lange Operationen als Job mit Status-Polling (statt synchronem Warten).
- Job-Ergebnisse in DB/Redis statt nur im Memory (restart-sicher).

---

## C-4: Memory / State bei vielen Connections (NIEDRIG–MITTEL)

**Problem:**
Alle Connections + Health-Check-State liegen im Memory des Servers. Bei
sehr vielen parallelen Clients wächst der Memory-Footprint; der
Health-Checker pingt alle Connections alle 30s (konfigurierbar).

**Empfohlene Gegenmaßnahmen (später):**
- Max-Connections-Limit (Konfig) + LRU-Eviction inaktiver Connections.
- Health-Check-Interval pro Connection konfigurierbar.

---

## Nicht betroffen (bereits sicher)

- ✅ Connection-Map-Zugriff: `sync.RWMutex` in `connection/manager.go`.
- ✅ JWT-Validation: stateless, nebenläufig sicher.
- ✅ Postgres-Plugin: `pgxpool` mit Connection-Limit.
- ✅ SQL-Guard (`internal/guard`): Command-Whitelist pro Permission → verhindert
  unerlaubte parallele DDL/Write durch Permission-Check, nicht durch Lock.

---

## Nächste Schritte (wann angehen?)

Diese Risiken sind **nicht** kritisch, solange:
- nur wenige Clients die API nutzen,
- primär lesend gearbeitet wird,
- keine öffentliche Exposition ohne Reverse-Proxy/Limit.

Sobald **parallele externe Nutzer** dazukommen (dein Ziel: "Datenbank-Schnittstelle
für mehrere Leute"), sollten **C-1** und **C-2** zuerst angegangen werden.
