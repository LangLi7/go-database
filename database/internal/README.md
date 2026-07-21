# go-database — Internal Database Files

Diese Dateien werden von go-database zur Laufzeit automatisch erstellt und verwaltet.

## Dateien

| Datei | Zweck | Technologie |
|-------|-------|------------|
| `auth.db` | User, Rollen, Permissions, API-Keys, Audit-Log, Design-Config | SQLite |

> Hinweis: Früher geplant waren `jobs.db` und `metrics.db`. Diese wurden
> entfernt — Jobs/Schedules liegen jetzt als JSON (`scheduled_jobs.json`),
> Traffic/Audit im `auth.db`. Siehe `DECISIONS.md` ADR-004.

## Hinweise

- Kein manuelles Editieren — nur über die API
- Die Datei wird beim ersten Start automatisch angelegt (Schema-Migration in `internal/internaldb/`)
- Für Production: Backup dieser Datei nicht vergessen!
- Kompatibel mit Linux, Windows, macOS
