#!/usr/bin/env python3
"""go-database API Test — ALLE auto-provisionierten DB-Typen testen.
   Starte vorher: bin/go-database.exe
   Testet PostgreSQL, MySQL, MariaDB, MongoDB, Redis (wenn online)."""

import json, os, sys, time

try:
    import requests
except ImportError:
    print("pip install requests"); sys.exit(1)

BASE = "http://localhost:8080/api/v1"
PAD = 50

OK  = "[OK]"
FAIL = "[!!]"

def ok(msg, body=None):
    print(f"  {OK} {msg.ljust(PAD)}", end="")
    if body is not None:
        s = json.dumps(body, indent=2, ensure_ascii=False) if isinstance(body, (dict, list)) else str(body)
        print(s[:140].replace("\n", " "))
    else:
        print()

def fail(msg, detail=""):
    print(f"  {FAIL} {msg.ljust(PAD)} {detail}")

def expect(r, code=200):
    if r.status_code != code:
        raise RuntimeError(f"Status {r.status_code} != {code}: {r.text[:200]}")
    return r.json()

# ---------------------------------------------------
print("\n" + "="*65)
print("  go-database API-Test  ALLE DB-Typen")
print("  (auto-provisionierte PostgreSQL, MySQL, MariaDB, MongoDB, Redis)")
print("="*65)

# 1) Server check
try:
    r = requests.get("http://localhost:8080/health", timeout=5)
    ok("Server erreichbar", r.json())
except Exception as e:
    fail("Server nicht erreichbar", "Starte: .\\bin\\go-database.exe"); sys.exit(1)

# 2) Login
r = requests.post(f"{BASE}/auth/login", json={"username":"admin","password":"admin"}, timeout=5)
data = expect(r)
if not data.get("success"):
    fail("Login"); sys.exit(1)
token = data["data"]["token"]
H = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
ok("Login als admin", {"token": token[:30]+"..."})

# 3) Connections auflisten
r = requests.get(f"{BASE}/connections", headers=H, timeout=5)
all_conns = r.json().get("data", [])
ok(f"Verbindungen: {len(all_conns)}", [c["name"] for c in all_conns])

if not all_conns:
    fail("Keine Verbindungen vorhanden", "Provisioner hat nichts gestartet")
    sys.exit(1)

# 4) Jede Verbindung testen
tests_run = 0
tests_ok  = 0
tests_fail = 0

for conn in all_conns:
    cid  = conn["id"]
    name = conn["name"]
    typ  = conn["type"]
    state = conn.get("state", "")
    print(f"\n  --- {typ.upper()}  ({name}) [{state}] ---")

    if state != "connected":
        fail("  Status nicht connected")
        tests_fail += 1
        tests_run += 1
        continue

    try:
        # a) Ping
        r = requests.get(f"{BASE}/connections/{cid}/ping", headers=H, timeout=10)
        expect(r)
        ok("  Ping")

        # b) DB-spezifische CREATE TABLE
        if typ == "postgres":
            create_sql = "CREATE TABLE IF NOT EXISTS api_test (id SERIAL PRIMARY KEY, label TEXT, val REAL, ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
        else:
            create_sql = "CREATE TABLE IF NOT EXISTS api_test (id INTEGER PRIMARY KEY AUTO_INCREMENT, label TEXT, val REAL, ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
        r = requests.post(f"{BASE}/connections/{cid}/execute", headers=H, timeout=10, json={"query": create_sql})
        expect(r)
        ok("  CREATE TABLE api_test")

        # c) Zeilen einfuegen (ohne id, DB generiert automatisch)
        rows_data = [
            {"label": "Alpha", "val": 1.1},
            {"label": "Beta",  "val": 2.2},
            {"label": "Gamma", "val": 3.3},
        ]
        for row in rows_data:
            r = requests.post(f"{BASE}/connections/{cid}/row/api_test", headers=H, timeout=10, json=row)
            expect(r, 201)
        ok("  3 Zeilen eingefuegt")

        # d) Query
        r = requests.post(f"{BASE}/connections/{cid}/query", headers=H, timeout=10, json={
            "query": "SELECT * FROM api_test ORDER BY id"
        })
        data = expect(r).get("data", {})
        cols = data.get("columns", [])
        rows = data.get("rows", [])
        ok(f"  Query: {len(rows)} Zeilen", f"Spalten: {cols}")
        for row in rows:
            print(f"         {row}")

        # e) Browse
        r = requests.get(f"{BASE}/connections/{cid}/browse/api_test", headers=H, timeout=10)
        br = expect(r).get("data", {})
        ok(f"  Browse: {br.get('total', 0)} total")

        # f) Schema
        r = requests.get(f"{BASE}/connections/{cid}/schema", headers=H, timeout=10)
        ok("  Schema")

        # g) Suggest
        r = requests.post(f"{BASE}/suggest", headers=H, timeout=10, json={
            "connection_id": cid, "input": "SELECT * FROM api", "current_table": ""
        })
        ok(f"  Suggest: {len(r.json().get('data',[]))} Treffer")

        # h) Tabelle droppen
        r = requests.post(f"{BASE}/connections/{cid}/execute", headers=H, timeout=10, json={
            "query": "DROP TABLE IF EXISTS api_test"
        })
        expect(r)
        ok("  DROP TABLE api_test")

        tests_ok += 1

    except Exception as e:
        fail(f"  Fehler", str(e))
        tests_fail += 1

    tests_run += 1

# Zusammenfassung
print("\n" + "="*65)
print(f"  {tests_run} DBs getestet: {tests_ok} OK, {tests_fail} Fehler")
print("="*65 + "\n")

if tests_fail > 0:
    sys.exit(1)
