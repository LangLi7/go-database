# go-database — Dokumentation

Zentrale Anlaufstelle für alle Docs. Die API ist generisch ("Hafen" für
Datenbanken); das Frontend (Dashboard/Admin-UI) ist ein **separater Client**
(siehe `DECISIONS.md` ADR-005).

## Schnellstart
- **README (Root)** — Landing Page, Features, Docker/Lokal-Quickstart, Protokoll-Überblick.
- **STRUCTURE.md** — *Wo ist was, wo kommt was her.* Paket-Karte + Request-Lifecycle.

## Konzept & Entscheidungen
- **PROJEKT.md** — Vision, Ziele, Architektur-Übersicht, Permission-Modell, Roadmap-Kontext.
- **DECISIONS.md** — ADRs (Architecture Decision Records): warum SQLite-Default,
  warum kein Frontend im Repo, Rust-Status, Concurrency-Modell.
- **RISKS.md** — Offene Risiken (Concurrency, Rate-Limit, async Tasks) für parallele externe Nutzer.
- **ROADMAP.md** — Meilensteine M1–M6.
- **TODO.md** — Status der Phasen (was ist implementiert).
- **AGENT_RULES.md** — Regeln für KI-Agenten / Mitarbeiter bei Änderungen.

## API
- **api.md** — Vollständige REST/WS/SSE-Referenz mit curl-Beispielen (implementiert).
- **PROTOCOLS.md** — Alle Protokolle: REST/WS/SSE (✅) + GraphQL/gRPC/OData/JSON-RPC/
  SOAP/MQTT/Webhooks/FIX (📋 Design-Spec).
- **CRYPTO.md** — Kryptographie-Anleitung: Algorithmen, Endpoints, Bedrohungsmodell, Zero-Trust, JtR.

## Projektstruktur (Kurzform)
```
go-database/
├── README.md            # Landing / Quickstart
├── Makefile             # make build / build-all / clean
├── Dockerfile           # Go-only, alpine runtime
├── docker-compose.yml   # api + optional sample DBs (profile: samples)
├── config/              # config.yaml (Default-Konfiguration)
├── cmd/server/          # Entrypoint (main.go)
├── internal/            # Alle Business-Logik (privat)
├── plugins/             # 6 DB-Plugins (postgres, mysql, mariadb, sqlite, mongodb, redis)
├── database/            # samples, external, docker-init, storage
└── docs/                # Diese Dokumentation
```
