# go-database API Dokumentation

**Basis-URL:** `http://localhost:8080/api/v1`  
**Response-Format:** `{"success": bool, "data": ..., "error": {...}, "meta": {"timestamp": "..."}}`

---

## Authentifizierung

### POST /auth/login

Authentifiziert einen Benutzer und gibt ein JWT zurück.

**Request:**
```json
{"username": "admin", "password": "admin"}
```

**Response (200):**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user_id": "admin-001",
    "username": "admin",
    "role": "admin"
  }
}
```

### POST /auth/refresh

Erneuert ein bestehendes Token.

**Request (Header):** `Authorization: Bearer <token>`  
**Request (Body):**
```json
{"token": "eyJhbGciOiJIUzI1NiIs..."}
```

**Response (200):** Wie Login

### POST /auth/change-password

Ändert das Passwort des authentifizierten Benutzers.

**Request:**
```json
{"old_password": "admin", "new_password": "neuesPasswort"}
```

**Response:** `204 No Content`

---

## Connections

### GET /connections

Listet alle registrierten Datenbank-Verbindungen.

**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "a1b2c3d4",
      "name": "Meine PG-DB",
      "type": "postgres",
      "source": "external",
      "state": "connected",
      "latency_ms": 2,
      "tags": ["production"]
    }
  ]
}
```

### POST /connections

Registriert eine neue Datenbank-Verbindung.

**Request:**
```json
{
  "name": "Meine DB",
  "type": "postgres",
  "source": "external",
  "host": "localhost",
  "port": 5432,
  "database": "sampledb",
  "user": "dev",
  "password": "dev123",
  "ssl": false,
  "tags": ["dev"]
}
```

**SQLite Variante:**
```json
{
  "name": "Lokale DB",
  "type": "sqlite",
  "filepath": "./database/samples/sqlite/sampledb.db",
  "tags": ["local"]
}
```

**Response:** `201 Created`

### GET /connections/:id

Details einer einzelnen Verbindung.

### DELETE /connections/:id

Entfernt eine Verbindung. **Response:** `204 No Content`

### GET /connections/:id/ping

Prüft die Verbindung und misst die Latenz.

**Response (200):**
```json
{"latency_ms": 2, "status": "ok"}
```

---

## Explorer

### GET /connections/:id/tables

Listet alle Tabellen/Collections.

### GET /connections/:id/schema

Gibt das vollständige Schema zurück (Tabellen, Spalten, Typen).

### GET /connections/:id/browse/:table

Daten aus einer Tabelle abrufen (paginierbar).

**Query-Parameter:**
| Parameter | Typ | Standard | Beschreibung |
|-----------|-----|----------|-------------|
| `page` | int | 1 | Seitenzahl |
| `per_page` | int | 50 | Einträge pro Seite (max 200) |
| `sort` | string | - | Spaltenname für Sortierung |
| `dir` | string | asc | Sortierrichtung (asc/desc) |
| `filter` | string | - | SQL-WHERE-Bedingung |

**Response (200):**
```json
{
  "data": [[1, "Alice", "alice@..."]],
  "columns": ["id", "name", "email"],
  "page": 1,
  "per_page": 50,
  "total": 100,
  "total_pages": 2,
  "duration_ms": 5
}
```

### POST /connections/:id/row/:table

Neue Zeile einfügen.

**Request:**
```json
{"name": "Neuer Eintrag", "email": "neu@example.com"}
```

**Response:** `201 Created`

### PUT /connections/:id/row/:table/:pk/:val

Zeile aktualisieren (`:pk` = Primärschlüssel-Spalte, `:val` = Wert).

**Request:**
```json
{"name": "Geändert", "email": "neu@example.com"}
```

### DELETE /connections/:id/row/:table/:pk/:val

Zeile löschen. **Response:** `204 No Content`

---

## Query

### POST /connections/:id/query

Führt eine SELECT-Query aus.

**Request:**
```json
{"query": "SELECT * FROM users WHERE id = 1"}
```

**Response (200):**
```json
{
  "columns": ["id", "name", "email"],
  "rows": [[1, "Alice", "alice@..."]],
  "rows_affected": 1,
  "duration_ms": 3
}
```

### POST /connections/:id/execute

Führt eine INSERT/UPDATE/DELETE-Query aus.

**Request:**
```json
{"query": "UPDATE users SET name = 'Bob' WHERE id = 1"}
```

---

## Admin

### GET /admin/stats

Dashboard-Statistiken.

**Response:**
```json
{
  "connections_total": 5,
  "connections_online": 3
}
```

### GET /admin/design

Aktives Design-Konfiguration.

