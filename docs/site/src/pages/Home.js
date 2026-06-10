import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { permissions, defaultRoles, plugins, frontendPages } from '../data/permissions';
export default function Home() {
    return (_jsxs("main", { className: "page home", children: [_jsxs("section", { className: "hero", children: [_jsx("h1", { children: "go-database" }), _jsxs("p", { className: "hero-sub", children: ["Universelle Datenbank-Middleware & Management-Plattform.", _jsx("br", {}), "Einheitliche REST-API f\u00FCr 6 Datenbank-Engines mit modernem React-Dashboard."] }), _jsxs("div", { className: "hero-cta", children: [_jsx("a", { href: "/api", className: "btn primary", children: "API Dokumentation" }), _jsx("a", { href: "/dashboard", className: "btn secondary", children: "Dashboard Guide" })] }), _jsxs("div", { className: "tags", children: [plugins.map(p => _jsx("span", { className: "tag", children: p.name }, p.name)), _jsx("span", { className: "tag", children: "Go 1.26" }), _jsx("span", { className: "tag", children: "React 19" })] })] }), _jsxs("div", { className: "home-content", children: [_jsx("h2", { children: "Architektur" }), _jsxs("div", { className: "arch-diagram", children: [_jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--accent)' }, children: [_jsx("span", { children: "React SPA" }), _jsx("small", { children: "Ant Design \u00B7 8 Pages" })] }), _jsx("div", { className: "arch-arrow", children: "\u2191 embed.FS \u2191" }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--green)' }, children: [_jsx("span", { children: "Gin HTTP Server" }), _jsx("small", { children: "RequestID \u00B7 CORS \u00B7 RateLimit \u00B7 Auth \u00B7 53 Routes" })] }), _jsxs("div", { className: "arch-split", children: [_jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--yellow)' }, children: [_jsx("span", { children: "Handler" }), _jsx("small", { children: "11 Files" })] }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--yellow)' }, children: [_jsx("span", { children: "Auth" }), _jsx("small", { children: "JWT + API-Keys" })] }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--yellow)' }, children: [_jsx("span", { children: "Guard" }), _jsx("small", { children: "RBAC" })] }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--yellow)' }, children: [_jsx("span", { children: "Suggest" }), _jsx("small", { children: "Trie + NL" })] }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--yellow)' }, children: [_jsx("span", { children: "Transfer" }), _jsx("small", { children: "Engine" })] })] }), _jsx("div", { className: "arch-arrow", children: "\u2191 Connection Manager \u2191" }), _jsxs("div", { className: "arch-layer", style: { borderColor: 'var(--orange)' }, children: [_jsx("span", { children: "Plugin Registry" }), _jsx("small", { children: "6 DB-Plugins" })] })] }), _jsx("h2", { children: "Tech Stack" }), _jsx("div", { className: "tech-grid", children: [
                            { name: 'Go 1.26', desc: 'Backend-Sprache', icon: '\u26A1', c: 'var(--blue)' },
                            { name: 'Gin', desc: 'HTTP-Framework', icon: '\u2630', c: 'var(--accent)' },
                            { name: 'React 19', desc: 'Frontend', icon: '\u2699', c: 'var(--cyan)' },
                            { name: 'TypeScript', desc: 'Frontend-Sprache', icon: '\u2605', c: 'var(--yellow)' },
                            { name: 'Vite 6', desc: 'Build-Tool', icon: '\u2603', c: 'var(--blue)' },
                            { name: 'Ant Design 6', desc: 'UI-Library', icon: '\u25C6', c: 'var(--orange)' },
                            { name: 'pgx/v5', desc: 'PostgreSQL-Treiber', icon: '\u2630', c: 'var(--green)' },
                            { name: 'modernc.org/sqlite', desc: 'SQLite (CGO-frei)', icon: '\u2665', c: 'var(--red)' },
                            { name: 'Koanf', desc: 'Konfiguration', icon: '\u2630', c: 'var(--accent)' },
                            { name: 'bcrypt', desc: 'Passwort-Hashing', icon: '\u2699', c: 'var(--yellow)' },
                            { name: 'AES-256-GCM', desc: 'JWT-Verschlüsselung', icon: '\u2699', c: 'var(--cyan)' },
                            { name: 'embed.FS', desc: 'Frontend-Einbettung', icon: '\u2699', c: 'var(--green)' },
                        ].map(t => (_jsxs("div", { className: "tech-card", children: [_jsx("span", { className: "tech-icon", style: { color: t.c }, children: t.icon }), _jsxs("div", { children: [_jsx("div", { className: "tech-name", children: t.name }), _jsx("div", { className: "tech-desc", children: t.desc })] })] }, t.name))) }), _jsx("h2", { children: "Datenbank-Plugins" }), _jsx("div", { className: "plugin-grid", children: plugins.map(p => (_jsxs("div", { className: "pcard", style: { borderTopColor: p.color }, children: [_jsx("div", { className: "pcard-name", children: p.name }), _jsx("div", { className: "pcard-driver", children: p.driver })] }, p.name))) }), _jsx("h2", { children: "Kern-Features" }), _jsx("div", { className: "card-grid", children: [
                            { title: 'Einheitliche REST-API', desc: '53 Endpunkte mit JSON-Response-Format { success, data, error, meta }', icon: '\u2699' },
                            { title: '6 DB-Engines', desc: 'PostgreSQL, MySQL, MariaDB, SQLite, MongoDB, Redis', icon: '\u2699' },
                            { title: 'RBAC Permission-System', desc: '19 Permission-Typen, 3 Default-Rollen, User-Overrides', icon: '\u2699' },
                            { title: 'JWT + API-Keys', desc: 'AES-256-GCM JWT, crypto/rand API-Keys, Login-Rate-Limit', icon: '\u2699' },
                            { title: 'SQL-Autocomplete', desc: 'Trie + Levenshtein + Schema-Aware + Natural-Language-Intents', icon: '\u2699' },
                            { title: 'Risikobewertung', desc: 'SQL-Klassifizierung LOW/MEDIUM/HIGH, Bestätigung für riskante Queries', icon: '\u2699' },
                            { title: 'Datentransfer', desc: 'Cross-DB-Transfer mit Schema-Mapping, Dry-Run, Batch-Streaming', icon: '\u2699' },
                            { title: 'Adaptives Dashboard', desc: 'Responsive SPA mit Dark/Light-Mode, DE/EN-i18n', icon: '\u2699' },
                            { title: 'Single Binary', desc: 'Frontend via embed.FS im Go-Binary, keine separaten Dateien', icon: '\u2699' },
                            { title: 'Audit-Logging', desc: 'Strukturiertes JSON-Logging mit slog + X-Request-ID-Tracing', icon: '\u2699' },
                        ].map(f => (_jsxs("div", { className: "card", children: [_jsx("div", { className: "card-icon", children: f.icon }), _jsx("h4", { children: f.title }), _jsx("p", { children: f.desc })] }, f.title))) }), _jsx("h2", { children: "Frontend \u2014 Dashboard SPA" }), _jsx("div", { className: "card-grid", children: frontendPages.map(p => (_jsxs("div", { className: "card", children: [_jsx("h4", { children: p.name }), _jsx("p", { children: p.desc })] }, p.name))) }), _jsx("h2", { children: "Permission-Modell" }), _jsxs("div", { className: "info-box", children: ["Standard-Login: ", _jsx("code", { children: "admin / admin" }), ". Vor Produktion Passwort \u00E4ndern!"] }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Rolle" }), _jsx("th", { children: "Permissions" }), _jsx("th", { children: "Beschreibung" })] }) }), _jsx("tbody", { children: defaultRoles.map(r => _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: r.name }) }), _jsx("td", { children: _jsx("code", { children: r.perms }) }), _jsx("td", { children: r.desc })] }, r.name)) })] }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Permission" }), _jsx("th", { children: "Gruppe" }), _jsx("th", { children: "Beschreibung" })] }) }), _jsx("tbody", { children: permissions.map(p => _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: p.key }) }), _jsx("td", { children: p.group }), _jsx("td", { children: p.desc })] }, p.key)) })] }), _jsx("h2", { children: "Python Beispiel" }), _jsxs("p", { children: ["Vollst\u00E4ndiges Skript: ", _jsx("code", { children: "docs/examples/python/api_test.py" })] }), _jsxs("div", { className: "code-block", children: [_jsx("div", { className: "code-hdr", children: _jsx("span", { children: "python" }) }), _jsx("pre", { children: _jsx("code", { children: `import requests

BASE = "http://localhost:8080/api/v1"

# Login → JWT
r = requests.post(f"{BASE}/auth/login", json={"username":"admin","password":"admin"})
token = r.json()["data"]["token"]
headers = {"Authorization": f"Bearer {token}"}

# SQLite anlegen
r = requests.post(f"{BASE}/connections", headers=headers,
    json={"name":"Test","type":"sqlite","filepath":":memory:"})
conn_id = r.json()["data"]["id"]

# Query
r = requests.post(f"{BASE}/connections/{conn_id}/query", headers=headers,
    json={"query":"SELECT 1 AS x"})
print(r.json()["data"]["rows"])  # [[1]]` }) })] }), _jsx("h2", { children: "Quickstart" }), _jsxs("div", { className: "code-block", children: [_jsx("div", { className: "code-hdr", children: _jsx("span", { children: "bash" }) }), _jsx("pre", { children: _jsx("code", { children: `# Docker
git clone https://github.com/LangLi7/go-database
cd go-database
docker compose --profile samples up -d
# UI: http://localhost:8080 | Login: admin / admin

# Lokale Entwicklung
go build -o bin/go-database ./cmd/server/
./bin/go-database

# Tests
go test ./internal/... -count=1 -v

# Frontend Build + Einbetten
make build` }) })] }), _jsx("h2", { children: "Konfiguration" }), _jsxs("div", { className: "code-block", children: [_jsx("div", { className: "code-hdr", children: _jsx("span", { children: "yaml" }) }), _jsx("pre", { children: _jsx("code", { children: `server:
  host: "127.0.0.1"
  port: 8080
auth:
  jwt_secret: "change-me-in-production"
  token_duration: 60  # minutes

# Oder Umgebungsvariablen:
# export GODB_AUTH_JWT_SECRET=mein-geheimer-schluessel
# export GODB_SERVER_PORT=9090` }) })] })] })] }));
}
