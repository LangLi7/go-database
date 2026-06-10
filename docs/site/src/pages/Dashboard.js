import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import CodeBlock from '../components/CodeBlock';
const sec = (id) => {
    setTimeout(() => {
        const el = document.getElementById(id);
        if (el)
            el.scrollIntoView({ behavior: 'smooth' });
    }, 50);
};
export default function DashboardPage() {
    return (_jsxs("main", { className: "page dash-page", children: [_jsx("h1", { children: "Dashboard Guide" }), _jsx("p", { className: "page-sub", children: "Die Dashboard-Seite ist der Startbildschirm der Web-Oberfl\u00E4che. Sie liefert auf einen Blick alle wichtigen Kennzahlen und Verbindungsstatus." }), _jsx("div", { className: "dash-nav", children: ['Übersicht', 'Layout & Struktur', 'Statistik-Karten', 'Verbindungstabelle', 'Datenfluss', 'Routing', 'Embedding', 'i18n'].map(s => (_jsx("a", { href: `#${s.toLowerCase().replace(/[^a-z]/g, '')}`, onClick: e => { e.preventDefault(); sec(s.toLowerCase().replace(/[^a-z]/g, '')); }, className: "dash-nav-item", children: s }, s))) }), _jsxs("section", { id: "\u00DCbersicht", className: "dash-section", children: [_jsx("h2", { children: "\u00DCbersicht" }), _jsxs("p", { children: ["Das Dashboard unter ", _jsx("code", { children: "/" }), " wird nach dem Login angezeigt und kombiniert Statistik-Karten mit einer detaillierten Verbindungstabelle. Es ist in React 19 + TypeScript mit Ant Design 6 umgesetzt und responsiv gestaltet."] }), _jsxs("div", { className: "dash-mock", children: [_jsx("div", { className: "mock-grid", children: ['Verbindungen', 'Aktiv', 'Benutzer', 'Datenbanken'].map(s => _jsxs("div", { className: "mock-card", children: [_jsx("div", { className: "mock-num", children: "0" }), _jsx("div", { className: "mock-label", children: s })] }, s)) }), _jsxs("div", { className: "mock-table", children: [_jsx("div", { className: "mt-hdr", children: "Verbindungen" }), _jsx("div", { className: "mt-row", children: _jsx("span", { children: "Keine Daten" }) })] })] })] }), _jsxs("section", { id: "layoutstruktur", className: "dash-section", children: [_jsx("h2", { children: "Layout & Struktur" }), _jsxs("p", { children: ["Das Dashboard verwendet ein responsives Grid-Layout mit Ant Design ", _jsx("code", { children: "Row" }), " / ", _jsx("code", { children: "Col" }), ":"] }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Breakpoint" }), _jsx("th", { children: "Karten pro Reihe" })] }) }), _jsxs("tbody", { children: [_jsxs("tr", { children: [_jsxs("td", { children: [_jsx("code", { children: "lg" }), " (\u2265 1200px)"] }), _jsx("td", { children: "4" })] }), _jsxs("tr", { children: [_jsxs("td", { children: [_jsx("code", { children: "sm" }), " (\u2265 576px)"] }), _jsx("td", { children: "2" })] }), _jsxs("tr", { children: [_jsxs("td", { children: [_jsx("code", { children: "xs" }), " (< 576px)"] }), _jsx("td", { children: "1 (gestapelt)" })] })] })] })] }), _jsxs("section", { id: "statistikkarten", className: "dash-section", children: [_jsx("h2", { children: "Statistik-Karten" }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Karte" }), _jsx("th", { children: "Datenquelle" }), _jsx("th", { children: "Farbe" })] }) }), _jsxs("tbody", { children: [_jsxs("tr", { children: [_jsx("td", { children: "Verbindungen" }), _jsx("td", { children: _jsx("code", { children: "api.listConnections().length" }) }), _jsx("td", { children: "Indigo" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Aktiv" }), _jsx("td", { children: _jsx("code", { children: ".filter(c => c.state === 'connected')" }) }), _jsx("td", { children: "Gr\u00FCn" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Benutzer" }), _jsx("td", { children: _jsx("code", { children: "api.listUsers().length" }) }), _jsx("td", { children: "Gelb" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Datenbanken" }), _jsx("td", { children: _jsx("code", { children: "api.listConnections().length" }) }), _jsx("td", { children: "Blau" })] })] })] }), _jsx(CodeBlock, { code: `const [connections, setConnections] = useState<any[]>([])
const [users, setUsers] = useState<any[]>([])
const [loading, setLoading] = useState(true)

const connected = connections.filter(c => c.state === 'connected')`, lang: "tsx" })] }), _jsxs("section", { id: "verbindungstabelle", className: "dash-section", children: [_jsx("h2", { children: "Verbindungstabelle" }), _jsxs("p", { children: ["Unter den Karten folgt eine Ant Design ", _jsx("code", { children: "Table" }), " mit allen Verbindungen:"] }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Spalte" }), _jsx("th", { children: "Darstellung" })] }) }), _jsxs("tbody", { children: [_jsxs("tr", { children: [_jsx("td", { children: "Name" }), _jsx("td", { children: "Text" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Typ" }), _jsx("td", { children: "Tag (PostgreSQL, MySQL, ...)" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Status" }), _jsx("td", { children: "Tag farbcodiert (gr\u00FCn/rot/orange)" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Latenz" }), _jsx("td", { children: "ms" })] }), _jsxs("tr", { children: [_jsx("td", { children: "Quelle" }), _jsx("td", { children: "external / local / docker / file" })] })] })] }), _jsx(CodeBlock, { code: `<Table
  dataSource={connections}
  columns={columns}
  rowKey="id"
  pagination={false}
  size="small"
/>`, lang: "tsx" })] }), _jsxs("section", { id: "datenfluss", className: "dash-section", children: [_jsx("h2", { children: "Datenfluss" }), _jsxs("p", { children: ["Zwei parallele API-Aufrufe via ", _jsx("code", { children: "Promise.all" }), " beim Mount:"] }), _jsxs("div", { className: "flow", children: [_jsx("span", { className: "flow-step", children: "Mount" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", style: { borderColor: 'var(--blue)' }, children: "Promise.all" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", children: "listConnections" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", style: { borderColor: 'var(--green)' }, children: "setState" })] }), _jsxs("div", { className: "flow", children: [_jsx("span", { className: "flow-step", children: "Mount" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", style: { borderColor: 'var(--blue)' }, children: "Promise.all" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", children: "listUsers" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", style: { borderColor: 'var(--green)' }, children: "setState" })] }), _jsx(CodeBlock, { code: `useEffect(() => {
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
}, [])`, lang: "tsx" })] }), _jsxs("section", { id: "routing", className: "dash-section", children: [_jsx("h2", { children: "Routing" }), _jsx(CodeBlock, { code: `<Routes>
  <Route path="/" element={<Dashboard />} />
  <Route path="/connections" ... />
  <Route path="/explorer" ... />
  <Route path="/query" ... />
  <Route path="/admin/users" ... />
  <Route path="/admin/roles" ... />
  <Route path="/admin/apikeys" ... />
  <Route path="/admin/settings" ... />
  <Route path="*" element={<Navigate to="/" replace />} />
</Routes>`, lang: "tsx" })] }), _jsxs("section", { id: "embedding", className: "dash-section", children: [_jsx("h2", { children: "Embedding & Deployment" }), _jsxs("div", { className: "flow", children: [_jsx("span", { className: "flow-step", children: "web/src/" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", children: "vite build" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", children: "web/dist/" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", children: "kopieren" }), _jsx("span", { className: "flow-arrow", children: "\u2192" }), _jsx("span", { className: "flow-step", style: { borderColor: 'var(--accent)' }, children: "internal/dashboard/dist/" })] }), _jsx(CodeBlock, { code: `// internal/dashboard/embed.go
package dashboard
import "embed"

//go:embed dist/*
var distFS embed.FS

func FS() (http.FileSystem, error) {
    sub, _ := fs.Sub(distFS, "dist")
    return http.FS(sub), nil
}`, lang: "go" })] }), _jsxs("section", { id: "i18n", className: "dash-section", children: [_jsx("h2", { children: "Internationalisierung" }), _jsx("p", { children: "96 Key/Value-Paare in Deutsch und Englisch. Automatische Spracherkennung via Browser, umschaltbar im UI." }), _jsxs("table", { children: [_jsx("thead", { children: _jsxs("tr", { children: [_jsx("th", { children: "Key" }), _jsx("th", { children: "Englisch" }), _jsx("th", { children: "Deutsch" })] }) }), _jsxs("tbody", { children: [_jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: "dashboard.title" }) }), _jsx("td", { children: "Dashboard" }), _jsx("td", { children: "\u00DCbersicht" })] }), _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: "dashboard.connections" }) }), _jsx("td", { children: "Connections" }), _jsx("td", { children: "Verbindungen" })] }), _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: "dashboard.active" }) }), _jsx("td", { children: "Active" }), _jsx("td", { children: "Aktiv" })] }), _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: "dashboard.users" }) }), _jsx("td", { children: "Users" }), _jsx("td", { children: "Benutzer" })] }), _jsxs("tr", { children: [_jsx("td", { children: _jsx("code", { children: "dashboard.databases" }) }), _jsx("td", { children: "Databases" }), _jsx("td", { children: "Datenbanken" })] })] })] })] })] }));
}
