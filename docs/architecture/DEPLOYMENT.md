# Deployment & Scaling

go-database is API-only (no embedded UI). This covers Docker deployment,
local LLM offload, and the hard limits when scaling to multiple containers.

## Docker (single instance)

```bash
# Local .gguf models via llama-server auto-started in container
GODB_MCP_PROVIDER=llamacpp docker compose up

# Cloud model (OpenRouter) — no GPU needed
docker compose up

# Sample DBs (postgres/mysql/mariadb/mongo/redis)
docker compose --profile samples up
```

The image builds `llama-server` from **llama.cpp `master`** (not a pinned
tag) so recent model architectures (qwen3.5 / ornith) load correctly. Verified:
`model loaded` + `/health → {"status":"ok"}` for Ornith-1.0-9B Q4_K_M.

### Volume mount gotcha (Windows / Docker Desktop)
The `models/` mount needs the **Windows path**, not an MSYS path:
```bash
docker run -v "H:\Projekt\Programmieren\go_database\models:/app/models:ro" ...   # ✓
docker run -v "$(pwd)/models:/app/models:ro" ...                                  # ✗ empty volume, model not found
```

## Local LLM Offload (RAM ↔ VRAM)

`n_gpu_layers` (llamacpp) controls distribution:
- `0` → all CPU/RAM
- `99` (or `-1`) → all layers on GPU/VRAM
- `20/40/60` → partial offload (large models on small GPU)

```bash
GODB_MCP_LLAMACPP_NGPU=99 docker compose up   # GPU offload (needs NVIDIA Container Toolkit)
```

CPU offload works out of the box. VRAM offload needs the GPU `deploy` block
uncommented in `docker-compose.yml` + NVIDIA Container Toolkit (`--gpus all`).

## Scaling — hard limits

### ⚠️ SQLite is single-writer
`internal-db` volume holds `auth.db` + agent memory (SQLite). **Do NOT share
it across multiple API replicas.** With `--scale api=3` or `deploy.replicas>1`
all pods write the same SQLite file → lock contention / corruption.

**To scale out:**
- **Option A (recommended):** one API instance per `internal-db` volume
  (separate named volumes, no shared state), or
- **Option B:** external Postgres for auth (`GODB_AUTH_*` env) so SQLite is
  not the bottleneck.

Solo / small-group use: single instance, `replicas: 1`. This is fine.

### Agent memory cross-container safety
`agent_memory.json` is protected by an **OS-level `flock`** on Linux (see
`internal/agent/memory_lock_linux.go`), so multiple containers mounting the
*同一* file serialize writes. On non-Linux (dev/Windows) only the per-process
`sync.Mutex` applies — multi-container there is unsupported (use Linux
containers, the deploy target). Still: prefer separate volumes per replica.

### Multi-tenant isolation
- `db_access` scope is enforced per-request in Agent/MCP (GuardGate.WithScope).
  A non-admin caller only sees connections whose ID is in their `db_access`.
- SQL-Guard parses **all** statements (splits on `;`, ignores `;` in string
  literals) and classifies by highest-risk command — `SELECT 1; DROP TABLE
  users;` is blocked unless the caller has exec permission.

## Pre-flight check
```bash
# cookbook recipe: is the environment start-ready?
recipe.Run("system_check", {"model": "models/Ornith-1.0-9B-GGUF/ornith-1.0-9b-Q4_K_M.gguf"})
# → docker, llama_server, agent_model, database_sqlite, database_provisioner
```
