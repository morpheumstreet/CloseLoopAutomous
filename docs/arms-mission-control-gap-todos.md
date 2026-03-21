# Arms backend — gap todo list (vs [mission-control](https://github.com/crshdn/mission-control/tree/main))

Use this as the master backlog for bringing `arms` toward Autensa/Mission Control backend parity. Check items off as you implement them.

_Re-checked against the `arms/` tree (2026-03): baseline vs “full MC” is called out so unchecked rows are not misread as “missing entirely” when a slim table or route already exists._

_See also [recomendeddesign.md](recomendeddesign.md) (earlier “GoAutensa” outline); this file is the live parity checklist + locked target architecture._

---

## Target architecture (single Go binary)

Direction for **100% MC-style behavior** while staying hexagonal (`ports` / `adapters` / `domain` / `application` / `platform`):

```
CloseLoopAutomous / arms (e.g. :8080)
├── cmd/arms/                    # main, graceful shutdown, config; + Asynq worker process when added
├── internal/adapters/httpapi    # REST (+ alias routes for MC clients)
├── internal/application/        # autopilot, convoy, costs, workspace, learner, scheduler, events
├── internal/domain/
├── internal/ports/              # repos, openclaw, pr publisher, event bus, workspace, …
├── internal/adapters/
│   ├── sqlite/
│   ├── gateway/openclaw/        # WS client (existing; path: internal/adapters/gateway/openclaw)
│   ├── shipping/                # GitHub PR (replace noop)
│   ├── workspace/               # worktrees, ports 4200–4299, merge queue (new)
│   └── notifier/                # SSE wired from domain events (evolve from shell)
└── migrations/
```

**Observability (later):** structured logs today (`slog`); optional **zap + OpenTelemetry + Prometheus** when ops requirements land.

---

## Locked design decisions

These resolve open questions from the backlog; implement against this table.

