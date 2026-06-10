import CodeBlock from '../components/CodeBlock'

const sec = (id: string) => {
  setTimeout(() => {
    const el = document.getElementById(id)
    if (el) el.scrollIntoView({ behavior: 'smooth' })
  }, 50)
}

export default function DashboardPage() {
  return (
    <main className="page dash-page">
      <h1>Dashboard Guide</h1>
      <p className="page-sub">Die Dashboard-Seite ist der Startbildschirm der Web-Oberfläche. Sie liefert auf einen Blick alle wichtigen Kennzahlen und Verbindungsstatus.</p>

      <div className="dash-nav">
        {['Übersicht', 'Layout & Struktur', 'Statistik-Karten', 'Verbindungstabelle', 'Datenfluss', 'Routing', 'Embedding', 'i18n'].map(s => (
          <a key={s} href={`#${s.toLowerCase().replace(/[^a-z]/g, '')}`} onClick={e => { e.preventDefault(); sec(s.toLowerCase().replace(/[^a-z]/g, '')) }} className="dash-nav-item">{s}</a>
        ))}
      </div>

      <section id="Übersicht" className="dash-section">
        <h2>Übersicht</h2>
        <p>Das Dashboard unter <code>/</code> wird nach dem Login angezeigt und kombiniert Statistik-Karten mit einer detaillierten Verbindungstabelle. Es ist in React 19 + TypeScript mit Ant Design 6 umgesetzt und responsiv gestaltet.</p>
        <div className="dash-mock">
          <div className="mock-grid">
            {['Verbindungen', 'Aktiv', 'Benutzer', 'Datenbanken'].map(s => <div key={s} className="mock-card"><div className="mock-num">0</div><div className="mock-label">{s}</div></div>)}
          </div>
          <div className="mock-table"><div className="mt-hdr">Verbindungen</div><div className="mt-row"><span>Keine Daten</span></div></div>
        </div>
      </section>

      <section id="layoutstruktur" className="dash-section">
        <h2>Layout &amp; Struktur</h2>
        <p>Das Dashboard verwendet ein responsives Grid-Layout mit Ant Design <code>Row</code> / <code>Col</code>:</p>
        <table>
          <thead><tr><th>Breakpoint</th><th>Karten pro Reihe</th></tr></thead>
          <tbody>
            <tr><td><code>lg</code> (&ge; 1200px)</td><td>4</td></tr>
            <tr><td><code>sm</code> (&ge; 576px)</td><td>2</td></tr>
            <tr><td><code>xs</code> (&lt; 576px)</td><td>1 (gestapelt)</td></tr>
          </tbody>
        </table>
      </section>

      <section id="statistikkarten" className="dash-section">
        <h2>Statistik-Karten</h2>
        <table>
          <thead><tr><th>Karte</th><th>Datenquelle</th><th>Farbe</th></tr></thead>
          <tbody>
            <tr><td>Verbindungen</td><td><code>api.listConnections().length</code></td><td>Indigo</td></tr>
            <tr><td>Aktiv</td><td><code>.filter(c =&gt; c.state === 'connected')</code></td><td>Grün</td></tr>
            <tr><td>Benutzer</td><td><code>api.listUsers().length</code></td><td>Gelb</td></tr>
            <tr><td>Datenbanken</td><td><code>api.listConnections().length</code></td><td>Blau</td></tr>
          </tbody>
        </table>
        <CodeBlock code={`const [connections, setConnections] = useState<any[]>([])
const [users, setUsers] = useState<any[]>([])
const [loading, setLoading] = useState(true)

const connected = connections.filter(c => c.state === 'connected')`} lang="tsx" />
      </section>

      <section id="verbindungstabelle" className="dash-section">
        <h2>Verbindungstabelle</h2>
        <p>Unter den Karten folgt eine Ant Design <code>Table</code> mit allen Verbindungen:</p>
        <table>
          <thead><tr><th>Spalte</th><th>Darstellung</th></tr></thead>
          <tbody>
            <tr><td>Name</td><td>Text</td></tr>
            <tr><td>Typ</td><td>Tag (PostgreSQL, MySQL, ...)</td></tr>
            <tr><td>Status</td><td>Tag farbcodiert (grün/rot/orange)</td></tr>
            <tr><td>Latenz</td><td>ms</td></tr>
            <tr><td>Quelle</td><td>external / local / docker / file</td></tr>
          </tbody>
        </table>
        <CodeBlock code={`<Table
  dataSource={connections}
  columns={columns}
  rowKey="id"
  pagination={false}
  size="small"
/>`} lang="tsx" />
      </section>

      <section id="datenfluss" className="dash-section">
        <h2>Datenfluss</h2>
        <p>Zwei parallele API-Aufrufe via <code>Promise.all</code> beim Mount:</p>
        <div className="flow">
          <span className="flow-step">Mount</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step" style={{ borderColor: 'var(--blue)' }}>Promise.all</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step">listConnections</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step" style={{ borderColor: 'var(--green)' }}>setState</span>
        </div>
        <div className="flow">
          <span className="flow-step">Mount</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step" style={{ borderColor: 'var(--blue)' }}>Promise.all</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step">listUsers</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step" style={{ borderColor: 'var(--green)' }}>setState</span>
        </div>
        <CodeBlock code={`useEffect(() => {
  let cancelled = false
  setLoading(true)
  Promise.all([api.listConnections(), api.listUsers()])
    .then(([connRes, userRes]) => {
      if (cancelled) return
      if (connRes.success) setConnections(connRes.data || [])
      if (userRes.success) setUsers(userRes.data || [])
      setLoading(false)
    })
  return () => { cancelled = true }
}, [])`} lang="tsx" />
      </section>

      <section id="routing" className="dash-section">
        <h2>Routing</h2>
        <CodeBlock code={`<Routes>
  <Route path="/" element={<Dashboard />} />
  <Route path="/connections" ... />
  <Route path="/explorer" ... />
  <Route path="/query" ... />
  <Route path="/admin/users" ... />
  <Route path="/admin/roles" ... />
  <Route path="/admin/apikeys" ... />
  <Route path="/admin/settings" ... />
  <Route path="*" element={<Navigate to="/" replace />} />
</Routes>`} lang="tsx" />
      </section>

      <section id="embedding" className="dash-section">
        <h2>Embedding &amp; Deployment</h2>
        <div className="flow">
          <span className="flow-step">web/src/</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step">vite build</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step">web/dist/</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step">kopieren</span>
          <span className="flow-arrow">&rarr;</span>
          <span className="flow-step" style={{ borderColor: 'var(--accent)' }}>internal/dashboard/dist/</span>
        </div>
        <CodeBlock code={`// internal/dashboard/embed.go
package dashboard
import "embed"

//go:embed dist/*
var distFS embed.FS

func FS() (http.FileSystem, error) {
    sub, _ := fs.Sub(distFS, "dist")
    return http.FS(sub), nil
}`} lang="go" />
      </section>

      <section id="i18n" className="dash-section">
        <h2>Internationalisierung</h2>
        <p>96 Key/Value-Paare in Deutsch und Englisch. Automatische Spracherkennung via Browser, umschaltbar im UI.</p>
        <table>
          <thead><tr><th>Key</th><th>Englisch</th><th>Deutsch</th></tr></thead>
          <tbody>
            <tr><td><code>dashboard.title</code></td><td>Dashboard</td><td>&Uuml;bersicht</td></tr>
            <tr><td><code>dashboard.connections</code></td><td>Connections</td><td>Verbindungen</td></tr>
            <tr><td><code>dashboard.active</code></td><td>Active</td><td>Aktiv</td></tr>
            <tr><td><code>dashboard.users</code></td><td>Users</td><td>Benutzer</td></tr>
            <tr><td><code>dashboard.databases</code></td><td>Databases</td><td>Datenbanken</td></tr>
          </tbody>
        </table>
      </section>
    </main>
  )
}
