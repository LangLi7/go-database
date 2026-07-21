# Konzept-Sammlung go-database

Brainstorming, Ideen, Design-Skizzen. **Nichts hier ist Code** — nur
Gedankenstütze für spätere Features. Wenn ein Feature gebaut wird, wandert
der relevante Teil nach `docs/` (Implementierung) oder direkt in Code.

## Index
- `agent-api-awareness.md` — Agent soll die *komplette* DB/SQL-API verstehen (Guard-aware, DB-spezifisch, Schema-Change)
- `db-gateway-mittelsmann.md` — go-database als sicherer Proxy/Mittelsmann für externe DBs
- `multi-user-parallel.md` — Nebenläufigkeit / Multi-User (lokal vs. API-Provider)

---
*Regel: Konzepte bleiben hier, bis entschieden wird, sie umzusetzen. Dann
wandern sie raus (in docs/ oder Code) und die Datei wird gelöscht.*