| Topic | Decision |
|-------|-----------|
| **REST naming** | Keep **plural** canonical: `/api/convoys`, `/api/tasks`, … (Go/REST convention + current code). Add **optional alias** routes (`/api/convoy/{id}` → same handler) for Next.js MC clients that expect singular paths. |
| **Scheduling** | **Asynq (Redis) + cron** for `product_schedules` and delayed jobs; **restart-safe** vs in-process ticker. Deprecate `ARMS_AUTOPILOT_TICK_SEC` once scheduler is wired (keep env until cutover). |
| **Device identity** | **Ed25519 `connect` block optional** via env (e.g. `ARMS_DEVICE_SIGNING=enabled`); token-only remains default. |
| **Automation tiers** | **supervised** — PRs created; human approve/merge. **semi_auto** — auto-dispatch + **manual merge** (extend later if auto-merge desired). **full_auto** — end-to-end autopilot (dispatch + merge queue policy in autopilot + workspace). Cross-check [MC README — Automation tiers](https://github.com/crshdn/mission-control). |
| **Convoy DAG** | Full graph in domain + SQLite; **github.com/dominikbraun/graph** (or equivalent) for algorithms; **`convoy_subtasks` + `agent_mailbox`** persistence. |
| **Realtime** | **Domain events + transactional outbox** → SSE `/api/live/events` (and later operator chat); avoid polling DB from handlers. |
| **Cost caps** | **`cost_caps` table** + daily/monthly/product scope; atomic enforcement in **application/costs** (extends today’s `budget.Static`). |
| **Workspace** | Dedicated **workspace service**: git worktrees, sandbox paths, port allocator **4200–4299**, **serialized merge queue**, **product-scoped locks**. |
| **Preference learning** | Migrate from **`preference_model_json` append-only** → **`swipe_history` + `preference_models`**; optional embeddings later. |
| **PR shipping** | Real **`PullRequestPublisher`**: default **google/go-github** REST + PAT; optional **`gh pr create`** backend for local/Enterprise flows (`ARMS_GITHUB_PR_BACKEND=gh`). |

---

## API stubs, docs, and gateway session

**Human reference:** [api-ref.md](api-ref.md) — section *Stubs / placeholders*. **OpenAPI:** [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — tag **Stubs** plus `Product.preference_model_json` (stub for real preference learning, not under Stubs).

| Route | Status |
|-------|--------|
| `GET /api/agents` | Empty list until agent domain ships |
| `POST /api/openclaw/proxy` | Not implemented (501); use server env `OPENCLAW_GATEWAY_*` + WS from service |
| `GET /api/workspaces` | **Snapshot:** allocated ports + `merge_queue_pending` (not a stub when stores wired) |
| `GET /api/settings` | Minimal / empty JSON |

There is no REST “session” resource. OpenClaw dispatch uses env (e.g. `ARMS_OPENCLAW_SESSION_KEY` on the server). A **browser-facing** gateway proxy (MC-style `/api/openclaw/*`) is optional; see §3.

---

## Implementation roadmap (vertical slices)

Rough calendar: **~4 weeks core (A–C)** + **polish (D)**; optional future below.

| Phase | Time (guide) | Deliverables |
|-------|----------------|--------------|
| **A — Production safety** | 1–2 wk | **Done (when `AgentHealth` wired):** MC convoy singular aliases (`/api/convoy/...`); **`GET /api/products/{id}/stalled-tasks`**; completion webhook + **`POST /api/tasks/{id}/complete`** → **`task_agent_health`** **`completed`** + **`task_completed`** outbox in **one SQLite transaction** (`LiveActivityTX.CompleteTaskWithEvent`); task **`sandbox_path` / `worktree_path`** (008–009). **Manual stall nudge:** **`POST /api/tasks/{id}/stall-nudge`** (optional JSON `{ "note" }`) → `status_reason` prefix + optional agent-health `stall_nudges[]` + SSE **`task_stall_nudged`**. **Still open:** automated **merge/conflict** policy (git), **auto**-nudge/reassign, same-Tx outbox for remaining paths (e.g. PR opened), multi-instance **DB leases** for completion / product gates. |
| **B — Full autonomy** | ~2 wk | Convoy: full DAG + **mailbox** + deeper **agent health** (retries, convoy-aware dispatch). **GitHub PR** — **done:** REST (`go-github`) + optional **`gh` CLI** backend + env tokens. **TBD:** auto post-execution chain. Deeper **ideas** scoring/metadata; **`swipe_history`** table + list API (**done**); separate **`preference_models`** table / learning loop still **TBD**. |
| **C — Polish** | ~1 wk | **Agent** domain + listing/health APIs (replace stub). **`product_schedules`** on **Asynq** (Redis). Optional **Ed25519** on OpenClaw `connect`. **Maybe pool** resurface / batch re-eval. ~~**HTTP aliases** `/api/convoy/*`~~ (done in A). |
| **D — Optional future** | — | Embedded UI (e.g. HTMX/templ), Postgres adapter, pure-Go agent runtime (replace OpenClaw). |

**Done in-tree (former “first commits”):** Compose **redis** service (optional; not yet consumed by app code), transactional **outbox** + **`livefeed`** SSE hub, **workspace** ports + merge queue + optional git worktrees, **GitHub** / **`gh`** behind `PullRequestPublisher`, **swipe_history**, **cost_caps** + composite budget, **task agent health** APIs.

**Next vertical slices (suggested):** (1) **Asynq + Redis** + `product_schedules` / cron (add `ARMS_REDIS_ADDR` or equivalent to `internal/config` and worker entrypoint), (2) **preference_models** or ML pipeline consuming **`swipe_history`**, (3) **agent** aggregate + mailbox, (4) convoy **DAG + `agent_mailbox`**, (5) optional **`/api/openclaw/*`** HTTP proxy if the UI needs it.

---

## Mission Control references (behavior parity)

Use [crshdn/mission-control](https://github.com/crshdn/mission-control) for behavioral detail; **routing/scheduling decisions are locked above**.

| Topic | Where to look in MC |
|-------|---------------------|
| Automation tiers (merge / pause rules) | README — *Automation tiers* and autopilot pipeline |
| Live activity / SSE | README — *Live Activity Feed*; realtime docs under repo `docs/` |
| Workspace isolation, ports, merge queue | README — *Workspace isolation*; `src/lib/workspace-isolation.ts` (and related) |
| OpenClaw client / handshake | [`src/lib/openclaw/client.ts`](https://github.com/crshdn/mission-control/blob/main/src/lib/openclaw/client.ts) |
| Device identity / signing | `device-identity` patterns vs token-only `connect` |
| Convoy + API shape | `src/app/api/convoy/` — arms uses plural + **alias** routes (see **Locked design decisions**) |

---

## 1. API surface and transport

- [x] Optional **MC-compat alias routes** — `POST /api/convoy`, `GET /api/convoy/{id}`, `POST /api/convoy/{id}/dispatch-ready` → same handlers as `/api/convoys/...`
- [x] Add HTTP server driving adapter (REST or minimal RPC) for orchestration — `cmd/arms`, `internal/adapters/httpapi`
- [x] Map route groups analogous to MC: `tasks`, `products`, `agents`, `costs`, `convoy`, `openclaw`, `webhooks`, `events`/`live`, `workspaces`, `settings` — implemented or stubbed under `/api/...`
- [x] Bearer auth middleware (`MC_API_TOKEN`-style) — env `MC_API_TOKEN`; omitted = dev open access
- [x] SSE auth pattern (e.g. token query param) for live streams — `GET /api/live/events` uses `?token=` when auth enabled (`SSEQueryToken`)
- [x] Request validation layer (DTOs + schema validation) — JSON DTOs + `validate()` helpers (no external schema lib yet)
- [x] Agent-completion webhook receiver — `POST /api/webhooks/agent-completion`
- [x] HMAC verification for webhooks (`WEBHOOK_SECRET`-style) — header `X-Arms-Signature` = hex(HMAC-SHA256(secret, raw body))
- [x] Route catalog documenting public API — `GET /api/docs/routes`
- [x] Human-readable API reference — `docs/api-ref.md`
- [x] OpenAPI 3.1 spec (hand-maintained) — `docs/openapi/arms-openapi.yaml` (import into Swagger UI / Redoc; not codegen-generated)

---

## 2. Persistence and data model (SQLite)

- [x] SQLite adapter implementing repository ports — `internal/adapters/sqlite` (`ProductStore`, `IdeaStore`, `TaskStore`, `ConvoyStore`, `CostStore`, `CheckpointStore`)
- [x] Migration runner + versioned migrations — embedded `migrations/*.sql`, `arms_schema_version`, `ExpectedSchemaVersion` constant (bump when adding files)
- [x] Pre-migration backup — `ARMS_DB_BACKUP=1` runs `VACUUM INTO` to `{DATABASE_PATH}.pre-migrate-{UTC}.bak` before migrate
- [x] Server wiring — `DATABASE_PATH` set → `platform.OpenApp` uses SQLite; empty → in-memory (same as before)
- [x] Baseline schema in `001_initial.sql` + `002_kanban_tasks.sql` — `products`, `ideas`, `tasks` (TEXT Kanban `status` after v2), `convoys` / `convoy_subtasks`, `cost_events`, `checkpoints` (one payload row per task)
- [x] Partial FK cascade — `ideas`, `tasks`, `convoys`, `cost_events` reference `products` with `ON DELETE CASCADE` where declared in migrations (not equivalent to all MC safety / soft-delete behavior)

### Extend toward full Mission Control data model

- [x] `products`: baseline MC-style profile — `repo_url`, `repo_branch`, `description`, `program_document`, `settings_json`, `icon_url` (migration `003_product_mc_metadata.sql`); HTTP `POST /api/products` optional fields + `PATCH /api/products/{id}`; profile text/repo hints passed through `domain.Product` into research/ideation ports (stubs use `ai.ProductContextSnippet`; real LLM adapters TBD)
- [ ] `research_cycles` / research history (not only `Product.ResearchSummary`)
- [ ] `ideas`: full scoring/metadata as in MC (today: title, description, impact, feasibility, reasoning, swipe outcome)
- [x] `swipe_history` — migration `007_swipe_history.sql`; SQLite + memory stores; autopilot **Append** on swipe / promote-maybe; **`GET /api/products/{id}/swipe-history`** (`?limit=`)
- [ ] `preference_models` (per-product learning) — **no** dedicated table yet; today: `preference_model_json` on product + **`swipe_history`** audit trail
- [x] `maybe_pool` table + list/promote API — `MaybePoolRepository`; `GET /api/products/{id}/maybe-pool`, `POST /api/ideas/{id}/promote-maybe` (baseline; aligns with §5)
- [ ] Maybe pool **resurface** / batch re-eval workflow (MC-style; not just storage)
- [ ] `product_feedback`
- [x] `cost_events`: **agent**, **model** columns (`006_phase_a_safety.sql`); append + breakdown API
- [x] `cost_caps` (daily + monthly + cumulative per product) + **`budget.Composite`** at dispatch
- [ ] `product_schedules` (research/ideation cadence, cron)
- [ ] `operations_log` / audit trail
- [ ] `convoys` / `convoy_subtasks`: richer DAG metadata, mail, per-subtask status (today: matches slim `domain.Convoy` + HTTP create + dispatch-ready wave)
- [x] `task_agent_health` (per-task; not full MC agent registry) — migration `009_agent_health_repo_path.sql` (table + `products.repo_clone_path` + `workspace_merge_queue.completed_at`)
- [x] `tasks`: **`sandbox_path`**, **`worktree_path`** — migration `008_task_workspace_paths.sql` (metadata for isolation / worktrees; returned on task JSON; **`PATCH /api/tasks/{id}`** may set them)
- [x] Checkpoint **history** + restore — `checkpoint_history` + APIs (latest still in `checkpoints`); MC **`work_checkpoints`** naming parity optional
- [ ] `agent_mailbox`
- [x] `workspace_ports` (4200–4299) + HTTP allocate/release
- [x] `workspace_merge_queue` table + pending **count** in `GET /api/workspaces`; FIFO **head** completion + **`completed_at`** on done; **real ship** optional via **`ARMS_MERGE_BACKEND=github|local`** (lease columns, merge outcome fields, **`mergequeue` service**); query **`skip_ship=1`** for break-glass metadata-only advance
- [ ] Broader MC parity: soft deletes, extra cascade paths, concurrency guards, ops tooling

---

## 3. OpenClaw / execution plane

- [x] Real `AgentGateway` adapter: WebSocket client — `internal/adapters/gateway/openclaw` ([coder/websocket](https://github.com/coder/websocket))
- [x] Config: `OPENCLAW_GATEWAY_URL`, `OPENCLAW_GATEWAY_TOKEN` (env) + `OPENCLAW_DISPATCH_TIMEOUT_SEC` (default 30) + `ARMS_DEVICE_ID` (optional `X-Arms-Device-Id`)
- [x] Dispatch timeouts — per-call `context.WithTimeout` from `OpenClawDispatchTimeout`
- [x] Reconnect on failure — drop cached conn after read/write error; next dispatch dials again (`App.Close` also closes client)
- [x] Map gateway errors — task layer wraps adapter errors with `domain.ErrGateway` (existing `task.Service`)
- [x] Device identity hint — `ARMS_DEVICE_ID` header on WS handshake (full MC device file parity still TBD)
- [x] Native OpenClaw WebSocket framing (aligned with [mission-control `client.ts`](https://github.com/crshdn/mission-control/blob/main/src/lib/openclaw/client.ts)): `token` query param + optional Bearer, `connect.challenge` → `connect` RPC (protocol 3), dispatch via **`chat.send`** with `sessionKey`, `message`, `idempotencyKey`
- [ ] Ed25519 **device** block on `connect` (MC `device-identity.ts` signing) — **optional** behind env (e.g. `ARMS_DEVICE_SIGNING=enabled`); token-only default
- [ ] Optional HTTP proxy routes (`/api/openclaw/*` equivalent) if UI or ops need them

---

## 4. Task lifecycle and Kanban

- [x] Align `Task` status model with MC Kanban columns — string statuses (`planning` → `inbox` → `assigned` → `in_progress` → `testing` → `review` → `done`) plus `failed`, `convoy_active`; migration `002_kanban_tasks.sql`
- [x] Planning gate + opaque planning JSON — `Task.ClarificationsJSON`, `UpdatePlanningArtifacts`; HTTP `PATCH /api/tasks/{id}` with `clarifications_json` while in `planning` (structured Q&A UX / spec editor still TBD)
- [x] Plan approval + reject / recall — `ApprovePlan`, `ReturnToPlanning` (inbox or assigned before dispatch); HTTP `POST /api/tasks/{id}/plan/approve`, `POST /api/tasks/{id}/plan/reject` (optional `{ "status_reason" }`); Kanban moves via `PATCH /api/tasks/{id}` (`status`, `status_reason`)
- [x] List tasks per product (board feed) — `GET /api/products/{id}/tasks`, `ports.TaskRepository.ListByProduct` (SQLite + memory), `404` if product missing
- [ ] Task images / attachments storage + API
- [ ] Distinguish manual task flow vs autopilot-derived tasks where MC does

---

## 5. Autopilot pipeline (extended)

- [x] Product program / profile injection into research/ideation — stored on `Product` + HTTP; `ResearchPort` / `IdeationPort` godoc + `ai.ProductContextSnippet` + stub behavior (full MC “Product Program CRUD” UX still evolves with UI)
- [x] Scheduling (interim): `research_cadence_sec`, `ideation_cadence_sec`, `last_auto_*` on product; `autopilot.Service.TickScheduled`; `ARMS_AUTOPILOT_TICK_SEC` in-process ticker in `cmd/arms`. `auto_dispatch_enabled` + tier stored; **task auto-dispatch from tier not wired** (manual dispatch unchanged).
- [ ] **Asynq + Redis** worker — `product_schedules` + cron/delayed jobs; replaces in-process ticker for production (**Locked design decisions**)
- [x] Background job — **minimal** today: in-process ticker; separate **worker process** TBD with Asynq
- [x] Preference stub: each swipe appends an event to `preference_model_json` (JSON array); when **`SwipeHistoryRepository`** is wired, the same flow **also** persists **`swipe_history`** rows. Not full MC **`preference_models`** / ML.
- [x] Maybe pool (baseline): `maybe_pool` table + `MaybePoolRepository`; swipe `maybe` adds; `GET /api/products/{id}/maybe-pool`; `POST /api/ideas/{id}/promote-maybe` → yes + pool remove + stage advance when in swipe. Resurface / batch re-eval: still open (§2).
- [x] Automation tiers: `automation_tier` enum `supervised` | `semi_auto` | `full_auto` on product + create/patch/JSON (behavioral differences beyond storage/TBD for dispatch).
- [ ] Post-execution chain: test → review → **automatic** PR on transitions (today: explicit **`POST /api/tasks/{id}/pull-request`** only).
- [x] GitHub **`PullRequestPublisher`** — `adapters/shipping` GitHub client (go-github v66) + noop; **`POST /api/tasks/{id}/pull-request`** (`head_branch`, optional `title`/`body`); **`ARMS_GITHUB_TOKEN`** / **`GITHUB_TOKEN`**; SSE **`pull_request_opened`** when URL returned.

---

## 6. Convoy mode

- [x] Persist baseline convoy + subtasks — SQLite + memory `ConvoyRepository` (graph + dispatch flags/refs); not yet “full MC” metadata
- [ ] Persist full convoy DAG metadata as in MC (beyond current domain)
- [ ] Convoy mail / inter-subtask messaging (port + persistence)
- [ ] Integrate convoy dispatch with agent health and retries
- [x] Minimal HTTP — `POST /api/convoys`, `GET /api/convoys/{id}`, `GET /api/products/{id}/convoys`, `POST /api/convoys/{id}/dispatch-ready`; `convoy.Service.Get`, `ListByProduct`; `ports.ConvoyRepository.ListByProduct`
- [ ] API parity with MC convoy — mail, graph, richer status (**naming / singular aliases:** done — §1)
- [ ] Richer subtask model (agent config, retries, nudges) if required for parity

---

## 7. Safety, cost, workspace

- [x] Budget at dispatch — **`budget.Composite`**: per-product **`cost_caps`** (daily / monthly / cumulative) + default cumulative when **no** caps row via **`ARMS_BUDGET_DEFAULT_CAP`** (default 100; set `0` to disable default ceiling)
- [x] Cost breakdown — **`GET /api/products/{id}/costs/breakdown`** (`from` / `to` query RFC3339); aggregates `by_agent`, `by_model`
- [x] Workspace isolation: **optional git worktree** (`internal/adapters/workspace` + gated HTTP); paths still **metadata** on tasks + ports; operator must set **`repo_clone_path`** on product
- [x] Port allocation **4200–4299** — `workspace_ports` + **`POST /api/workspace/ports`** / **`DELETE /api/workspace/ports/{port}`**
- [x] Serialized merge queue **ordering** — only FIFO **head** per product can `POST .../merge-queue/complete` (`domain.ErrNotMergeQueueHead` → 409); not git merge / conflict detection
- [x] Product-scoped **in-process** lock on task **Complete** (`task.ProductGate`); multi-instance would need DB leases later
- [x] Checkpoint **history** + **restore** — `checkpoint_history` + **`GET /api/tasks/{id}/checkpoints`**, **`POST .../checkpoint/restore`** (`history_id`); latest row still in `checkpoints`
- [x] Agent health — **task-scoped** heartbeats + SQLite/memory + HTTP (not full MC **agent** aggregate yet)
- [x] Stalled detection — **`GET /api/products/{id}/stalled-tasks`** (`no_heartbeat` / `heartbeat_stale` for in_progress, testing, review, convoy_active)
- [x] **Manual** stall nudge — **`POST /api/tasks/{id}/stall-nudge`** (execution statuses); **`task_stall_nudged`** SSE + agent-health detail `stall_nudges[]`
- [ ] **Auto**-nudge / reassign policy for stalled tasks

---

## 8. Realtime and observability

- [x] Domain outbox baseline — table `event_outbox` (`005_event_outbox.sql`); `internal/application/livefeed` (**Hub**, **OutboxPublisher**, **RunOutboxRelay**); SQLite path relays to SSE; in-memory path publishes directly to hub
- [x] Same-transaction outbox for **SQLite** dispatch, checkpoint, cost, and **task completion** + agent-health **`completed`** (`LiveActivityTX`); other paths (e.g. PR opened) still best-effort after external I/O
- [x] SSE transport — `GET /api/live/events` (hello + ping + activity `data:` lines; `SSEQueryToken` when auth on)
- [x] SSE activity (partial) — **`task_dispatched`**, **`cost_recorded`**, **`checkpoint_saved`**, **`task_completed`** (SQLite same-tx + relay; in-memory hub on complete), **`task_stall_nudged`** (operator **`POST .../stall-nudge`**); **`?product_id=`** filter; broader catalog + agent/type filters still TBD
- [ ] Operator chat: queued notes + direct messages (ports + storage)
- [ ] Per-task chat history
- [ ] Learner / knowledge base port + storage + injection into future dispatches

---

## 9. Agents domain

- [ ] Agent aggregate + repository
- [ ] Registration and discovery/import flows (gateway-backed)
- [ ] Agent APIs: mailbox, full listing, configuration (beyond task heartbeats)
- [x] Health-style data — **task agent heartbeats** + per-task / per-product agent-health routes + `GET /api/agents` lists recent rows (`stub: true` only if handlers built with **`AgentHealth == nil`** — normal SQLite/memory apps wire it)

---

## 10. Cross-cutting platform

- [x] Dedicated `internal/config` — `LoadFromEnv()`, env vars documented on `Config`; `httpapi.Config` is a type alias + `LoadConfig()` wrapper for the HTTP adapter
- [x] Dockerfile for `arms` service — `arms/Dockerfile` (Alpine runtime, static `CGO_ENABLED=0` build)
- [x] docker-compose — `arms/docker-compose.yml` (port 8080, `DATABASE_PATH=/data/arms.db`, named volume; OpenClaw/token env commented)
- [x] **Redis** service in Compose — optional sidecar for **Asynq** when scheduler lands; app does **not** yet read Redis URL from env (add e.g. `ARMS_REDIS_ADDR` to `internal/config` with worker wiring)
- [x] Production hardening doc — `docs/arms-production-hardening.md` (secrets, TLS termination, `wss://`, `NO_PROXY` / webhooks, persistence, containers, logging)
- [x] Structured logging + request IDs — `log/slog` in `cmd/arms`, `X-Request-ID` + optional access log (`internal/adapters/httpapi/logging.go`); `ARMS_LOG_JSON`, `ARMS_ACCESS_LOG`
- [x] Automated tests touching persistence + HTTP wiring — SQLite `repos_test` / `migrate_test`, `sqlite_app_test`, `platform/router_test`, application tests with memory/SQLite + gateway stub (`openclaw` tests use real WS to test client)
- [x] Opt-in HTTP integration tests — `internal/integration/` with `//go:build integration`; run `go test -tags=integration ./internal/integration/...` (in-memory app + stub gateway; full product→task→dispatch flow)
- [x] CI — `.github/workflows/arms.yml` runs `go test ./...` and `-tags=integration` on `arms/**` changes
- [ ] Contract tests against **live** OpenClaw gateway (optional env-gated job)

---

## Quick reference

| Area            | Rough priority for a vertical slice                         |
|-----------------|--------------------------------------------------------------|
| SQLite + core tables | Unblocks everything else                                  |
| HTTP + auth + tasks/products | Makes the service usable from a UI or CLI            |
| Real OpenClaw WS   | Closes the execution-plane gap                            |
| Webhooks           | Completes the async completion loop                       |
| SSE + costs + workspace | Match MC ops and safety story                          |
| Stub routes → real domains | Agents, workspaces, settings, openclaw proxy (if needed) |
| Roadmap phases A→D | See **Implementation roadmap** above                      |
