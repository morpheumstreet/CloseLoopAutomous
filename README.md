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
- **Agent registry** and **gateway endpoints** in the database: each endpoint uses a **`driver`** string to select the integration below; field semantics (URL, token, `device_id`, `session_key`) match [`config/arms.toml`](config/arms.toml) and the domain constants in [`arms/internal/domain/gateway_endpoint.go`](arms/internal/domain/gateway_endpoint.go).
- Optional **Redis + arms-worker** for scheduled product ticks, autopilot, and **stall nudge / reassignment** automation.

### Supported gateway drivers (“claws”)

These are the **`gateway_endpoints.driver`** values **arms** recognizes today (aliases such as `openclaw` → `openclaw_ws` are normalized in code).

| Driver | Agent / runtime | How dispatch runs |
|--------|-----------------|-------------------|
| `stub` | Built-in stub | In-process no-op (seed endpoint `gw-stub`); for tests and dry runs. |
| `openclaw_ws` | [OpenClaw](https://github.com/openclaw/openclaw) | WebSocket RPC (`connect`, `chat.send`)—the reference OpenClaw-class wire. |
| `nemoclaw_ws` | NemoClaw / NVIDIA OpenShell | Same WebSocket shape as OpenClaw; optional `nemoclaw <sandbox> start` before dial (`ARMS_NEMOCLAW_*`). |
| `nullclaw_ws` | NullClaw (legacy) | OpenClaw-shaped WebSocket (not stock NullClaw HTTP). |
| `nullclaw_a2a` | [NullClaw](https://github.com/nullclaw/nullclaw) | HTTP JSON-RPC 2.0 `POST …/a2a` (`message/send`, Google A2A-style). |
| `picoclaw_ws` | [PicoClaw](https://github.com/sipeed/picoclaw) | Pico Protocol WebSocket (`message.send` + `session_id`). |
| `zeroclaw_ws` | [ZeroClaw](https://github.com/zeroclaw-labs/zeroclaw) | OpenClaw-compatible WebSocket (`connect` + `chat.send`). |
| `clawlet_ws` | [Clawlet](https://github.com/mosaxiv/clawlet) | OpenClaw-class WebSocket when a compatible control listener is configured. |
| `ironclaw_ws` | IronClaw | Rust OpenClaw-class WebSocket gateway. |
| `mimiclaw_ws` | [MimiClaw](https://github.com/memovai/mimiclaw) | JSON WebSocket (e.g. device on port 18789; `session_key` = `chat_id`). |
| `nanobot_cli` | [Nanobot](https://github.com/HKUDS/nanobot) | Subprocess: `nanobot agent -m` (not the nanobot channel gateway). |
| `inkos_cli` | [InkOS](https://github.com/Narcooo/inkos) | Subprocess: `inkos write next … --json`. |
| `zclaw_relay_http` | [zclaw](https://github.com/tnm/zclaw) | HTTP `POST …/api/chat` via the web relay. |
| `mistermorph_http` | [MisterMorph](https://github.com/quailyquaily/mistermorph) | HTTP task API: `POST …/tasks`, poll `GET …/tasks/{id}` (Bearer auth). |
| `copaw_http` | [CoPaw](https://github.com/agentscope-ai/CoPaw) (AgentScope) | JSON-RPC `POST …/console/api` (`chat.send`); `device_id` = workspace. |
| `metaclaw_http` | MetaClaw (or any OpenAI-compatible proxy) | `POST …/v1/chat/completions`; optional model via `device_id`. |

For per-driver URL/token/session notes and env toggles, see the comments in [`config/arms.toml`](config/arms.toml).

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
