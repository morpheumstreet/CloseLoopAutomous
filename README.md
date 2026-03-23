# CloseLoopAutomous

**Mission and product control for autonomous, self-iterating workflows**—plan products, triage ideas, run Kanban tasks through agents, ship with PRs and merge queues, and watch it all move on a live event stream. The stack pairs a production-minded **Go** API (**arms**) with an optional **Fishtank** browser UI, in the spirit of [Autensa / Mission Control](https://github.com/crshdn/mission-control) but delivered as a focused service you can run locally or in Docker.

---

## What you get

### Product and idea lifecycle

- **Products** with mission/vision, automation tier (`supervised` → `full_auto`), cadences for research and ideation, merge policy, and cost caps.
- **Research and ideation** phases that feed structured **ideas** with rich metadata (scores, complexity, tags, sources).
- **Swipe / preference flow** (`pass`, `maybe`, `yes`, `now`), a **maybe pool** with batch re-evaluation, **customer feedback** capture, and a **preference model** you can edit or recompute from swipe history.
- **TF-IDF tag suggestions** (no LLM) for ideas and product-scoped corpus context.

### Execution: tasks, convoys, and agents

- **Kanban tasks** tied to approved ideas—planning, dispatch, checkpoints, restore, and completion—with **budget checks** before spend.
- **Convoys**: DAG-style subtasks, mail between subtasks, wave dispatch with shared budget rules, plus **Mission Control–compatible** convoy routes on tasks.
- **Agent registry** and **gateway endpoints** in the database: plug in **OpenClaw-class WebSockets**, **PicoClaw**, **ZeroClaw**, **Clawlet**, **IronClaw**, **Nanobot CLI**, **MimiClaw**, **zclaw relay**, **NullClaw A2A**, and more (see [`config/arms.toml`](config/arms.toml) comments).
- Optional **Redis + arms-worker** for scheduled product ticks, autopilot, and **stall nudge / reassignment** automation.

### Shipping and operations

- **Pull requests** from tasks via GitHub REST or **`gh pr create`**, with GitHub.com and Enterprise-friendly configuration.
- **Serialized merge queue** with lease semantics; backends include **noop**, **GitHub merge**, or **local git**.
- **HMAC webhooks** for **agent completion** and **CI completion** so external runners can advance the board or finish convoy subtasks.
- **Server-Sent Events** (`/api/live/events`) for dispatch, costs, checkpoints, PRs, merge ship, convoy activity, and more—with **outbox + relay** when using SQLite for restart-safe delivery.

### Knowledge, costs, and observability

- **Knowledge** indexing with configurable backends (e.g. **FTS5**, optional **chromem** / embeddings) for dispatch-time snippets and ingestion controls.
- **Cost recording** and **per-product breakdowns** (by agent, model, time range) aligned with the same composite budget rules as dispatch.
- **Operations log** for auditing product, task, merge-queue, and related actions.
- **Operator endpoints**: health, version, fleet summary, and **host metrics** (CPU, memory, disk).

### User interface

- **Fishtank** (React + Vite) surfaces missions, docs, tasks, and system views—run with **Bun** for dev and production builds.

---

## At a glance

| Piece | Location |
|--------|-----------|
| HTTP API + domain | [`arms/`](arms/) (`cmd/arms`, hexagonal `internal/`) |
| Background worker | [`arms/cmd/arms-worker`](arms/cmd/arms-worker) |
| UI | [`fishtank/`](fishtank/) |
| Example config | [`config/arms.toml`](config/arms.toml) |

**Status:** APIs and behavior evolve; parity and roadmap notes live in [`docs/arms-mission-control-gap-todos.md`](docs/arms-mission-control-gap-todos.md). For deployment hardening, see [`docs/arms-production-hardening.md`](docs/arms-production-hardening.md).

---

## Appendix A — Technical reference

| Document | Contents |
|----------|-----------|
| [`docs/design/overview.md`](docs/design/overview.md) | High-level design |
| [`docs/design/part2.md`](docs/design/part2.md), [`docs/design/ui-design.md`](docs/design/ui-design.md) | Deeper design / UI notes |
| [`docs/recomendeddesign.md`](docs/recomendeddesign.md) | Recommended design notes |
| [`docs/add-multi-claw-types.md`](docs/add-multi-claw-types.md) | Multi-gateway / claw driver notes |
| [`docs/fishtank-ui-wiring-outstanding.md`](docs/fishtank-ui-wiring-outstanding.md), [`docs/fishtank-ui-todos.md`](docs/fishtank-ui-todos.md) | UI wiring backlog |
| [`arms/internal/domain/`](arms/internal/domain/) | Domain types (products, tasks, convoys, enums) |
| [`arms/internal/adapters/sqlite/migrations/`](arms/internal/adapters/sqlite/migrations/) | Versioned SQLite schema |

**Repository layout**

| Path | Role |
|------|------|
| `arms/` | Go module: API server, worker, ports, adapters, jobs |
| `docs/` | API reference, setup, production, gaps, design |
| `fishtank/` | React UI (Bun-only scripts) |
| `.github/workflows/` | CI for `arms` |

---

## Appendix B — API reference

| Resource | Location |
|----------|-----------|
| Human-readable route tables and semantics | [`docs/api-ref.md`](docs/api-ref.md) |
| OpenAPI 3.1 | [`docs/openapi/arms-openapi.yaml`](docs/openapi/arms-openapi.yaml) |
| Live route inventory | `GET /api/docs/routes` on a running server (same catalog as `internal/adapters/httpapi/routes_catalog.go`) |

**Auth:** Bearer token when `MC_API_TOKEN` is set; optional same-origin relaxation. Webhooks use **HMAC** (`WEBHOOK_SECRET`). SSE accepts Bearer header or `?token=` when auth is enabled.

---

## Appendix C — Setup guide

End-to-end instructions (Go, Bun, optional Redis/Docker, env vars, Fishtank) are in **[`docs/setup-guide.md`](docs/setup-guide.md)**.

**Minimal API run** (from repo root):

```bash
cd arms
go run ./cmd/arms
```

**Docker** (SQLite volume; add Redis and `arms-worker` when using Asynq-backed schedules/autopilot—see setup guide):

```bash
docker compose -f arms/docker-compose.yml up --build
```

Default listen address: **`http://localhost:8080`** (`ARMS_LISTEN`). Quick checks:

```bash
curl -s http://localhost:8080/api/health
curl -s http://localhost:8080/api/docs/routes
```

**Tests** (`arms/`):

```bash
cd arms
go test ./...
go test -tags=integration ./internal/integration/...
```

---

## Appendix D — License

This project is licensed under the **MIT License** — see [`LICENSE`](LICENSE) in the repository root (**SPDX:** `MIT`).
