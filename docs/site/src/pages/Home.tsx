import { permissions, defaultRoles, plugins, frontendPages } from '../data/permissions'

export default function Home() {
  return (
    <main className="page home">
      <section className="hero">
        <h1>go-database</h1>
        <p className="hero-sub">Universelle Datenbank-Middleware &amp; Management-Plattform.<br />Einheitliche REST-API f&uuml;r 6 Datenbank-Engines mit modernem React-Dashboard.</p>
        <div className="hero-cta">
          <a href="/api" className="btn primary">API Dokumentation</a>
          <a href="/dashboard" className="btn secondary">Dashboard Guide</a>
        </div>
        <div className="tags">{plugins.map(p => <span key={p.name} className="tag">{p.name}</span>)}<span className="tag">Go 1.26</span><span className="tag">React 19</span></div>
      </section>

      <div className="home-content">
        <h2>Architektur</h2>
        <div className="arch-diagram">
          <div className="arch-layer" style={{ borderColor: 'var(--accent)' }}><span>React SPA</span><small>Ant Design &middot; 8 Pages</small></div>
          <div className="arch-arrow">&#8593; embed.FS &#8593;</div>
          <div className="arch-layer" style={{ borderColor: 'var(--green)' }}><span>Gin HTTP Server</span><small>RequestID &middot; CORS &middot; RateLimit &middot; Auth &middot; 53 Routes</small></div>
          <div className="arch-split">
            <div className="arch-layer" style={{ borderColor: 'var(--yellow)' }}><span>Handler</span><small>11 Files</small></div>
            <div className="arch-layer" style={{ borderColor: 'var(--yellow)' }}><span>Auth</span><small>JWT + API-Keys</small></div>
            <div className="arch-layer" style={{ borderColor: 'var(--yellow)' }}><span>Guard</span><small>RBAC</small></div>
            <div className="arch-layer" style={{ borderColor: 'var(--yellow)' }}><span>Suggest</span><small>Trie + NL</small></div>
            <div className="arch-layer" style={{ borderColor: 'var(--yellow)' }}><span>Transfer</span><small>Engine</small></div>
          </div>
          <div className="arch-arrow">&#8593; Connection Manager &#8593;</div>
          <div className="arch-layer" style={{ borderColor: 'var(--orange)' }}><span>Plugin Registry</span><small>6 DB-Plugins</small></div>
        </div>

        <h2>Tech Stack</h2>
        <div className="tech-grid">
          {[
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
          ].map(t => (
            <div key={t.name} className="tech-card">
              <span className="tech-icon" style={{ color: t.c }}>{t.icon}</span>
              <div><div className="tech-name">{t.name}</div><div className="tech-desc">{t.desc}</div></div>
            </div>
          ))}
        </div>

        <h2>Datenbank-Plugins</h2>
        <div className="plugin-grid">
          {plugins.map(p => (
            <div key={p.name} className="pcard" style={{ borderTopColor: p.color }}>
              <div className="pcard-name">{p.name}</div>
              <div className="pcard-driver">{p.driver}</div>
            </div>
          ))}
        </div>

        <h2>Kern-Features</h2>
        <div className="card-grid">
          {[
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
          ].map(f => (
            <div key={f.title} className="card">
              <div className="card-icon">{f.icon}</div>
              <h4>{f.title}</h4>
              <p>{f.desc}</p>
            </div>
          ))}
        </div>

        <h2>Frontend &mdash; Dashboard SPA</h2>
        <div className="card-grid">
          {frontendPages.map(p => (
            <div key={p.name} className="card"><h4>{p.name}</h4><p>{p.desc}</p></div>
          ))}
        </div>

        <h2>Permission-Modell</h2>
        <div className="info-box">Standard-Login: <code>admin / admin</code>. Vor Produktion Passwort ändern!</div>
        <table>
          <thead><tr><th>Rolle</th><th>Permissions</th><th>Beschreibung</th></tr></thead>
          <tbody>{defaultRoles.map(r => <tr key={r.name}><td><code>{r.name}</code></td><td><code>{r.perms}</code></td><td>{r.desc}</td></tr>)}</tbody>
        </table>
        <table>
          <thead><tr><th>Permission</th><th>Gruppe</th><th>Beschreibung</th></tr></thead>
          <tbody>{permissions.map(p => <tr key={p.key}><td><code>{p.key}</code></td><td>{p.group}</td><td>{p.desc}</td></tr>)}</tbody>
        </table>

        <h2>Python Beispiel</h2>
        <p>Vollständiges Skript: <code>docs/examples/python/api_test.py</code></p>
        <div className="code-block">
          <div className="code-hdr"><span>python</span></div>
          <pre><code>{`import requests

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
print(r.json()["data"]["rows"])  # [[1]]`}</code></pre>
        </div>

        <h2>Quickstart</h2>
        <div className="code-block">
          <div className="code-hdr"><span>bash</span></div>
          <pre><code>{`# Docker
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
make build`}</code></pre>
        </div>

        <h2>Konfiguration</h2>
        <div className="code-block">
          <div className="code-hdr"><span>yaml</span></div>
          <pre><code>{`server:
  host: "127.0.0.1"
  port: 8080
auth:
  jwt_secret: "change-me-in-production"
  token_duration: 60  # minutes

# Oder Umgebungsvariablen:
# export GODB_AUTH_JWT_SECRET=mein-geheimer-schluessel
# export GODB_SERVER_PORT=9090`}</code></pre>
        </div>
      </div>
    </main>
  )
}
