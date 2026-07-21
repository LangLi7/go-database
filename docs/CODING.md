# go-database — Coding Standards & Modernisierungs-Notizen

**Stand:** 2026-07-21 · Go 1.26.2 · Ziel: lesbarer, moderner, wartbarer Code.

Diese Datei ist das laufende **Notizbuch** für Code-Qualität. Sie wird bei jeder
Modernisierungs-Runde ergänzt (siehe `CHANGELOG.md` für die chronologische Liste).

---

## 1. Sprach-Standards (mit Go 1.26)

| Thema | Regel | Status |
|-------|-------|--------|
| `interface{}` | ❌ verbannt → immer `any` | ✅ erledigt (transfer/) |
| `ioutil.*` | ❌ verbannt → `os`/`io` | ✅ nicht vorhanden |
| Fehler-Wrapping | `fmt.Errorf("...: %w", err)` | ✅ 27× im Code |
| Fehler-Ketten | `errors.Join` wo mehrere Fehler | 📋 optional (selten nötig) |
| `context` | Handler nutzen Request-Context, nicht `context.Background()` | ⚠️ 8× (6× in Tests OK, 2× legitimer Langlauf-Kontext) |
| Generics | `slices`/`maps`/`cmp` aus stdlib statt `golang.org/x/exp` | ✅ stdlib verfügbar (Go 1.21+) |
| Formatierung | `gofmt -w .` (tab-indent, kein trailing space) | ✅ alle Dateien |
| Imports | `goimports` mit `local-prefixes: go-database` | ✅ .golangci.yml |

---

## 2. Architektur-Regeln (nicht verhandelbar)

1. **Keine Logik in `cmd/`.** `main.go` = DI + Router-Registrierung + Shutdown.
2. **Handler sind dünn.** JSON ↔ Manager-Aufruf. Keine SQL, kein Business-Code.
3. **Kein direkter DB-Zugriff aus Handlern.** Immer über `connection.Manager`
   → `plugin.DBPlugin` aus der Registry.
4. **Middleware-Kette für Sicherheit:** Auth → Permission → DB-Access → Guard.
5. **Response-Format:** `{success, data, error, meta}` (siehe `internal/api/response`).
6. **Keine rohen DB-Fehler nach außen** — generisch + Details nur im Log.

---

## 3. Lesbarkeit & Dokumentation

- **Exportierte Symbole brauchen Doc-Comments** (`// Func does ...`).
  `revive:exported` prüft das.
- **Komplexe Funktionen:** kurzer Kommentar *warum*, nicht *was*.
- **Magic Numbers vermeiden** → const-Block (z.B. `const maxConnections = 10`).
- **Fehler-Variablen** sprechend benennen (`errConnNotFound`, nicht `e`).
- **Naming:** `camelCase` lokal, `PascalCase` exportiert, Akronyme `DBName`/`URL` (nicht `DbName`/`Url`).

---

## 4. Concurrency & Thread-Safety (später kritisch)

> Siehe `RISKS.md` C-1 bis C-4 für Details. Hier nur die Code-Regeln:

- **Shared State** immer hinter `sync.Mutex`/`RWMutex` (Connection-Manager macht das ✅).
- **SQLite-Plugin:** `SetMaxOpenConns(1)` → serielle Writer. Bei vielen parallelen
  externen Nutzern: Flaschenhals. Gegenmaßnahme: Postgres für den Internal-Store
  empfehlen (`GODB_INTERNAL_DB_AUTH_URL`).
- **Keine globalen Variablen** für request-scoped Daten — Context nutzen.
- **Goroutines:** immer mit `context` + kontrolliertem Exit (kein leak).

---

## 5. Linting & CI

```bash
make fmt      # gofmt -w .
make vet      # go vet ./...
make lint     # golangci-lint run ./...   (Config: .golangci.yml)
make test     # go test -count=1 ./...
make build    # CGO_ENABLED=0 go build ./cmd/server/
```

`.golangci.yml` aktiviert: errcheck, staticcheck, revive, gocritic, gosec (info),
bodyclose, unconvert, durationcheck, wsl, goimports, misspell, u.a.

> **Hinweis:** `golangci-lint` ist nicht im Repo gebundelt. Bei Bedarf:
> `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
> (oder per GitHub Action in CI).

---

## 6. Offene Modernisierungs-Ideen (Backlog)

- 📋 **Structured errors:** eigener `AppError`-Typ mit Code+HTTP-Status+Details
  (statt generischer `response.Error`).
- 📋 **OpenTelemetry:** Tracing für Request-Lifecycle (wenn Multi-Service-Szenarien).
- 📋 **`errors.Join`** bei Batch-Operationen (transfer/migration) nutzen.
- 📋 **Table-Driven Tests** ausbauen (derzeit nur `smoke_test.go` als Integration).
- 📋 **`go:generate`** für Mock-Generierung in Tests (falls Interfaces wachsen).
- 📋 **CI Pipeline:** GitHub Action (lint + vet + test + build + docker build).

---

## 7. Changelog-Verweis

Chronologische Liste aller Modernisierungs-Schritte: **`CHANGELOG.md`**.
