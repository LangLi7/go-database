# go-database API Dokumentation

**Basis-URL:** `http://localhost:8080/api/v1`  
**Response-Format:** `{"success": bool, "data": ..., "error": {"code": "...", "message": "..."}, "meta": {"timestamp": "..."}}`

---

## Setup (First-Time)

### GET /setup/status

Prüft, ob die Erstkonfiguration abgeschlossen ist.

**Response (200):**
```json
{
  "success": true,
  "data": {"setup_complete": false}
}
```

### POST /setup/initialize

Setzt Admin-Passwort und E-Mail beim ersten Start.  
**Erforderlich** nach der Erstinstallation, bevor der Login funktioniert.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "meinSicheresPasswort123"
}
```

**Response:** `204 No Content`

**Fehler:** `409 ALREADY_SETUP` falls bereits konfiguriert.

---

## Authentifizierung

### POST /auth/login

Authentifiziert einen Benutzer und gibt ein JWT zurück.

**Erststart (First-Time Setup):**
- Beim **allerersten** Start ist der Admin-User `admin` mit Passwort `admin`
  voreingestellt. Ein Login mit `admin:admin` gibt dann `403 SETUP_REQUIRED`
  zurück. Du MUSST zuerst `/setup/initialize` mit einem neuen Passwort aufrufen.
- Nach dem Setzen gilt nur noch das neue Passwort. `admin:admin` funktioniert
  danach nicht mehr (die `auth.db` merkt sich den Zustand — siehe `internal/internaldb`).

**Request:**
```json
{"username": "admin", "password": "meinSicheresPasswort123"}
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

**Response (403 — Setup erforderlich):**
```json
{
  "success": false,
  "error": {"code": "SETUP_REQUIRED", "message": "default admin password must be changed"}
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

### Passkeys (WebAuthn) — plattformübergreifend (Windows Hello / TouchID / Android)

Passkeys ersetzen das Passwort durch kryptografische Schlüssel auf dem
Endgerät (TPM/Secure Enclave). Registrierung und Login laufen als
WebAuthn-Ceremony; der Server speichert nur die public credential.

**Voraussetzung:** Frontend muss die Browser WebAuthn-API nutzen
(`navigator.credentials.create` / `.get`). RPID = Host des Servers,
Origin = exakt `scheme://host:port`.

**Registrieren (eingeloggt):**
```http
POST /api/v1/auth/passkeys/register/begin   # → Creation-Options (JSON)
POST /api/v1/auth/passkeys/register/finish?name=Laptop   # → 201 {id,name,aaguid}
GET  /api/v1/auth/passkeys                   # eigene Passkeys auflisten
DELETE /api/v1/auth/passkeys/:id             # Passkey entfernen
```

**Login (öffentlich, kein Token nötig):**
```http
POST /api/v1/auth/passkeys/login/begin       # → {session, options}
POST /api/v1/auth/passkeys/login/finish?session=<token>  # → {token, user_id, ...}
```
Der `token` aus `login/finish` ist ein normales JWT und gilt für alle
weiteren Endpoints. Ein Passkey-Login berechtigt auch zum
`POST /auth/change-password` (selbes Prinzip wie Passwort-Login).

**Speicher:** separate Tabelle `user_passkeys` (public_key, credential_id,
aaguid, sign_count). Passwörter und Passkeys sind unabhängig — ein Konto
kann beides haben.

**LuckPerms-artige Permission-Ansicht:**
```http
GET /api/v1/admin/permission-groups   # hierarchische Permission-Tree (Gruppen + Keys)
```
Entspricht LuckPerms' `permission info` — zeigt alle Permission-Nodes
(`connections:*`, `users:*`, `roles:manage`, …) gruppiert, inkl.
resourcenspezifischer Nodes (`list:connection.<id>`).

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

**Auto-Erkennung (`type: "auto"`):** Wenn du `type` nicht kennst, setze es auf
`"auto"`. Das System erkennt den Typ heuristisch:
- aus dem **DSN** in `params.dsn` oder `host` (z.B. `postgres://...`, `mongodb://...`, `sqlserver://...`)
- aus dem **Well-Known-Port** (5432→postgres, 3306→mysql, 1433→mssql, 27017→mongodb, 6379→redis, 9200→elasticsearch, 8123→clickhouse)
- aus einer gesetzten **Datei** (`filepath`→sqlite)

```json
{
  "name": "Unbekannt",
  "type": "auto",
  "host": "db.example.com",
  "port": 1433,
  "user": "sa",
  "password": "***"
}
```

---

### POST /connections/test
Testet eine Verbindung **ohne sie zu speichern** (nutzt ebenfalls `type: "auto"`).
Request siehe oben (`/connections`). Response bei Erfolg:
```json
{ "success": true, "data": { "success": true, "latency_ms": 12, "message": "connection successful" } }
```

---

### Per-Type Endpoints (alias für Direktzugriff)

Du kannst eine Datenbank auch **ohne vorherige Registrierung** direkt über einen
typ-spezifischen Endpoint ansprechen. Die Verbindung wird pro Request erzeugt
und sofort geschlossen ("throwaway").

| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| POST | `/api/v1/db/{type}/query` | SELECT auf {type} (Throwaway) |
| POST | `/api/v1/db/{type}/execute` | WRITE auf {type} (Throwaway) |
| POST | `/api/v1/db/{type}/test` | Konnektivitäts-Check |

