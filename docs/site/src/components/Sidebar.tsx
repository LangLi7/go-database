import { endpoints, type Endpoint } from '../data/endpoints'

const groups: { label: string; ids: string[] }[] = [
  { label: 'Auth', ids: ['login', 'refresh', 'verify', 'change-pw'] },
  { label: 'Connections', ids: ['list-conn', 'create-conn', 'get-conn', 'del-conn', 'ping-conn'] },
  { label: 'Explorer', ids: ['tables', 'schema', 'browse', 'insert', 'update', 'delete'] },
  { label: 'Query', ids: ['query', 'execute', 'safe'] },
  { label: 'Datenbanken', ids: ['standalone', 'list-db', 'create-db', 'drop-db', 'create-tbl', 'drop-tbl'] },
  { label: 'Admin', ids: ['stats', 'activity', 'get-design', 'save-design', 'list-users', 'create-user', 'update-user', 'delete-user', 'get-perms', 'set-perms', 'get-db', 'set-db', 'list-roles', 'create-role', 'update-role', 'delete-role', 'role-perms', 'perm-groups'] },
  { label: 'API Keys', ids: ['list-keys', 'create-key', 'delete-key'] },
  { label: 'Transfer', ids: ['start-transfer', 'get-transfer', 'cancel-transfer', 'transfer-log'] },
  { label: 'WebSocket', ids: ['ws-query'] },
  { label: 'SSE', ids: ['sse-activity', 'sse-stats'] },
  { label: 'Weitere', ids: ['suggest', 'traffic-stats', 'traffic-reqs', 'health'] },
]

const map = new Map<string, Endpoint>()
endpoints.forEach(e => map.set(e.id, e))

export default function Sidebar({ current }: { current: string }) {
  return (
    <aside className="sidebar">
      {groups.map(g => (
        <div className="sg" key={g.label}>
          <div className="sgt">{g.label}</div>
          {g.ids.map(id => {
            const ep = map.get(id)
            if (!ep) return null
            return (
              <a
                key={id}
                href={`/api#${id}`}
                className={`sl ${current === id ? 'active' : ''}`}
              >
                <span className={`m method-${ep.method}`}>{ep.method}</span>
                {ep.short}
              </a>
            )
          })}
        </div>
      ))}
    </aside>
  )
}
