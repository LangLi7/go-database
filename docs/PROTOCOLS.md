# go-database — API-Protokoll-Referenz

**Stand:** Code-Audit. Status pro Protokoll ist markiert:
- ✅ **Implementiert** — im Repo vorhanden, gegen den Code dokumentiert
- 📋 **Geplant** — Design-Spezifikation, Beispiele sind Ziel-Kontrakte (noch nicht im Code)

Basis-URL für alle HTTP-Protokolle: `http://localhost:8080/api/v1`
Einheitliches Response-Format (REST): `{ "success": bool, "data": ..., "error": {"code","message"}, "meta": {"timestamp","request_id"} }`

---

# TEIL A — Implementiert (✅)

## A1. REST (HTTP/JSON) ✅

Die Haupt-API. Vollständig in `internal/api/handler/*` + `internal/api/router/routes.go`.

### A1.1 Authentifizierung

Alle Endpoints ausser `/health`, `/setup/*`, `/auth/login` brauchen einen Header:
```
Authorization: Bearer <JWT>
```
oder einen API-Key:
```
Authorization: Bearer <raw_api_key>
```

**Setup-Status prüfen** (vor dem ersten Login):
```bash
curl http://localhost:8080/api/v1/setup/status
# → {"success":true,"data":{"setup_complete":false}}
```

**Setup initialisieren** (nur beim ersten Start; setzt Admin-Passwort):
```bash
curl -X POST http://localhost:8080/api/v1/setup/initialize \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"meinSicheresPasswort123"}'
# → 204 No Content
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"meinSicheresPasswort123"}'
# → {"success":true,"data":{"token":"eyJ...","user_id":"admin-001","username":"admin","role":"admin"}}
```

**Token refreshen:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Authorization: Bearer <JWT>" \
  -d '{}'
```

**Passwort ändern:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"alt","new_password":"neuSicher123"}'
# → 204 No Content
```

### A1.2 Connections (Datenbank-Verbindungen)

```bash
# Liste aller Connections
curl http://localhost:8080/api/v1/connections -H "Authorization: Bearer <JWT>"

# Neue Connection anlegen (PostgreSQL)
curl -X POST http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"Mein PG","type":"postgres","host":"localhost","port":5432,
       "database":"appdb","user":"dev","password":"dev123","tags":["prod"]}'

# SQLite (nur filepath)
curl -X POST http://localhost:8080/api/v1/connections \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"Lokal","type":"sqlite","filepath":"./database/samples/sqlite/sample.db"}'

# Verbindung testen (ohne zu speichern)
curl -X POST http://localhost:8080/api/v1/connections/test \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"type":"postgres","host":"localhost","port":5432,"database":"appdb","user":"dev","password":"dev123"}'

# Details / Ping / Tabellen / Schema
curl http://localhost:8080/api/v1/connections/<ID> -H "Authorization: Bearer <JWT>"
curl http://localhost:8080/api/v1/connections/<ID>/ping -H "Authorization: Bearer <JWT>"
curl http://localhost:8080/api/v1/connections/<ID>/tables -H "Authorization: Bearer <JWT>"
curl http://localhost:8080/api/v1/connections/<ID>/schema -H "Authorization: Bearer <JWT>"

# Löschen
curl -X DELETE http://localhost:8080/api/v1/connections/<ID> -H "Authorization: Bearer <JWT>"
```

### A1.3 Explorer (Daten-CRUD)

```bash
# Tabelle browsen (paginierbar, filter/sort)
curl "http://localhost:8080/api/v1/connections/<ID>/browse/users?page=1&per_page=50&sort=created_at&dir=desc&filter=active=1" \
  -H "Authorization: Bearer <JWT>"

# Nur SELECT erlaubt auf /query (SQL-Guard)
curl -X POST http://localhost:8080/api/v1/connections/<ID>/query \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"query":"SELECT id,name,email FROM users WHERE active=1 LIMIT 10"}'

# INSERT/UPDATE/DELETE auf /execute
curl -X POST http://localhost:8080/api/v1/connections/<ID>/execute \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"query":"UPDATE users SET name='\''Bob'\'' WHERE id=1"}'

# Zeile einfügen / ändern / löschen (Explorer-Helper, kein rohes SQL nötig)
curl -X POST http://localhost:8080/api/v1/connections/<ID>/row/users \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'   # → 201

curl -X PUT http://localhost:8080/api/v1/connections/<ID>/row/users/id/1 \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated"}'

curl -X DELETE http://localhost:8080/api/v1/connections/<ID>/row/users/id/1 \
  -H "Authorization: Bearer <JWT>"                        # → 204
```

