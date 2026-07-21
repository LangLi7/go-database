# Konzept: Multi-Agent Council / Collaboration

_Status:_ Brainstorming — nicht gebaut. Trigger: User will, dass mehrere
Agenten (lokal + API-Provider) als "Council" mit Struktur/Ziel zusammenarbeiten.
Verwandt: `multi-user-parallel.md` (Durchsatz), `agent-api-awareness.md` (Tool-Sicht).

## Ausgangslage (Architektur heute)
- `internal/agent/handler.go`: ein globaler `agent`-Singleton, ein LLM-Client,
  eine Session-Map. Requests sind **unabhängig**, kein Agent-zu-Agent-Kontakt.
- Lokales Parallel (`mcp.llamacpp.parallel: N`) = N Slots in EINEM llama-server
  → N gleichzeitige Aufrufe derselben Agent-Logik. **Kein Council**, nur
  Durchsatz.
- Mehrere Provider = mehrere go-database-Instanzen (Container) mit je eigenem
  `provider`. Teilen aber keinen State/Ziel.

## Was "Council" bedeutet
Koordinator-Agent zerlegt Task → spezialisierte Agenten bearbeiten Sub-Tasks
→ Council aggregiert/Einigt sich. Bekannt aus AutoGen / CrewAI / LangGraph.

### Varianten
1. **Provider-Failover** (einfach): lokales Modell antwortet nicht →
   API-Provider übernimmt. Teilweise schon da (`fallback_paid`).
2. **Spezialisten-Council** (komplex): Modell A schreibt SQL, Modell B prüft
   gegen Guard, Modell C erklärt Ergebnis. Braucht Orchestrierung + geteilten
   State.
3. **Parallel-Ensemble** (mittel): gleiche Frage an N Modelle, Mehrheit/Review
   entscheidet. Gut für kritische SQL (Fehlervermeidung), teuer.

## Docker-Scaling + Port-Identifikation (das genannte Issue)
- N go-database-Instanzen in Compose: Service-Naming löst "wer ist wer"
  (`godb-agent-local`, `godb-agent-deepseek`), aber Cross-Container-Koordination
  braucht:
  - Service Discovery (Docker DNS: `godb-agent-local:8080`) ODER
  - API-Gateway/Router davor (entscheidet, welcher Agent die Anfrage kriegt) ODER
  - Shared State (Redis Pub/Sub) für Agent-zu-Agent-Kommunikation.
- **Problem:** N volle go-database-Instanzen nur fürs Council ist schwer.
  Besser: 1 Koordinator-Instanz + N leichtgewichtige Modell-Endpoints
  (llama-server `--parallel`, oder OpenRouter-Modell-Routing).

## Empfehlung (lazy)
- **Jetzt:** Einzel-Agent + Tool-Use reicht für DB-Verwaltung. Council = YAGNI.
- **Wenn doch:** zuerst Variante 1 (Failover) —已有 `fallback_paid`.
- **Echtes Council:** erst wenn konkreter Spezialisten-Use-Case da. Dann:
  - Koordinator als neuer Agent-Modus (`agent.go` erweitern, nicht neu bauen)
  - Sub-Agenten als LLM-Clients mit unterschiedlichem `provider`/`model`
  - Geteilter State über `internaldb` oder Redis (kein neues System)

## Offene Fragen
- Wer entscheidet im Council bei Dissens? (Mehrheit? Gewichtung? Mensch im Loop?)
- Kosten/Latency akzeptabel bei Ensemble (N-fache Token)?
- Braucht der Koordinator eigene Guard-Regeln (darf er DDL delegieren)?
