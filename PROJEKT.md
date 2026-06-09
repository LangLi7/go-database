# go-database – Projekt-Dokumentation

## Vision

**go-database** ist eine universelle Datenbank-Middleware / Management-Plattform.
Es fungiert als zentrale Schnittstelle („Hafen") zwischen beliebigen Anwendungen
und verschiedensten Datenbanksystemen.

### Metapher: Der Hafen

```
Anwendungen                          go-database                       Datenbanken
─────────────                        ───────────                       ──────────
Website ─────┐                                         ┌─── PostgreSQL
Discord Bot ─┤                                         ├─── MySQL
Mobile App ──┼──── API (REST/gRPC/WS) ───► go-database ┼─── MariaDB
AI/Code ─────┤                           │             ├─── SQLite
Minecraft ───┘                           │             ├─── MongoDB
                                         │             ├─── Redis
                                    Dashboard          ├─── MSSQL
                                    (Admin-UI)          └─── ... (erweiterbar)
```

- **Anwendungen** müssen sich nicht um DB-Verbindungen, Credentials oder Treiber kümmern
- **go-database** verwaltet Connections, Auth, Permissions, Query-Routing, Caching, Replikation
- **Dashboard** dient der Administration – Connections verwalten, Daten browsen, Queries ausführen, User/Permissions steuern, Traffic überwachen

---

## Ziele

1. **Universelle DB-Schnittstelle** – Eine API für alle Datenbanktypen (SQL + NoSQL)
2. **Sicher & Multi-Tenant** – Volles Permission-System (Rollen + User-spezifische Rechte) für mehrere Teams/Entwickler
3. **Admin-Dashboard** – Webbasierte Verwaltungsoberfläche (kein phpMyAdmin nur für MySQL, sondern für alle DBs)
4. **Erweiterbar** – Plugins für jeden DB-Typ, einfach per `init()` registrierbar
5. **Bidirektionale Sync** – Daten zwischen verschiedenen DB-Typen synchronisieren (z.B. PostgreSQL → SQLite)
6. **API-First** – Alles über API steuerbar, Dashboard ist nur ein Client
7. **Embedded / Single Binary** – Dashboard in Go-Binary eingebettet, keine externen Abhängigkeiten
8. **Adaptives Design** – Netflix/Spotify-artiges Dashboard-Design, via Datenbank konfigurierbar

---

## Zielgruppe & Use Cases

| Use Case | Beschreibung |
|----------|-------------|
| **Webentwickler** | Mehrere Projekte mit verschiedenen DBs über eine API verwalten |
| **Discord Bot** | Bot braucht eine DB – go-database als Middleware, kein direkter DB-Zugriff |
| **Minecraft Server** | Plugin verbindet sich per API statt direktem MySQL-JDBC |
| **AI/Code Generator** | AI generiert Queries → go-database validiert & executed sicher |
| **Team/Abteilung** | Mehrere Entwickler mit abgestuften Rechten auf verschiedenen DBs |
| **Dashboard/Admin** | Nicht-technische User browsen Daten über das Dashboard |
| **CI/CD / Automation** | Backup/Restore, Migrationen, Schema-Änderungen per API |

---

## Architektur

```
┌─────────────────────────────────────────────────────────┐
│                    go-database                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │ REST API │  │   gRPC   │  │ WebSocket│  │   SSE  │ │
│  │  (Gin)   │  │          │  │          │  │        │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬────┘ │
│       │             │             │             │      │
│  ┌────▼─────────────▼─────────────▼─────────────▼────┐ │
│  │              Auth Middleware                       │ │
│  │  JWT / API-Key / Session + Permission Check       │ │
│  └───────────────────────┬───────────────────────────┘ │
│                          │                              │
│  ┌───────────────────────▼───────────────────────────┐ │
│  │              Connection Manager                    │ │
│  │      Verwaltet alle DB-Connections (Pooling)      │ │
│  └────┬──────┬──────┬──────┬──────┬──────┬──────┬────┘ │
│       │      │      │      │      │      │      │      │
│  ┌────▼──┐┌──▼───┐┌──▼───┐┌──▼──┐┌──▼───┐┌──▼──┐    │
│  │PG     ││MySQL ││Maria ││SQL  ││Mongo ││Redis│ ... │
│  │Plugin ││Plugin││Plugin││ite  ││Plugin││Plug.│     │
│  │       ││      ││      ││Plug.││      ││     │     │
│  └───────┘└──────┘└──────┘└─────┘└──────┘└─────┘    │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │           Auth Store (SQLite intern)              │  │
│  │  Users │ Roles │ Permissions │ Activity │ Config │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │        Key Vault (API-Key crypto/rand+SHA256)     │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │    Traffic Monitor (in-memory Request-Log)       │  │
│  └──────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Embedded Dashboard SPA               │  │
│  │         (HTML + CSS + JS, via embed.FS)          │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## DB-Typen (geplant)

### Aktuell implementiert
| Typ | Plugin | Status |
|-----|--------|--------|
| PostgreSQL | `plugins/postgres` | ✅ |
| MySQL | `plugins/mysql` | ✅ |
| MariaDB | `plugins/mariadb` | ✅ |
| SQLite | `plugins/sqlite` | ✅ |
| MongoDB | `plugins/mongodb` | ✅ |
| Redis | `plugins/redis` | ✅ |

### Geplant (müssen noch implementiert werden)
| Typ | Anmerkung |
|-----|-----------|
| **Microsoft SQL Server (MSSQL)** | `github.com/denisenkom/go-mssqldb` |
| **Oracle** | `github.com/godror/godror` |
| **CockroachDB** | PG-kompatibel, eigenes Plugin für spezielle Features |
| **Cassandra / ScyllaDB** | `github.com/gocql/gocql` |
| **ClickHouse** | `github.com/ClickHouse/clickhouse-go` |
| **Elasticsearch** | REST-API-Plugin |
| **Neo4j** | Graph-DB via Bolt-Protocol |
| **Firebird** | `github.com/nakagami/firebirdsql` |
| **InfluxDB** | Time-Series |
| **DuckDB** | Embedded OLAP |
| **SQL Server via sqlcmd** | Lightweight Variante |

Jeder DB-Typ wird als **Plugin** implementiert (Interface `DBPlugin` in `internal/plugin/interface.go`)
und per `init()` automatisch registriert.

---

## Permission-System

### Hierarchie

```
Globale Ebene:
  ├── Rollen (admin, developer, readonly)
  │    ├── Berechtigungen (Permissions): connections:list, queries:run, users:create, ...
  │    └── DB-Zugriff (db_permissions): Welche Connections darf die Rolle sehen?
  │
  └── User-spezifische Overrides (optional)
       ├── Zusätzliche Berechtigungen
       └── Zusätzlicher DB-Zugriff

Datenbank-Ebene (pro Connection):
  ├── Tabellen-Rechte: SELECT, INSERT, UPDATE, DELETE
  ├── Spalten-Rechte: visibility, read-only
  └── Row-Level-Security: Nur bestimmte Zeilen sichtbar
```

### Permission-Typen

| Permission | Beschreibung |
|-----------|-------------|
| `connections:list` | Connections sehen |
| `connections:create` | Neue Connections anlegen |
| `connections:delete` | Connections löschen |
| `connections:query` | Queries ausführen |
| `connections:execute` | Write-Queries ausführen (INSERT/UPDATE/DELETE) |
| `users:list` | User auflisten |
| `users:create` | User anlegen |
| `users:edit` | User bearbeiten |
| `users:delete` | User löschen |
| `settings:read` | Settings lesen |
| `settings:write` | Settings schreiben |
| `jobs:list` | Jobs auflisten |
| `jobs:create` | Jobs anlegen |
| `jobs:cancel` | Jobs abbrechen |
| `backup:create` | Backups erstellen |
| `backup:restore` | Backups wiederherstellen |
| `traffic:view` | Traffic-Monitoring sehen |
| `apikeys:manage` | API-Keys verwalten |
| `roles:manage` | Rollen verwalten |
| `*` | Admin (alle Rechte) |

### User-spezifische Overrides
- Zusätzlich zur Rolle kann ein User eigene Permissions haben
- Beispiel: User hat Rolle "developer" (darf queries ausführen) + override "users:create" (darf auch User anlegen)
- User kann nur Connections sehen, für die er (oder seine Rolle) explizit berechtigt ist

---

## Dashboard-Features (vollständige Liste)

### 1. Dashboard-Startseite
- Stats-Cards: Connections (online/total), Users, Actions Today, Queries Today
- Recent Activity Feed
- Connections-Übersicht mit Status

### 2. Connections
- Grid-Ansicht mit Status (online/offline/error), Typ-Badge, Latenz
- Connection hinzufügen (alle DB-Typen)
- Create Local SQLite DB
- Ping / Delete
- Verbindungsdetails anzeigen

### 3. Explorer
- **Connection auswählen** → Tabellen werden geladen
- **Tabellen**:
  - Liste mit Row-Count + Column-Count
  - Filter/Suche
- **Daten browsern**:
  - Paginierung (50er-Schritte, First/Prev/Next/Last)
  - Spalten-Sortierung (Klick auf Header)
  - Textfilter (LIKE-Suche über alle Spalten)
  - Inline-Editing (Klick auf Zelle → Edit-Modus)
  - Delete Row
  - Insert Row (Formular mit allen Spalten)
  - NULL-Werte visualisieren
- **Tabellen-Struktur**:
  - Spalten anzeigen (Name + Typ)
  - Indizes / Primary Keys
  - Fremdschlüssel

### 4. Query Editor
- Connection-Selector
- SQL-Editor mit Syntax-Highlighting (zukünftig)
- **Autocomplete / Command-Vorschläge** (geplant)
  - Tabellen- und Spaltennamen vorschlagen
  - SQL-Keywords vorschlagen
  - Tastatur-Navigation durch Vorschläge
- AI-Check (OpenRouter-Integration für Query-Validierung)
- Ergebnisse als Tabelle
- Ausführungszeit + Row-Count
- Tastenkürzel: Ctrl+Enter

### 5. Traffic & Activity
- **Live-Stats**: Requests/min, Active Users, Queries/h, Uptime, Total Requests
- **Status-Codes**: Verteilung (2xx, 4xx, 5xx)
- **Top-Endpoints**: Meistaufgerufene Pfade
- **Request-Log**: Letzte 200 Requests mit Methode, Path, Status, Latenz, User
- **Audit-Log**: Wer hat wann was gemacht (aus der Datenbank)

### 6. Users & Roles
- **User-CRUD**: Anlegen, Bearbeiten, Löschen
- **Rollen**: Anlegen, Permissions zuweisen, DB-Zugriff steuern
- **User-spezifische Permissions**: Override für einzelne User
- **DB-Access-Matrix**: Pro Rolle/User Checkboxen für Connections (Discord-Channel-Stil)

### 7. Security
- **API-Keys**: Generieren (crypto/rand + SHA256), Revoke, Prefix-basierte Verwaltung
- **System-Keys**: OpenRouter/AI-Keys, verschlüsselt speichern
- **Audit-Log**: Alle Aktionen protokolliert

### 8. Settings
- **Design-Config** (adaptiv, via DB):
  - Primärfarbe
  - Sidebar-Breite
  - Font-Size
  - Compact Mode
  - Overflow-Verhalten
  - Dark/Light Mode (geplant)
- **About**: Version, Engine-Info

---

## API-Übersicht

### Auth
```
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/change-password
```

### Admin
```
GET/POST   /api/v1/admin/config
GET/POST   /api/v1/admin/design
GET        /api/v1/admin/stats
GET/POST   /api/v1/admin/users
PUT/DELETE /api/v1/admin/users/:id
GET/PUT    /api/v1/admin/users/:id/permissions
GET/POST   /api/v1/admin/roles
PUT/DELETE /api/v1/admin/roles/:id
PUT        /api/v1/admin/roles/:id/permissions
GET        /api/v1/admin/activity
```

### Traffic
```
GET /api/v1/traffic/stats
GET /api/v1/traffic/requests
```

### API Keys
```
GET/POST   /api/v1/apikeys
DELETE     /api/v1/apikeys/:prefix
```

### Connections
```
GET        /api/v1/connections
POST       /api/v1/connections
GET/DELETE /api/v1/connections/:id
GET        /api/v1/connections/:id/ping
GET        /api/v1/connections/:id/tables
GET        /api/v1/connections/:id/schema
POST       /api/v1/connections/:id/query
POST       /api/v1/connections/:id/execute
```

### Explorer (Data)
```
GET    /api/v1/connections/:id/browse/:table
POST   /api/v1/connections/:id/row/:table
PUT    /api/v1/connections/:id/row/:table/:pk/:val
DELETE /api/v1/connections/:id/row/:table/:pk/:val
```

### Jobs / Schedules / Backups / etc.
```
GET/POST   /api/v1/jobs
GET/DELETE /api/v1/jobs/:id
GET/POST/DELETE /api/v1/schedules/...
GET/POST   /api/v1/backups/...
```

---

## Kritik & Lessons Learned (aus dem bisherigen Prozess)

### Was nicht gut war
1. **Zu viele Änderungen auf einmal** – Statt strukturiertem Vorgehen wurde wild gecodet
2. **Kein klares Zielbild** – Es wurde entwickelt, ohne die Vision zu dokumentieren
3. **Dashboard zu komplex** – Wollte zu viele Features auf einmal, keins richtig
4. **User wurde nicht genug einbezogen** – Annahmen getroffen statt nachzufragen
5. **Keine Priorisierung** – Explorer, Traffic, Permissions, Design alles gleichzeitig

### Was wir besser machen
1. **Erst dokumentieren, dann coden**
2. **Inkrementell entwickeln** – Ein Feature nach dem anderen, fertig machen
3. **User-Feedback einholen** – Nach jedem Feature fragen, ob es passt
4. **Klare Prioritäten** – Was ist MVP, was ist Nice-to-have?

---

## Roadmap (Neustart)

### Phase 1: Fundament (Backend)
- [ ] Alle DB-Plugins fertig (auch MSSQL, Oracle, CockroachDB, Cassandra, ...)
- [ ] Einheitliches Query-Interface (SQL normalisieren)
- [ ] Connection-Pool mit Health-Checks
- [ ] Auth-System mit vollem Permission-Modell

### Phase 2: API (REST)
- [ ] Vollständige REST-API für alle Features
- [ ] API-Dokumentation (OpenAPI/Swagger)
- [ ] Rate-Limiting pro User/Key

### Phase 3: Dashboard (SPA)
- [ ] Nützliches, fokussiertes Dashboard
- [ ] Query-Editor mit Autocomplete
- [ ] Explorer mit vollem CRUD
- [ ] Traffic-Monitoring mit Diagrammen
- [ ] User/Rollen-Verwaltung

### Phase 4: Advanced
- [ ] Bidirektionale Sync zwischen DBs
- [ ] Backup/Restore mit Scheduling
- [ ] gRPC API
- [ ] WebSocket für Echtzeit-Updates
- [ ] Multi-Node / Cluster-Betrieb

---

## Technologie-Stack

| Komponente | Technologie |
|-----------|------------|
| Sprache | Go 1.26 |
| HTTP-Framework | Gin |
| Auth | JWT (golang-jwt) + bcrypt |
| API-Keys | crypto/rand + SHA256 |
| Auth-Store | SQLite (modernc.org/sqlite) |
| DB-Plugins | pgx/v5, go-sql-driver/mysql, mongo-driver, go-redis/v9, ... |
| Dashboard | Vanilla HTML/CSS/JS (embedded via embed.FS) |
| Logging | slog (strukturiertes JSON) |
| Realtime | gorilla/websocket + SSE |
| Testing | Standard library testing |

---

## Nächste Schritte

1. **Du sagst mir, ob diese Dokumentation in die richtige Richtung geht**
2. **Wir priorisieren gemeinsam die Features**
3. **Dann bauen wir Feature für Feature – mit deinem Feedback nach jedem Schritt**

**Frage:** Ist diese Vision das, was du dir vorgestellt hast? Was fehlt, was ist anders?
