# go-database — Internal Database Files

Diese Dateien werden von go-database zur Laufzeit automatisch erstellt und verwaltet.

## Dateien

| Datei | Zweck | Technologie |
|-------|-------|------------|
| `auth.db` | User, Rollen, Permissions, API-Keys | SQLite |
| `jobs.db` | Job-Queue, Cron-Schedules, Historie | SQLite |
| `metrics.db` | Traffic-Logs, Audit-Logs, Metriken | SQLite |

## Hinweise

- Kein manuelles Editieren — nur über die API/Dashboard
- Die Dateien werden beim ersten Start automatisch angelegt (Schema-Migration in `internal/internaldb/`)
- Für Production: Backup dieser Dateien nicht vergessen!
- Kompatibel mit Linux, Windows, macOS
