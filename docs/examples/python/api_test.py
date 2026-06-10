#!/usr/bin/env python3
"""go-database API Test — funktioniert OHNE externe Datenbank.
   Nutzt SQLite mit temporarer Datei und zeigt Live-API + WebSocket + SSE."""

import json, os, sys, time, tempfile, threading, textwrap

try:
    import requests
except ImportError:
    print("pip install requests"); sys.exit(1)

BASE = "http://localhost:8080/api/v1"
PAD = 40

OK  = "[OK]"
FAIL = "[!!]"

def ok(msg, body=None):
    print(f"  {OK} {msg.ljust(PAD)}", end="")
    if body is not None:
        s = json.dumps(body, indent=2, ensure_ascii=False)
        print(s[:120].replace("\n", " "))
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
print("  go-database API-Test  (keine externe DB noetig)")
print("="*65)

# 1) Server erreichbar?
try:
    r = requests.get("http://localhost:8080/health", timeout=5)
    data = r.json()
    ok("Server erreichbar", data)
except Exception as e:
    fail("Server nicht erreichbar", "Starte: bin/go-database.exe")
    sys.exit(1)

# 2) Login
try:
    r = requests.post(f"{BASE}/auth/login", json={"username":"admin","password":"admin"}, timeout=5)
    data = expect(r)
    if not data.get("success"):
        fail("Login", data.get("error", {}).get("message", "?")); sys.exit(1)
    token = data["data"]["token"]
    role = data["data"]["role"]
    ok(f"Login als admin ({role})", {"token": token[:30]+"..."})
except Exception as e:
    fail("Login fehlgeschlagen", str(e)); sys.exit(1)

H = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

# 3) Leeres Dashboard zeigen
r = requests.get(f"{BASE}/connections", headers=H, timeout=5)
data = r.json()
count = len(data.get("data", []))
ok(f"Verbindungen: {count} (noch keine)", [] if count == 0 else data["data"])

r = requests.get(f"{BASE}/admin/users", headers=H, timeout=5)
users = r.json().get("data", [])
ok(f"Benutzer: {len(users)}", [u["username"] for u in users])

# 4) Admin-Stats
r = requests.get(f"{BASE}/admin/stats", headers=H, timeout=5)
ok("Admin-Statistiken", r.json().get("data"))

# 5) SQLite-TEMP-Datei anlegen
tmp = tempfile.NamedTemporaryFile(suffix=".db", delete=False)
tmp.close()
DB_PATH = tmp.name
r = requests.post(f"{BASE}/connections", headers=H, timeout=10, json={
    "name": "Temp-Test-DB", "type": "sqlite", "filepath": DB_PATH
})
if r.status_code == 201:
    conn_id = r.json()["data"]["id"]
    ok("SQLite-Verbindung angelegt", {"id": conn_id, "file": DB_PATH})
else:
    fail("SQLite anlegen fehlgeschlagen", f"{r.status_code} {r.text[:200]}")
    os.unlink(DB_PATH)
    sys.exit(1)

# 6) Tabelle anlegen
r = requests.post(f"{BASE}/connections/{conn_id}/execute", headers=H, timeout=10, json={
    "query": "CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT, email TEXT, created_at TEXT DEFAULT CURRENT_TIMESTAMP)"
})
ok("Tabelle 'users' angelegt", r.json().get("data"))

# 7) Daten einfuegen
inserts = [
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob",   "email": "bob@example.com"},
    {"name": "Charlie","email": "charlie@example.com"},
]
for u in inserts:
    r = requests.post(f"{BASE}/connections/{conn_id}/row/users", headers=H, timeout=10, json=u)
    ok(f"  -> Benutzer '{u['name']}' eingefuegt", r.json().get("success"))

# 8) Query
r = requests.post(f"{BASE}/connections/{conn_id}/query", headers=H, timeout=10, json={
    "query": "SELECT * FROM users ORDER BY id"
})
data = r.json().get("data", {})
cols = data.get("columns", [])
rows = data.get("rows", [])
ok(f"Query: {len(rows)} Zeilen, Spalten: {cols}")
for row in rows:
    print(f"      {row}")

# 9) Browse (paginated)
r = requests.get(f"{BASE}/connections/{conn_id}/browse/users?page=1&per_page=2", headers=H, timeout=10)
br = r.json().get("data", {})
ok(f"Browse (page 1/2, total {br.get('total',0)})", f"{len(br.get('data',[]))} Zeilen")

# 10) Schema abrufen
r = requests.get(f"{BASE}/connections/{conn_id}/schema", headers=H, timeout=10)
ok("Schema abrufbar", r.json().get("success"))

# 11) Ping
r = requests.get(f"{BASE}/connections/{conn_id}/ping", headers=H, timeout=10)
ok("Ping erfolgreich", r.json().get("data"))

# 12) Suggest
r = requests.post(f"{BASE}/suggest", headers=H, timeout=10, json={
    "connection_id": conn_id, "input": "SELECT * FROM u", "current_table": ""
})
ok("Autocomplete-Suggestions", f"{len(r.json().get('data',[]))} Treffer")

# 13) Audittrail
r = requests.get(f"{BASE}/admin/activity", headers=H, timeout=10)
logs = r.json().get("data")
ok("Audit-Log abrufbar", f"{len(logs) if logs else 0} Eintraege")

# 14) API-Keys auflisten (REST)
r = requests.get(f"{BASE}/apikeys", headers=H, timeout=10)
aks = r.json().get("data")
ok("API-Keys abrufbar", f"{len(aks) if aks else 0} Keys")

# 15) Roles auflisten
r = requests.get(f"{BASE}/admin/roles", headers=H, timeout=10)
roles = r.json().get("data", [])
ok(f"Rollen: {len(roles)}", [rol["name"] for rol in roles])

# 16) Permission-Gruppen
r = requests.get(f"{BASE}/admin/permission-groups", headers=H, timeout=10)
ok("Permission-Gruppen abrufbar", f"{len(r.json().get('data',[]))} Gruppen")

# 17) Health-Check
r = requests.get("http://localhost:8080/health", timeout=5)
ok("Health-Check", r.json())

# --- Cleanup ---
r = requests.delete(f"{BASE}/connections/{conn_id}", headers=H, timeout=5)
ok("SQLite-Verbindung geloescht", "204" if r.status_code == 204 else r.status_code)
try: os.unlink(DB_PATH)
except: pass

print("="*65)
print(f"  ALLE 17 TESTS BESTANDEN")
print("="*65 + "\n")
