# go-database — Kryptographie-Anleitung (Crypto Manual)

**Stand:** 2026-07-21. Dieses Dokument ist die **Bedienungsanleitung** für die
Krypto-API. Ein Werkzeug ohne Anleitung ist keins — also: wie, wann, wofür.

Alle Endpoints liegen unter `/api/v1/crypto` und benötigen Auth (JWT/API-Key)
sowie die Permission `connections:exec`.

---

## 1. Welcher Algorithmus wann? (Entscheidungshilfe)

| Ziel | Algorithmus | Warum |
|------|-------------|------|
| **Datenfeld verschlüsseln** (empfohlen) | `aes-256-gcm` | Schnell + authentifiziert (Tamper-Erkennung) |
| Mobile/ARM ohne AES-HW | `chacha20-poly1305` | Genauso sicher, keine AES-Beschleunigung nötig |
| **Spalte in DB verschlüsseln** | `aes-256-gcm` | Via `/crypto/encrypt/:table/:column` |
| Legacy-Kompatibilität | `aes-256-cbc` | Encrypt-then-MAC (HMAC-SHA256) — nur wenn nötig |
| Schlüsselaustausch / Transport | `x25519-aes-256-gcm` | ECDH + AEAD, kein statischer Key |
| An jemanden senden (Public Key) | `rsa-oaep-4096` | Nur mit Public Key verschlüsseln |
| **Passwörter hashen** (neu) | `argon2id` | OWASP-Standard, memory-hard (JtR-resistent) |
| **Unterschreiben** (Integrität) | `ed25519` oder `ecdsa-p256` | Signatur, nicht verschlüsseln! |
| **Prüfsumme / Digest** | `sha256` / `sha512` / `blake2b` / `sha3-256` | Integrität, kein Geheimnis |

> **Verschlüsselung ≠ Signatur ≠ Hash.** Das ist die häufigste Verwechslung:
> - **Encrypt**: nur Berechtigte können lesen.
> - **Sign**: jeder kann lesen, aber jeder kann prüfen, *wer* es geschrieben hat (und dass es unverändert ist).
> - **Hash**: Einweg-Prüfsumme, kein Geheimnis.

---

## 2. Endpoints (Übersicht)

| Methode | Pfad | Zweck |
|---------|------|-------|
| GET | `/crypto/algorithms` | Liste aller Algos + Metadaten |
| POST | `/crypto/keys` | Key erzeugen |
| GET | `/crypto/keys` | Eigene Keys auflisten |
| DELETE | `/crypto/keys/:id` | Key löschen |
| POST | `/crypto/keys/rotate` | Neuen Key (Rotation) erzeugen |
| POST | `/crypto/encrypt` | Text verschlüsseln |
| POST | `/crypto/decrypt` | Text entschlüsseln |
| POST | `/crypto/sign` | Daten signieren (ed25519/ecdsa) |
| POST | `/crypto/verify` | Signatur verifizieren |
| POST | `/crypto/hash` | Digest berechnen |
| POST | `/connections/:id/crypto/encrypt/:table/:column` | Spalte verschlüsseln |
| POST | `/connections/:id/crypto/decrypt/:table/:column` | Spalte entschlüsseln |

---

## 3. Beispiele (curl)

### Algorithmen entdecken
```bash
curl http://localhost:8080/api/v1/crypto/algorithms \
  -H "Authorization: Bearer $TOK"
```

### Key erzeugen (AES-256-GCM)
```bash
KEY=$(curl -s -X POST http://localhost:8080/api/v1/crypto/keys \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"algorithm":"aes-256-gcm","purpose":"data-encryption"}')
echo "$KEY"
# -> {"id":"k_abc...","algorithm":"aes-256-gcm", ...}  (PrivKey NICHT im Response)
```

### Text verschlüsseln
```bash
curl -s -X POST http://localhost:8080/api/v1/crypto/encrypt \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"key_id":"k_abc...","plaintext":"Geheimnis 123"}'
# -> {"ciphertext":"...","nonce":"...","tag":"...","algorithm":"aes-256-gcm","key_id":"k_abc..."}
```