`{type}` ∈ `postgres | mysql | mariadb | sqlite | mongodb | redis | mssql | ...`
(alle registrierten Plugins, siehe `GET /api/v1/connections`).

**Beispiel — direkt PostgreSQL abfragen:**
```bash
curl -X POST http://localhost:8080/api/v1/db/postgres/query \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d '{
    "host": "localhost", "port": 5432,
    "database": "sampledb", "user": "dev", "password": "dev123",
    "query": "SELECT * FROM users LIMIT 10"
  }'
```

**Beispiel — MSSQL direkt testen (MC-Plugin erkennt automatisch SQL Server):**
```bash
curl -X POST http://localhost:8080/api/v1/db/mssql/test \
  -H "Authorization: Bearer $TOK" \
  -H "Content-Type: application/json" \
  -d '{"host":"sqlhost","port":1433,"user":"sa","password":"***","database":"master"}'
```

> Diese Endpoints sind ein **Alias** zum generischen `/connections/:id/*`-Modell.
> Beide teilen sich dieselbe Plugin-/Guard-/Permission-Logik.

---

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

## LLM / AI — Modell-Entdeckung

Die go-database API kann verfügbare AI-Modelle aus zwei Quellen abfragen –
lokal (LM Studio) und remote (OpenRouter FREE-Modelle).

### `GET /api/v1/models/local`

Lokale Modelle aus LM Studio (läuft auf `http://localhost:1234`).

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "key": "deepseek-r1-distill-qwen-14b",
      "display_name": "DeepSeek R1 Distill Qwen 14B",
      "publisher": "lmstudio-community",
      "architecture": "qwen2",
      "quantization": {"name": "Q4_K_M", "bits_per_weight": 4},
      "size_bytes": 8988109952,
      "params_string": "14B",
      "format": "gguf",
      "capabilities": {"vision": false, "trained_for_tool_use": false}
    }
  ]
}
```

**Fehlerfall** (LM Studio nicht erreichbar): leeres Array `[]`.

---

### `GET /api/v1/models/remote`

Kostenlose Modelle von OpenRouter (FREE). `Authorization: Bearer <key>` optional für vollständige Liste.

**Response:**
```json
{
  "success": true,
  "data": [
    {"id": "google/gemma-4-31b-it:free", "pricing": {"prompt": "0", "completion": "0"}},
    {"id": "nvidia/nemotron-3-nano-30b-a3b:free", "pricing": {"prompt": "0", "completion": "0"}}
  ]
}
```

**Fallback:** Wenn OpenRouter nicht erreichbar ist → hartcodierte FREE-Liste.

---

### MCP-Server

Der MCP-Server (separater Stdio-Prozess oder optionaler HTTP-Endpoint)
stellt 7 Tools bereit: `list_connections`, `query`, `execute`, `list_tables`,
`schema`, `list_databases`, `nl2sql`. Details → **`docs/MCP.md`**.

### LLM-Client (Provider)

Einheitliches Interface für OpenRouter (FREE→Paid Fallback), LM Studio und
Ollama. Konfiguration via `config/config.yaml` (`mcp.*`) oder
`GODB_MCP_*` Umgebungsvariablen. Details → **`docs/LLM.md`**.

---

## Health

### GET /health

Server-Health-Check.

**Response:**
```json
{"status": "ok", "version": "0.1.0"}
```

---

## Transfer (DB-Migration)

### POST /transfer

Vollständige Datenmigration zwischen zwei beliebigen Datenbank-Typen.  
Unterstützt: PostgreSQL ↔ MySQL ↔ MariaDB ↔ SQLite ↔ MongoDB ↔ Redis

**Schema + Daten** werden automatisch konvertiert (Typ-Mapping, Batch-Insert).

**Request:**
```json
{
  "source_conn": "conn-sqlite",
  "target_conn": "conn-pg",
  "tables": ["users", "products"],
  "dry_run": false,
  "batch_size": 500,
  "on_conflict": "error"
}
```

**Parameter:**
| Feld | Typ | Standard | Beschreibung |
|------|-----|----------|-------------|
| `source_conn` | string | - | Quell-Verbindungs-ID |
| `target_conn` | string | - | Ziel-Verbindungs-ID |
| `tables` | string[] | alle | Zu migrierende Tabellen |
| `dry_run` | bool | false | Nur Schema generieren, nichts schreiben |
| `batch_size` | int | 100 | Zeilen pro INSERT |
| `on_conflict` | enum | "error" | `error` (abbruch), `skip` (überspringen), `overwrite` (überschreiben) |

**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "transfer-1712345678",
    "status": "pending",
    "source_type": "sqlite",
    "target_type": "postgres"
  }
}
```

### GET /transfer/:id

Status einer Migration abfragen.

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "transfer-1712345678",
    "status": "running",
    "tables": ["users", "products"],
    "source_type": "sqlite",
    "target_type": "postgres"
  }
}
```

### DELETE /transfer/:id

Laufende Migration abbrechen. **Response:** `204 No Content`

---

## Status-Codes

| Code | Bedeutung |
|------|-----------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden (inkl. SETUP_REQUIRED) |
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
| `SETUP_REQUIRED` | Admin-Passwort muss zuerst gesetzt werden |
| `NOT_FOUND` | Resource nicht gefunden |
| `CONFLICT` | Resource existiert bereits / Setup schon abgeschlossen |
| `ALREADY_SETUP` | Setup wurde bereits durchgeführt |
| `SAME_TYPE` | Quelle und Ziel haben den gleichen DB-Typ |
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
