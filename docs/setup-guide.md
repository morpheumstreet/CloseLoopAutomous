# Setup guide — CloseLoopAutomous (arms + Fishtank)

This guide gets **arms** (HTTP API + SQLite) and **Fishtank** (browser UI) running on your machine. Optional pieces: **Redis**, **arms-worker** (background jobs), and **Docker**.

For API details after the server is up, see [api-ref.md](api-ref.md) and `GET /api/docs/routes` on a running instance.

---

## What you need

| Component | Purpose |
|-----------|---------|
| **Go 1.22+** (1.26+ matches `arms/go.mod`) | Build and run **arms** and **arms-worker** |
| **Bun** | Run **Fishtank** dev server and production build (`bun run dev` / `bun run build`) |
| **Redis** (optional) | Autopilot schedules, Asynq queues, stall auto-nudge — **required** if you use those features |
| **Docker** (optional) | Run arms + Redis from `arms/docker-compose.yml` |

**Fishtank tooling:** This project uses **Bun** for the UI (`fishtank/package.json` scripts). **Do not use npm or Node** for Fishtank here—use `bun install`, `bun run dev`, and `bun run build` only.

---

## Repository layout

- **`arms/`** — Go module: `cmd/arms` (API server), `cmd/arms-worker` (Asynq consumer)
- **`fishtank/`** — React UI; env vars follow the `VITE_*` convention; dev/build via **Bun** (not npm/Node)
- **`Makefile`** (repo root) — `make build` / `make run` for **arms** only

---

## 1. arms (API server)

### Build

From the **repository root**:

```bash
make build
```

Binary: `arms/bin/arms`.

Or from `arms/`:

```bash
cd arms
GOWORK=off CGO_ENABLED=0 go build -trimpath -o bin/arms ./cmd/arms
```

### Configuration

Settings are read from **environment variables** (see comments in `arms/internal/config/config.go`). You can also pass a **JSON or TOML** file; **environment always overrides** the file when a variable is set in the process environment.

```bash
# From repo root (create ./data first: mkdir -p data)
./arms/bin/arms -c config/arms.toml
```

The repo includes **`config/arms.toml`** — same shape as below (flat keys = env names). Copy to **`config/arms.local.toml`** for machine-specific secrets (that path is gitignored); environment variables still override either file.

Minimal excerpt:

```toml
ARMS_LISTEN = ":8080"
DATABASE_PATH = "./data/arms.db"
ARMS_CORS_ALLOW_ORIGIN = "http://localhost:5173"
```

Example **`config.json`**:

```json
{
  "ARMS_LISTEN": ":8080",
  "DATABASE_PATH": "./data/arms.db"
}
```

Override a file value at runtime:

```bash
ARMS_LISTEN=:9090 ./arms/bin/arms -c ./config.toml
```

### Minimal local run (SQLite on disk)

```bash
mkdir -p data
export DATABASE_PATH="$(pwd)/data/arms.db"
export ARMS_CORS_ALLOW_ORIGIN="http://localhost:5173"   # Fishtank dev origin; adjust if yours differs
./arms/bin/arms
```

- **`DATABASE_PATH` unset** → in-memory stores (data lost on restart).
- **`MC_API_TOKEN` unset** → HTTP API auth is off (dev-friendly). Set it for production.
- **`WEBHOOK_SECRET`** — required for HMAC webhooks when you use them.

Smoke checks:

```bash
curl -sS http://127.0.0.1:8080/api/health
curl -sS http://127.0.0.1:8080/api/version
```

### Docker (arms + optional Redis)

From **repo root**:

```bash
docker compose -f arms/docker-compose.yml up --build
```

Compose sets `DATABASE_PATH=/data/arms.db` and publishes **8080**. Uncomment `ARMS_REDIS_ADDR` (and related vars) in `arms/docker-compose.yml` when you need Redis-backed scheduling; then run **arms-worker** (below).

---

## 2. arms-worker (optional, Redis required)

When **`ARMS_REDIS_ADDR`** is set, background work (product schedules, autopilot ticks, stall auto-nudge, etc.) is driven by **Asynq**; **arms** enqueues tasks and **arms-worker** runs them.

Build:

```bash
cd arms
GOWORK=off CGO_ENABLED=0 go build -trimpath -o bin/arms-worker ./cmd/arms-worker
```

Run (same env / `-c` as arms so Redis and DB match):

```bash
export ARMS_REDIS_ADDR=127.0.0.1:6379
export DATABASE_PATH="$(pwd)/data/arms.db"
./bin/arms-worker -c ./config.toml
```

If `ARMS_REDIS_ADDR` is empty, the worker exits immediately (by design).

---

## 3. Fishtank (UI)

**Use Bun only** (`bun install` / `bun run …`). Do not use **npm** or **Node** for this tree.

Install dependencies and start the dev server:

```bash
cd fishtank
bun install
```

Create **`.env`** or **`.env.local`** in `fishtank/` (Vite/Bun convention — variables must be prefixed with `VITE_`):

```env
VITE_ARMS_URL=http://localhost:8080
# Optional — only if arms uses Bearer auth:
# VITE_ARMS_TOKEN=your-mc-api-token
```

If `VITE_ARMS_URL` is unset, the UI defaults to **`http://localhost:8080`** (same host and port as arms’ default `ARMS_LISTEN`).

Start:

```bash
bun run dev
```

Open the URL Bun prints (often `http://localhost:5173`). The dashboard loads products from `GET /api/products`.

**CORS:** arms must allow the UI origin via **`ARMS_CORS_ALLOW_ORIGIN`** (see arms config). Use `*` only for quick local experiments.

Production build:

```bash
bun run build
```

Serve the `fishtank/dist/` static files with any HTTP server; set `VITE_ARMS_*` at **build time** for the API URL the browser will call.

---

## 4. Typical dev workflow

1. Start **Redis** (if you use worker / schedules): e.g. `docker run -p 6379:6379 redis:7-alpine`
2. Start **arms** with `DATABASE_PATH`, `ARMS_CORS_ALLOW_ORIGIN`, and optionally `ARMS_REDIS_ADDR`
3. Start **arms-worker** if Redis is enabled
4. Start **Fishtank** with `VITE_ARMS_URL` pointing at arms

---

## 5. Troubleshooting

| Symptom | Things to check |
|--------|------------------|
| Fishtank “not connected” / fetch fails | arms running? `VITE_ARMS_URL` correct? **CORS** (`ARMS_CORS_ALLOW_ORIGIN`) includes the UI origin? |
| Autopilot / schedules never run | **`ARMS_REDIS_ADDR`** set on arms? **arms-worker** running? |
| `arms -c` errors | File extension must be **`.toml`** or **`.json`**; keys must match env var names |
| Secrets in repo | Prefer **environment** for tokens; file is fine for non-secrets. Remember **env overrides file** |

---

## 6. Further reading

- [api-ref.md](api-ref.md) — HTTP routes and behavior  
- [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — OpenAPI 3.1  
- [arms-production-hardening.md](arms-production-hardening.md) — hardening notes  
- [fishtank-ui-todos.md](fishtank-ui-todos.md) / [fishtank-ui-wiring-outstanding.md](fishtank-ui-wiring-outstanding.md) — UI roadmap  

---

_Last updated for `arms` `-c` config files, env override semantics, and Fishtank `VITE_ARMS_*`._
