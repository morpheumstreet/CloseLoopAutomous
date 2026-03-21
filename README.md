# CloseLoopAutomous

Self-iterating autonomous **mission / product control** backend—conceptually aligned with [Autensa / Mission Control](https://github.com/crshdn/mission-control), implemented as a **single Go service** with hexagonal architecture.

## What’s here (backend: `arms/`)

Most capability lives under **`arms/`** (not always obvious from a shallow GitHub tree view):

| Layer | Contents |
|--------|-----------|
| **HTTP API** | Products, ideas, Kanban tasks, convoys, costs, agents, merge queue, preference model, operations log, webhooks, SSE — see [`docs/api-ref.md`](docs/api-ref.md) |
| **Domain** | `Product`, `Idea`, `Task` (MC-style statuses), `Convoy`, costs, autopilot tier enums — `arms/internal/domain/` |
| **Persistence** | SQLite + versioned migrations through **016** (`arms/internal/adapters/sqlite/migrations/`), optional in-memory mode |
| **Execution** | OpenClaw WebSocket client (`arms/internal/adapters/gateway/openclaw/`), gateway stub for tests |
| **Realtime** | `GET /api/live/events` (SSE); **`event_outbox`** + relay when using SQLite; in-memory hub otherwise |
| **Shipping** | `PullRequestPublisher` — REST (`go-github` + `ARMS_GITHUB_TOKEN`) or **`gh pr create`** (`ARMS_GITHUB_PR_BACKEND=gh`) |

Parity and roadmap: [`docs/arms-mission-control-gap-todos.md`](docs/arms-mission-control-gap-todos.md).  
OpenAPI: [`docs/openapi/arms-openapi.yaml`](docs/openapi/arms-openapi.yaml).

## Quick start

**Run the API** (from repo root):

```bash
cd arms
go run ./cmd/arms
```

**Docker** (SQLite volume + optional Redis for Asynq-backed autopilot when you set **`ARMS_REDIS_ADDR`** and run **`cmd/arms-worker`** alongside **`cmd/arms`**):

```bash
docker compose -f arms/docker-compose.yml up --build
```

Service listens on **`http://localhost:8080`** by default (`ARMS_LISTEN`).

**Smoke checks:**

```bash
curl -s http://localhost:8080/api/health
curl -s http://localhost:8080/api/docs/routes
```

**SSE** (use `?token=…` if `MC_API_TOKEN` is set):

```bash
curl -N http://localhost:8080/api/live/events
```

With persistence, set `DATABASE_PATH` (e.g. `./data/arms.db`); empty uses in-memory stores. Optional **`ARMS_BUDGET_DEFAULT_CAP`** (default `100`, use `0` to disable) applies when no per-product `cost_caps` row exists. **GitHub PRs** (`POST /api/tasks/{id}/pull-request`): either set **`ARMS_GITHUB_TOKEN`** (or `GITHUB_TOKEN`) for the REST API, or set **`ARMS_GITHUB_PR_BACKEND=gh`** and use the [GitHub CLI](https://cli.github.com/) (`gh auth login`); optional **`ARMS_GH_BIN`**, **`ARMS_GITHUB_HOST`** (Enterprise). See [`docs/api-ref.md`](docs/api-ref.md).

## Repo layout

| Path | Role |
|------|------|
| `arms/` | Go module: `cmd/arms`, `cmd/arms-worker`, `internal/{domain,ports,adapters,application,platform,config,jobs}` |
| `docs/` | API reference, gap analysis, production notes |
| `fishtank/` | Separate area (see that tree) |
| `.github/workflows/` | CI for `arms` |

## Tests

```bash
cd arms
go test ./...
go test -tags=integration ./internal/integration/...
```

## License / status

Work in progress; behavior and APIs evolve with the gap checklist. For deployment hardening, see [`docs/arms-production-hardening.md`](docs/arms-production-hardening.md).
