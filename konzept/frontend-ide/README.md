# Konzept: go-database Web-IDE (Frontend Client)

_Status:_ Brainstorming — nicht gebaut. Trigger: User will eine moderne,
industrielle DB-IDE (VSCode-Stil) als Frontend für go-database. Fokus:
Rust/Tauri v2 vs Go. Inkl. aller APIs + Agent + Terminal.

## 1. Sprache/Stack-Entscheidung (Rust vs Go)

### Option A: Rust + Tauri v2 (EMPFOHLEN)
- **Warum:** kleinste Binary (~15MB), nativer WebView (system WebView2/
  WKWebView), kein Chromium-Bundle wie Electron. Industriell/professionell.
- **Frontend:** React/Vue/Svelte (TS) im WebView. Gleiche Skills wie eine
  Web-IDE, aber native Shell.
- **Backend-Sidecar:** go-database läuft als Sidecar (Tauri `externalBin`),
  Frontend spricht REST/WS/SSE gegen `localhost:8080`.
- **Terminal:** `rustyline` oder xterm.js im WebView + `pty`-Crate im Rust-
  Backend (Tauri v2 Command).
- **Docker:** Tauri-App ist Standalone; go-database kommt per Docker-Compose
  (siehe unten). App kann `docker compose up` triggern.
- **Nachteil:** Rust-Lernkurve, 2 Sprachen (Rust-Shell + TS-UI).

### Option B: Go + Wails v2
- **Warum:** eine Sprache (Go) fürs Frontend-Shell + kann go-database direkt
  importieren (kein Sidecar nötig!). `wails dev/build` nutzt WebView2.
- **Frontend:** gleiches TS-UI wie A.
- **Terminal:** `creack/pty` + xterm.js.
- **Nachteil:** Wails-Binary größer (~20-30MB), weniger "native feel" als
  Tauri, kleinere Community.

### Empfehlung
- **Tauri v2** für die "industriell/professionell/native" Anforderung +
  kleinste Distribution. go-database als Sidecar (entkoppelt, API-only).
- **Wails** nur wenn du eine-Sprache-Go bevorzugst (kein RPC-Overhead,
  direkter Import). Aber: go-database ist schon API-only (kein UI), also ist
  Tauri + Sidecar der sauberere Schnitt.

→ **Decision: Tauri v2 (Rust-Shell + TS-Webview), go-database als Sidecar.**
  Wails als Alternative dokumentiert, nicht gebaut.

## 2. Verbindungsmodi (Docker / ohne Docker)

### A. Mit Docker (empfohlen für Server/Team)
- `docker-compose up` startet `godb-api` (go-database) + Sample-DBs.
- IDE verbindet via `http://localhost:8080` (oder Remote-Host).
- IDE triggert Compose via Tauri-Command (`os/exec docker compose`).

### B. Ohne Docker (Standalone/Desktop)
- Tauri bundled go-database als `externalBin` Sidecar.
- App startet go-database beim Launch (Port 8080), killt beim Quit.
- Kein Docker nötig — reines Desktop-App.

### C. Remote
- IDE konfiguriert beliebige go-database-URL (Self-Hosted/Server).

## 3. API-Mapping (alle Endpoints aus routes.go)

