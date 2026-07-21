# Konzept: Cookbook als berechenbares Werkzeug (Recipes API)

_Status:_ Brainstorming — nicht gebaut. Trigger: User will "Cookbook-Feature
zum Berechnen" als API, die auch **extern** ladbar/aufrufbar ist (Werkzeug-Hilfe).

## Was existiert (Wiederverwendung!)
- `POST /api/v1/templates/apply`, `GET /api/v1/templates` (`routes.go:274-275`)
  → **verdrahtet, aber Handler leer** (Scaffolding in `internal/templates/`).
- `GET /api/v1/models/local` + `/models/remote` (`models.go`) → listen nur Modelle,
  **kein Hardware-Scan**.
- `POST /api/v1/models/download` + `/models/start` (`templates.go:104,150`) →
  Download/Start vorhanden, aber kein Compatibility-Check davor.
- `internal/suggest/` — Risiko-Klassifizierung + SQL-Autocomplete. Logik-Vorbild.
- `LOCAL_MODELS.md` — statische RAM/Quant-Tabellen (von Hand), keine API.

## Idee A: Recipe = benannte Berechnungsvorschrift
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
Input: strukturierte Werte (model_gb, slots, ctx, system_ram). Output: Berechnung.

## Idee B: Hardware-Compatibility-Cookbook (PewDiePie/Odysseus-Stil)
User will: Hardware scannen → passendes Modell empfehlen → Settings berechnen
→ externen Downloader mitschicken. Massiv angelehnt an Odysseus/local-LLM-Finder.

### Bausteine
1. **Scan** (vorhanden? Nein — neu): RAM/CPU/GPU/VRAM auslesen
   - Go `runtime.NumCPU`, `runtime.MemStats` (RAM)
   - GPU/VRAM: `golang.org/x/sys` oder `nvml` (NVIDIA) — Windows: `wmic`/`powershell`
2. **Match**: welche `.gguf` passt? Größe vs. RAM/VRAM, Quant-Stufe wählen
   - nutzt `models/`-Verzeichnis + strukturierte RAM-Tabelle (aus `LOCAL_MODELS.md`)
3. **Recommend**: bestes Modell + Settings
   - `ctx_size`, `n_gpu_layers`, `parallel`, `batch_size` pro Hardware-Profil
4. **Download**: externer Downloader
   - `POST /models/download` nimmt URL + Name (schon da) → nur Scan/Recommend fehlt davor
5. **Ornith-spezifisch**: perfekte Settings pro Hardware-Profil als Recipe ablegen
   - z.B. Ornith-9B-Q4: "8GB RAM → ctx 4096, gpu-layers -1, parallel 1"

### Design (lazy, ~150 Z.)
- `internal/hardware` (neu, klein): `Scan() -> Spec{RAM, CPU, GPU, VRAM}`
- `internal/recipe` (neu, klein): Registry + `ram_slots_estimate`, `model_fit`
- `GET /api/v1/hardware` → Spec JSON
- `GET /api/v1/recipes/recommend?model=...` → Settings + Download-URL
- Rezept-Daten als `cookbook/*.json` (portabel, extern ladbar)

## Extern nutzbar (Werkzeug-Hilfe)
- Andere go-database-Instanzen oder externe Tools rufen `POST /recipes/:name`
- Rezept als portable JSON beschreibbar → kein Re-Code nötig
- Downloader: `POST /models/download` mit URL+Target → extern triggerbar

## Offene Fragen
- GPU-Erkennung: `nvml` (C-Binding) schwer auf Windows? Ersatz: `powershell`
  Get-CimInstance Win32_VideoController. Oder nur RAM/CPU scannen (VRAM schätzen).
- Reichen reine Go-Funktionen (Compute in Code) oder sollen Rezepte DSL/sandbox
  sein? → Erstereres zuerst (sicher).
- Braucht Recipe Zugriff auf System-RAM? (Go `runtime.MemStats` oder `exec`).

## Nächster Schritt
Erst EIN konkretes Recipe bauen (z.B. `ram_slots_estimate` + `model_fit`),
dann Hardware-Scan. Nicht das ganze Framework vorab.
