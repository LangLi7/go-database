export interface Endpoint {
  id: string
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'WS' | 'SSE'
  path: string
  short: string
  desc: string
  group: string
  perm: string | null
  req?: string
  res?: string
}

export const endpoints: Endpoint[] = [
  // Auth
  { id: 'login', method: 'POST', path: '/auth/login', short: '/auth/login', desc: 'Authentifiziert einen Benutzer und gibt ein JWT zurück.', group: 'Auth', perm: null, req: '{"username":"admin","password":"admin"}', res: '{"success":true,"data":{"token":"eyJ...","user_id":"admin-001","username":"admin","role":"admin"}}' },
  { id: 'refresh', method: 'POST', path: '/auth/refresh', short: '/auth/refresh', desc: 'Erneuert ein bestehendes Token.', group: 'Auth', perm: null, req: '{"token":"eyJ..."}' },
  { id: 'verify', method: 'GET', path: '/auth/verify', short: '/auth/verify', desc: 'Verifiziert ein Token und gibt Benutzerinformationen zurück.', group: 'Auth', perm: null },
  { id: 'change-pw', method: 'POST', path: '/auth/change-password', short: '/auth/change-password', desc: 'Ändert das Passwort des authentifizierten Benutzers.', group: 'Auth', perm: null, req: '{"old_password":"admin","new_password":"neuesPasswort"}', res: '204 No Content' },

  // Connections
  { id: 'list-conn', method: 'GET', path: '/connections', short: '/connections', desc: 'Listet alle registrierten Datenbank-Verbindungen.', group: 'Connections', perm: 'connections:list', res: '[{"id":"a1b2c3d4","name":"Meine PG-DB","type":"postgres","state":"connected","latency_ms":2}]' },
  { id: 'create-conn', method: 'POST', path: '/connections', short: '/connections', desc: 'Registriert eine neue Datenbank-Verbindung.', group: 'Connections', perm: 'connections:create', req: '{"name":"Meine DB","type":"postgres","host":"localhost","port":5432,"database":"sampledb","user":"dev","password":"dev123"}', res: '201 Created' },
  { id: 'get-conn', method: 'GET', path: '/connections/:id', short: '/connections/:id', desc: 'Details einer einzelnen Verbindung.', group: 'Connections', perm: 'connections:list' },
  { id: 'del-conn', method: 'DELETE', path: '/connections/:id', short: '/connections/:id', desc: 'Entfernt eine Verbindung.', group: 'Connections', perm: 'connections:delete', res: '204 No Content' },
  { id: 'ping-conn', method: 'GET', path: '/connections/:id/ping', short: '/connections/:id/ping', desc: 'Prüft die Verbindung und misst die Latenz.', group: 'Connections', perm: 'connections:list', res: '{"latency_ms":2,"status":"ok"}' },

  // Explorer
  { id: 'tables', method: 'GET', path: '/connections/:id/tables', short: '/connections/:id/tables', desc: 'Listet alle Tabellen/Collections.', group: 'Explorer', perm: 'connections:list', res: '["users","products","orders"]' },
  { id: 'schema', method: 'GET', path: '/connections/:id/schema', short: '/connections/:id/schema', desc: 'Vollständiges Schema mit Tabellen, Spalten, Typen und Constraints.', group: 'Explorer', perm: 'connections:list' },
  { id: 'browse', method: 'GET', path: '/connections/:id/browse/:table', short: '/connections/:id/browse/:table', desc: 'Daten paginiert abrufen mit Sortierung und Filter.', group: 'Explorer', perm: 'connections:query', req: '?page=1&per_page=50&sort=id&dir=asc', res: '{"data":[[1,"Alice"]],"columns":["id","name"],"page":1,"per_page":50,"total":100,"total_pages":2,"duration_ms":5}' },
  { id: 'insert', method: 'POST', path: '/connections/:id/row/:table', short: '/connections/:id/row/:table', desc: 'Neue Zeile einfügen.', group: 'Explorer', perm: 'connections:query', req: '{"name":"Alice","email":"alice@example.com"}', res: '201 Created' },
  { id: 'update', method: 'PUT', path: '/connections/:id/row/:table/:pk/:val', short: '/connections/:id/row/:table/:pk/:val', desc: 'Zeile aktualisieren.', group: 'Explorer', perm: 'connections:query' },
  { id: 'delete', method: 'DELETE', path: '/connections/:id/row/:table/:pk/:val', short: '/connections/:id/row/:table/:pk/:val', desc: 'Zeile löschen.', group: 'Explorer', perm: 'connections:query', res: '204 No Content' },

  // Query
  { id: 'query', method: 'POST', path: '/connections/:id/query', short: '/connections/:id/query', desc: 'SELECT-Query mit automatischer SQL-Typerkennung.', group: 'Query', perm: 'connections:query', req: '{"query":"SELECT * FROM users WHERE id = 1"}', res: '{"columns":["id","name"],"rows":[[1,"Alice"]],"rows_affected":1,"duration_ms":3}' },
  { id: 'execute', method: 'POST', path: '/connections/:id/execute', short: '/connections/:id/execute', desc: 'INSERT/UPDATE/DELETE/DDL-Query.', group: 'Query', perm: 'connections:execute', req: '{"query":"UPDATE users SET name=\'Bob\' WHERE id=1"}' },
  { id: 'safe', method: 'POST', path: '/execute/safe', short: '/execute/safe', desc: 'Query mit Risikobewertung und Sicherheitsprüfung.', group: 'Query', perm: 'connections:query', req: '{"connection_id":"...","sql":"DROP TABLE users","confirm_high":true}' },

  // DB
  { id: 'standalone', method: 'POST', path: '/databases/standalone', short: '/databases/standalone', desc: 'Erzeugt eine Standalone-Datenbank (SQLite-Datei).', group: 'Datenbanken', perm: 'connections:create' },
  { id: 'list-db', method: 'GET', path: '/connections/:id/databases', short: '/connections/:id/databases', desc: 'Listet alle Datenbanken einer Verbindung.', group: 'Datenbanken', perm: 'connections:list' },
  { id: 'create-db', method: 'POST', path: '/connections/:id/databases', short: '/connections/:id/databases', desc: 'Erzeugt eine neue Datenbank.', group: 'Datenbanken', perm: 'connections:execute', req: '{"name":"new_database"}' },
  { id: 'drop-db', method: 'DELETE', path: '/connections/:id/databases/:name', short: '/connections/:id/databases/:name', desc: 'Löscht eine Datenbank.', group: 'Datenbanken', perm: 'connections:execute', res: '204 No Content' },
  { id: 'create-tbl', method: 'POST', path: '/connections/:id/tables', short: '/connections/:id/tables', desc: 'Erzeugt eine neue Tabelle.', group: 'Datenbanken', perm: 'connections:execute', req: '{"name":"users","columns":"id INT PK, name TEXT"}' },
  { id: 'drop-tbl', method: 'DELETE', path: '/connections/:id/tables/:name', short: '/connections/:id/tables/:name', desc: 'Löscht eine Tabelle.', group: 'Datenbanken', perm: 'connections:execute', res: '204 No Content' },

  // Admin
  { id: 'stats', method: 'GET', path: '/admin/stats', short: '/admin/stats', desc: 'Dashboard-Statistiken über Verbindungen, Queries, Benutzer und Datenbanken.', group: 'Admin', perm: 'traffic:view', res: '{"connections":5,"queries":1234,"users":3,"databases":8}' },
  { id: 'activity', method: 'GET', path: '/admin/activity', short: '/admin/activity', desc: 'Audit-Log der letzten Aktionen.', group: 'Admin', perm: 'traffic:view' },
  { id: 'get-design', method: 'GET', path: '/admin/design', short: '/admin/design', desc: 'Aktive Design-Konfiguration.', group: 'Admin', perm: 'settings:read' },
  { id: 'save-design', method: 'POST', path: '/admin/design', short: '/admin/design', desc: 'Design speichern (Theme, Farben, Layout).', group: 'Admin', perm: 'settings:write', req: '{"name":"Winter-Theme","config":"{\\"dark_mode\\":true}","active":true}' },
  { id: 'list-users', method: 'GET', path: '/admin/users', short: '/admin/users', desc: 'Listet alle Benutzer (ohne Passwort-Hashes).', group: 'Admin', perm: 'users:list' },
  { id: 'create-user', method: 'POST', path: '/admin/users', short: '/admin/users', desc: 'Neuen Benutzer anlegen.', group: 'Admin', perm: 'users:create', req: '{"username":"dev1","password":"secret","role":"developer"}' },
  { id: 'update-user', method: 'PUT', path: '/admin/users/:id', short: '/admin/users/:id', desc: 'Benutzer aktualisieren.', group: 'Admin', perm: 'users:edit' },
  { id: 'delete-user', method: 'DELETE', path: '/admin/users/:id', short: '/admin/users/:id', desc: 'Benutzer löschen.', group: 'Admin', perm: 'users:delete', res: '204 No Content' },
  { id: 'get-perms', method: 'GET', path: '/admin/users/:id/permissions', short: '/admin/users/:id/permissions', desc: 'Effektive Berechtigungen (Rolle + Overrides).', group: 'Admin', perm: 'users:list' },
  { id: 'set-perms', method: 'PUT', path: '/admin/users/:id/permissions', short: '/admin/users/:id/permissions', desc: 'User-spezifische Permission-Overrides.', group: 'Admin', perm: 'users:edit', req: '{"extra_perm":["users:create"],"extra_db_access":["conn-123"]}' },
  { id: 'get-db', method: 'GET', path: '/admin/users/:id/db-access', short: '/admin/users/:id/db-access', desc: 'DB-Zugriffsrechte abrufen.', group: 'Admin', perm: 'roles:manage' },
  { id: 'set-db', method: 'PUT', path: '/admin/users/:id/db-access', short: '/admin/users/:id/db-access', desc: 'DB-Zugriffsrechte setzen.', group: 'Admin', perm: 'roles:manage' },
  { id: 'list-roles', method: 'GET', path: '/admin/roles', short: '/admin/roles', desc: 'Listet alle Rollen.', group: 'Admin', perm: 'roles:manage' },
  { id: 'create-role', method: 'POST', path: '/admin/roles', short: '/admin/roles', desc: 'Neue Rolle anlegen.', group: 'Admin', perm: 'roles:manage', req: '{"name":"DBA","permissions":["connections:list","connections:query","backup:create"]}' },
  { id: 'update-role', method: 'PUT', path: '/admin/roles/:id', short: '/admin/roles/:id', desc: 'Rolle aktualisieren.', group: 'Admin', perm: 'roles:manage' },
  { id: 'delete-role', method: 'DELETE', path: '/admin/roles/:id', short: '/admin/roles/:id', desc: 'Rolle löschen.', group: 'Admin', perm: 'roles:manage' },
  { id: 'role-perms', method: 'PUT', path: '/admin/roles/:id/permissions', short: '/admin/roles/:id/permissions', desc: 'Rollen-Berechtigungen setzen.', group: 'Admin', perm: 'roles:manage', req: '{"permissions":["connections:*","users:list"],"db_access":["*"]}' },
  { id: 'perm-groups', method: 'GET', path: '/admin/permission-groups', short: '/admin/permission-groups', desc: 'Hierarchische Permission-Gruppen für die UI.', group: 'Admin', perm: 'roles:manage' },

  // API Keys
  { id: 'list-keys', method: 'GET', path: '/apikeys', short: '/apikeys', desc: 'Listet alle API-Keys (nur Prefixe).', group: 'API Keys', perm: 'apikeys:manage' },
  { id: 'create-key', method: 'POST', path: '/apikeys', short: '/apikeys', desc: 'Generiert einen neuen API-Key.', group: 'API Keys', perm: 'apikeys:manage', req: '{"name":"CI/CD Pipeline","permissions":["connections:list","connections:query"]}', res: '{"raw_key":"a1b2...","prefix":"a1b2c3d4","name":"CI/CD Pipeline","formatted":"a1b2********cdef"}' },
  { id: 'delete-key', method: 'DELETE', path: '/apikeys/:prefix', short: '/apikeys/:prefix', desc: 'API-Key widerrufen.', group: 'API Keys', perm: 'apikeys:manage', res: '204 No Content' },

  // Transfer
  { id: 'start-transfer', method: 'POST', path: '/transfer', short: '/transfer', desc: 'Startet Datentransfer zwischen zwei Verbindungen.', group: 'Transfer', perm: 'connections:execute', req: '{"source_conn":"conn-sqlite","target_conn":"conn-pg","tables":["users"],"dry_run":true,"batch_size":1000}' },
  { id: 'get-transfer', method: 'GET', path: '/transfer/:id', short: '/transfer/:id', desc: 'Status eines Transfers.', group: 'Transfer', perm: 'connections:execute' },
  { id: 'cancel-transfer', method: 'DELETE', path: '/transfer/:id', short: '/transfer/:id', desc: 'Laufenden Transfer abbrechen.', group: 'Transfer', perm: 'connections:execute' },
  { id: 'transfer-log', method: 'GET', path: '/transfer/:id/log', short: '/transfer/:id/log', desc: 'Fehler-Log eines Transfers.', group: 'Transfer', perm: 'connections:execute' },

  // Suggest
  { id: 'suggest', method: 'POST', path: '/suggest', short: '/suggest', desc: 'SQL-Autocomplete: Vorschläge basierend auf Eingabe und Schema.', group: 'Weitere', perm: 'connections:list', req: '{"connection_id":"...","input":"SELECT * FROM u"}', res: '[{"text":"users","type":"table","confidence":0.95}]' },
  { id: 'traffic-stats', method: 'GET', path: '/traffic/stats', short: '/traffic/stats', desc: 'Traffic-Statistiken (Requests/s, Latenz, Fehlerrate).', group: 'Weitere', perm: 'traffic:view' },
  { id: 'traffic-reqs', method: 'GET', path: '/traffic/requests', short: '/traffic/requests', desc: 'Letzte API-Requests.', group: 'Weitere', perm: 'traffic:view' },
  { id: 'health', method: 'GET', path: '/health', short: '/health', desc: 'Server-Health-Check (keine Auth nötig).', group: 'Weitere', perm: null, res: '{"status":"ok","version":"0.1.0"}' },
  { id: 'ws-query', method: 'WS', path: '/ws/query/:id', short: '/ws/query/:id', desc: 'WebSocket: Streaming-Query-Endpunkt. Sende {"type":"query","query":"SELECT..."}', group: 'WebSocket', perm: 'connections:query' },
  { id: 'sse-activity', method: 'SSE', path: '/sse/activity', short: '/sse/activity', desc: 'SSE: Live Audit-Log-Stream mit Heartbeat alle 3s.', group: 'SSE', perm: 'traffic:view' },
  { id: 'sse-stats', method: 'SSE', path: '/sse/stats', short: '/sse/stats', desc: 'SSE: Live-Statistiken alle 2s (aktive/gesamte Verbindungen).', group: 'SSE', perm: 'traffic:view' },
]