Response-Format (`/browse`):
```json
{
  "data": [[1,"Alice","alice@example.com"]],
  "columns": ["id","name","email"],
  "page": 1, "per_page": 50, "total": 100, "total_pages": 2, "duration_ms": 5
}
```

### A1.4 Transfer (Datenbank-Typ-Konverter) ✅

Genau dein "Hafen-Konverter"-Ziel — funktionsfähig zwischen allen 6 Plugin-Typen.

```bash
# SQLite → PostgreSQL migrieren
curl -X POST http://localhost:8080/api/v1/transfer \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{
    "source_conn": "<SQLITE_ID>",
    "target_conn": "<PG_ID>",
    "tables": ["users","products"],
    "dry_run": false,
    "batch_size": 500,
    "on_conflict": "overwrite"
  }'
# → 201: {"success":true,"data":{"id":"transfer-...","status":"pending",
#       "source_type":"sqlite","target_type":"postgres"}}

# Status / Abbrechen
curl http://localhost:8080/api/v1/transfer/<ID> -H "Authorization: Bearer <JWT>"
curl -X DELETE http://localhost:8080/api/v1/transfer/<ID> -H "Authorization: Bearer <JWT>"
```

Unterstützte Konvertierungen (beide Richtungen):
`postgres ↔ mysql ↔ mariadb ↔ sqlite ↔ mongodb ↔ redis`

### A1.5 Admin (User, Rollen, Permissions, Design)

```bash
curl http://localhost:8080/api/v1/admin/users -H "Authorization: Bearer <JWT>"
curl -X POST http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"username":"dev1","password":"secret123","role":"developer"}'
curl http://localhost:8080/api/v1/admin/roles -H "Authorization: Bearer <JWT>"
curl -X POST http://localhost:8080/api/v1/admin/roles \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"DBA","permissions":["connections:list","connections:query","backup:create"]}'
curl http://localhost:8080/api/v1/admin/activity -H "Authorization: Bearer <JWT>"
```

### A1.6 API-Keys

```bash
curl -X POST http://localhost:8080/api/v1/apikeys \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{"name":"CI Pipeline","permissions":["connections:list","connections:query"]}'
# → {"success":true,"data":{"raw_key":"a1b2c3...","prefix":"a1b2c3","formatted":"a1b2c3****cdef"}}
# ACHTUNG: raw_key nur EINMAL sichtbar!
curl -X DELETE http://localhost:8080/api/v1/apikeys/<PREFIX> -H "Authorization: Bearer <JWT>"
```

### A1.7 Health

```bash
curl http://localhost:8080/health
# → {"status":"ok","version":"0.1.0"}
```

---

## A2. WebSocket ✅

Echtzeit-Query-Streaming pro Connection. Handler: `internal/api/handler/ws.go`.
Endpoint: `GET /api/v1/ws/query/:id` (Upgrade auf `ws://`, Auth via `?token=` oder Header).

**Client-Nachricht (JSON):**
```json
{ "type": "query", "query": "SELECT * FROM users LIMIT 5", "req_id": "r1" }
{ "type": "execute", "query": "INSERT INTO users(name) VALUES('x')", "req_id": "r2" }
{ "type": "ping" }
```

**Server-Antwort (JSON):**
```json
{ "type": "connected", "success": true }
{ "type": "result", "req_id": "r1", "success": true,
  "data": {"columns":["id","name"],"rows":[[1,"Alice"]],"rows_affected":0,"duration_ms":3} }
{ "type": "result", "req_id": "r2", "success": false, "error": "execute failed: ..." }
{ "type": "pong", "req_id": "r3", "success": true }
{ "type": "notification", "event": "row.insert", "data": {...}, "time": "2026-..." }
```

**Minimal-Beispiel (Node.js):**
```js
const ws = new WebSocket("ws://localhost:8080/api/v1/ws/query/<ID>?token=<JWT>");
ws.on("message", m => console.log(JSON.parse(m)));
ws.send(JSON.stringify({ type:"query", query:"SELECT 1", req_id:"1" }));
```
> Hinweis: `hub` broadcastet an alle Clients derselben Connection (kein 1:1).
> `CheckOrigin` ist aktuell `return true` — für Production einschränken.

---

## A3. Server-Sent Events (SSE) ✅

Server-push von Live-Events. Handler: `internal/api/handler/sse.go`.
Endpoints: `GET /api/v1/sse/activity`, `GET /api/v1/sse/stats` (beide Auth-pflichtig).