| Bereich | Endpoint | IDE-Feature |
|---------|----------|-------------|
| Auth | POST /auth/login, /refresh, /verify | Login-Screen, Token-Store |
| Passkeys | /auth/passkeys/* | Passkey-Login (WebAuthn) |
| Connections | GET/POST/DELETE /connections, /test | Connection-Tree, Wizard |
| Explore | GET /connections/:id/tables, /schema, /databases | Schema-Browser |
| Browse | GET /connections/:id/browse/:table | Data-Grid (VSCode-Table) |
| CRUD | POST/PUT/DELETE /connections/:id/row/:table/* | Inline-Edit Grid |
| Query | POST /connections/:id/query, /execute | SQL-Editor (Monaco) |
| DB-Type | POST /db/:type/{query,execute,test} | Quick-Connect (kein Save) |
| WS | GET /ws/query/:id | Streaming-Results |
| SSE | GET /sse/{activity,stats} | Live-Activity-Feed |
| Transfer | POST /transfer, /:id, /log | DB-Migration-Wizard |
| Suggest | POST /suggest, /suggest/ai | Autocomplete + AI-SQL |
| Execute | POST /execute/safe | Guarded-Run |
| Agent | POST /agent/chat, /agent/stream | **AI-Chat-Panel** |
| Models | GET /models/local, /remote | Modell-Picker |
| Templates | GET/POST /templates, /apply | Schema-Templates |
| Download | POST /models/download, /start | Modell-Manager |
| Hardware | GET /hardware, /recipes, /recipes/:name | **Cookbook-Panel** |
| Admin | /admin/users, /roles, /stats, /activity | User/Role-Mgmt |
| APIKeys | GET/POST/DELETE /apikeys | Key-Verwaltung |
| Crypto | /crypto/* (encrypt/decrypt/sign/...) | Vault-Panel |
| Schedules | GET/POST/PUT/DELETE /schedules | Job-Scheduler |
| Samples | GET /samples, POST /samples/:sample | Sample-Loader |

→ **Alle 70+ Endpoints werden vom Client konsumiert.** Keine neue API nötig.

## 4. Feature-Matrix (nach IDE-Vorbildern)

Recherche (2025/2026): DBeaver (universal, aber UI dated), DataGrip (heavy,
beste IntelliSense), TablePlus (clean, fast, macOS), Beekeeper Studio (OSS,
tabbed, simple), DBCode (VSCode-Extension, AI-first).

| Feature | Vorbild | IDE-Implementierung |
|---------|---------|---------------------|
| Tabbed SQL-Editor | Beekeeper/DataGrip | Monaco Editor, multi-tab |
| Schema-Browser (Tree) | DBeaver | Connection-Explorer links |
| Data-Grid Inline-Edit | TablePlus | Virtuelle Tabelle (TanStack) |
| AI-Chat + NL→SQL | DBCode | Agent-Panel (SSE) |
| Autocomplete/Suggest | DataGrip | /suggest + Monaco LSP |
| ER-Diagramm | DBeaver | (später) /schema → Graph |
| Terminal (SQL/Shell) | VSCode | xterm.js + pty |
| Crypto-Vault | — | /crypto-Panel |
| Live-Activity | — | SSE-Feed |
| Multi-Connection | DBeaver | Tabs pro Connection |

## 5. Terminal (VSCode-Stil)
- **xterm.js** im WebView (TS).
- **Rust-Backend:** `portable-pty` (Tauri Command) spawned Shell
  (pwsh/bash), stdout/stderr → xterm via Event.
- **Modi:** (a) System-Shell, (b) SQL-REPL gegen go-database
  (`/connections/:id/query` im Loop), (c) `godb`-CLI.
- ponytail: System-Shell reicht zuerst; SQL-REPL ist nur Wrapper um /query.

## 6. Architektur (Tauri v2)
```
godb-ide/                      (Rust + TS)
├── src-tauri/                 (Rust: sidecar-mgmt, pty, docker-trigger)
│   ├── Cargo.toml
│   ├── tauri.conf.json        (externalBin: go-database sidecar)
│   └── src/main.rs
├── src/                       (TS: React/Vue + xterm + Monaco)
│   ├── api/                   (typed client für alle /api/v1/*)
│   ├── components/            (Explorer, Grid, Editor, AgentPanel, Terminal)
│   └── App.tsx
└── package.json
```
- **Sidecar:** `go-database` binary im `src-tauri/bin/` (per CI gebaut,
  siehe release.yml-Refactor).
- **Kein eigener Server** im Frontend — alles geht durch go-database (Single
  Source of Truth, Auth/RBAC/Crypto bereits server-seitig).

## 7. Offene Fragen / Risiken
- Binär-Größe: Tauri+Sidecar ~40MB (ok für Desktop).
- Auto-Update: Tauri `updater` oder ghcr-Image für Server-Modus.
- Plattform: Windows (dein Host) primär → WebView2 vorinstalliert.
- ER-Diagramm: aufwändig, nur wenn User will (später).

## 8. Nächster Schritt
- **Spike:** Tauri-v2-Scaffold + Sidecar-Start + Login + einen Query-Tab.
  Nicht das ganze Feature-Set vorab.
- Wahl: Tauri v2 (Rust) bestätigt? Dann Spike bauen.
```