### Text entschlüsseln
```bash
curl -s -X POST http://localhost:8080/api/v1/crypto/decrypt \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"key_id":"k_abc...","ciphertext":"...","nonce":"...","tag":"...","algorithm":"aes-256-gcm"}'
# -> {"plaintext":"Geheimnis 123"}
```

### Signieren & Verifizieren (ed25519)
```bash
# 1. Key für Signaturen erzeugen
SIGKEY=$(curl -s -X POST http://localhost:8080/api/v1/crypto/keys \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"algorithm":"ed25519","purpose":"signing"}')

# 2. Signieren
curl -s -X POST http://localhost:8080/api/v1/crypto/sign \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d "{\"key_id\":\"$(echo $SIGKEY | jq -r .id)\",\"data\":\"wichtige Transaktion #42\"}"
# -> {"key_id":"...","algorithm":"ed25519","signature":"base64..."}

# 3. Verifizieren (z.B. später durch Dritten)
curl -s -X POST http://localhost:8080/api/v1/crypto/verify \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"key_id":"...","data":"wichtige Transaktion #42","signature":"base64..."}'
# -> {"valid":true}
```

### Hash berechnen
```bash
curl -s -X POST http://localhost:8080/api/v1/crypto/hash \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"algorithm":"sha256","data":"hallo welt"}'
# -> {"algorithm":"sha256","digest":"09ca... (hex)"}
```

### Spalte in Datenbank verschlüsseln (Column-Level Encryption)
```bash
# Verschlüsselt ALLE Werte in users.email mit Key k_abc
curl -s -X POST http://localhost:8080/api/v1/connections/conn1/crypto/encrypt/users/email \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"key_id":"k_abc..."}'
# -> {"rows_affected": 150}

# Wieder lesbar machen:
curl -s -X POST http://localhost:8080/api/v1/connections/conn1/crypto/decrypt/users/email \
  -H "Authorization: Bearer $TOK" -H "Content-Type: application/json" \
  -d '{"key_id":"k_abc..."}'
```
> Hinweis: Verschlüsselte Spaltenwerte werden als JSON-Envelope
> `{"ct":"...","n":"...","t":"...","a":"...","k":"..."}` in der DB gespeichert.
> Suchanfragen (`WHERE email = '...'`) funktionieren dann NICHT mehr direkt —
> das ist der Trade-off von Application-Level Encryption.

---

## 4. Wo wird Verschlüsselung angewendet?

```
                 ┌──────────────────────────────────────────┐
App/Client ──────►  go-database API  (/crypto/*)            │
                 │   Service → Engine → Crypter(Algo)      │
                 │   KeyStore (Keys pro User, isoliert)    │
                 └──────────────────────────────────────────┘
                          │
        ┌─────────────────┴─────────────────┐
        ▼                                    ▼
  A) App-Level Encryption              B) DB-Spalte verschlüsseln
  (einzelne Werte/Text)                (/crypto/encrypt/:table/:column)
        │                                    │
        ▼                                    ▼
  Klartext NUR bei Abruf mit Key       Verschlüsselt in der DB liegen
  sichtbar                              (z.B. PII, Kreditkarten, Tokens)
```

**Typische Anwendungsfälle:**
- **PII schützen** (E-Mail, Telefon, Adresse) → Spalten-Verschlüsselung.
- **API-Tokens / Secrets** in der DB → aes-256-gcm, niemals Klartext.
- **Audit-Logs fälschungssicher** → ed25519-Signaturen über Log-Einträge.
- **Downloads/Exports** → Hash (sha256) mitschicken zur Integritätsprüfung.
- **Passwörter** → argon2id (nicht bcrypt, nicht sha256!).

---

## 5. Threat Model & Zero-Trust

