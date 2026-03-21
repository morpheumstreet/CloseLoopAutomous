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

Common `code` values from domain mapping include `not_found`, `invalid_transition`, `conflict`, `merge_queue_head`, `budget_exceeded`, `gateway`, `shipping`, `invalid_signature`, etc.

---

## Products and ideas

| Method | Path | Body (JSON) | Notes |
|--------|------|-------------|--------|
| POST | `/api/products` | `name`, `workspace_id`; optional profile fields; optional `research_cadence_sec`, `ideation_cadence_sec` (≥0, 0=off), `automation_tier` (`supervised` \| `semi_auto` \| `full_auto`), `auto_dispatch_enabled` | Create product (Mission Control–style profile + autopilot metadata). |
| GET | `/api/products` | — | `{ "products": [ … ] }` — list products (dashboards / UIs). |
| GET | `/api/products/{id}` | — | Response includes profile fields, cadence/tier, `preference_model_json` (JSON string), optional `last_auto_*` timestamps. |
| PATCH | `/api/products/{id}` | Any subset of profile + autopilot fields above | At least one field required. Does not change pipeline `stage` (use research/ideation). |
| PATCH | `/api/products/{id}/cost-caps` | At least one of: `daily_cap`, `monthly_cap`, `cumulative_cap` (numbers) | **Negative** value for an axis **clears** that limit (unlimited on that axis). Upserts `cost_caps` row. |
| GET | `/api/products/{id}/costs/breakdown` | — | Query: optional `from`, `to` (RFC3339 / RFC3339Nano). JSON: `total`, `events[]`, `by_agent`, `by_model`. |
| POST | `/api/products/{id}/research` | — | Run research phase. The full product record (including `program_document`, `description`, repo fields) is passed to the research port for prompt context. |
| POST | `/api/products/{id}/ideation` | — | Run ideation phase. Same product context plus stored `research_summary`. |
| GET | `/api/products/{id}/ideas` | — | `{ "ideas": [ … ] }` |
| GET | `/api/products/{id}/maybe-pool` | — | `{ "ideas": [ … ] }` for ideas swiped `maybe`. |
| GET | `/api/products/{id}/swipe-history` | — | Query: optional `limit` (default 100, max 500). `{ "swipes": [ { id, idea_id, product_id, decision, created_at } ] }`, newest first (audit log; complements `preference_model_json`). |
| GET | `/api/products/{id}/merge-queue` | — | Query: optional `limit` (default 50, max 500). `{ "merge_queue": [ { id, product_id, task_id, status, created_at } ] }` — **pending** rows only, FIFO by `id`. **503** if merge queue is not configured. |
| POST | `/api/ideas/{id}/swipe` | `decision`: `pass` \| `maybe` \| `yes` \| `now` | Appends to `preference_model_json`; `maybe` enqueues pool. |
| POST | `/api/ideas/{id}/promote-maybe` | — | Requires prior `maybe` swipe; sets decision to yes and removes from pool. |

---

## Tasks (Kanban)

