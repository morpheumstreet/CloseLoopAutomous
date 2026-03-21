# Arms HTTP API reference

REST surface for the `arms` service (`cmd/arms`). **JSON** request and response bodies unless noted. Path parameters use `{id}` style as in the router.

**Canonical machine-readable list:** `GET /api/docs/routes` returns `{ "routes": [ { "method", "path", "description" }, ... ] }` (same inventory as `internal/adapters/httpapi/routes_catalog.go`).

**OpenAPI 3.1:** [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — use with Swagger UI, Redocly, or Postman import.

---

## Authentication

| Mode | When |
|------|------|
| **None** | `MC_API_TOKEN` is unset — all protected routes are open (dev default). |
| **Bearer** | Set `MC_API_TOKEN`. Send `Authorization: Bearer <token>` on API calls. |
| **Same-origin** | If `ARMS_ALLOW_SAME_ORIGIN=1` or `true`, browser requests from the same origin may omit Bearer when a token is configured. |

**Unauthenticated by design:** `GET /api/health`, `GET /api/docs/routes`, `POST /api/webhooks/agent-completion`, `GET /api/live/events` (see SSE below).

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

Common `code` values from domain mapping include `not_found`, `invalid_transition`, `conflict`, `budget_exceeded`, `gateway`, `invalid_signature`, etc.

---

## Products and ideas

| Method | Path | Body (JSON) | Notes |
|--------|------|-------------|--------|
| POST | `/api/products` | `name`, `workspace_id`; optional profile fields; optional `research_cadence_sec`, `ideation_cadence_sec` (≥0, 0=off), `automation_tier` (`supervised` \| `semi_auto` \| `full_auto`), `auto_dispatch_enabled` | Create product (Mission Control–style profile + autopilot metadata). |
| GET | `/api/products/{id}` | — | Response includes profile fields, cadence/tier, `preference_model_json` (JSON string), optional `last_auto_*` timestamps. |
| PATCH | `/api/products/{id}` | Any subset of profile + autopilot fields above | At least one field required. Does not change pipeline `stage` (use research/ideation). |
| POST | `/api/products/{id}/research` | — | Run research phase. The full product record (including `program_document`, `description`, repo fields) is passed to the research port for prompt context. |
| POST | `/api/products/{id}/ideation` | — | Run ideation phase. Same product context plus stored `research_summary`. |
| GET | `/api/products/{id}/ideas` | — | `{ "ideas": [ … ] }` |
| GET | `/api/products/{id}/maybe-pool` | — | `{ "ideas": [ … ] }` for ideas swiped `maybe`. |
| POST | `/api/ideas/{id}/swipe` | `decision`: `pass` \| `maybe` \| `yes` \| `now` | Appends to `preference_model_json`; `maybe` enqueues pool. |
| POST | `/api/ideas/{id}/promote-maybe` | — | Requires prior `maybe` swipe; sets decision to yes and removes from pool. |

---

## Tasks (Kanban)

Task JSON includes at least: `id`, `product_id`, `idea_id`, `spec`, `status` (string: `planning`, `inbox`, `assigned`, `in_progress`, `testing`, `review`, `done`, `failed`, `convoy_active`), `status_reason`, `plan_approved`, `clarifications_json`, `checkpoint`, `external_ref`, `created_at`, `updated_at`.

| Method | Path | Body | Notes |
|--------|------|------|--------|
| GET | `/api/products/{id}/tasks` | — | `{ "tasks": [ … ] }`, newest first. 404 if product missing. |
| POST | `/api/tasks` | `idea_id`, `spec` | Creates task in **`planning`** (idea must be approved). |
| GET | `/api/tasks/{id}` | — | |
| PATCH | `/api/tasks/{id}` | Any of: `status`, `status_reason`, `clarifications_json` | At least one field required. Clarifications only while in `planning`. |
| POST | `/api/tasks/{id}/plan/approve` | Optional `{ "spec" }` | `planning` → `inbox`, `plan_approved: true`. |
| POST | `/api/tasks/{id}/plan/reject` | Optional `{ "status_reason" }` | Back to **`planning`** from **`inbox`** or **`assigned`** (blocked after dispatch / `external_ref` set). |
| POST | `/api/tasks/{id}/dispatch` | `estimated_cost` (number) | Requires **`assigned`** + approved plan. |
| POST | `/api/tasks/{id}/checkpoint` | `payload` | |
| POST | `/api/tasks/{id}/complete` | — | |

**Typical flow:** create task → (optional) PATCH `clarifications_json` → POST `plan/approve` → PATCH `status` to `assigned` → POST `dispatch` → checkpoint / complete / webhook.

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

| Method | Path | Body |
|--------|------|------|
| POST | `/api/costs` | `product_id`, `task_id`, `amount`, optional `note` |

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
| GET | `/api/live/events` | `text/event-stream`. If `MC_API_TOKEN` is set, use **`?token=<same value>`** (query) instead of Bearer. |

---

## Stubs / placeholders

These exist for route parity; behavior is minimal or not implemented for production use:

- `GET /api/agents` — empty list.
- `POST /api/openclaw/proxy` — not implemented (use WebSocket gateway env from server config).
- `GET /api/workspaces`, `GET /api/settings` — empty or minimal JSON.

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
| `ARMS_AUTOPILOT_TICK_SEC` | Positive integer → in-process cadence tick interval (seconds) for scheduled research/ideation; unset or invalid → disabled. |

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
