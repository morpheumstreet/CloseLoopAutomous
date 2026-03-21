# Arms HTTP API reference

REST surface for the `arms` service (`cmd/arms`). **JSON** request and response bodies unless noted. Path parameters use `{id}` style as in the router.

**Canonical machine-readable list:** `GET /api/docs/routes` returns `{ "routes": [ { "method", "path", "description" }, ... ] }` (same inventory as `internal/adapters/httpapi/routes_catalog.go`).

**OpenAPI 3.1:** [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — use with Swagger UI, Redocly, or Postman import.

---

## Authentication

| Mode | When |
|------|------|
| **None** | `MC_API_TOKEN` is unset — all protected routes are open (dev default). |
| **Bearer** | Set `MC_API_TOKEN`. Send `Authorization: Bearer <token>` on API calls. The same header is accepted on **`GET /api/live/events`** when auth is enabled (for fetch-based or custom SSE clients). |
| **Same-origin** | If `ARMS_ALLOW_SAME_ORIGIN=1` or `true`, browser requests from the same origin may omit Bearer when a token is configured. |

**Unauthenticated by design:** `GET /api/health`, `GET /api/docs/routes`, `POST /api/webhooks/agent-completion`. **`GET /api/live/events`** is open only when **`MC_API_TOKEN` is unset** and **`ARMS_ACL`** is empty; otherwise see **SSE** below.

### Request correlation

- Every response includes **`X-Request-ID`**. Send the same header on the request to propagate a client-generated trace id; otherwise the server generates a UUID.
- With access logging enabled (default), each request emits one **`http_request`** line via `log/slog` (stdout): `method`, `path`, `status`, `duration_ms`, `request_id`.
- **`ARMS_LOG_JSON`**: `1` or `true` → JSON log lines; default is text.
- **`ARMS_ACCESS_LOG`**: `0`, `false`, `off`, or `no` → disable per-request access logs (request id header is still set).

---

## Errors

Failed requests typically return JSON:

```json
{ "error": "message", "code": "optional_code" }
```

Common `code` values from domain mapping include `not_found`, `invalid_transition`, `conflict`, `merge_queue_head`, `merge_conflict`, `merge_lease_busy`, `budget_exceeded`, `gateway`, `shipping`, `invalid_signature`, etc.

---

## Products and ideas

| Method | Path | Body (JSON) | Notes |
|--------|------|-------------|--------|
| POST | `/api/products` | `name`, `workspace_id`; optional profile fields; optional **`merge_policy_json`** (must parse as JSON); optional `research_cadence_sec`, `ideation_cadence_sec` (≥0, 0=off), `automation_tier` (`supervised` \| `semi_auto` \| `full_auto`), `auto_dispatch_enabled` | Create product (Mission Control–style profile + autopilot metadata). Invalid **`merge_policy_json`** → **400** `invalid_input`. |
| GET | `/api/products` | — | `{ "products": [ … ] }` — list products (dashboards / UIs). |
| GET | `/api/products/{id}` | — | Response includes profile fields, cadence/tier, `preference_model_json`, **`merge_policy_json`**, parsed **`merge_policy`** (`merge_method`, optional `merge_backend_override`), **`merge_queue_pending`** (when merge queue is wired), optional `last_auto_*` timestamps. |
| PATCH | `/api/products/{id}` | Any subset of profile + autopilot fields above; optional **`merge_policy_json`** (string, JSON for `merge_method` / `merge_backend`) | At least one field required. Does not change pipeline `stage` (use research/ideation). Appends **`operations_log`** row **`product.patch`** (fields touched). |
| PATCH | `/api/products/{id}/cost-caps` | At least one of: `daily_cap`, `monthly_cap`, `cumulative_cap` (numbers) | **Negative** value for an axis **clears** that limit (unlimited on that axis). Upserts `cost_caps` row. |
| GET | `/api/products/{id}/costs/breakdown` | — | Query: optional `from`, `to` (RFC3339 / RFC3339Nano). JSON: `total`, `events[]`, `by_agent`, `by_model`. |
| POST | `/api/products/{id}/research` | — | Run research phase. The full product record (including `program_document`, `description`, repo fields) is passed to the research port for prompt context. |
| POST | `/api/products/{id}/ideation` | — | Run ideation phase. Same product context plus stored `research_summary`. |
| GET | `/api/products/{id}/ideas` | — | `{ "ideas": [ … ] }` |
| GET | `/api/products/{id}/maybe-pool` | — | `{ "ideas": [ … ] }` for ideas swiped `maybe`. |
| GET | `/api/products/{id}/swipe-history` | — | Query: optional `limit` (default 100, max 500). `{ "swipes": [ { id, idea_id, product_id, decision, created_at } ] }`, newest first (audit log; complements `preference_model_json`). |
| GET | `/api/products/{id}/preference-model` | — | `{ "product_id", "model_json", "source": "preference_models" \| "legacy", "updated_at" }`. Uses dedicated **`preference_models`** row when present; otherwise falls back to **`preference_model_json`** on the product (default empty legacy → `[]`). |
| PUT | `/api/products/{id}/preference-model` | `{ "model_json": "<JSON string>" }` | Upserts **`preference_models`** for the product (`model_json` must parse as JSON). Response matches GET with **`source`: `preference_models`**. |
| POST | `/api/products/{id}/preference-model/recompute` | — | Query: optional **`limit`** (default 500, max 5000). Aggregates **`swipe_history`** decision counts into **`preference_models`** JSON (`source`, `counts`, `sample_size`). |
| GET | `/api/products/{id}/product-schedule` | — | **`product_schedules`** row or defaults: **`enabled`** (default true when no row), **`spec_json`**, optional **`updated_at`**. |
| PATCH | `/api/products/{id}/product-schedule` | At least one of **`enabled`**, **`spec_json`** | Upserts schedule. When **`enabled`** is **`false`**, **`TickScheduled`** skips that product (autopilot cadence). **503** if schedule store not wired. |
| GET | `/api/products/{id}/research-cycles` | — | Query: optional `limit` (default 50, max 500). `{ "research_cycles": [ { id, product_id, summary_snapshot, created_at } ] }` — append-only history when each **`POST …/research`** succeeds (SQLite/memory with persistence). |
| GET | `/api/products/{id}/merge-queue` | — | Query: optional `limit` (default 50, max 500). JSON: **`merge_queue`**, **`pending_count`** (full count, not capped by `limit`), **`head_task_id`** (first pending). Each row: **`queue_position`** (1-based), **`is_head`**, plus **`lease_owner`**, **`lease_expires_at`**, **`merge_ship_state`**, **`merged_sha`**, **`merge_error`**, **`conflict_files`** when relevant. **503** if merge queue is not configured. |
| POST | `/api/ideas/{id}/swipe` | `decision`: `pass` \| `maybe` \| `yes` \| `now` | Appends to `preference_model_json`; `maybe` enqueues pool. |
| POST | `/api/ideas/{id}/promote-maybe` | — | Requires prior `maybe` swipe; sets decision to yes and removes from pool. |

---

## Tasks (Kanban)

Task JSON includes at least: `id`, `product_id`, `idea_id`, `spec`, `status` (string: `planning`, `inbox`, `assigned`, `in_progress`, `testing`, `review`, `done`, `failed`, `convoy_active`), `status_reason`, `plan_approved`, `clarifications_json`, `checkpoint`, `external_ref`, `sandbox_path`, `worktree_path`, optional **`pull_request_url`**, **`pull_request_number`**, **`pull_request_head_branch`** (after `POST …/pull-request`), `created_at`, `updated_at`.

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/products/{id}/tasks` | — | `{ "tasks": [ … ] }`, newest first. 404 if product missing. |
| POST | `/api/tasks` | `idea_id`, `spec` | Creates task in **`planning`** (idea must be approved). |
| GET | `/api/tasks/{id}` | — | |
| PATCH | `/api/tasks/{id}` | Any of: `status`, `status_reason`, `clarifications_json` | At least one field required. Clarifications only while in `planning`. **Full-auto:** when product `automation_tier` is **`full_auto`**, moving **`testing` → `review`** or **`in_progress` → `review`** may **auto-open a PR** if the task already has **`pull_request_head_branch`** set and **`pull_request_url`** is still empty (best-effort; uses same shipper as `POST …/pull-request`). When a task reaches **`done`** under **`full_auto`**, the server **best-effort** calls merge-queue **`Complete`** (same as **`POST …/merge-queue/complete`**) so **`ARMS_MERGE_BACKEND=github|local`** can ship without a separate HTTP call if this task owns the FIFO head. |
| POST | `/api/tasks/{id}/plan/approve` | Optional `{ "spec" }` | `planning` → `inbox`, `plan_approved: true`. |
| POST | `/api/tasks/{id}/plan/reject` | Optional `{ "status_reason" }` | Back to **`planning`** from **`inbox`** or **`assigned`** (blocked after dispatch / `external_ref` set). |
| POST | `/api/tasks/{id}/dispatch` | `estimated_cost` (number) | Requires **`assigned`** + approved plan. Enforces **`budget.Composite`** (caps + default cumulative). **402** + **`budget_exceeded`** if `estimated_cost` would exceed allowed spend. |
| POST | `/api/tasks/{id}/pull-request` | `head_branch` (required), optional `title`, `body` | Opens a PR using `product.repo_url` (GitHub.com or GitHub-like path on GHES) and `product.repo_branch` as base (default `main`). **REST** / **`gh`** backends as in config. Persists **`pull_request_*`** on the task when a URL is returned. Response `{ "pr_url": "...", "pr_number": <int> }` (`pr_number` omitted if unknown). Allowed while task is `in_progress`, `testing`, `review`, or `done`. |
| POST | `/api/tasks/{id}/merge-queue` | — | Enqueues the task on the product’s **serialized merge queue** (FIFO by row `id`). **201** `{ "status": "queued" }`. **409** `conflict` if this task already has a **pending** row. **503** if merge queue is not configured. **`operations_log`:** **`merge_queue.enqueue`**. |
| DELETE | `/api/tasks/{id}/merge-queue` | — | Removes this task’s **pending** queue row. Allowed for **non-head** entries anytime; for the **head**, allowed only when there is **no active merge ship lease** (otherwise **503** **`merge_lease_busy`**). **404** if not queued. **`operations_log`:** **`merge_queue.cancel`**. |
| POST | `/api/tasks/{id}/merge-queue/complete` | Query: optional **`skip_ship=1`** or **`skip_real_merge=1`** to advance the queue without calling GitHub/git | **Head-only** (same as before). With **`ARMS_MERGE_BACKEND=github`**, merges **`pull_request_number`** via REST (needs token + PR opened first). With **`local`**, runs **`git merge`** in **`product.repo_clone_path`** (needs **`pull_request_head_branch`**). **409** **`merge_conflict`** on conflict; **503** **`merge_lease_busy`** if another instance holds the lease. Default backend **`noop`** keeps metadata-only completion. **`operations_log`:** **`merge_queue.complete`** (body includes **`skip_ship`**). |
| GET | `/api/tasks/{id}/checkpoints` | — | Query: optional `limit` (default 50, max 500). `{ "checkpoints": [ { id, task_id, payload, created_at } ] }` newest first. |
| POST | `/api/tasks/{id}/checkpoint/restore` | `{ "history_id": <int> }` | Restores payload via same rules as recording a checkpoint. |
| POST | `/api/tasks/{id}/checkpoint` | `payload` | Appends **`checkpoint_history`** and updates latest `checkpoints` row + task. |
| POST | `/api/tasks/{id}/complete` | — | **`done`** from `in_progress` / `testing` / `review`. With SQLite, **`task_agent_health`** **`completed`** and SSE **`task_completed`** are written in the **same transaction** as the task row. **`full_auto`:** after success, same **best-effort** merge-queue **`Complete`** as Kanban **`done`**. |
| POST | `/api/tasks/{id}/stall-nudge` | Optional `{ "note": "..." }` (empty body OK) | **Phase A** operator nudge for **`in_progress`**, **`testing`**, **`review`**, **`convoy_active`**: prepends `[stall_nudge <ts>]` to **`status_reason`**, appends **`stall_nudges[]`** to agent-health **`detail`** when health is wired, emits **`task_stall_nudged`** on the live stream. |

**Typical flow:** create task → (optional) PATCH `clarifications_json` → POST `plan/approve` → PATCH `status` to `assigned` → POST `dispatch` → checkpoint / complete / webhook.

**Merge queue:** enqueue / **cancel** / complete via the task routes above; **`POST …/merge-queue/complete` succeeds only for the FIFO head** (see **`merge_queue_head`** in Errors). **`DELETE …/merge-queue`** drops a waiter or dequeues the head when safe (no active lease). Env: **`ARMS_MERGE_BACKEND`** (`noop` \| `github` \| `local`), **`ARMS_MERGE_METHOD`** (`merge` \| `squash` \| `rebase`), **`ARMS_MERGE_LEASE_SEC`**, **`ARMS_MERGE_LEASE_OWNER`**. List pending rows under **Products and ideas**.

---

## Convoys

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/products/{id}/convoys` | — | `{ "convoys": [ … ] }` |
| POST | `/api/convoys` | `parent_task_id`, `product_id`, `subtasks[]` with `agent_role`, optional `id`, `depends_on` | **`depends_on`** must reference subtask ids in the same payload; **cycles** and **unknown deps** → **400** **`invalid_input`**. |
| GET | `/api/convoys/{id}` | — | |
| GET | `/api/convoys/{id}/mail` | — | Query: optional **`limit`**. **`{ "messages": [ { id, convoy_id, subtask_id, body, created_at } ] }`**. **503** **`not_configured`** if mail store not wired. |
| POST | `/api/convoys/{id}/mail` | `{ "subtask_id", "body" }` | Append-only convoy mail (**201** `{ "status": "ok" }`). **503** if mail not configured. |
| POST | `/api/convoys/{id}/dispatch-ready` | JSON **`{ "estimated_cost": <number> }`** (optional; empty body = **0**) | Dispatches one ready wave of subtasks. Each subtask about to be sent is checked with **`budget.Composite`** using **`estimated_cost`** (same semantics as **`POST …/dispatch`** per dispatch). **402** **`budget_exceeded`** when caps would be exceeded. |

---

## Costs

| Method | Path | Body | Notes |
|--------|------|------|--------|
| POST | `/api/costs` | `product_id`, `task_id`, `amount`, optional `note`, optional **`agent`**, **`model`** | Same **`budget.Composite`** rules as task dispatch: **`cost_caps`** (daily / monthly / cumulative) plus default cumulative when no caps row (**`ARMS_BUDGET_DEFAULT_CAP`**). **402** + code **`budget_exceeded`** if the new amount would exceed the allowed spend. |

---

## Operations log (audit)

| Method | Path | Body | Notes |
|--------|------|------|-------|
| GET | `/api/operations-log` | — | Query: **`limit`**, **`product_id`**, **`action`**, **`resource_type`**, **`since`** (RFC3339 / RFC3339Nano lower bound). `{ "entries": [ … ] }` newest first. Appends on product/task/dispatch/preference/schedule/convoy mail actions (coverage grows over time). |

---

## Workspace (ports)

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/workspaces` | — | `{ "ports": [ { port, product_id, task_id, allocated_at } ], "merge_queue_pending": <int> }`. If workspace ports or merge queue are not wired, response is **`{ "ports": [], "merge_queue_pending": 0, "stub": true }`**. |
| POST | `/api/workspace/ports` | `product_id`, `task_id` | Allocates first free port in **4200–4299**. **503** if exhausted or ports store not configured. |
| DELETE | `/api/workspace/ports/{port}` | — | Releases port; **404** if not allocated. |

---

## Webhook (agent completion)

| Method | Path | Auth |
|--------|------|------|
| POST | `/api/webhooks/agent-completion` | **HMAC**, not Bearer |

- Header: **`X-Arms-Signature`** = lowercase hex **HMAC-SHA256**(`WEBHOOK_SECRET`, raw request body).
- Body (parent task completion): `{ "task_id": "<id>" }`.
- Body (convoy subtask, without completing parent): `{ "task_id": "<parent_task_id>", "convoy_id": "<id>", "subtask_id": "<id>" }` — both **`convoy_id`** and **`subtask_id`** are required together; marks the subtask **completed** for DAG gating.
- Requires `WEBHOOK_SECRET` set; otherwise **503**.

---

## SSE (live stream)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/live/events` | `text/event-stream`. When **`MC_API_TOKEN`** is set: **`Authorization: Bearer <token>`** or **`?token=<same value>`** (native **`EventSource`** only supports the query form). When only **`ARMS_ACL`** is configured: **`?basic=<base64(user:password)>`**. Optional **`product_id=`** — only forward `data:` lines whose JSON `product_id` matches (or lacks `product_id`). |

After the initial `hello` object, each **`data:`** line is JSON with at least `type`, `ts` (RFC3339 nano), and optional `product_id`, `task_id`, `data` (object). Types include **`task_dispatched`**, **`cost_recorded`**, **`checkpoint_saved`**, **`task_completed`** (`data.source` e.g. `api_task_complete` / `agent_completion_webhook`), **`task_stall_nudged`**, **`pull_request_opened`** (includes `data.html_url`, optional `data.number`), **`merge_ship_completed`** (`data.state`, `data.merged_sha`, `data.error`, `data.conflict_files`, `data.merge_queue_row_id`), **`convoy_subtask_dispatched`** (`data.convoy_id`, `data.subtask_id`, `data.agent_role`, `data.external_ref`), **`convoy_subtask_completed`**. With **`DATABASE_PATH`** set, events are persisted in **`event_outbox`** and relayed to subscribers (restart-safe delivery of pending rows). In-memory mode broadcasts directly from the hub.

---

## Agents (registry + task heartbeats)

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/agents` | — | **`registry`**: registered execution agents (`id`, `display_name`, optional `product_id`, `source`, `external_ref`, `created_at`). **`items`**: recent **task agent health** rows (same shape as before). **`stub: true`** on **`items`** only when agent health is not wired. |
| POST | `/api/agents` | `display_name`; optional `product_id`, `source`, `external_ref` | Creates a logical agent slot (**201**). |
| GET | `/api/agents/{id}/mailbox` | — | Query: optional `limit`. **`{ "messages": [ { id, agent_id, body, optional task_id, created_at } ] }`**. |
| POST | `/api/agents/{id}/mailbox` | `body`; optional `task_id` | Append-only mailbox message (**201**). |

## Stubs / placeholders

These exist for route parity; behavior is minimal or not implemented for production use:

- `POST /api/openclaw/proxy` — not implemented (use WebSocket gateway env from server config).
- `GET /api/settings` — empty or minimal JSON.

---

## Environment (server)

Loaded via `internal/config` (`LoadFromEnv`). Commonly:

| Variable | Purpose |
|----------|---------|
| `ARMS_LISTEN` | Bind address (default `:8080`). |
| `MC_API_TOKEN` | API Bearer secret. |
| `WEBHOOK_SECRET` | Webhook HMAC key. |
| `DATABASE_PATH` | SQLite file; empty = in-memory only. |
| `ARMS_DB_BACKUP` | `1` / `true` → backup DB before migrate. |
| `OPENCLAW_GATEWAY_URL`, `OPENCLAW_GATEWAY_TOKEN`, `ARMS_OPENCLAW_SESSION_KEY`, `OPENCLAW_DISPATCH_TIMEOUT_SEC`, `ARMS_DEVICE_ID` | OpenClaw WebSocket dispatch. |
| `ARMS_LOG_JSON` | `1` / `true` → JSON logs to stdout. |
| `ARMS_ACCESS_LOG` | `0` / `false` / `off` / `no` → disable per-request access log lines. |
| `ARMS_CORS_ALLOW_ORIGIN` | Optional. When set (e.g. `http://localhost:3000`), arms sends `Access-Control-Allow-Origin` for browser UIs such as Fishtank. |
| `ARMS_AUTOPILOT_TICK_SEC` | Positive integer → cadence tick interval (seconds) for scheduled research/ideation; unset or invalid → disabled. **With `ARMS_REDIS_ADDR` set**, **`cmd/arms`** enqueues **`arms:autopilot_tick`** on that interval and **`cmd/arms-worker`** runs **`TickScheduled`**; **without Redis**, **`cmd/arms`** runs the tick **in-process** (same as before). |
| `ARMS_REDIS_ADDR` | Optional Redis (e.g. `localhost:6379`) for Asynq: worker consumes **`arms`** queue; HTTP server enqueues autopilot ticks when **`ARMS_AUTOPILOT_TICK_SEC`** is also set. |
| `ARMS_BUDGET_DEFAULT_CAP` | Default **cumulative** spend ceiling per product when **no** `cost_caps` row exists (default **100**). Set **`0`** to disable that default (no cumulative check until caps are configured). |
| `ARMS_GITHUB_TOKEN` | PAT for PR creation when using the REST backend. If empty, **`GITHUB_TOKEN`** is used. |
| `ARMS_GITHUB_API_URL` | Optional GitHub Enterprise API base for REST backend (e.g. `https://github.mycompany.com/api/v3/`). |
| `ARMS_GITHUB_PR_BACKEND` | `api` (default) or empty → REST + token; **`gh`** → `gh pr create` (see **`ARMS_GH_BIN`**, **`ARMS_GITHUB_HOST`**). |
| `ARMS_GH_BIN` | Optional path to the `gh` executable (default: resolve `gh` on `PATH`). |
| `ARMS_GITHUB_HOST` | Optional `GH_HOST` for GitHub Enterprise when using the `gh` backend. |

See `internal/config/config.go` for the full list and comments.

---

## Docker

From the `arms/` module directory:

```bash
docker build -t arms:local .
docker run --rm -p 8080:8080 -e DATABASE_PATH=/data/arms.db -v arms-db:/data arms:local
```

Or use **`arms/docker-compose.yml`** (named volume + defaults). Set `MC_API_TOKEN` / `WEBHOOK_SECRET` / OpenClaw variables in `environment` or an env file as needed.

For production deployment (TLS, secrets, webhooks behind proxies), see **[arms-production-hardening.md](arms-production-hardening.md)**.

**Worker binary:** from `arms/`, `go build -o arms-worker ./cmd/arms-worker` — run with **`ARMS_REDIS_ADDR`** and the same **`DATABASE_PATH`** (and related env) as **`cmd/arms`**; consumes queue **`arms`**, handling **`arms:autopilot_tick`** (runs **`autopilot.Service.TickScheduled`**) and **`arms:ping`** (no-op smoke task).

**Integration tests (module `arms/`):** `go test -tags=integration ./internal/integration/...` — end-to-end HTTP against an in-memory app and stub agent gateway. CI runs the same via `.github/workflows/arms.yml` when `arms/` changes.
