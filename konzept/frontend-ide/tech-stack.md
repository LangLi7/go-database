# Frontend-IDE: Tech-Stack & Komponenten

## Stack (Decision: Tauri v2)
- **Shell:** Rust (Tauri v2), WebView2 (Windows), WKWebView (macOS/Linux)
- **UI:** TypeScript + React 18 + Vite
- **Editor:** Monaco Editor (VSCode-Engine) für SQL
- **Grid:** TanStack Table (virtuell,百万行)
- **Terminal:** xterm.js + Rust `portable-pty`
- **State:** Zustand (leichtgewichtig)
- **HTTP:** native fetch + EventSource (SSE) + WebSocket
- **Styles:** Tailwind CSS (industriell, dark-mode-first)
- **Sidecar:** go-database binary (`src-tauri/bin/go-database-server.exe`)

## Komponenten-Baum
```
App
├── LoginScreen          (POST /auth/login, Passkey)
├── MainLayout
│   ├── ConnectionExplorer   (Tree: /connections, /:id/tables, /schema)
│   ├── TabBar
│   │   ├── QueryTab         (Monaco + /:id/query, WS-Stream)
│   │   ├── DataTableTab     (/:id/browse/:table, inline CRUD)
│   │   ├── AgentTab         (/agent/chat + SSE, NL→SQL)
│   │   ├── TransferTab      (/transfer Wizard)
│   │   ├── CryptoTab        (/crypto/* Vault)
│   │   ├── CookbookTab      (/hardware, /recipes)
│   │   └── TerminalTab      (xterm + pty)
│   ├── ActivityFeed         (SSE /sse/activity)
│   └── StatusBar            (User/Role, Connection-Status)
```

## Layout (VSCode-Stil)
```
┌────────────┬──────────────────────────────────────┐
│ Explorer    │  Tab: [Query] [Users] [Agent] [Term] │
│ ├ Connections│ ┌──────────────────────────────────┐ │
│ │ ├ pg-dev   │ │ Monaco SQL-Editor                 │ │
│ │ │ ├ users  │ │ SELECT * FROM users LIMIT 100;   │ │
│ │ │ ├ orders │ └──────────────────────────────────┘ │
│ ├ Samples   │ ┌──────────────────────────────────┐ │
│ ├ Cookbook  │ │ Result-Grid (TanStack)           │ │
│ └ Admin     │ │ [row] [row] [row]  ← inline edit │ │
│             │ └──────────────────────────────────┘ │
├────────────┴──────────────────────────────────────┤
│ StatusBar: user@role | pg-dev:ok | 42ms            │
└────────────────────────────────────────────────────┘
```

## Tauri Commands (Rust → Frontend)
| Command | Zweck |
|---------|-------|
| `start_sidecar()` | go-database launch (Port 8080) |
| `stop_sidecar()` | kill beim Quit |
| `spawn_pty(shell)` | Terminal PTY zurückgeben |
| `docker_compose_up()` | `os/exec docker compose up -d` |
| `open_external(url)` | Browser/Model-Pfad |

## Docker / Standalone (Mode-Auswahl im Login)
- **Standalone:** Sidecar auto-start (kein Docker).
- **Docker:** App triggert `docker compose up`, verbindet localhost:8080.
- **Remote:** URL manuell (Self-Host).

## Build (CI-Anpassung nötig)
- `release.yml` baut go-database (Sidecar) → `src-tauri/bin/`
- Tauri-Build (rust-toolchain) → `.msi`/`.deb`/`.appimage`
- **Hinweis:** aktueller `release.yml` hat Tauri-Schritte (vor ADR-005
  entfernt). Wieder aktivieren + Sidecar-Build verknüpfen.

## Dependencies (Minimal)
- npm: react, vite, monaco-editor, @tanstack/react-table, xterm,
  zustand, tailwindcss
- cargo: tauri (v2), portable-pty, tokio
- **Kein** eigenes Backend-Framework — go-database ist die API.

## Risk/Ceiling
- ponytail: Sidecar-Mgmt (start/kill) global pro App — ok für Single-
  Instance. Multi-Instance-Lock (`.lock`-File) wenn nötig.
- ER-Diagramm: nicht in v1 (später, /schema → Graph-View).