**Response:**
```json
{
  "id": "custom",
  "name": "Mein Theme",
  "config": "{\"primary_color\":\"#6366f1\",\"dark_mode\":false}",
  "active": true
}
```

### POST /admin/design

Design speichern (adaptives Theme, Netflix-Stil).

**Request:**
```json
{
  "name": "Winter-Theme",
  "config": "{\"primary_color\":\"#06b6d4\",\"dark_mode\":true,\"compact\":false,\"sidebar_width\":\"260px\"}",
  "active": true
}
```

### GET /admin/users

Listet alle Benutzer (ohne Passwort-Hashes).

### POST /admin/users

Neuen Benutzer anlegen.

**Request:**
```json
{"username": "dev1", "password": "secret", "role": "developer"}
```

### PUT /admin/users/:id

Benutzer aktualisieren.

### DELETE /admin/users/:id

Benutzer löschen. **Response:** `204 No Content`

### GET /admin/users/:id/permissions

Effektive Berechtigungen eines Users (Rolle + Overrides).

### PUT /admin/users/:id/permissions

User-spezifische Overrides setzen.

**Request:**
```json
{"extra_perm": ["users:create"], "extra_db_access": ["conn-123"]}
```

### GET /admin/roles

Listet alle Rollen.

### POST /admin/roles

Neue Rolle anlegen.

**Request:**
```json
{
  "name": "DBA",
  "permissions": ["connections:list", "connections:query", "backup:create"]
}
```

### PUT /admin/roles/:id

Rolle aktualisieren.

### DELETE /admin/roles/:id

Rolle löschen.

### PUT /admin/roles/:id/permissions

Rollen-Berechtigungen setzen.

**Request:**
```json
{"permissions": ["connections:*", "users:list"]}
```

### GET /admin/activity

Audit-Log der letzten Aktionen.

---

## API Keys

### GET /apikeys

Listet alle API-Keys (ohne Hashes).

### POST /apikeys

Neuen API-Key generieren.

**Request:**
```json
{"name": "CI/CD Pipeline", "permissions": ["connections:list", "connections:query"]}
```

**Response (201):**
```json
{
  "raw_key": "a1b2c3d4...",
  "prefix": "a1b2c3d4",
  "name": "CI/CD Pipeline",
  "formatted": "a1b2c3d4********cdef"
}
```

**Wichtig:** Der `raw_key` wird nur einmal angezeigt!

### DELETE /apikeys/:prefix

API-Key widerrufen. **Response:** `204 No Content`

---

## Traffic

### GET /traffic/stats

Traffic-Statistiken.

### GET /traffic/requests

Letzte Requests.

---

## Health

### GET /health

Server-Health-Check.

**Response:**
```json
{"status": "ok", "version": "0.1.0"}
```

---

## Transfer

### POST /transfer

Datentransfer zwischen zwei Verbindungen starten.

**Request:**
```json
{
  "source_conn": "conn-sqlite",
  "target_conn": "conn-pg",
  "tables": ["users", "products"],
  "dry_run": true,
  "batch_size": 1000
}
```

### GET /transfer/:id

Status eines Transfers.

### DELETE /transfer/:id

Transfer abbrechen.

### GET /transfer/:id/log

Fehler-Log eines Transfers.

---

## Status-Codes

| Code | Bedeutung |
|------|-----------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 500 | Internal Server Error |
| 502 | Bad Gateway (DB-Verbindungsfehler) |

---

## Error-Codes

| Code | Bedeutung |
|------|-----------|
| `BAD_REQUEST` | Validierungsfehler |
| `UNAUTHORIZED` | Fehlende/gültige Authentifizierung |
| `FORBIDDEN` | Nicht genügend Berechtigungen |
| `NOT_FOUND` | Resource nicht gefunden |
| `CONFLICT` | Resource existiert bereits |
| `CONNECTION_FAILED` | DB-Verbindung fehlgeschlagen |
| `QUERY_FAILED` | SQL-Query-Fehler |
| `INTERNAL_ERROR` | Server-interner Fehler |

---

## Permission-Modell

### Default-Rollen

| Rolle | Berechtigungen |
|-------|---------------|
| `admin` | `*` (alles) |
| `developer` | connections:list/query/execute, users:list, jobs:list/create, traffic:view |
| `readonly` | connections:list, connections:query |

### Verfügbare Permissions

`connections:list`, `connections:create`, `connections:delete`, `connections:query`, `connections:execute`, `users:list`, `users:create`, `users:edit`, `users:delete`, `settings:read`, `settings:write`, `jobs:list`, `jobs:create`, `jobs:cancel`, `backup:create`, `backup:restore`, `traffic:view`, `apikeys:manage`, `roles:manage`
