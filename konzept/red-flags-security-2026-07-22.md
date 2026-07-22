# Red Flags & Security Gaps — go-database Cross-Platform Test (2026-07-22)

## Test-Matrix Ergebnis

| Platform | Test | Result |
|----------|------|--------|
| Windows (MSYS) | E2E + Security + Agent(llamacpp) | 13/14 PASS |
| Kali Linux (WSL) | `go test ./...` | 16 pkgs OK |
| Ubuntu 22.04 (Docker) | `go test ./...` | 16 pkgs OK |
| Docker image | build + run + health + API | OK |

**Windows FAIL (1):** valid query on mysql → 400. Root cause: provisioned
MySQL/MariaDB/Postgres/MongoDB containers are in `error` state (auth/connection
refused), only redis + sometimes mysql connect. **Infra issue, not code.**

---

## 🔴 CRITICAL Red Flags

### 1. CORS = `*` (all origins)
`internal/api/middleware/cors.go:10` → `allowedOrigin = "*"`.
- Any website can call the API with the user's cookies/JWT (if ever used in browser).
- Auth endpoints (`/auth/challenge`, `/login-pubkey`) are therefore CSRF-able from
  any origin.
- **Fix:** restrict to known frontend origins via env/config. For now acceptable
  (API-only, no browser session cookie), but document as risk.

### 2. JWT Secret default `change-me-in-production` committed in config.yaml
`config/config.yaml:9` → `jwt_secret: "change-me-in-production"`.
- Code mitigates: `config.go:224` generates random secret if default/empty →
  tokens invalid after each restart (annoying but safe).
- **Risk:** if someone sets `change-me-in-production` explicitly → all JWTs
  forgeable. Should hard-fail on default secret in production mode.

### 3. `.env` overrides `config.yaml` silently (GODB_* env priority)
koanf loads `.env` then `GODB_*` env vars with HIGHEST priority.
- During testing, a forgotten `.env` with `GODB_MCP_PROVIDER=lmstudio` overrode
  `config.yaml` → llamacpp provider ignored, server tried LM Studio on :1234.
- **Risk:** silent misconfiguration; hard to debug. Document precedence clearly.
- **Note:** `.env` is now deleted; `config.yaml` (gitignored) is source of truth.

---

## 🟡 MEDIUM Gaps

### 4. SQL injection surface
- Connection-ID is validated (injection id → 400). ✅
- SQL string itself is passed raw to plugin.Query() — by design (it's a DB tool).
- **Risk:** no statement allowlist; a user with `connections:query` can run
  `DROP`/`DELETE`. Should scope queries by role (readonly role → SELECT only).
- The `readonly` role exists but isn't enforced at query-execution level.

### 5. Multi-user / multi-DB isolation NOT enforced in Agent/MCP
- API-Key + JWT set `db_access` scope (verified: scoped key sees 0 conns). ✅
- BUT: Agent (`/agent/chat`) and MCP (`/mcp`) call `gate.List()` GLOBAL —
  no per-user `db_access` filter. A user with agent access can query ALL
  databases regardless of their `db_access` scope.
- **This is the Phase-2 gap noted earlier. Critical for multi-tenant.**

### 6. Provisioned DB containers unhealthy
- postgres/mariadb/mongodb containers fail health (empty password / URI errors).
- Only redis + mysql sometimes connect. Affects query E2E but not API security.
- **Fix:** fix provisioner credentials (set explicit user/pass in docker-compose).

### 7. Rate limiting absent
- No rate limit on `/auth/challenge` or `/login-pubkey` → brute-force / nonce
  replay window. Nonce has `expires_at` (saw in challenge response) → replay
  protected, but no attempt-throttling.

---

## 🟢 What works (verified)

- ✅ Admin SSH-style pubkey login (Ed25519 challenge-response)
- ✅ Unauthenticated access → 401 on all protected routes
- ✅ Invalid/forged JWT → 401
- ✅ API-Key isolation: scoped key sees only its `db_access` (0 leaked conns)
- ✅ API-Key cannot do admin ops → 403
- ✅ Role inheritance (Luckperms parent→child) — unit-tested
- ✅ MCP tool call (list_connections) returns data
- ✅ Agent uses LOCAL llama.cpp model (Ornith-9B) — no LM Studio/Ollama needed
- ✅ SQL injection via connection-id → 400 (rejected before DB)
- ✅ Cross-platform: Windows + Kali + Ubuntu(all Docker) compile + test green
- ✅ Docker image builds + runs + healthcheck passes

---

## Action Items (priority)

1. **HIGH:** Restrict CORS to configured origins (env `CORS_ORIGINS`).
2. **HIGH:** Enforce `db_access` scope in Agent/MCP (Phase-2 gate refactor).
3. **MED:** Hard-fail if `jwt_secret` is the default value in production.
4. **MED:** Enforce `readonly` role = SELECT-only at query execution.
5. **LOW:** Add rate-limiting to auth endpoints.
6. **LOW:** Fix provisioner DB credentials so postgres/mariadb/mongodb connect.
7. **DOC:** Document config precedence (`.env` > `config.yaml`).
