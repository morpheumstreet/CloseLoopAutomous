# Arms backend — gap todo list (vs [mission-control](https://github.com/crshdn/mission-control/tree/main))

Use this as the master backlog for bringing `arms` toward Autensa/Mission Control backend parity. Check items off as you implement them.

**Backlog checklist (§1–§10):** `94` done · `13` open · **~88%** complete — see **[Master backlog (all checklist items)](#master-backlog-all-checklist-items)** for the full table; _grep_ `- [x]` / `- [ ]` in this file to refresh counts after edits._

**Next priority:** **Convoy** — **`dominikbraun/graph`** (or equivalent) + **MC-exact DAG** on top of migration **`022`**. **Knowledge:** auto-ingest from **product_feedback** / swipes / agent summaries still **TBD** (CRUD + dispatch injection **done**). **Stalled-task reassign** policy still **TBD** when task↔agent binding exists (**#93**/**#94**). **Preference:** **embeddings / trained model** on **`preference_models`** still **TBD**. Optional polish: stable PR correlation keys for extreme dedupe.

**Asynq scheduling (steady state):** Set **`ARMS_REDIS_ADDR`** and run **`cmd/arms-worker`** alongside **`cmd/arms`**. **`ARMS_AUTOPILOT_TICK_SEC`** and **`ARMS_USE_ASYNQ_SCHEDULER`** are **deprecated** (ignored; warnings if set). **`cmd/arms`** runs startup + **5m** resync (**`product_schedules`** + per-product reconcile) plus HTTP hooks; worker runs **`product:schedule:tick`**, **`arms:product_autopilot_tick`**, optional **`arms:autopilot_tick`**, and (when **`ARMS_AUTO_STALL_NUDGE_ENABLED`**) **`arms:stall_autonudge_tick`**.

**What this is:** a single checklist + design locks for **backend parity** with [mission-control](https://github.com/crshdn/mission-control): API routes, SQLite schema, OpenClaw wiring, safety/cost/workspace, realtime, and convoy/autopilot gaps. It is **not** a fishtank/UI spec; pair with [api-ref.md](api-ref.md) for HTTP details and [recomendeddesign.md](recomendeddesign.md) for the broader architecture sketch.

_Re-checked against the `arms/` tree (2026-03-23): SQLite schema **v24** (`ExpectedSchemaVersion` in `internal/adapters/sqlite/migrate.go`); baseline vs “full MC” is called out so unchecked rows are not misread as “missing entirely” when a slim table or route already exists._

_See also [recomendeddesign.md](recomendeddesign.md) (earlier “GoAutensa” outline); this file is the live parity checklist + locked target architecture._

### Remarks — merge-queue autopilot policy & shipping (done, 2026-03)

The backlog previously called out **“autopilot-driven merge policy”** and **same-transaction outbox** around external ship paths. The following is now implemented in `arms/`:

| Topic | What shipped | Where to look |
|-------|----------------|---------------|
| **Tier → merge gates** | Defaults from **`automation_tier`**: supervised / semi_auto require **approved GitHub review** + **`mergeable_state: clean`** for *unattended* ship; **full_auto** does not. Overrides: **`merge_policy_json`** fields **`require_approved_review`**, **`require_clean_mergeable`**. Product JSON **`merge_policy`** exposes effective booleans. | `internal/domain/merge.go` — `EffectiveMergeExecutionGates`, `MergePolicy` |
| **Semi-auto auto-ship** | On task **→ done**, **`MergeShip.CompleteIfPolicyAllowsAuto`**: runs real merge only if gates pass (silent skip if not); **`full_auto`** still uses **`Complete`** (no gate enforcement). | `internal/application/task/service.go` — `maybeAutoMergeShip`; `internal/application/mergequeue/service.go` |
| **GitHub gate check** | **`PullRequestMergeGateChecker`** on **`GitHubPRMerger`**: latest review per user must include **APPROVED** when required; **clean** mergeable state when required. | `internal/adapters/shipping/github_merge.go` — `CheckMergeGates` |
| **Operator override** | **`POST /api/tasks/{id}/merge-queue/complete`** (and **resolve** routes below) **do not** apply merge gates — human/operator can still force progression. | `mergequeue.Service.Complete` vs `CompleteIfPolicyAllowsAuto` |
| **Resolve after conflict** | **`POST /api/tasks/{id}/merge-queue/resolve`** and **`POST /api/merge-queue/{rowId}/resolve`** with optional body **`{"action":"retry_merge"\|"skip_ship"}`** (default retry). | `internal/adapters/httpapi/handlers.go`, `server.go`; OpenAPI + `routes_catalog` |
| **Same-Tx outbox on ship finish** | When live events use **`OutboxPublisher`** and the store supports it, **`FinishShipWithOutbox`** commits **merge_queue row update + `event_outbox` insert** in one SQLite transaction (no race vs relay). In-memory / hub-only paths still **finish then Publish**. | `internal/adapters/sqlite/workspace.go`; `ports.MergeShipOutboxFinisher`; `mergequeue` + `livefeed` |

**Still not done** (unchanged from Phase A bullets): **reassign** policy for stalls (auto-nudge shipped); **DB leases** for task completion / product gates beyond merge-queue lease. ~~**same-Tx outbox for PR opened**~~ — after GitHub returns, task row + **`pull_request_opened`** commit together when **`LiveTX`** is wired (forge HTTP remains out-of-band).

---

## Target architecture (single Go binary)

Direction for **100% MC-style behavior** while staying hexagonal (`ports` / `adapters` / `domain` / `application` / `platform`):

```
CloseLoopAutomous / arms (e.g. :8080)
├── cmd/arms/                    # HTTP API, graceful shutdown; with Redis: startup + 5m Asynq resync (schedules + per-product reconcile), HTTP hooks
├── cmd/arms-worker/             # Asynq consumer: arms:autopilot_tick, arms:product_autopilot_tick, product:schedule:tick (same DB/env as API)
├── internal/jobs/               # Shared Asynq task types + queue name **arms** (default queue)
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
| **Scheduling** | **Asynq (Redis) + cron** for `product_schedules` and delayed jobs; **`ARMS_AUTOPILOT_TICK_SEC` / in-process ticker deprecated** (2026-03); startup + **5m** API resync + worker chains. |
| **Device identity** | **Ed25519 `connect` block optional** via env (e.g. `ARMS_DEVICE_SIGNING=enabled`); token-only remains default. |
| **Automation tiers** | **supervised** — PRs created; human approve/merge; no unattended merge-queue ship on task **done**. **semi_auto** — merge-queue **auto-ship on task done** only when **GitHub merge gates** pass (approved review + `mergeable_state: clean` by default), overridable via **`merge_policy_json`**; manual **`POST …/merge-queue/complete`** still bypasses gates. **full_auto** — end-to-end autopilot including **ungated** merge-queue **`MergeShip.Complete`** on **done** (when queue/backend configured). Cross-check [MC README — Automation tiers](https://github.com/crshdn/mission-control). |
| **Convoy DAG** | Full graph in domain + SQLite; **github.com/dominikbraun/graph** (or equivalent) for algorithms; **`convoy_subtasks` + `agent_mailbox`** persistence. |
| **Realtime** | **Domain events + transactional outbox** → SSE `/api/live/events` (and later operator chat); avoid polling DB from handlers. |
| **Cost caps** | **`cost_caps` table** + daily/monthly/product scope; atomic enforcement in **application/costs** (extends today’s `budget.Static`). |
| **Workspace** | Dedicated **workspace service**: git worktrees, sandbox paths, port allocator **4200–4299**, **serialized merge queue**, **product-scoped locks**. |
| **Preference learning** | **`swipe_history`** (audit) + dedicated **`preference_models`** table + **`GET/PUT /api/products/{id}/preference-model`** (baseline); legacy **`preference_model_json`** on product still updated on swipe; **ML / embeddings** later. |
| **PR shipping** | Real **`PullRequestPublisher`**: default **google/go-github** REST + PAT; optional **`gh pr create`** backend for local/Enterprise flows (`ARMS_GITHUB_PR_BACKEND=gh`). |

---

## API stubs, docs, and gateway session

**Human reference:** [api-ref.md](api-ref.md) — section *Stubs / placeholders*. **OpenAPI:** [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — tag **Stubs** plus `Product.preference_model_json` (swipe append trail); dedicated preference payload also via **`GET/PUT …/preference-model`** (tag **Ideas**). Tag **Ops**: **`GET /api/operations-log`**.

| Route | Progress | Details |
|-------|----------|---------|
| `GET /api/agents` | **Partial** | **`registry[]`** execution agents + **`items[]`** recent task heartbeats (`stub: true` on **`items`** only when agent health is disabled) |
| `POST /api/openclaw/proxy` | **Not implemented** | Returns **501**; use server env `OPENCLAW_GATEWAY_*` + WS from service |
| `GET /api/workspaces` | **Live** | Allocated ports + `merge_queue_pending` when stores are wired (not a stub) |
| `GET /api/settings` | **Stub** | Minimal / empty JSON |

_**Progress** labels: **Live** = behavior backed by real stores/handlers; **Partial** = mixed real + conditional stub; **Stub** = placeholder payload; **Not implemented** = route returns 501 or equivalent._

There is no REST “session” resource. OpenClaw dispatch uses env (e.g. `ARMS_OPENCLAW_SESSION_KEY` on the server). A **browser-facing** gateway proxy (MC-style `/api/openclaw/*`) is optional; see §3.

---

## Implementation roadmap (vertical slices)

Rough calendar: **~4 weeks core (A–C)** + **polish (D)**; optional future below.

| Phase | Time (guide) | Deliverables |
|-------|----------------|--------------|
| **A — Production safety** | 1–2 wk | **Done (when `AgentHealth` wired):** MC convoy singular aliases (`/api/convoy/...`); **`GET /api/products/{id}/stalled-tasks`**; completion webhook + **`POST /api/tasks/{id}/complete`** → **`task_agent_health`** **`completed`** + **`task_completed`** outbox in **one SQLite transaction** (`LiveActivityTX.CompleteTaskWithEvent`); task **`sandbox_path` / `worktree_path`** (008–009). **Manual stall nudge:** **`POST /api/tasks/{id}/stall-nudge`** (optional JSON `{ "note" }`) → `status_reason` prefix + optional agent-health `stall_nudges[]` + SSE **`task_stall_nudged`**. **Merge queue ship:** FIFO head + **lease** + optional **real merge** (`ARMS_MERGE_BACKEND=github|local`), conflict/failure persisted on row; **`merge_ship_completed`** SSE; **autopilot merge policy** (tier + **`merge_policy_json`** gates, **semi_auto** gated auto-ship, **resolve** routes, **same-Tx outbox** on merge finish when SQLite outbox is wired) — see **Remarks — merge-queue autopilot policy** above. **Still open:** **auto**-nudge/reassign, multi-instance **DB leases** for task completion / product gates beyond merge queue. |
| **B — Full autonomy** | ~2 wk | Convoy: **done (baseline DAG semantics):** `convoy_subtasks.completed` (migration 011); dependents **`dispatch-ready`** only after upstream **completed**; webhook **`convoy_id` + `subtask_id`** + parent **`task_id`**; SSE **`convoy_subtask_dispatched`** / **`convoy_subtask_completed`**. **Baseline:** parent-task **agent-health gate** + **`dispatch_attempts`** / gateway retry cap (migration **018**). **Partial (2026-03):** migration **`022`** — convoy **`metadata_json`**, subtask **`title`** / **`metadata_json`** / **`dag_layer`**, **`convoy_mail`** **`kind`** / **`from_subtask_id`** / **`to_subtask_id`**; HTTP graph summary + subtask **`status`**, enriched mail POST. **TBD:** **`dominikbraun/graph`** (or equivalent), MC-exact convoy API/fields, cross-agent mailbox unification. **GitHub PR + post-execution (#60)** — **done (baseline):** REST + **`gh`**, duplicate recovery, agent + **CI** HMAC webhooks, auto PR/merge path for tiers. Deeper **ideas** scoring/metadata; **`swipe_history`** table + list API (**done**); **`preference_models`** — **heuristic learning loop** (**aggregate** swipe + **`product_feedback`** + maybe-pool → **`preference_models`**, ideation bias, event-driven recompute + **`POST …/recompute`**) **done**; **ML / embeddings** still **TBD**. |
| **C — Polish** | ~1 wk | **Agent** domain + listing/health APIs (replace stub). **`product_schedules`** on **Asynq** (Redis) — **done:** migration **017**, **`product:schedule:tick`**, cron + one-shot **`delay_seconds`**, HTTP fields on **`GET/PATCH …/product-schedule`**; optional follow-up: cancel/replace stale Redis tasks on schedule edits. Autopilot tick offload via Redis **done**. Optional **Ed25519** on OpenClaw `connect`. ~~**Maybe pool** batch re-eval~~ (**done:** migration **020**, **`POST …/maybe-pool/batch-reeval`**). ~~**`product_feedback`**~~ (**done:** migration **021**, **`POST`/`GET …/feedback`**, **`PATCH /api/product-feedback/{id}`**). ~~**HTTP aliases** `/api/convoy/*`~~ (done in A). |
| **D — Optional future** | — | Embedded UI (e.g. HTMX/templ), Postgres adapter, pure-Go agent runtime (replace OpenClaw). |

**Done in-tree (former “first commits”):** Compose **redis** service (optional); **`ARMS_REDIS_ADDR`** + **`cmd/arms-worker`** → **`product:schedule:tick`** + **`arms:product_autopilot_tick`** (+ optional **`arms:autopilot_tick`**), **`cmd/arms`** startup + **5m** resync, transactional **outbox** + **`livefeed`** SSE hub, **workspace** ports + merge queue + optional git worktrees, **GitHub** / **`gh`** behind `PullRequestPublisher`, **swipe_history**, **cost_caps** + composite budget, **task agent health** APIs, **`preference_models`** + **`operations_log`** (migrations 014–015).

**Next vertical slices (suggested):** **(1)** Convoy **`dominikbraun/graph`** + MC-exact DAG/mail parity (build on **`022`**). **(2)** **Knowledge auto-ingest** (feedback, swipes, completion summaries) + optional **semantic** store swap (**chromem** / Mem0 HTTP). **(3)** **Stalled-task reassign** when execution-agent binding exists (**§7**). **(4)** **ML / embeddings** on **`preference_models`**. **(5)** Optional **`/api/openclaw/*`** HTTP proxy, **operations_log** breadth, PR idempotency keys (#60 polish).

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

## Master backlog (all checklist items)

Flat index of every §1–§10 row below. **Workflow:** update `- [ ]` / `- [x]` in the numbered sections first, then set **Status** here to **Open** or **Done** for the matching `#` row so the table stays accurate.

| # | § | Status | Item |
|---:|---|:--:|------|
| 1 | 1 | Done | Optional **MC-compat alias routes** — `POST /api/convoy`, `GET /api/convoy/{id}`, `POST /api/convoy/{id}/dispatch-ready` → same handlers as `/api/convoys/...` |
| 2 | 1 | Done | Add HTTP server driving adapter (REST or minimal RPC) for orchestration — `cmd/arms`, `internal/adapters/httpapi` |
| 3 | 1 | Done | Map route groups analogous to MC: `tasks`, `products`, `agents`, `costs`, `convoy`, `openclaw`, `webhooks`, `events`/`live`, `workspaces`, `settings` — implemented or stubbed under `/api/...` (+ **`preference-model`**, **`operations-log`**, **`research-cycles`**, **`merge-queue`**, **`stalled-tasks`**, **`stall-nudge`**, etc. — see **`GET /api/docs/routes`**) |
| 4 | 1 | Done | Bearer auth middleware (`MC_API_TOKEN`-style) — env `MC_API_TOKEN`; omitted = dev open access |
| 5 | 1 | Done | SSE auth pattern (e.g. token query param) for live streams — `GET /api/live/events` uses `?token=` when auth enabled (`SSEQueryToken`) |
| 6 | 1 | Done | Request validation layer (DTOs + schema validation) — JSON DTOs + `validate()` helpers (no external schema lib yet) |
| 7 | 1 | Done | Agent-completion webhook receiver — `POST /api/webhooks/agent-completion` (parent task: `{ "task_id" }`; convoy subtask: add **`convoy_id`** + **`subtask_id`** with same **`task_id`** = parent) |
| 8 | 1 | Done | HMAC verification for webhooks (`WEBHOOK_SECRET`-style) — header `X-Arms-Signature` = hex(HMAC-SHA256(secret, raw body)) |
| 9 | 1 | Done | Route catalog documenting public API — `GET /api/docs/routes` |
| 10 | 1 | Done | Human-readable API reference — `docs/api-ref.md` |
| 11 | 1 | Done | OpenAPI 3.1 spec (hand-maintained) — `docs/openapi/arms-openapi.yaml` (import into Swagger UI / Redoc; not codegen-generated) |
| 12 | 2 | Done | SQLite adapter implementing repository ports — `internal/adapters/sqlite` (`ProductStore`, `IdeaStore`, `TaskStore`, `ConvoyStore`, `CostStore`, `CheckpointStore`) |
| 13 | 2 | Done | Migration runner + versioned migrations — embedded `migrations/*.sql`, `arms_schema_version`, `ExpectedSchemaVersion` constant (bump when adding files) |
| 14 | 2 | Done | Pre-migration backup — `ARMS_DB_BACKUP=1` runs `VACUUM INTO` to `{DATABASE_PATH}.pre-migrate-{UTC}.bak` before migrate |
| 15 | 2 | Done | Server wiring — `DATABASE_PATH` set → `platform.OpenApp` uses SQLite; empty → in-memory (same as before) |
| 16 | 2 | Done | Baseline schema in `001_initial.sql` + `002_kanban_tasks.sql` — `products`, `ideas`, `tasks` (TEXT Kanban `status` after v2), `convoys` / `convoy_subtasks`, `cost_events`, `checkpoints` (one payload row per task) |
| 17 | 2 | Done | Partial FK cascade — `ideas`, `tasks`, `convoys`, `cost_events` reference `products` with `ON DELETE CASCADE` where declared in migrations (not equivalent to all MC safety / soft-delete behavior) |
| 18 | 2 | Done | `products`: baseline MC-style profile — `repo_url`, `repo_branch`, `description`, `program_document`, `settings_json`, `icon_url` (migration `003_product_mc_metadata.sql`); HTTP `POST /api/products` optional fields + `PATCH /api/products/{id}`; profile text/repo hints passed through `domain.Product` into research/ideation ports (stubs use `ai.ProductContextSnippet`; real LLM adapters TBD) |
| 19 | 2 | Done | `research_cycles` — migration **`012_research_cycles.sql`**; append on successful **`RunResearch`**; **`GET /api/products/{id}/research-cycles`** (full MC “research graph” / analytics still TBD) |
| 20 | 2 | Done | **`ideas`** MC metadata — migration **019** (category, scores, complexity, effort, analysis fields, tags JSON, source, **`status`** + **`swiped_at`**, **`task_id`**, research cycle link, resurfacing fields, **`updated_at`**); swipe syncs status; task create → **`building`** + **`task_id`**; **`PATCH /api/ideas/{id}`** for operator edits |
| 21 | 2 | Done | `swipe_history` — migration `007_swipe_history.sql`; SQLite + memory stores; autopilot **Append** on swipe / promote-maybe; **`GET /api/products/{id}/swipe-history`** (`?limit=`) |
| 22 | 2 | Done | `preference_models` — migration **`014_preference_models.sql`**; **`GET` / `PUT /api/products/{id}/preference-model`**; **`POST …/preference-model/recompute`** — **heuristic** aggregate (**`swipe_history`** + **`product_feedback`** + maybe-pool) persisted as **`preference_aggregate_v1`**, ideation bias from stored model; refresh on swipe / feedback / maybe batch-reeval; **ML / embeddings** still **TBD** |
| 23 | 2 | Done | `maybe_pool` table + list/promote API — `MaybePoolRepository`; `GET /api/products/{id}/maybe-pool`, `POST /api/ideas/{id}/promote-maybe` (baseline; aligns with §5) |
| 24 | 2 | Done | Maybe pool **batch re-eval** — migration **020** (eval columns on **`maybe_pool`**); **`POST /api/products/{id}/maybe-pool/batch-reeval`** (`note`, `next_evaluate_delay_sec`); **`autopilot.BatchReevaluateMaybePool`**; **`GET …/maybe-pool`** returns **`maybe_*`** fields per idea (**resurface** automation vs MC still TBD) |
| 25 | 2 | Done | **`product_feedback`** — migration **021**; **`ProductFeedbackRepository`** (SQLite + memory); **`POST`/`GET /api/products/{id}/feedback`**, **`PATCH /api/product-feedback/{id}`** (`processed`); **`application/feedback`** |
| 26 | 2 | Done | `cost_events`: **agent**, **model** columns (`006_phase_a_safety.sql`); append + breakdown API |
| 27 | 2 | Done | `cost_caps` (daily + monthly + cumulative per product) + **`budget.Composite`** at dispatch |
| 28 | 2 | Done | `product_schedules` — **012** + **017**; **`GET` / `PATCH …/product-schedule`** (cron/delay + metadata); **`TickScheduled`** skips **`enabled: false`**; per-row **`product:schedule:tick`** when Redis (**#55**) |
| 29 | 2 | Done | `operations_log` — migration **`015_operations_log.sql`**; **`GET /api/operations-log`** with **`?action=`**, **`?resource_type=`**, **`?since=`** (RFC3339); append on key actions (extend coverage over time) |
| 30 | 2 | Open | Convoy **graph algorithms** + MC-exact DAG — **partial:** migration **`022`** (`metadata_json`, subtask **`title`**/**`metadata_json`**/**`dag_layer`**, mail **`kind`**/from/to); **`dominikbraun/graph`** (or equivalent) + stricter MC parity **TBD** |
| 31 | 2 | Done | `task_agent_health` (per-task; not full MC agent registry) — migration `009_agent_health_repo_path.sql` (table + `products.repo_clone_path` + `workspace_merge_queue.completed_at`) |
| 32 | 2 | Done | `tasks`: **`sandbox_path`**, **`worktree_path`** — migration `008_task_workspace_paths.sql` (metadata for isolation / worktrees; returned on task JSON; **`PATCH /api/tasks/{id}`** may set them) |
| 33 | 2 | Done | Checkpoint **history** + restore — `checkpoint_history` + APIs (latest still in `checkpoints`); MC **`work_checkpoints`** naming parity optional |
| 34 | 2 | Done | `agent_mailbox` — migration **`013_agents_mailbox.sql`** + **`GET/POST /api/agents/{id}/mailbox`** (baseline); **convoy / cross-agent mail** still **TBD** (§6) |
| 35 | 2 | Done | `workspace_ports` (4200–4299) + HTTP allocate/release |
| 36 | 2 | Done | `workspace_merge_queue` table + pending **count** in `GET /api/workspaces`; FIFO **head** completion + **`completed_at`** on done; **real ship** optional via **`ARMS_MERGE_BACKEND=github\|local`** (lease columns, merge outcome fields, **`mergequeue` service**); query **`skip_ship=1`**; **`DELETE …/merge-queue`**; **resolve** routes; enriched **GET …/merge-queue** + product **merge_queue_pending** / **merge_policy** (effective gates); **operations_log** merge / resolve + **product.patch**; overlap **#106** (policy + same-Tx finish outbox) |
| 37 | 2 | Open | Broader MC parity: soft deletes, extra cascade paths, concurrency guards, ops tooling |
| 38 | 3 | Done | Real `AgentGateway` adapter: WebSocket client — `internal/adapters/gateway/openclaw` ([coder/websocket](https://github.com/coder/websocket)) |
| 39 | 3 | Done | Config: `OPENCLAW_GATEWAY_URL`, `OPENCLAW_GATEWAY_TOKEN` (env) + `OPENCLAW_DISPATCH_TIMEOUT_SEC` (default 30) + `ARMS_DEVICE_ID` (optional `X-Arms-Device-Id`) |
| 40 | 3 | Done | Dispatch timeouts — per-call `context.WithTimeout` from `OpenClawDispatchTimeout` |
| 41 | 3 | Done | Reconnect on failure — drop cached conn after read/write error; next dispatch dials again (`App.Close` also closes client) |
| 42 | 3 | Done | Map gateway errors — task layer wraps adapter errors with `domain.ErrGateway` (existing `task.Service`) |
| 43 | 3 | Done | Device identity hint — `ARMS_DEVICE_ID` header on WS handshake (full MC device file parity still TBD) |
| 44 | 3 | Done | Native OpenClaw WebSocket framing (aligned with [mission-control `client.ts`](https://github.com/crshdn/mission-control/blob/main/src/lib/openclaw/client.ts)): `token` query param + optional Bearer, `connect.challenge` → `connect` RPC (protocol 3), dispatch via **`chat.send`** with `sessionKey`, `message`, `idempotencyKey` |
| 45 | 3 | Open | Ed25519 **device** block on `connect` (MC `device-identity.ts` signing) — **optional** behind env (e.g. `ARMS_DEVICE_SIGNING=enabled`); token-only default |
| 46 | 3 | Open | Optional HTTP proxy routes (`/api/openclaw/*` equivalent) if UI or ops need them |
| 47 | 4 | Done | Align `Task` status model with MC Kanban columns — string statuses (`planning` → `inbox` → `assigned` → `in_progress` → `testing` → `review` → `done`) plus `failed`, `convoy_active`; migration `002_kanban_tasks.sql` |
| 48 | 4 | Done | Planning gate + opaque planning JSON — `Task.ClarificationsJSON`, `UpdatePlanningArtifacts`; HTTP `PATCH /api/tasks/{id}` with `clarifications_json` while in `planning` (structured Q&A UX / spec editor still TBD) |
| 49 | 4 | Done | Plan approval + reject / recall — `ApprovePlan`, `ReturnToPlanning` (inbox or assigned before dispatch); HTTP `POST /api/tasks/{id}/plan/approve`, `POST /api/tasks/{id}/plan/reject` (optional `{ "status_reason" }`); Kanban moves via `PATCH /api/tasks/{id}` (`status`, `status_reason`) |
| 50 | 4 | Done | List tasks per product (board feed) — `GET /api/products/{id}/tasks`, `ports.TaskRepository.ListByProduct` (SQLite + memory), `404` if product missing |
| 51 | 4 | Open | Task images / attachments storage + API |
| 52 | 4 | Open | Distinguish manual task flow vs autopilot-derived tasks where MC does |
| 53 | 5 | Done | Product program / profile injection into research/ideation — stored on `Product` + HTTP; `ResearchPort` / `IdeationPort` godoc + `ai.ProductContextSnippet` + stub behavior (full MC “Product Program CRUD” UX still evolves with UI) |
| 54 | 5 | Done | Scheduling (interim): `research_cadence_sec`, `ideation_cadence_sec`, `last_auto_*` on product; `autopilot.Service.TickScheduled` / `TickProduct`; **production cadence** via Redis + **`cmd/arms-worker`**; **`cmd/arms`** startup + **5m** resync + HTTP hooks. **`ARMS_AUTOPILOT_TICK_SEC`** deprecated (ignored). `auto_dispatch_enabled` + tier stored; **task auto-dispatch from tier not wired** (manual dispatch unchanged). |
| 55 | 5 | Done | **`product_schedules`** **Asynq** — migration **017** (`cron_expr`, `delay_seconds`, task metadata); task **`product:schedule:tick`** on **`arms-worker`**; **`cmd/arms`** startup + 5m resync + PATCH hook **`ResyncProductSchedule`**; chains **`TickProduct`** then next enqueue. **`ARMS_AUTOPILOT_TICK_SEC`** follow-up **done** (deprecated); optional Inspector cancel on schedule edits. |
| 56 | 5 | Done | Background job — **`cmd/arms-worker`**: **`product:schedule:tick`**, **`arms:product_autopilot_tick`**, optional **`arms:autopilot_tick`** → **`TickScheduled`**; **`cmd/arms`** no in-process autopilot ticker. |
| 57 | 5 | Done | Preference data: swipe append to **`preference_model_json`** + **`swipe_history`**; **`GET/PUT …/preference-model`**; **`POST …/preference-model/recompute`** + event-driven refresh — **heuristic** **`preference_aggregate_v1`** (swipes + **`product_feedback`** + maybe-pool), ideation prompt bias; **ML / embeddings** still **TBD**. |
| 58 | 5 | Done | Maybe pool: baseline + **batch re-eval** — **`POST /api/products/{id}/maybe-pool/batch-reeval`** updates eval timestamps / counts / notes / **`next_evaluate_at`** (§2 **#24**); list includes **`maybe_*`** keys. |
| 59 | 5 | Done | Automation tiers: `automation_tier` enum `supervised` \| `semi_auto` \| `full_auto` on product + create/patch/JSON (behavioral differences beyond storage/TBD for dispatch). |
| 60 | 5 | Done | Post-execution chain: test → review → **automatic** PR + merge path — **`full_auto`/`semi_auto`** agent webhook **`next_board_status`**; Kanban → **`review`** auto-PR when head branch set; REST **`gh`** duplicate recovery; merge retries; **`POST /api/webhooks/ci-completion`** (same **`WEBHOOK_SECRET`**) for CI-driven **`testing`/`review`/`done`/`failed`**. Optional follow-up: stronger stable PR idempotency keys. |
| 61 | 5 | Done | GitHub **`PullRequestPublisher`** — `adapters/shipping` GitHub client (go-github v66) + noop; **`POST /api/tasks/{id}/pull-request`** (`head_branch`, optional `title`/`body`); **`ARMS_GITHUB_TOKEN`** / **`GITHUB_TOKEN`**; SSE **`pull_request_opened`** when URL returned. |
| 62 | 6 | Done | Persist baseline convoy + subtasks — SQLite + memory `ConvoyRepository` (deps + **dispatch** + **completion** + refs); not yet “full MC” metadata |
| 63 | 6 | Done | **Dependency gating** — a subtask is eligible for **`dispatch-ready`** only when all **`depends_on`** ids are **`completed`** (not merely dispatched); avoids firing downstream agents before upstream work is done |
| 64 | 6 | Done | **Subtask completion webhook** — `POST /api/webhooks/agent-completion` with **`task_id`** (parent) + **`convoy_id`** + **`subtask_id`** marks one subtask completed without completing the parent task |
| 65 | 6 | Done | **SSE** — **`convoy_subtask_dispatched`**, **`convoy_subtask_completed`** (same hub/outbox path as other live events when wired) |
| 66 | 6 | Open | Persist **full** convoy DAG as in MC — **partial:** **`022`** + domain **`ConvoySubtaskDagLayers`** + HTTP **`graph`** block; MC-complete schema/routes **TBD** |
| 67 | 6 | Done | Convoy mail — migrations **`016`** + **`022`**; **`GET`/`POST /api/convoys/{id}/mail`**; **`note`**/**`handoff`**/**`blocker`**, from/to subtask IDs |
| 68 | 6 | Done | Convoy **`dispatch-ready`**: optional **`AgentHealth`** gate on **parent** task (`stalled` / `error` / `failed` / `offline` / `dead` → no-op); **`dispatch_attempts`** on **`convoy_subtasks`** (migration **018**); cap (default 5) then **`ErrGateway`** |
| 69 | 6 | Done | Minimal HTTP — `POST /api/convoys`, `GET /api/convoys/{id}`, `GET /api/products/{id}/convoys`, `POST /api/convoys/{id}/dispatch-ready`; `convoy.Service.Get`, `ListByProduct`; `ports.ConvoyRepository.ListByProduct`; subtask **`completed`** in JSON |
| 70 | 6 | Open | API parity with MC convoy — **partial** after **`022`** (metadata, mail, computed **`status`**, **`graph`** summary); full MC field/route parity **TBD** |
| 71 | 6 | Open | Richer subtask model (agent config, retries, nudges) if required for parity |
| 72 | 7 | Done | Budget at **single-task** dispatch — **`budget.Composite`**: per-product **`cost_caps`** (daily / monthly / cumulative) + default cumulative when **no** caps row via **`ARMS_BUDGET_DEFAULT_CAP`** (default 100; set `0` to disable default ceiling) |
| 73 | 7 | Done | Budget at **convoy** `dispatch-ready` — **`POST …/dispatch-ready`** body **`estimated_cost`** (optional, default 0); **`budget.Composite`** per subtask dispatched in the wave |
| 74 | 7 | Done | Cost breakdown — **`GET /api/products/{id}/costs/breakdown`** (`from` / `to` query RFC3339); aggregates `by_agent`, `by_model` |
| 75 | 7 | Done | Workspace isolation: **optional git worktree** (`internal/adapters/workspace` + gated HTTP); paths still **metadata** on tasks + ports; operator must set **`repo_clone_path`** on product |
| 76 | 7 | Done | Port allocation **4200–4299** — `workspace_ports` + **`POST /api/workspace/ports`** / **`DELETE /api/workspace/ports/{port}`** |
| 77 | 7 | Done | Serialized merge queue **ordering** — only FIFO **head** per product can `POST .../merge-queue/complete` (`domain.ErrNotMergeQueueHead` → 409); optional **real merge** via **`ARMS_MERGE_BACKEND=github\|local`** (lease, conflict/failure left on pending row + **`merge_ship_completed`** SSE; **`skip_ship=1`** advances without forge); **`DELETE …/merge-queue`** operator dequeue |
| 78 | 7 | Done | Product-scoped **in-process** lock on task **Complete** (`task.ProductGate`); multi-instance would need DB leases later |
| 79 | 7 | Done | Checkpoint **history** + **restore** — `checkpoint_history` + **`GET /api/tasks/{id}/checkpoints`**, **`POST .../checkpoint/restore`** (`history_id`); latest row still in `checkpoints` |
| 80 | 7 | Done | Agent health — **task-scoped** heartbeats + SQLite/memory + HTTP (not full MC **agent** aggregate yet) |
| 81 | 7 | Done | Stalled detection — **`GET /api/products/{id}/stalled-tasks`** (`no_heartbeat` / `heartbeat_stale` for in_progress, testing, review, convoy_active) |
| 82 | 7 | Done | **Manual** stall nudge — **`POST /api/tasks/{id}/stall-nudge`** (execution statuses); **`task_stall_nudged`** SSE + agent-health detail `stall_nudges[]` |
| 83 | 7 | Done | **Auto**-nudge for stalled tasks — **`ARMS_AUTO_STALL_NUDGE_ENABLED`** + **`ARMS_AUTO_STALL_NUDGE_INTERVAL_SEC`** (enqueue from **`cmd/arms`**) + **`ARMS_AUTO_STALL_NUDGE_COOLDOWN_SEC`** / **`ARMS_AUTO_STALL_NUDGE_MAX_PER_DAY`**; Asynq **`arms:stall_autonudge_tick`**; reuses **`task_stall_nudged`** with **`source":"auto"`**; auto path preserves **`last_heartbeat_at`**. **Reassign** policy still **TBD** |
| 84 | 8 | Done | Domain outbox baseline — table `event_outbox` (`005_event_outbox.sql`); `internal/application/livefeed` (**Hub**, **OutboxPublisher**, **RunOutboxRelay**); SQLite path relays to SSE; in-memory path publishes directly to hub |
| 85 | 8 | Done | Same-transaction outbox for **SQLite** dispatch, checkpoint, cost, and **task completion** + agent-health **`completed`** (`LiveActivityTX`); other paths (e.g. PR opened) still best-effort after external I/O |
| 86 | 8 | Done | SSE transport — `GET /api/live/events` (hello + ping + activity `data:` lines; `SSEQueryToken` when auth on) |
| 87 | 8 | Done | SSE activity (partial) — **`task_dispatched`**, **`cost_recorded`**, **`checkpoint_saved`**, **`task_completed`** (SQLite same-tx + relay; in-memory hub on complete), **`task_stall_nudged`** (operator **`POST .../stall-nudge`**), **`pull_request_opened`**, **`merge_ship_completed`**, **`convoy_subtask_dispatched`**, **`convoy_subtask_completed`**; **`?product_id=`** filter; broader catalog + agent/type filters still TBD |
| 88 | 8 | Done | Operator chat queue — migration **`023`** **`task_chat_messages`**; **`GET /api/products/{id}/chat-queue`**; **`POST …/chat-queue/{messageId}/ack`**; queued notes via **`POST /api/tasks/{id}/chat`** `{queue:true}` |
| 89 | 8 | Done | Per-task chat — **`GET`/`POST /api/tasks/{id}/chat`** (chronological history, `author` operator\|agent\|system); SSE **`task_chat_message`**, **`task_chat_queue_ack`** |
| 90 | 8 | Done | Knowledge base — migration **`024`** **`knowledge_entries`** + **`knowledge_fts`** (FTS5); **`ports.KnowledgeRepository`**; **`knowledge.Service`**; product CRUD **`/api/products/{id}/knowledge`**; OpenClaw **`KnowledgeForDispatch`** injects ranked snippets on **task** + **convoy subtask** dispatch; **`ARMS_KNOWLEDGE_DISPATCH_SNIPPETS`**, **`ARMS_KNOWLEDGE_DISABLE_DISPATCH`** |
| 91 | 9 | Done | **Execution agent** registry + repository — **`execution_agents`** table; **`POST /api/agents`**, **`GET /api/agents`** includes **`registry[]`** |
| 92 | 9 | Done | **Mailbox** (stub) — **`agent_mailbox`** + **`GET/POST /api/agents/{id}/mailbox`** |
| 93 | 9 | Open | Registration and discovery/import flows (gateway-backed auto-provision) |
| 94 | 9 | Open | Full MC parity: agent config, gateway import, aggregate health across tasks |
| 95 | 9 | Done | Health-style data — **task agent heartbeats** + per-task / per-product agent-health routes + `GET /api/agents` **`items`** (`stub: true` only if **`AgentHealth == nil`**) |
| 96 | 10 | Done | Dedicated `internal/config` — `LoadFromEnv()`, env vars documented on `Config`; `httpapi.Config` is a type alias + `LoadConfig()` wrapper for the HTTP adapter |
| 97 | 10 | Done | Dockerfile for `arms` service — `arms/Dockerfile` (Alpine runtime, static `CGO_ENABLED=0` build) |
| 98 | 10 | Done | docker-compose — `arms/docker-compose.yml` (port 8080, `DATABASE_PATH=/data/arms.db`, named volume; OpenClaw/token env commented) |
| 99 | 10 | Done | **Redis** service in Compose — optional sidecar for **Asynq**; **`ARMS_REDIS_ADDR`** wired in **`internal/config`** for autopilot enqueue (**`cmd/arms`**) + worker (**`cmd/arms-worker`**) |
| 100 | 10 | Done | Production hardening doc — `docs/arms-production-hardening.md` (secrets, TLS termination, `wss://`, `NO_PROXY` / webhooks, persistence, **`arms-worker`** + Redis, containers, logging) |
| 101 | 10 | Done | Structured logging + request IDs — `log/slog` in `cmd/arms`, `X-Request-ID` + optional access log (`internal/adapters/httpapi/logging.go`); `ARMS_LOG_JSON`, `ARMS_ACCESS_LOG` |
| 102 | 10 | Done | Automated tests touching persistence + HTTP wiring — SQLite `repos_test` / `migrate_test`, `sqlite_app_test`, `platform/router_test`, application tests with memory/SQLite + gateway stub (`openclaw` tests use real WS to test client) |
| 103 | 10 | Done | Opt-in HTTP integration tests — `internal/integration/` with `//go:build integration`; run `go test -tags=integration ./internal/integration/...` (in-memory app + stub gateway; full product→task→dispatch flow) |
| 104 | 10 | Done | CI — `.github/workflows/arms.yml` runs `go test ./...` and `-tags=integration` on `arms/**` changes |
| 105 | 10 | Open | Contract tests against **live** OpenClaw gateway (optional env-gated job) |
| 106 | 5–7 | Done | **Autopilot merge policy + merge-queue polish** — `EffectiveMergeExecutionGates` + **`merge_policy_json`** overrides; **`semi_auto`** **`CompleteIfPolicyAllowsAuto`** + GitHub **`CheckMergeGates`**; **`full_auto`** ungated ship on **done**; **`POST …/merge-queue/resolve`** + **`POST /api/merge-queue/{id}/resolve`**; SQLite **`FinishShipWithOutbox`** (merge row + **`event_outbox`** one transaction). Operator **`…/complete`** bypasses gates. |

---

## 1. API surface and transport

- [x] Optional **MC-compat alias routes** — `POST /api/convoy`, `GET /api/convoy/{id}`, `POST /api/convoy/{id}/dispatch-ready` → same handlers as `/api/convoys/...`
- [x] Add HTTP server driving adapter (REST or minimal RPC) for orchestration — `cmd/arms`, `internal/adapters/httpapi`
- [x] Map route groups analogous to MC: `tasks`, `products`, `agents`, `costs`, `convoy`, `openclaw`, `webhooks`, `events`/`live`, `workspaces`, `settings` — implemented or stubbed under `/api/...` (+ **`preference-model`**, **`operations-log`**, **`research-cycles`**, **`merge-queue`**, **`stalled-tasks`**, **`stall-nudge`**, etc. — see **`GET /api/docs/routes`**)
- [x] Bearer auth middleware (`MC_API_TOKEN`-style) — env `MC_API_TOKEN`; omitted = dev open access
- [x] SSE auth pattern (e.g. token query param) for live streams — `GET /api/live/events` uses `?token=` when auth enabled (`SSEQueryToken`)
- [x] Request validation layer (DTOs + schema validation) — JSON DTOs + `validate()` helpers (no external schema lib yet)
- [x] Agent-completion webhook receiver — `POST /api/webhooks/agent-completion` (parent task: `{ "task_id" }`; convoy subtask: add **`convoy_id`** + **`subtask_id`** with same **`task_id`** = parent)
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
- [x] `research_cycles` — migration **`012_research_cycles.sql`**; append on successful **`RunResearch`**; **`GET /api/products/{id}/research-cycles`** (full MC “research graph” / analytics still TBD)
- [x] `ideas`: full scoring/metadata as in MC — migration **019**; **`status`** / **`swiped_at`** / **`task_id`** / research cycle / category / tags / analysis fields; **`PATCH /api/ideas/{id}`**; legacy **`impact`** / **`feasibility`** / swipe **`decided`** / **`decision`** retained
- [x] `swipe_history` — migration `007_swipe_history.sql`; SQLite + memory stores; autopilot **Append** on swipe / promote-maybe; **`GET /api/products/{id}/swipe-history`** (`?limit=`)
- [x] `preference_models` — migration **`014_preference_models.sql`**; **`GET` / `PUT /api/products/{id}/preference-model`**; **`POST …/preference-model/recompute`** + event-driven refresh — **heuristic** **`preference_aggregate_v1`** (swipes + **`product_feedback`** + maybe-pool), ideation bias; **ML / embeddings** still **TBD**
- [x] `maybe_pool` table + list/promote API — `MaybePoolRepository`; `GET /api/products/{id}/maybe-pool`, `POST /api/ideas/{id}/promote-maybe` (baseline; aligns with §5)
- [x] Maybe pool **batch re-eval** — migration **020**; **`POST /api/products/{id}/maybe-pool/batch-reeval`**; **`GET …/maybe-pool`** exposes **`maybe_*`** metadata (**automated resurface** vs full MC still TBD)
- [x] **`product_feedback`** — migration **021**; append/list/patch-processed APIs (**#25**)
- [x] `cost_events`: **agent**, **model** columns (`006_phase_a_safety.sql`); append + breakdown API
- [x] `cost_caps` (daily + monthly + cumulative per product) + **`budget.Composite`** at dispatch
- [x] `product_schedules` — table in **012** + **017** (Asynq fields); **`GET` / `PATCH /api/products/{id}/product-schedule`** (incl. **`cron_expr`**, **`delay_seconds`**, schedule metadata); **`TickScheduled`** skips products with **`enabled: false`**; **per-row Asynq** via **`product:schedule:tick`** when Redis + worker
- [x] `operations_log` — migration **`015_operations_log.sql`**; **`GET /api/operations-log`** with **`?action=`**, **`?resource_type=`**, **`?since=`** (RFC3339); append on key actions (extend coverage over time)
- [x] `convoys` / `convoy_subtasks` / **`convoy_mail`** — migration **`022`** + HTTP: convoy **`metadata_json`**, subtask **`title`** / **`metadata_json`** / **`dag_layer`** (computed on create), mail **`kind`** / from / to; **`GET`/`POST …/convoys/{id}/mail** — **TBD:** **`dominikbraun/graph`** (or equivalent), MC-exact DAG spec + cross-agent mailbox unification
- [x] `task_agent_health` (per-task; not full MC agent registry) — migration `009_agent_health_repo_path.sql` (table + `products.repo_clone_path` + `workspace_merge_queue.completed_at`)
- [x] `tasks`: **`sandbox_path`**, **`worktree_path`** — migration `008_task_workspace_paths.sql` (metadata for isolation / worktrees; returned on task JSON; **`PATCH /api/tasks/{id}`** may set them)
- [x] Checkpoint **history** + restore — `checkpoint_history` + APIs (latest still in `checkpoints`); MC **`work_checkpoints`** naming parity optional
- [x] `agent_mailbox` — migration **`013_agents_mailbox.sql`** + **`GET/POST /api/agents/{id}/mailbox`** (baseline); **convoy / cross-agent mail** still **TBD** (§6)
- [x] `workspace_ports` (4200–4299) + HTTP allocate/release
- [x] `workspace_merge_queue` table + pending **count** in `GET /api/workspaces`; FIFO **head** completion + **`completed_at`** on done; **real ship** optional via **`ARMS_MERGE_BACKEND=github|local`** (lease columns, merge outcome fields, **`mergequeue` service**); query **`skip_ship=1`** for break-glass metadata-only advance; **`DELETE /api/tasks/{id}/merge-queue`** cancels a pending row (non-head anytime; head when no active ship lease); **`POST …/merge-queue/resolve`** + **`POST /api/merge-queue/{id}/resolve`**; **`GET …/merge-queue`** returns **`head_task_id`**, **`pending_count`**, per-row **`queue_position`** / **`is_head`**; **`GET /api/products/{id}`** adds **`merge_queue_pending`** + parsed **`merge_policy`** (incl. effective **gate** flags); **`operations_log`** on enqueue / complete / cancel / resolve / **`product.patch`**
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
- [x] Scheduling (interim): `research_cadence_sec`, `ideation_cadence_sec`, `last_auto_*` on product; `autopilot.Service.TickScheduled` / `TickProduct`; **production cadence** via Redis + **`cmd/arms-worker`** (**`product:schedule:tick`**, **`arms:product_autopilot_tick`**); **`cmd/arms`** startup + **5m** resync + HTTP hooks. **`ARMS_AUTOPILOT_TICK_SEC`** deprecated (ignored). `auto_dispatch_enabled` + tier stored; **task auto-dispatch from tier not wired** (manual dispatch unchanged).
- [x] **Asynq + Redis** — **`product_schedules`** per-row **cron** + **delayed** jobs (**`product:schedule:tick`**); **`ARMS_AUTOPILOT_TICK_SEC` / in-process ticker deprecated** (2026-03).
- [x] Background job — **`cmd/arms-worker`** handles **`product:schedule:tick`**, **`arms:product_autopilot_tick`**, optional **`arms:autopilot_tick`** → **`TickScheduled`**; **`cmd/arms`** no longer runs an in-process autopilot ticker.
- [x] Preference data: swipe append to **`preference_model_json`** + **`swipe_history`**; **`GET/PUT …/preference-model`**; **`POST …/preference-model/recompute`** + event-driven refresh — **heuristic** aggregate (**swipes** + **`product_feedback`** + maybe-pool) → **`preference_models`**, ideation prompt bias; **ML / embeddings** still **TBD**.
- [x] Maybe pool: `maybe_pool` + **`MaybePoolRepository`**; swipe **`maybe`** adds; **`GET /api/products/{id}/maybe-pool`** (with **`maybe_*`** fields); **`POST /api/ideas/{id}/promote-maybe`**; **`POST …/maybe-pool/batch-reeval`** (§2 **#24**).
- [x] Automation tiers: `automation_tier` enum `supervised` | `semi_auto` | `full_auto` on product + create/patch/JSON (behavioral differences beyond storage/TBD for dispatch).
- [x] **Merge-queue autopilot policy** — tier-derived **merge execution gates** (`require_approved_review`, `require_clean_mergeable` defaults + **`merge_policy_json`** overrides); **`full_auto`** → **`MergeShip.Complete`** on task **done**; **`semi_auto`** → **`CompleteIfPolicyAllowsAuto`** (GitHub gates when **`ARMS_MERGE_BACKEND=github`**); **`supervised`** no unattended ship; operator **`POST …/merge-queue/complete`** / **resolve** routes **ignore** gates (**#106**).
- [ ] Post-execution chain: test → review → **automatic** PR + merge — **partial:** **`full_auto`** / **`semi_auto`**: Kanban **`testing`/`in_progress`/`convoy_active` → `review`** opens PR when head branch set and no URL; **`done`** runs **`ensurePullRequestForAutoMerge`** then merge-queue ship with **merge retries** (transient **`ErrShipping`**, not conflict / **`ErrShippingNonRetryable`**); **`OpenPullRequest`** — **SQLite `LiveTX`**: task + **`pull_request_opened`** outbox; **PR create retries**; **GitHub 422 duplicate open PR** → list-by-head recovery; **webhook** optional **`next_board_status`**: **`testing`** / **`review`** for tiered board advance (post-exec “test phase” hook). **Still TBD:** CI-driven status signals beyond webhook, **gh** duplicate-PR parity, stronger PR dedupe keys.
- [x] GitHub **`PullRequestPublisher`** — `adapters/shipping` GitHub client (go-github v66) + noop; **`POST /api/tasks/{id}/pull-request`** (`head_branch`, optional `title`/`body`); **`ARMS_GITHUB_TOKEN`** / **`GITHUB_TOKEN`**; SSE **`pull_request_opened`** when URL returned.

---

## 6. Convoy mode

- [x] Persist convoy + subtasks — SQLite + memory `ConvoyRepository` (deps + **dispatch** + **completion** + refs); migration **`022`** adds convoy/subtask metadata + **`dag_layer`** (see §2)
- [x] **Dependency gating** — a subtask is eligible for **`dispatch-ready`** only when all **`depends_on`** ids are **`completed`** (not merely dispatched); avoids firing downstream agents before upstream work is done
- [x] **Subtask completion webhook** — `POST /api/webhooks/agent-completion` with **`task_id`** (parent) + **`convoy_id`** + **`subtask_id`** marks one subtask completed without completing the parent task
- [x] **SSE** — **`convoy_subtask_dispatched`**, **`convoy_subtask_completed`** (same hub/outbox path as other live events when wired)
- [ ] Persist **full** convoy DAG / graph algorithms as in MC — **partial:** **`022`** + computed **`dag_layer`** + HTTP **`graph`** summary; **`dominikbraun/graph`** (or equivalent) + MC-complete payloads **TBD**
- [x] Convoy mail / inter-subtask messaging — migrations **`016`** + **`022`**; **`GET`/`POST …/convoys/{id}/mail`**; kinds + from/to subtask (**cross-agent** unification with **`agent_mailbox`** still **TBD** — §2)
- [x] Integrate convoy dispatch with agent health and retries — parent-task health gate + **`dispatch_attempts`** + retry cap (**018**)
- [x] Minimal HTTP — `POST /api/convoys`, `GET /api/convoys/{id}`, `GET /api/products/{id}/convoys`, `POST /api/convoys/{id}/dispatch-ready`; `convoy.Service.Get`, `ListByProduct`; `ports.ConvoyRepository.ListByProduct`; subtask **`completed`** in JSON
- [ ] API parity with MC convoy — **partial** after **`022`** (metadata, mail, **`status`**, **`graph`**); full MC routes/fields **TBD** (**naming / singular aliases:** done — §1)
- [ ] Richer subtask model (agent config, retries, nudges) if required for parity

---

## 7. Safety, cost, workspace

- [x] Budget at **single-task** dispatch — **`budget.Composite`**: per-product **`cost_caps`** (daily / monthly / cumulative) + default cumulative when **no** caps row via **`ARMS_BUDGET_DEFAULT_CAP`** (default 100; set `0` to disable default ceiling)
- [x] Budget at **convoy** `dispatch-ready` — **`POST …/dispatch-ready`** body **`estimated_cost`** (optional, default 0); **`budget.Composite`** per subtask dispatched in the wave
- [x] Cost breakdown — **`GET /api/products/{id}/costs/breakdown`** (`from` / `to` query RFC3339); aggregates `by_agent`, `by_model`
- [x] Workspace isolation: **optional git worktree** (`internal/adapters/workspace` + gated HTTP); paths still **metadata** on tasks + ports; operator must set **`repo_clone_path`** on product
- [x] Port allocation **4200–4299** — `workspace_ports` + **`POST /api/workspace/ports`** / **`DELETE /api/workspace/ports/{port}`**
- [x] Serialized merge queue **ordering** — only FIFO **head** per product can `POST .../merge-queue/complete` (`domain.ErrNotMergeQueueHead` → 409); optional **real merge** via **`ARMS_MERGE_BACKEND=github|local`** (lease, conflict/failure left on pending row + **`merge_ship_completed`** SSE; **`skip_ship=1`** advances without forge); operator **`DELETE .../merge-queue`** to dequeue pending; **`POST …/merge-queue/resolve`** + **`POST /api/merge-queue/{id}/resolve`** (`retry_merge` / `skip_ship`); SQLite **same-Tx** queue finish + outbox when **`OutboxPublisher`** wired (**#106**)
- [x] Product-scoped **in-process** lock on task **Complete** (`task.ProductGate`); multi-instance would need DB leases later
- [x] Checkpoint **history** + **restore** — `checkpoint_history` + **`GET /api/tasks/{id}/checkpoints`**, **`POST .../checkpoint/restore`** (`history_id`); latest row still in `checkpoints`
- [x] Agent health — **task-scoped** heartbeats + SQLite/memory + HTTP (not full MC **agent** aggregate yet)
- [x] Stalled detection — **`GET /api/products/{id}/stalled-tasks`** (`no_heartbeat` / `heartbeat_stale` for in_progress, testing, review, convoy_active)
- [x] **Manual** stall nudge — **`POST /api/tasks/{id}/stall-nudge`** (execution statuses); **`task_stall_nudged`** SSE + agent-health detail `stall_nudges[]`
- [x] **Auto**-nudge for stalled tasks — **`ARMS_AUTO_STALL_NUDGE_*`** (default **off**); **`arms:stall_autonudge_tick`** via Redis + **`cmd/arms-worker`**; **`StalledTaskState`** + per-task cooldown and optional rolling 24h cap from **`stall_nudges`** entries with **`auto:`** note prefix; **`task_stall_nudged`** includes **`source":"auto"`** when applicable; auto nudge does **not** advance **`last_heartbeat_at`** so tasks stay stalled until a real heartbeat
- [ ] **Reassign** policy for stalled tasks (deferred — needs task↔execution-agent model / **#93**/**#94**)

---

## 8. Realtime and observability

- [x] Domain outbox baseline — table `event_outbox` (`005_event_outbox.sql`); `internal/application/livefeed` (**Hub**, **OutboxPublisher**, **RunOutboxRelay**); SQLite path relays to SSE; in-memory path publishes directly to hub
- [x] Same-transaction outbox for **SQLite** dispatch, checkpoint, cost, **task completion** + agent-health **`completed`** (`LiveActivityTX`), and **merge-queue ship finish** (`FinishShipWithOutbox` when events go through **`OutboxPublisher`**); other paths (e.g. **PR opened** after forge round-trip) still best-effort after external I/O
- [x] SSE transport — `GET /api/live/events` (hello + ping + activity `data:` lines; `SSEQueryToken` when auth on)
- [x] SSE activity (partial) — **`task_dispatched`**, **`cost_recorded`**, **`checkpoint_saved`**, **`task_completed`** (SQLite same-tx + relay; in-memory hub on complete), **`task_stall_nudged`** (manual **`POST .../stall-nudge`** or auto **`arms:stall_autonudge_tick`** with optional **`source":"auto"`**), **`pull_request_opened`**, **`merge_ship_completed`**, **`convoy_subtask_dispatched`**, **`convoy_subtask_completed`**; **`?product_id=`** filter; broader catalog + agent/type filters still TBD
- [x] Operator chat: product-scoped **queue** of pending operator notes — migration **`023`**; **`GET /api/products/{id}/chat-queue`**; **`POST /api/products/{id}/chat-queue/{messageId}/ack`**; append via **`POST /api/tasks/{id}/chat`** with **`queue: true`**
- [x] Per-task chat history — **`GET` / `POST /api/tasks/{id}/chat`** (`?limit=`); authors **`operator`** \| **`agent`** \| **`system`**; realtime **`task_chat_message`** / **`task_chat_queue_ack`** on same publisher path as other live events (outbox when **`OutboxPublisher`** wired)
- [x] Learner / knowledge base — SQLite **FTS5** (**`024`**), **`KnowledgeRepository`**, HTTP **`POST/GET/PATCH/DELETE …/knowledge`**, dispatch injection via **`openclaw.Options.KnowledgeForDispatch`** (task + convoy subtask); disable with **`ARMS_KNOWLEDGE_DISABLE_DISPATCH`**; **auto-ingest** from feedback/swipes and **semantic** backends still optional follow-ups

---

## 9. Agents domain

- [x] **Execution agent** registry + repository — **`execution_agents`** table; **`POST /api/agents`**, **`GET /api/agents`** includes **`registry[]`**
- [x] **Mailbox** (stub) — **`agent_mailbox`** + **`GET/POST /api/agents/{id}/mailbox`**
- [ ] Registration and discovery/import flows (gateway-backed auto-provision)
- [ ] Full MC parity: agent config, gateway import, aggregate health across tasks
- [x] Health-style data — **task agent heartbeats** + per-task / per-product agent-health routes + `GET /api/agents` **`items`** (`stub: true` only if **`AgentHealth == nil`**)

---

## 10. Cross-cutting platform

- [x] Dedicated `internal/config` — `LoadFromEnv()`, env vars documented on `Config`; `httpapi.Config` is a type alias + `LoadConfig()` wrapper for the HTTP adapter
- [x] Dockerfile for `arms` service — `arms/Dockerfile` (Alpine runtime, static `CGO_ENABLED=0` build)
- [x] docker-compose — `arms/docker-compose.yml` (port 8080, `DATABASE_PATH=/data/arms.db`, named volume; OpenClaw/token env commented)
- [x] **Redis** service in Compose — optional sidecar for **Asynq**; **`ARMS_REDIS_ADDR`** wired in **`internal/config`** for autopilot enqueue (**`cmd/arms`**) + worker (**`cmd/arms-worker`**)
- [x] Production hardening doc — `docs/arms-production-hardening.md` (secrets, TLS termination, `wss://`, `NO_PROXY` / webhooks, persistence, **`arms-worker`** + Redis, containers, logging)
- [x] Structured logging + request IDs — `log/slog` in `cmd/arms`, `X-Request-ID` + optional access log (`internal/adapters/httpapi/logging.go`); `ARMS_LOG_JSON`, `ARMS_ACCESS_LOG`
- [x] Automated tests touching persistence + HTTP wiring — SQLite `repos_test` / `migrate_test`, `sqlite_app_test`, `platform/router_test`, application tests with memory/SQLite + gateway stub (`openclaw` tests use real WS to test client)
- [x] Opt-in HTTP integration tests — `internal/integration/` with `//go:build integration`; run `go test -tags=integration ./internal/integration/...` (in-memory app + stub gateway; full product→task→dispatch flow)
- [x] CI — `.github/workflows/arms.yml` runs `go test ./...` and `-tags=integration` on `arms/**` changes
- [ ] Contract tests against **live** OpenClaw gateway (optional env-gated job)

---

## Quick reference

| Area            | Rough priority for a vertical slice                         |
|-----------------|--------------------------------------------------------------|
| SQLite + core tables | Unblocks everything else (current **v24** migrations)     |
| HTTP + auth + tasks/products | Makes the service usable from a UI or CLI            |
| Real OpenClaw WS   | Closes the execution-plane gap                            |
| Webhooks           | Completes the async completion loop                       |
| SSE + costs + workspace | Match MC ops and safety story                          |
| Asynq + **`cmd/arms-worker`** | **`product:schedule:tick`**, **`arms:product_autopilot_tick`**, optional **`arms:autopilot_tick`** — **`ARMS_AUTOPILOT_TICK_SEC` deprecated**; **`cmd/arms`** **5m** resync |
| Operations log + preference API | Audit trail + structured preference storage (ML later)   |
| Stub routes → real domains | Settings stub, **`/api/openclaw/*`** proxy (if needed)   |
| Roadmap phases A→D | See **Implementation roadmap** above                      |
