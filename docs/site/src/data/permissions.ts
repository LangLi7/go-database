export interface Permission {
  key: string
  desc: string
  group: string
}

export const permissions: Permission[] = [
  { key: 'connections:list', desc: 'Verbindungen und Schema auflisten', group: 'Connection' },
  { key: 'connections:create', desc: 'Neue Verbindungen registrieren', group: 'Connection' },
  { key: 'connections:delete', desc: 'Verbindungen löschen', group: 'Connection' },
  { key: 'connections:query', desc: 'SELECT, Browse, Row-CRUD, Safe-Execute', group: 'Connection' },
  { key: 'connections:execute', desc: 'INSERT/UPDATE/DELETE, DDL, Transfer', group: 'Connection' },
  { key: 'users:list', desc: 'Benutzer auflisten', group: 'Users' },
  { key: 'users:create', desc: 'Benutzer anlegen', group: 'Users' },
  { key: 'users:edit', desc: 'Benutzer bearbeiten + Permissions setzen', group: 'Users' },
  { key: 'users:delete', desc: 'Benutzer löschen', group: 'Users' },
  { key: 'settings:read', desc: 'Design/Einstellungen lesen', group: 'Admin' },
  { key: 'settings:write', desc: 'Design/Einstellungen speichern', group: 'Admin' },
  { key: 'traffic:view', desc: 'Stats, Activity, Traffic einsehen', group: 'Admin' },
  { key: 'apikeys:manage', desc: 'API-Keys verwalten', group: 'Admin' },
  { key: 'roles:manage', desc: 'Rollen + Permission-Gruppen verwalten', group: 'Admin' },
  { key: 'jobs:list', desc: 'Jobs auflisten', group: 'Jobs' },
  { key: 'jobs:create', desc: 'Jobs erstellen', group: 'Jobs' },
  { key: 'jobs:cancel', desc: 'Jobs abbrechen', group: 'Jobs' },
  { key: 'backup:create', desc: 'Backup erstellen', group: 'Backup' },
  { key: 'backup:restore', desc: 'Backup wiederherstellen', group: 'Backup' },
]

export const defaultRoles = [
  { name: 'admin', perms: '*', desc: 'Vollzugriff auf alle Funktionen' },
  { name: 'developer', perms: 'connections:list/query/execute, users:list, jobs:*', desc: 'Entwickler mit eingeschränktem Admin-Zugriff' },
  { name: 'readonly', perms: 'connections:list, connections:query', desc: 'Nur-Lese-Zugriff' },
]

export const plugins = [
  { name: 'PostgreSQL', driver: 'pgx/v5', color: '#336791' },
  { name: 'MySQL', driver: 'go-sql-driver/mysql', color: '#4479A1' },
  { name: 'MariaDB', driver: 'go-sql-driver/mysql', color: '#C0765A' },
  { name: 'SQLite', driver: 'modernc.org/sqlite', color: '#003B57' },
  { name: 'MongoDB', driver: 'mongo-driver', color: '#47A248' },
  { name: 'Redis', driver: 'go-redis/v9', color: '#DC382D' },
]

export const frontendPages = [
  { name: 'Dashboard', route: '/', desc: 'Startseite mit Statistik-Karten und Verbindungstabelle' },
  { name: 'Connections', route: '/connections', desc: 'DB-Verbindungen verwalten, anlegen, pingen' },
  { name: 'Explorer', route: '/explorer', desc: 'Tabellen browsen und Daten paginiert anzeigen' },
  { name: 'Query Editor', route: '/query', desc: 'SQL-Editor mit Autocomplete und Risikoanalyse' },
  { name: 'Admin Users', route: '/admin/users', desc: 'Benutzer-CRUD mit Permission-Overrides' },
  { name: 'Admin Roles', route: '/admin/roles', desc: 'Rollen-CRUD mit Permission-Matrix' },
  { name: 'API Keys', route: '/admin/apikeys', desc: 'API-Keys erstellen, auflisten, widerrufen' },
  { name: 'Settings', route: '/admin/settings', desc: 'Passwort ändern' },
]