Task JSON includes at least: `id`, `product_id`, `idea_id`, `spec`, `status` (string: `planning`, `inbox`, `assigned`, `in_progress`, `testing`, `review`, `done`, `failed`, `convoy_active`), `status_reason`, `plan_approved`, `clarifications_json`, `checkpoint`, `external_ref`, `sandbox_path`, `worktree_path`, `created_at`, `updated_at`.

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/products/{id}/tasks` | — | `{ "tasks": [ … ] }`, newest first. 404 if product missing. |
| POST | `/api/tasks` | `idea_id`, `spec` | Creates task in **`planning`** (idea must be approved). |
| GET | `/api/tasks/{id}` | — | |
| PATCH | `/api/tasks/{id}` | Any of: `status`, `status_reason`, `clarifications_json` | At least one field required. Clarifications only while in `planning`. |
| POST | `/api/tasks/{id}/plan/approve` | Optional `{ "spec" }` | `planning` → `inbox`, `plan_approved: true`. |
| POST | `/api/tasks/{id}/plan/reject` | Optional `{ "status_reason" }` | Back to **`planning`** from **`inbox`** or **`assigned`** (blocked after dispatch / `external_ref` set). |
| POST | `/api/tasks/{id}/dispatch` | `estimated_cost` (number) | Requires **`assigned`** + approved plan. Enforces **`budget.Composite`** (caps + default cumulative). **402** + **`budget_exceeded`** if `estimated_cost` would exceed allowed spend. |
| POST | `/api/tasks/{id}/pull-request` | `head_branch` (required), optional `title`, `body` | Opens a PR using `product.repo_url` (`owner/repo`) and `product.repo_branch` as base (default `main`). **REST backend** (default, or `ARMS_GITHUB_PR_BACKEND=api`): **`ARMS_GITHUB_TOKEN`** or **`GITHUB_TOKEN`** with `repo` scope; optional **`ARMS_GITHUB_API_URL`** for GitHub Enterprise. **`ARMS_GITHUB_PR_BACKEND=gh`**: runs `gh pr create` (optional **`ARMS_GH_BIN`**, **`ARMS_GITHUB_HOST`**). Response `{ "pr_url": "..." }` (empty string if publisher is noop). Allowed while task is `in_progress`, `testing`, `review`, or `done`. |
| POST | `/api/tasks/{id}/merge-queue` | — | Enqueues the task on the product’s **serialized merge queue** (FIFO by row `id`). **201** `{ "status": "queued" }`. **409** `conflict` if this task already has a **pending** row. **503** if merge queue is not configured. |
| POST | `/api/tasks/{id}/merge-queue/complete` | — | Marks the **pending** row for this task **done** only when it is the **head** of the queue for that product; otherwise **409** with code **`merge_queue_head`**. **404** if the task has no pending row. **503** if merge queue is not configured. |
| GET | `/api/tasks/{id}/checkpoints` | — | Query: optional `limit` (default 50, max 500). `{ "checkpoints": [ { id, task_id, payload, created_at } ] }` newest first. |
| POST | `/api/tasks/{id}/checkpoint/restore` | `{ "history_id": <int> }` | Restores payload via same rules as recording a checkpoint. |
| POST | `/api/tasks/{id}/checkpoint` | `payload` | Appends **`checkpoint_history`** and updates latest `checkpoints` row + task. |
| POST | `/api/tasks/{id}/complete` | — | **`done`** from `in_progress` / `testing` / `review`. With SQLite, **`task_agent_health`** **`completed`** and SSE **`task_completed`** are written in the **same transaction** as the task row. |
| POST | `/api/tasks/{id}/stall-nudge` | Optional `{ "note": "..." }` (empty body OK) | **Phase A** operator nudge for **`in_progress`**, **`testing`**, **`review`**, **`convoy_active`**: prepends `[stall_nudge <ts>]` to **`status_reason`**, appends **`stall_nudges[]`** to agent-health **`detail`** when health is wired, emits **`task_stall_nudged`** on the live stream. |

**Typical flow:** create task → (optional) PATCH `clarifications_json` → POST `plan/approve` → PATCH `status` to `assigned` → POST `dispatch` → checkpoint / complete / webhook.

**Merge queue:** enqueue / complete via the task routes above; **`POST …/merge-queue/complete` succeeds only for the FIFO head** (see **`merge_queue_head`** in Errors). List pending rows under **Products and ideas**.

---

## Convoys

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/products/{id}/convoys` | — | `{ "convoys": [ … ] }` |
| POST | `/api/convoys` | `parent_task_id`, `product_id`, `subtasks[]` with `agent_role`, optional `id`, `depends_on` | |
| GET | `/api/convoys/{id}` | — | |
| POST | `/api/convoys/{id}/dispatch-ready` | — | Dispatches one ready wave of subtasks. |

---

## Costs

| Method | Path | Body | Notes |
|--------|------|------|--------|
| POST | `/api/costs` | `product_id`, `task_id`, `amount`, optional `note`, optional **`agent`**, **`model`** | Same **`budget.Composite`** rules as task dispatch: **`cost_caps`** (daily / monthly / cumulative) plus default cumulative when no caps row (**`ARMS_BUDGET_DEFAULT_CAP`**). **402** + code **`budget_exceeded`** if the new amount would exceed the allowed spend. |

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
- Body: `{ "task_id": "<id>" }` (JSON).
- Requires `WEBHOOK_SECRET` set; otherwise **503**.

---

## SSE (live stream)

| Method | Path | Notes |
|--------|------|--------|
| GET | `/api/live/events` | `text/event-stream`. When **`MC_API_TOKEN`** is set: **`Authorization: Bearer <token>`** or **`?token=<same value>`** (native **`EventSource`** only supports the query form). When only **`ARMS_ACL`** is configured: **`?basic=<base64(user:password)>`**. Optional **`product_id=`** — only forward `data:` lines whose JSON `product_id` matches (or lacks `product_id`). |

After the initial `hello` object, each **`data:`** line is JSON with at least `type`, `ts` (RFC3339 nano), and optional `product_id`, `task_id`, `data` (object). Types include **`task_dispatched`**, **`cost_recorded`**, **`checkpoint_saved`**, **`task_completed`** (`data.source` e.g. `api_task_complete` / `agent_completion_webhook`), **`task_stall_nudged`**, **`pull_request_opened`** (includes `data.html_url`). With **`DATABASE_PATH`** set, events are persisted in **`event_outbox`** and relayed to subscribers (restart-safe delivery of pending rows). In-memory mode broadcasts directly from the hub.

---

## Stubs / placeholders

These exist for route parity; behavior is minimal or not implemented for production use:

- `GET /api/agents` — empty list.
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
| `ARMS_AUTOPILOT_TICK_SEC` | Positive integer → in-process cadence tick interval (seconds) for scheduled research/ideation; unset or invalid → disabled. |
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

**Integration tests (module `arms/`):** `go test -tags=integration ./internal/integration/...` — end-to-end HTTP against an in-memory app and stub agent gateway. CI runs the same via `.github/workflows/arms.yml` when `arms/` changes.