```bash
curl -N http://localhost:8080/api/v1/sse/stats -H "Authorization: Bearer <JWT>"
```
```
event: connected
data: {"status":"ok"}

event: stats
data: {"timestamp":"2026-...","total_connections":3,"active_connections":2}

event: heartbeat
data: {"timestamp":"2026-...","connections":3}
```
> Intervalle: stats alle 2s, heartbeat/activity alle 3s. Client trennt → Stream endet.

---

# TEIL B — Geplant (📋 Design-Spezifikation)

Diese Protokolle sind **noch nicht im Code**. Die folgenden Beispiele sind
**Ziel-Kontrakte**, damit du / dein Rust-Frontend / andere Clients wissen,
welches Verhalten erwartet wird. Priorität unten (B10).

Gemeinsame Regel für alle: sie sprechen denselben **Core-Service** an
(`connection.Manager` + `plugin.DBPlugin`), nur der Transport unterscheidet sich.
Auth: JWT/API-Key wird je Protokoll-idiomatisch übertragen.

---

## B1. GraphQL 📋

**Motivation:** Typsichere, deklarative Queries — ideal für dein späteres
phpMyAdmin-ähnliches Rust/Tauri-Frontend (kann genau die Felder wählen).

**Empfohlener Endpoint:** `POST /api/v1/graphql` (+ `GET` für Playground)

**Schema-Skizze:**
```graphql
type Query {
  connections: [Connection!]!
  connection(id: ID!): Connection
  tables(connId: ID!): [String!]!
  rows(connId: ID!, table: String!, page: Int=1, perPage: Int=50,
       filter: String, sort: String, dir: SortDir=ASC): RowPage!
  runQuery(connId: ID!, query: String!): QueryResult!
}

type Mutation {
  createConnection(input: ConnectionInput!): Connection!
  insertRow(connId: ID!, table: String!, values: JSON!): Row!
  updateRow(connId: ID!, table: String!, pk: String!, val: String!, values: JSON!): Row!
  deleteRow(connId: ID!, table: String!, pk: String!, val: String!): Boolean!
  transfer(input: TransferInput!): TransferJob!
}
```

**Beispiel-Request:**
```bash
curl -X POST http://localhost:8080/api/v1/graphql \
  -H "Authorization: Bearer <JWT>" -H "Content-Type: application/json" \
  -d '{
    "query": "query { rows(connId:\"<ID>\", table:\"users\", page:1, perPage:10) { columns rows total } }"
  }'
```
**Beispiel-Response:**
```json
{ "data": { "rows": { "columns":["id","name"], "rows":[[1,"Alice"]], "total":42 } } }
```
> Implementierungs-Hinweis: `github.com/graphql-go/graphql` oder `gqlgen`.
> RowPage.rows als `[JSON!]!` (heterogene Typen aus SQL).

---

## B2. SOAP 📋

**Motivation:** Enterprise-Interop (legacy Java/.NET-Clients, Banken-Systeme).
**Endpoint:** `POST /api/v1/soap` (Content-Type `text/xml; charset=utf-8`)
SOAP 1.1/1.2 Envelope; WSDL unter `GET /api/v1/soap?wsdl`.

**Beispiel-Request (Query ausführen):**
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Header>
    <AuthToken>eyJ...JWT...</AuthToken>
  </soap:Header>
  <soap:Body>
    <RunQuery>
      <connId>conn-abc</connId>
      <query>SELECT * FROM users LIMIT 5</query>
    </RunQuery>
  </soap:Body>
</soap:Envelope>
```

**Beispiel-Response:**
```xml
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <RunQueryResponse>
      <columns>id,name,email</columns>
      <rows>
        <row><id>1</id><name>Alice</name><email>a@x.com</email></row>
      </rows>
      <durationMs>4</durationMs>
    </RunQueryResponse>
  </soap:Body>