**Was wir garantieren:**
- ✅ Starke, geprüfte Algorithmen (AES-GCM, ChaCha20-Poly1305, Argon2id, ed25519).
- ✅ Keys werden **pro User isoliert** gespeichert; ein User kann nicht die Keys
  eines anderen nutzen (Server-seitiger Check, nicht nur Client-Vertrauen).
- ✅ `PrivKey` wird **nie** im JSON-Response serialisiert.
- ✅ Nonces/Zufallswerte kommen aus `crypto/rand` (nicht math/rand).
- ✅ AEAD-Algorithmen erkennen Tampering (Manipulation) beim Decrypt.

**Was wir (noch) NICHT tun — und du beachten musst:**
- ⚠️ **Key-at-Rest**: Der KeyStore liegt in `auth.db`. Wenn jemand `auth.db`
  + den Server-Zugriff hat, kann er decrypten. → `auth.db` schützen
  (Dateiberechtigungen, ggf. eigener Verschlüsselungs-Key via KMS).
- ⚠️ **Transport**: API läuft über HTTP. Im Produktivbetrieb **TLS terminieren**
  (Reverse-Proxy / Load-Balancer), sonst fliegen Keys/Token im Klartext.
- ⚠️ **Client ist nicht vertrauenswürdig** (Zero-Trust): Der Server nimmt
  `algorithm` aus dem Request. Ein Client könnte `aes-256-cbc` statt `gcm`
  fordern (schwächer). Für Hochsicherheit: erlaubte Algorithmen serverseitig
  einschränken (TODO: Allowlist in Config).
- ⚠️ **Key-Rotation**: Bei Kompromittierung musst du alte Daten manuell
  re-encrypten. Ein automatisiertes Rotation+Re-Encrypt ist geplant (RISKS.md).

**Don't trust the client — Prinzipien:**
1. Auth immer serverseitig prüfen (JWT-Signatur, nicht nur `user_id` aus Body).
2. Permissions serverseitig (RBAC), nicht client-seit.
3. Keys gehören zum User — serverseitiger Ownership-Check (✅ implementiert).

---

## 6. John the Ripper (aus Spaß / Security-Testing)

JtR knackt **Passwort-Hashes**, keine AES-/ChaCha-Dateninhalte.

- **Was JtR kann:** bcrypt-/argon2-Hashes von *eigenen* Accounts angreifen.
- **Was JtR NICHT kann:** symmetrisch verschlüsselte Daten (kein Hash, sondern
  zufälliger Key) — dafür bräuchte es den Key, nicht brute-force.
- **Ethisch:** Nur auf **EIGENEN** Hashes testen. Fremde Hashes knacken = illegal.
- **Argon2id vs bcrypt:** argon2id ist memory-hard → JtR braucht viel RAM und
  ist deutlich langsamer als bei bcrypt. Deshalb ist argon2id der bessere
  Standard für neue Passwörter. (Bestehende bcrypt-Hashes im System bleiben
  sicher, sind aber schwächer als argon2id.)

**Test-Szenario (eigenes Passwort):**
```bash
# 1. User mit Passwort anlegen, bcrypt-Hash aus auth.db extrahieren (eigene DB!)
# 2. john --format=bcrypt hash.txt
# 3. Vergleich: gleiches Passwort mit argon2id → john --format=argon2 ...
#    → argon2id dauert spürbar länger (Demo für "warum memory-hard").
```

---

## 7. Fehlend / Roadmap

- 📋 **Envelope Encryption** (DEK wrapping mit KEK) für KMS-Anbindung.
- 📋 **KMS-Integration** (HashiCorp Vault / AWS KMS) statt lokalem KeyStore.
- 📋 **Key-Rotation + automatisches Re-Encrypt** bestehender Daten.
- 📋 **Algorithm-Allowlist** in Config (Zero-Trust-Härtung).
- 📋 **TLS native** im Server (aktuell extern via Proxy).
- 📋 **HSM-Support** für Enterprise/Banken.
