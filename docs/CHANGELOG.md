# Changelog — go-database Modernisierung

Alle Code-Qualitäts- und Modernisierungs-Schritte, chronologisch.
Format: `## [Datum] — Thema`.

---

## [2026-07-21] — Code-Modernisierung (Stand 21.07.2026)

**Ziel:** Lesbarkeit, moderne Go-Patterns, Dokumentation.

### Geändert
- **gofmt:** 3 Dateien formatiert (`internal/api/handler/connections.go`,
  `databases.go`, `suggest_ai.go`) → Repo ist jetzt 100% `gofmt`-clean.
- **`interface{}` → `any`:** in `internal/transfer/typemap.go` + `engine.go`
  (4 Stellen). Go 1.18+ Alias, kürzer, moderner.
- **`.golangci.yml`** hinzugefügt: moderne Linter-Suite
  (errcheck, staticcheck, revive, gocritic, gosec, bodyclose, unconvert,
  durationcheck, wsl, goimports, misspell, …). `golangci-lint v2`-Format.
- **Makefile** erweitert: `fmt`, `vet`, `lint`, `test`, `tidy` Targets
  (zuvor nur `build`/`build-all`/`clean`).
- **`docs/CODING.md`** neu: Coding-Standards + Notizbuch (Sprachregeln,
  Architektur-Regeln, Concurrency, Linting, Backlog).
- **`docs/CHANGELOG.md`** (diese Datei): Modernisierungs-Log.

### Befund (Audit vor der Änderung)
- Go 1.26.2 — aktuell ✅
- Keine `ioutil`-Nutzung ✅
- `errors.Wrap`/`%w` bereits 27× vorhanden ✅
- `interface{}` nur in 2 transfer-Dateien (behoben)
- `gofmt`-Issues in 3 Dateien (behoben)
- `golangci-lint` zuvor nicht konfiguriert (jetzt `.golangci.yml`)

### Bewusst NICHT geändert
- `context.Background()` in 8 Stellen: 6× in `_test.go` (OK), 1× in
  `provisioner/docker.go` (Docker-Provisioning, legitimer Langlauf-Kontext),
  1× in `transfer.go` (Migration-Job-Start). Kein Blind-Rewrite — würde nur
  Risiko ohne Nutzen bringen.
- `internal/` + `plugins/` Package-Layout: sauber (Standard-Go), Umbau würde
  Dutzende Import-Pfade brechen.

### Verifikation
```
go build ./...        → exit 0
go vet ./...          → exit 0
go test ./internal/... → exit 0 (alle Pakete grün)
gofmt -l .            → leer (alle formatiert)
docker compose config → VALID
```

---

## [2026-07-20] — Frontend-Entfernung & Docker-Cleanup
- `web/` (14 GB) + `internal/dashboard/` gelöscht (Frontend = separater Client, ADR-005)
- `main.go` auf API-only umgestellt
- `Makefile`/`Dockerfile` auf Go-only reduziert
- `docs/site/node_modules` (1,4 GB) gelöscht, Scaffold behalten
- Stray-Artefakte entfernt: `server.exe`, `stderr.log`, `stdout.log`, `tree.txt`,
  `build.sh`/`build.bat`, `IDEA.md`, `database/storage/`
- `.gitignore` + `.dockerignore` vervollständigt
- `docs/*` Planungs-Docs nach `docs/` verschoben, `docs/STRUCTURE.md` + `docs/README.md` neu
- `docs/PROTOCOLS.md` neu: alle 11 Protokolle (REST/WS/SSE ✅, 8 weitere 📋 Design-Spec)
- `RISKS.md` + ADR-011 (Concurrency) neu

---

## Versionierung (SemVer, geplant)
- **0.1.0** — aktueller Stand (API-only, 6 DB-Plugins, REST/WS/SSE)
- **0.2.0** (geplant) — GraphQL/gRPC/OData als Transportschichten
- **1.0.0** (geplant) — stabiles API + Tauri-v2 Frontend (separates Repo)
