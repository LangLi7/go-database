import { useState, useEffect } from 'react'
import Sidebar from '../components/Sidebar'
import Search from '../components/Search'
import CodeBlock from '../components/CodeBlock'
import { endpoints } from '../data/endpoints'
import { permissions, defaultRoles } from '../data/permissions'

export default function ApiDocs() {
  const [hash, setHash] = useState('')

  useEffect(() => {
    const onHash = () => setHash(window.location.hash.slice(1))
    onHash()
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  useEffect(() => {
    if (hash) {
      const el = document.getElementById(hash)
      if (el) el.scrollIntoView({ behavior: 'smooth' })
    }
  }, [hash])

  return (
    <div className="api-layout">
      <Sidebar current={hash} />
      <main className="page api-page">
        <div className="api-top">
          <h1>API Referenz</h1>
          <Search />
        </div>
        <p className="api-base">Basis-URL: <code>http://localhost:8080/api/v1</code> &mdash; Einheitliches Format: <code>{'{ success, data, error, meta }'}</code></p>

        <table className="auth-table">
          <thead><tr><th>Methode</th><th>Header</th><th>Verwendung</th></tr></thead>
          <tbody>
            <tr><td>JWT (Session)</td><td><code>Authorization: Bearer &lt;token&gt;</code></td><td>Erhalten via <code>/auth/login</code></td></tr>
            <tr><td>API Key</td><td><code>X-API-Key: &lt;key&gt;</code></td><td>Erstellt via <code>/apikeys</code></td></tr>
          </tbody>
        </table>

        {endpoints.map(ep => (
          <div key={ep.id} id={ep.id} className="endpoint">
            <div className="ep-head">
              <span className={`method method-${ep.method}`}>{ep.method}</span>
              <span className="ep-path">{ep.path}</span>
              {ep.perm && <span className="ep-perm">{ep.perm}</span>}
            </div>
            <p className="ep-desc">{ep.desc}</p>
            {ep.req && (
              <div className="ep-section">
                <div className="ep-section-label">Request</div>
                <CodeBlock code={ep.req} />
              </div>
            )}
            {ep.res && (
              <div className="ep-section">
                <div className="ep-section-label">Response</div>
                <CodeBlock code={ep.res} />
              </div>
            )}
          </div>
        ))}

        <section className="api-section">
          <h2>Status-Codes</h2>
          <table><thead><tr><th>Code</th><th>Bedeutung</th></tr></thead><tbody>
            {[[200, 'Success'],[201, 'Created'],[204, 'No Content'],[400, 'Bad Request — Validierungsfehler'],[401, 'Unauthorized'],[403, 'Forbidden'],[404, 'Not Found'],[409, 'Conflict'],[500, 'Internal Server Error'],[502, 'Bad Gateway — DB-Fehler']].map(([c, d]) => <tr key={c}><td>{c}</td><td>{d}</td></tr>)}
          </tbody></table>
        </section>

        <section className="api-section">
          <h2>Error-Codes</h2>
          <table><thead><tr><th>Code</th><th>Bedeutung</th></tr></thead><tbody>
            {[['BAD_REQUEST','Validierungsfehler'],['UNAUTHORIZED','Fehlende/ungültige Auth'],['FORBIDDEN','Permission verweigert'],['NOT_FOUND','Resource nicht gefunden'],['CONFLICT','Resource existiert'],['CONNECTION_FAILED','DB-Fehler'],['QUERY_FAILED','SQL-Fehler'],['NETWORK_ERROR','Netzwerkfehler'],['PARSE_ERROR','Parsing-Fehler'],['INTERNAL_ERROR','Serverfehler']].map(([c, d]) => <tr key={c}><td><code>{c}</code></td><td>{d}</td></tr>)}
          </tbody></table>
        </section>

        <section className="api-section" id="perms">
          <h2>Permission-Modell</h2>
          <table><thead><tr><th>Rolle</th><th>Permissions</th></tr></thead><tbody>
            {defaultRoles.map(r => <tr key={r.name}><td><code>{r.name}</code></td><td><code>{r.perms}</code></td></tr>)}
          </tbody></table>
          <table><thead><tr><th>Permission</th><th>Gruppe</th><th>Beschreibung</th></tr></thead><tbody>
            {permissions.map(p => <tr key={p.key}><td><code>{p.key}</code></td><td>{p.group}</td><td>{p.desc}</td></tr>)}
          </tbody></table>
        </section>
      </main>
    </div>
  )
}
