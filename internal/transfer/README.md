# Transfer Engine — go-database

## Konzept

Die Transfer Engine erlaubt Daten zwischen **beliebigen DB-Typen** zu verschieben:

```
SQLite ──┐
Postgres ─┤                        ┌── MySQL
MySQL ────┤   Transfer Engine      ├── MariaDB
MariaDB ──┼── Source → Target ────┼── SQLite
MongoDB ──┤   (Type Mapping)      ├── MongoDB
Redis ────┘                        └── Postgres
```

## Ablauf

1. **Source** liest Schema + Daten (streaming, batch-weise)
2. **TypeMapper** konvertiert Typen (z.B. `TEXT → VARCHAR(255)`)
3. **Schema Mapping** — automatisch + manuelles Override
4. **Target** erstellt Tabellen + schreibt Daten
5. **ProgressTracker** reported Fortschritt

## Unterstützte Transformationen

| Source Type | Zielsysteme |
|-------------|-------------|
| `TEXT` | `VARCHAR(255)` (MySQL), `TEXT` (PG), `string` (Mongo) |
| `INTEGER` | `INT` (MySQL), `INTEGER` (PG), `int32` (Mongo) |
| `BOOLEAN` | `TINYINT(1)` (MySQL), `BOOLEAN` (PG), `bool` (Mongo) |
| `TIMESTAMP` | `DATETIME` (MySQL), `TIMESTAMP` (PG), `Date` (Mongo) |
| `DECIMAL(10,2)` | `DECIMAL(10,2)` (MySQL), `NUMERIC(10,2)` (PG), `double` (Mongo) |

## Dry-Run Modus

```json
{
  "dryRun": true,
  "sourceConn": "sqlite-dev",
  "targetConn": "pg-dev",
  "tables": ["users", "products"]
}
```

→ Zeigt Schema + Row-Count, ohne zu schreiben.
