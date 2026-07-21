# go-database — Agent Rules

## Für Goose Agent: Lies diese Datei zuerst, dann PROJEKT.md, dann TODO.md, dann DECISIONS.md.

---

## Arbeitsweise

### Phasen
- Arbeite TODO.md Phasen **sequenziell** ab (Phase 1 → 2 → 3 ...)
- Hake jeden Task mit `[x]` ab wenn fertig
- Starte die nächste Phase automatisch ohne zu fragen
- **Frage nur wenn**: kritische Entscheidung nötig die nicht in DECISIONS.md steht

### Innerhalb einer Phase
- Tasks ohne Abhängigkeiten können zusammen implementiert werden
- Erstelle immer alle Files einer Phase bevor du zur nächsten gehst
- Teste jeden Task kurz (compile check minimum)

---

## Code Style (Go)

```
- Paketname: lowercase, kein Underscore (manager, nicht connection_manager)
- Interfaces: enden auf -er wenn möglich (DBPlugin, nicht IDBPlugin)
- Errors: wrappen mit fmt.Errorf("context: %w", err)
- Keine panic() ausser in init() für kritische Konfigurationsfehler
- Context überall wo IO stattfindet (ctx context.Context als erster Parameter)
- Structured Logging: slog (Go standard, kein Logrus/Zap)
- Keine globalen Variablen ausser Logger und Config (nach Init)
```

## Code Style (Rust)

```
- Kein unwrap() in Production Code, immer ? oder match
- thiserror für Error Types
- tokio für async Runtime
- tonic für gRPC
- Clippy warnings = errors (kein Code mit Warnings committen)
```

---

## File Struktur Regeln

- **Nie** Business Logic in `cmd/` — nur Startup und Dependency Injection
- **Nie** direkte DB Calls ausserhalb von `internal/plugins/`
- **Nie** HTTP Handler Logik in `internal/api/routes.go` — Routes nur registrieren, Logik in Handler Files
- Config immer über `internal/config/` lesen, nie direkt os.Getenv in Business Logic

---

## API Conventions

### Response Format (immer einheitlich)
```json
// Success
{
  "success": true,
  "data": { ... },
  "meta": { "timestamp": "...", "request_id": "..." }
}

// Error
{
  "success": false,
  "error": {
    "code": "CONNECTION_NOT_FOUND",
    "message": "Connection with id 'xyz' not found",
    "details": { ... }
  },
  "meta": { "timestamp": "...", "request_id": "..." }
}
```

### HTTP Status Codes
- 200: Success (GET, PUT)
- 201: Created (POST)
- 204: No Content (DELETE)
- 400: Bad Request (Validation Error)
- 401: Unauthorized
- 403: Forbidden
- 404: Not Found
- 409: Conflict
- 500: Internal Server Error

### Route Naming
```
GET    /api/v1/connections          → Liste
POST   /api/v1/connections          → Erstellen
GET    /api/v1/connections/:id      → Einzeln
PUT    /api/v1/connections/:id      → Update
DELETE /api/v1/connections/:id      → Löschen
POST   /api/v1/connections/:id/ping → Action
```

---

## Fehlerbehandlung

- Alle Errors loggen mit slog (level: Error) inkl. context
- Nie raw errors zum Client senden (internal details verstecken)
- SQL Errors: generic "query failed" zum Client, details nur im Log
- Connection Errors: retry 3x mit exponential backoff bevor Error zurück

---

## Security Regeln

- **Niemals** Credentials im Log ausgeben
- **Niemals** SQL Queries unvalidiert durchlassen (Security Gate Middleware)
- API Keys immer gehasht vergleichen (constant time comparison)
- JWT Secret aus Environment Variable, nie hardcoded

---

## Testing

- Jeder Plugin braucht minimum einen Unit Test (mock connection)
- Integration Tests für DB Plugins: nur wenn lokale DB verfügbar (build tag: integration)
- API Handler Tests: httptest.NewRecorder
- Minimum: Compile check + Unit Tests müssen grün sein

---

## Entscheidungen

- Wenn eine Entscheidung nicht in DECISIONS.md steht und du nicht sicher bist: **frage**
- Wenn du eine Entscheidung getroffen hast die wichtig ist: **trage sie in DECISIONS.md ein**
- Folge immer den bestehenden Decisions, weiche nicht ab ohne Rückfrage

---

## Was du NICHT tun sollst

- Keine Dependencies hinzufügen die nicht in ARCHITECTURE.md oder DECISIONS.md stehen ohne Rückfrage
- Kein Code generieren der nicht in der aktuellen Phase steht (nicht vorwärts springen)
- Keine Dateien ausserhalb der definierten Verzeichnisstruktur erstellen
- Nicht die Plugin Interface Signatur ändern ohne alle bestehenden Plugins anzupassen
- Keine TODO Kommentare im Code lassen — entweder implementieren oder in TODO.md als Backlog eintragen

---

## Wie du Fortschritt zeigst

Nach jeder abgeschlossenen Phase:
1. TODO.md updaten (alle Tasks der Phase mit [x])
2. Kurze Zusammenfassung was erstellt wurde
3. Direkt mit nächster Phase anfangen

---

## Projekt Kontext

- Dieses Projekt wird später als Komponente für **Goose Agent** verwendet
- Deshalb ist API-first wichtig — alles muss programmatisch nutzbar sein
- Docker deployment ist Pflicht — muss auf einem VPS laufen können
- Self-hosted, kein Cloud-Dependency
