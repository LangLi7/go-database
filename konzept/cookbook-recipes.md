# Konzept: Cookbook als berechenbares Werkzeug (Recipes API)

_Status:_ Brainstorming — nicht gebaut. Trigger: User will "Cookbook-Feature
zum Berechnen" als API, die auch **extern** ladbar/aufrufbar ist (Werkzeug-Hilfe).

## Was existiert (Wiederverwendung!)
- `POST /api/v1/templates/apply`, `GET /api/v1/templates` (`routes.go:274-275`)
  → **verdrahtet, aber Handler leer** (Scaffolding in `internal/templates/`).
  Das ist das natürliche Zuhause für Recipes.
- `internal/suggest/` — Risiko-Klassifizierung + SQL-Autocomplete. Logik-Vorbild.
- `internal/guard/rbac.go` — Command-Detection. Wiederverwendbar für Recipe-Validierung.

## Idee: Recipe = benannte Berechnungsvorschrift
```
Recipe {
  name:        "ram_slots_estimate"
  description: "RAM-Bedarf für lokales Modell mit N Slots"
  input:       { model_gb: float, slots: int, ctx_size: int }
  compute:     func(input) -> { ram_gb: float, feasible: bool }
}
```
Aufruf: `POST /api/v1/recipes/{name}` mit JSON-Input → JSON-Output.
Extern: Rezepte als `.json` in `cookbook/` ladbar (wie `ApplyTemplate`).

## Konkreter Anwendungsfall (User genannt)
Lokales Parallel-Problem: "wie viel RAM braucht Modell X mit N Slots?"
→ Recipe `ram_slots_estimate`:
```
ram_gb = model_gb + slots * ctx_size/1000 * 0.5   # grobe Faustformel
feasible = ram_gb < system_ram * 0.8
```
Verhindert das `--parallel 4`-Overcommit-Desaster (3m Timeout) im Vorfeld.

## Design (lazy, ~50 Z. + Handler)
1. `internal/templates` (oder neues `internal/recipe`):
   - `type Recipe struct { Name, Description string; InputSchema json; Compute func(map[string]any) (map[string]any, error) }`
   - `registry := map[string]Recipe{}` + `Register(name, r)`
2. HTTP: `POST /api/v1/recipes/:name` → Input parsen → `Compute` → JSON out
3. Extern laden: `POST /api/v1/recipes/load` nimmt Recipe-JSON, registriert es
4. Built-in Recipes: `ram_slots_estimate`, ggf. `quant_ram` (Q4→RAM)

## Extern nutzbar (Werkzeug-Hilfe)
- Andere go-database-Instanzen oder externe Tools rufen `POST /recipes/:name`
- Rezept als portable JSON beschreibbar → kein Re-Code nötig

## Offene Fragen
- Reichen reine Go-Funktionen (Compute in Code) oder sollen Rezepte
  **DSL/sandbox** sein (User schreibt Formel)? → Erstereres zuerst (sicher).
- Braucht Recipe Zugriff auf System-RAM? (Go `runtime.MemStats` oder `exec`).

## Nächster Schritt
Erst EIN konkretes Recipe bauen (z.B. `ram_slots_estimate`), dann erweitern.
Nicht das ganze Framework vorab.
