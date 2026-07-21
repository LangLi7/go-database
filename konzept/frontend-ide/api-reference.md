# API-Referenz für die Web-IDE (Client-Spec)

Generiert aus `internal/api/router/routes.go`. Alle Endpoints, die der
Frontend-Client konsumieren muss. Basis: `http://localhost:8080/api/v1`
(oder Remote-Host). Auth: `Authorization: Bearer <jwt>` oder
`X-API-Key: <key>`.

## Auth (public)
| Methode | Pfad | Body | Beschreibung |
|---------|------|------|--------------|
| POST | /auth/login | {username,password} | JWT holen |
| POST | /auth/refresh | {refresh_token} | Token erneuern |
| GET | /auth/verify | — | Token prüfen |
| POST | /auth/change-password | {old,new} | Passwort ändern |
| POST | /auth/passkeys/login/begin | — | WebAuthn login start |
| POST | /auth/passkeys/login/finish | — | WebAuthn login finish |
| GET/POST/DELETE | /auth/passkeys[/:id] | — | Passkey verwalten |
| GET | /setup/status | — | Erst-Setup Status |
| POST | /setup/initialize | — | Admin initialisieren |
| GET | /health | — | Health-Check |

## Connections
| Methode | Pfad | Perm | Beschreibung |
|---------|------|------|--------------|
| GET | /connections | List | Alle Connections |
| POST | /connections | Create | Connection anlegen |
| POST | /connections/test | Create | Verbindung testen |
| GET | /connections/:id | List | Detail |
| GET | /connections/:id/ping | List | Ping |
| GET | /connections/:id/tables | List | Tabellen |
| GET | /connections/:id/schema | List | Schema (Spalten) |
| GET | /connections/:id/databases | List | Datenbanken |
| POST | /connections/:id/query | Query | SELECT |
| POST | /connections/:id/execute | Exec | WRITE/DDL |
| POST | /connections/:id/databases | Exec | DB anlegen |
| DELETE | /connections/:id/databases/:name | Exec | DB droppen |
| POST | /connections/:id/tables | Exec | Tabelle anlegen |
| DELETE | /connections/:id/tables/:name | Exec | Tabelle droppen |
| DELETE | /connections/:id | Delete | Connection löschen |
| POST | /databases/standalone | Create | Standalone-DB |

## Data-Browse / CRUD (pro Connection)
| Methode | Pfad | Perm | Beschreibung |
|---------|------|------|--------------|
| GET | /connections/:id/browse/:table | Query | Grid (paginiert) |
| POST | /connections/:id/row/:table | Exec | Insert |
| PUT | /connections/:id/row/:table/:pk/:val | Exec | Update |
| DELETE | /connections/:id/row/:table/:pk/:val | Exec | Delete |

## Per-Type Direct (kein Save)
| Methode | Pfad | Perm |
|---------|------|------|
| POST | /db/:type/query | Query |
| POST | /db/:type/execute | Exec |
| POST | /db/:type/test | Create |

## Streaming / Real-time
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET (WS) | /ws/query/:id | Streaming-Query |
| GET (WS) | /ws/transfer/:id | Transfer-Progress |
| GET (SSE) | /sse/activity | Live-Activity |
| GET (SSE) | /sse/stats | Live-Stats |

## Transfer (Migration)
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| POST | /transfer | Transfer starten |
| GET | /transfer/:id | Status |
| DELETE | /transfer/:id | Cancel |
| GET | /transfer/:id/log | Log |

## Suggest / AI-SQL
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| POST | /suggest | SQL-Autocomplete |
| POST | /suggest/ai | AI-Vorschlag |
| POST | /execute/safe | Guarded-Run |

## Agent (NL→SQL + Tools)
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| POST | /agent/chat | Chat (JSON) |
| GET (SSE) | /agent/stream | Streaming-Antwort |

## Models / LLM
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | /models/local | LM Studio Modelle |
| GET | /models/remote | OpenRouter Free-Modelle |
| POST | /models/download | huggingface-cli Download |
| POST | /models/start | llama-server starten |

## Hardware Cookbook (neu)
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | /hardware | Host-Spec (RAM/CPU/GPU) |
| GET | /recipes | Recipe-Liste |
| POST | /recipes/:name | Recipe ausführen |

## Templates / Samples
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | /templates | SQL-Templates |
| POST | /templates/apply | Template anwenden |
| GET | /samples | Sample-DBs |
| POST | /connections/:id/samples/:sample | Sample laden |
| POST | /connections/:id/import | Daten importieren |
| POST | /connections/:id/import/csv | CSV import |

## Admin
| Methode | Pfad | Perm |
|---------|------|------|
| GET | /admin/stats | TrafficView |
| GET | /admin/activity | TrafficView |
| GET/POST | /admin/design | SettingsRead/Write |
| GET/POST/PUT/DELETE | /admin/users[/:id] | Users* |
| GET/PUT | /admin/users/:id/permissions | UsersEdit |
| GET/POST/PUT/DELETE | /admin/roles[/:id] | RolesManage |
| GET | /admin/permission-groups | RolesManage |
| GET/PUT | /admin/users/:id/db-access | RolesManage |
| GET/POST/DELETE | /apikeys[/:prefix] | APIKeysManage |
| GET/POST/PUT/DELETE | /schedules[/:id] | ConnectionsExec |

## Crypto (Vault)
| Methode | Pfad | Beschreibung |
|---------|------|--------------|
| GET | /crypto/keys | Keys listen |
| POST | /crypto/keys | Key anlegen |
| DELETE | /crypto/keys/:id | Key löschen |
| POST | /crypto/encrypt | Encrypt |
| POST | /crypto/decrypt | Decrypt |
| POST | /crypto/sign | Sign |
| POST | /crypto/verify | Verify |
| POST | /crypto/hash | Hash |
| POST | /crypto/keys/rotate | Rotation |
| GET | /crypto/algorithms | Algos |
| POST | /connections/:id/crypto/encrypt/:table/:column | Col-Encrypt |
| POST | /connections/:id/crypto/decrypt/:table/:column | Col-Decrypt |

## Permissions (Client-Verwendung)
- `PermConnectionsList/Create/Delete/Query/Exec`
- `PermUsersList/Create/Edit/Delete`
- `PermRolesManage`, `PermTrafficView`, `PermSettingsRead/Write`,
  `PermAPIKeysManage`
- Client zeigt Menüs basierend auf Role-Permissions (via /auth/verify Role).
