# Konzept: go-database als DB-Gateway / Mittelsmann

_Status:_ Teilweise schon Realität — hier gesammelt, was fehlt.

## Was schon da ist (der "Mittelsmann" existiert)
- `connection.Manager` registriert **beliebige externe DBs** (Postgres, MySQL,
  MariaDB, MSSQL, Mongo, Redis, SQLite) über `plugin.Register`.
- Alle Zugriffe laufen durch Schichten:
  - `executor` — Query-Ausführung
  - `guard/rbac.go` — SQL-Allow/Blocklist + Permissions
  - `internaldb` — Audit-Log (`store.LogAudit`)
- API-only: Clients reden NUR mit go-database, nie direkt mit der DB.
  → Credential-Hiding + zentrales Routing ist bereits gebaut.

## Was fehlt / offen
1. **Credential-Verschlüsselung im Store**: `plugin.Config.Password` ist
   `json:"-"` (nicht in API-Ausgabe), aber beim Speichern in `internaldb`
   landet es vermutlich im Klartext in `database/internal/auth.db`.
   `internal/crypto` (AES-256-GCM, RSA-OAEP, X25519) ist da → beim `Add()`
   verschlüsseln (≈15 Z.).
2. **RBAC für Connections**: `users`-Tabelle hat `extra_db_access` (pro User:
   welche Connections sichtbar). Ist das im Handler durchgesetzt? Prüfen!
3. **Keine eigene Container-Orchestrierung**: Docker Compose hat Sample-DBs,
   aber keine "DB per API hoch/runter"-Engine. YAGNI — dafür ist echtes
   K8s/Docker die Plattform, nicht unsere App.

## Idee: "Interne DB als Connection-Registry"
User will: externe DB anbinden, aber mit Login/Sichtbarkeit gesteuert über
eine interne Verwaltungs-DB.
→ Das ist exakt `internaldb` + `connection.Manager` + RBAC. Fehlt nur:
- Credentials verschlüsselt speichern (siehe 1)
- `extra_db_access` im Handler erzwingen (siehe 2)
- Optional: Connection-Groups/Rollen für feingranulare Sichtbarkeit

## Nicht bauen (YAGNI)
- Eigenes K8s-ähnliches Orchestrierungssystem in go-database.
- Eigener User-Store neben `internaldb` (der ist schon da).

## Nächster Schritt
Kleiner, sicherer Schritt: Credential-Verschlüsselung im `Add()`-Pfad
nachrüsten + prüfen ob RBAC `extra_db_access` greift. Dann ist go-database
ein geprüfter, verschlüsselter DB-Mittelsmann.