</soap:Envelope>
```
> Faults im `soap:Fault` mit `faultcode` = Error-Code (z.B. `QUERY_FAILED`).

---

## B3. JSON-RPC 2.0 📋

**Motivation:** Schlanke RPC-Schnittstelle, gut für CLI-Tools und Scripting.
**Endpoint:** `POST /api/v1/rpc` (Content-Type `application/json`)

**Beispiel-Request (Batch möglich):**
```json
[
  { "jsonrpc":"2.0", "id":1, "method":"connection.list", "params":{} },
  { "jsonrpc":"2.0", "id":2, "method":"query.run",
    "params":{ "connId":"conn-abc", "query":"SELECT 1" } },
  { "jsonrpc":"2.0", "id":3, "method":"row.insert",
    "params":{ "connId":"conn-abc", "table":"users", "values":{ "name":"Bob" } } }
]
```

**Beispiel-Response:**
```json
[
  { "jsonrpc":"2.0", "id":1, "result":[ {"id":"conn-abc","name":"PG","type":"postgres"} ] },
  { "jsonrpc":"2.0", "id":2, "result":{ "columns":["?column?"],"rows":[[1]],"duration_ms":1 } },
  { "jsonrpc":"2.0", "id":3, "result":{ "rows_affected":1 } }
]
```
**Fehler:**
```json
{ "jsonrpc":"2.0", "id":2, "error":{ "code":-32000, "message":"QUERY_FAILED: ..." } }
```
> Methoden-Namensraum: `connection.*`, `query.*`, `row.*`, `transfer.*`, `admin.*`.

---

## B4. XML-RPC 📋

**Motivation:** Maximale Kompatibilität für alte Clients (Classic-PHP, Python xmlrpc).
**Endpoint:** `POST /api/v1/xmlrpc` (Content-Type `text/xml`)

**Beispiel-Request:**
```xml
<?xml version="1.0"?>
<methodCall>
  <methodName>query.run</methodName>
  <params>
    <param><value><string>conn-abc</string></value></param>
    <param><value><string>SELECT * FROM users LIMIT 1</string></value></param>
    <param><value><string>eyJ...JWT...</string></value></param>
  </params>
</methodCall>
```
**Beispiel-Response:**
```xml
<?xml version="1.0"?>
<methodResponse>
  <params>
    <param><value><struct>
      <member><name>columns</name><value><array>...</array></value></member>
      <member><name>rows</name><value><array>...</array></value></member>
    </struct></value></param>
  </params>
</methodResponse>
```
> Auth als letzter Param (Token) oder HTTP-Header.

---

## B5. gRPC 📋

**Motivation:** Hochperformant, typsicher, ideal für Service-to-Service
(z.B. dein Rust-Frontend-Backend oder Microservices). Passt zu ADR-008
(Go↔Rust localhost gRPC).

**Service-Definition (protobuf-Skizze):**
```protobuf
service GoDatabase {
  rpc ListConnections(Empty) returns (ConnectionList);
  rpc GetConnection(GetRequest) returns (Connection);
  rpc CreateConnection(CreateRequest) returns (Connection);
  rpc RunQuery(QueryRequest) returns (QueryResult);
  rpc RunExecute(ExecuteRequest) returns (ExecuteResult);
  rpc StreamQuery(QueryRequest) returns (stream QueryResult);  // analog zu WS/SSE
  rpc StartTransfer(TransferRequest) returns (TransferJob);
}

message QueryRequest { string conn_id=1; string query=2; string auth_token=3; }
message QueryResult  { repeated string columns=1; repeated Row rows=2; int64 duration_ms=3; }
```

**Transport:** `grpc-server` auf eigener Port (z.B. `:8081`), neben der HTTP-API.
> Ist in ADR-003/008 als Go↔Rust-Modell vorgesehen; aktuell NICHT im Code.

---

## B6. MQTT 📋

**Motivation:** IoT / Pub-Sub — viele Clients subscriben auf DB-Events ohne Pollen.
**Broker:** Integrierter MQTT-Broker (z.B. `github.com/mochi-mqtt/server`) oder
Bridge zu externem Broker.

**Topics:**
```
godb/<connId>/rows/<table>/insert
godb/<connId>/rows/<table>/update
godb/<connId>/rows/<table>/delete
godb/<connId>/status           # connected/disconnected/error
godb/global/stats             # heartbeat-ähnlich
```

**Publish-Beispiel (Server → Client bei INSERT):**
```json
{ "event":"insert", "connId":"conn-abc", "table":"users",
  "row":[7,"Alice"], "columns":["id","name"], "ts":"2026-..." }
```
**Subscribe-Beispiel (Client → Server, Command via topic):**
```
PUBLISH godb/<connId>/cmd/query  {"reqId":"r1","query":"SELECT 1"}
# Antwort auf godb/<connId>/cmd/query/resp/r1
```
> Auth: MQTT v5 User/Password = API-Key, oder JWT im Connect-Properties.

---

## B7. Webhooks 📋

**Motivation:** Outbound-Benachrichtigung — go-database pusht Events an
registrierte URLs (z.B. dein Rust-Frontend, Slack, CI).

**Endpoint (Registrierung):** `POST /api/v1/webhooks`
```json
{ "url":"https://myapp.example.com/hook", "events":["row.insert","connection.error"],
  "secret":"whsec_xxx" }
