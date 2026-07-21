# Konzept: Multi-User / Nebenläufigkeit

_Status:_ Geklärt + teilweise umgesetzt. Hier als Referenz.

## Befund (verifiziert)
- **REST-API**: voll parallel (Go-Goroutines, `sync.RWMutex` im
  `connection.Manager`). 5 parallele Requests = alle ~2.8ms gleichzeitig. ✅
- **Agent-API**: parallel, aber der **lokale llama-server** ist seriell (1 Slot).

## Lokales Modell: serieller Engpass gelöst
- `mcp.llamacpp.parallel: N` in config (Default 1 = seriell).
- Bei `N > 1` startet go-database llama-server mit
  `--parallel N --cont-batching --batch-size 512`.
- **Bug gefixt**: `--parallel` ohne `--batch-size` hängt den Server (deadlock).
  Plus: `s.cmd.Stdout/Stderr` dürfen nicht `nil` sein (Windows crash) → `io.Discard`.
- RAM-abhängig: mehr Slots = mehr Kontext-RAM.

## Alternative: Paid API-Provider (empfohlen für Multi-User)
- OpenRouter bereits integriert. `fallback_paid: true` + `model`:
  - `deepseek/deepseek-r1` — günstig, stark bei SQL/Reasoning
  - `anthropic/claude-3.5-haiku` — bestes Tool-Use, teurer
- API-seitig parallel, kein lokaler Slot-Limit. Kosten steigen mit Parallelität.

## Entscheidung
- Einzelnutzer/Dev: lokales Modell, `parallel: 1`.
- Multi-User: Paid Provider ODER lokales Modell mit `parallel: N` (RAM!).