```

**Payload (Server → deine URL):**
```json
{
  "event": "row.insert",
  "connId": "conn-abc",
  "table": "users",
  "row": [7,"Alice"],
  "columns": ["id","name"],
  "timestamp": "2026-...",
  "signature": "sha256=..."
}
```
> `signature` = HMAC-SHA256 des Body mit `secret` (Verifizierung deinerseits).
> Events aus `NotifyWebSocket` + Audit-Log speisen.

---

## B8. FIX Protocol 📋

**Motivation:** Finanz-/Trading-Systeme (falls DB Marktdaten/Orders speichert).
FIX 4.2/4.4 über TCP session layer.

**Session:** Initiator/Acceptor auf Port `:8082`.
**Mapping (Beispiel — MarketData-Message → DB-Row):**
```
8=FIX.4.2|9=...|35=W|34=2|49=GODB|56=CLIENT|55=AAPL|270=191.45|271=100|10=...
```
→ go-database schreibt `INSERT INTO market_data(symbol,price,qty) VALUES('AAPL',191.45,100)`
bei `35=W` (MarketDataSnapshot).
> Sehr spezifisch; nur relevant wenn Finanz-Use-Case. `github.com/quickfixgo/quickfix` als Basis.

---

## B9. OData 📋

**Motivation:** Standardisiertes Querying über REST (Filter/Sort/Pagination via URL).
Ideal für Grid-Controls in deinem Rust-phpMyAdmin (AG-Grid, etc.).

**Endpoint:** `GET /api/v1/odata/<connId>/<table>`

**Beispiel:**
```
GET /api/v1/odata/conn-abc/users?$filter=name eq 'Alice' and active eq true
      &$orderby=created_at desc&$top=10&$skip=20&$select=id,name,email
```
**Response (OData-JSON):**
```json
{
  "@odata.context": ".../$metadata#users",
  "value": [ { "id":1, "name":"Alice", "email":"a@x.com" } ],
  "@odata.count": 42
}
```
> `$filter` wird auf `sanitizeFilter`-Regeln abgebildet (SQL-Injection-Schutz).
> `$metadata` unter `GET /api/v1/odata/$metadata`.

---

# TEIL C — Empfehlung: Implementierungs-Priorität

| # | Protokoll | Aufwand | Nutzen (dein Ziel) | Priorität |
|---|-----------|---------|---------------------|-----------|
| 1 | **JSON-RPC 2.0** | niedrig | CLI/Scripting, nah an REST | 🔴 hoch |
| 2 | **GraphQL** | mittel | Rust-Tauri-Frontend typsicher | 🔴 hoch |
| 3 | **OData** | mittel | Grid-Frontend (phpMyAdmin-Style) | 🟠 mittel |
| 4 | **gRPC** | mittel | Service-to-Service, Rust-Bridge | 🟠 mittel |
| 5 | **Webhooks** | niedrig | Event-Push an dein Frontend/CI | 🟠 mittel |
| 6 | **MQTT** | mittel | IoT/Pub-Sub | 🟡 low (nur bei Bedarf) |
| 7 | **XML-RPC** | niedrig | Legacy-Clients | 🟡 low |
| 8 | **SOAP** | mittel | Enterprise/Legacy | 🟡 low |
| 9 | **FIX** | hoch | nur bei Finanz-Use-Case | ⚪ optional |

**Empfehlung:** Zuerst **JSON-RPC** (schnell, deckt RPC-Clients ab) + **GraphQL**
(für dein Rust-Frontend), dann **OData** (Grid-UI). gRPC/Webhooks als Nächstes.

---

# TEIL D — Gemeinsame Hinweise (alle Protokolle)

- **Auth:** JWT oder API-Key. Bei Nicht-HTTP (gRPC/MQTT/FIX) im jeweiligen
  Transport (Metadata / Password / Session).
- **Permission-Check:** Jeder Transport läuft durch dieselbe RBAC-Middleware
  (`middleware.AuthMiddleware` + Permission-Check), damit Sicherheit identisch ist.
- **SQL-Guard:** Auch bei GraphQL/gRPC/SOAP gilt der Command-Whitelist-Check
  (SELECT auf `query`, Write auf `execute`).
- **Rate-Limit:** Aktuell nur Login-limitiert (siehe RISKS.md C-2); für alle
  neuen Protokolle globales Limit pro Key/User empfohlen.
- **Concurrency:** SQLite-Connections auf 1 Writer (`SetMaxOpenConns(1)`) →
  bei paralleler Last Postgres-Connections bevorzugen (RISKS.md C-1).
