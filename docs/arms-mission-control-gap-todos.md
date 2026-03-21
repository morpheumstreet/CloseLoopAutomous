# Arms backend — gap todo list (vs [mission-control](https://github.com/crshdn/mission-control/tree/main))

Use this as the master backlog for bringing `arms` toward Autensa/Mission Control backend parity. Check items off as you implement them.

_Re-checked against the `arms/` tree: baseline vs “full MC” is called out so unchecked rows are not misread as “missing entirely” when a slim table or route already exists._

---

## API stubs, docs, and gateway session

**Human reference:** [api-ref.md](api-ref.md) — section *Stubs / placeholders*. **OpenAPI:** [openapi/arms-openapi.yaml](openapi/arms-openapi.yaml) — tag **Stubs** plus `Product.preference_model_json` (stub for real preference learning, not under Stubs).

| Route | Status |
|-------|--------|
| `GET /api/agents` | Empty list until agent domain ships |
| `POST /api/openclaw/proxy` | Not implemented (501); use server env `OPENCLAW_GATEWAY_*` + WS from service |
| `GET /api/workspaces`, `GET /api/settings` | Minimal / empty JSON |

There is no REST “session” resource. OpenClaw dispatch uses env (e.g. `ARMS_OPENCLAW_SESSION_KEY` on the server). A **browser-facing** gateway proxy (MC-style `/api/openclaw/*`) is optional; see §3.

---

## Suggested roadmap (continue toward MC parity)

Rough ordering; adjust sprint length to taste.

| Phase | Focus | Notes |
|-------|--------|--------|
| **A** | SSE + costs | Domain events or outbox → `GET /api/live/events` with real payloads; filters by product/agent/type. Add `cost_caps`, extend `cost_events` dimensions; cost breakdown queries. |
| **B** | Ship pipeline | Real `PullRequestPublisher` + test → review → PR orchestration; wire `auto_dispatch_enabled` + **tier behavior** (supervised / semi-auto / full-auto). |
| **C** | Workspace + safety | Worktree/sandbox port, `workspace_ports`, `workspace_merges`, product-scoped locks; checkpoint **history** + restore API. |
| **D** | Convoy + agents | Full DAG metadata, convoy mail, health/retries; agent aggregate, discovery, replace stub `GET /api/agents`. Decide **`/api/convoys` vs `/api/convoy`** for MC client parity. |
| **E** | Autopilot depth | `research_cycles`, `swipe_history`, real `preference_models`, maybe-pool **resurface** / batch re-eval; `product_schedules` with **cron/outbox vs in-process ticker** (today: ticker only). |
| **Ongoing** | Optional | Ed25519 device block on `connect`; env-gated live OpenClaw contract tests. |

---

## Mission Control references (open decisions)

Use [crshdn/mission-control](https://github.com/crshdn/mission-control) for precedent when designing behavior, not only table names.

| Topic | Where to look in MC |
|-------|---------------------|
| Automation tiers (merge / pause rules) | README — *Automation tiers* and autopilot pipeline |
| Live activity / SSE | README — *Live Activity Feed*; realtime docs under repo `docs/` |
| Workspace isolation, ports, merge queue | README — *Workspace isolation*; `src/lib/workspace-isolation.ts` (and related) |
| OpenClaw client / handshake | [`src/lib/openclaw/client.ts`](https://github.com/crshdn/mission-control/blob/main/src/lib/openclaw/client.ts) |
| Device identity / signing | `device-identity` patterns vs token-only `connect` |
| Convoy + API shape | `src/app/api/convoy/` (and related routes) vs arms `/api/convoys` |

---

## 1. API surface and transport

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
- [ ] `swipe_history`
- [ ] `preference_models` (per-product learning)
- [x] `maybe_pool` table + list/promote API — `MaybePoolRepository`; `GET /api/products/{id}/maybe-pool`, `POST /api/ideas/{id}/promote-maybe` (baseline; aligns with §5)
- [ ] Maybe pool **resurface** / batch re-eval workflow (MC-style; not just storage)
- [ ] `product_feedback`
- [ ] `cost_events`: add dimensions (agent, model, billing period, …); baseline table + append API already exist (`id`, `product_id`, `task_id`, `amount`, `note`, `at`)
- [ ] `cost_caps` (daily + monthly + product scope); today: in-process `budget.Static` cumulative cap only
- [ ] `product_schedules` (research/ideation cadence, cron)
- [ ] `operations_log` / audit trail
- [ ] `convoys` / `convoy_subtasks`: richer DAG metadata, mail, per-subtask status (today: matches slim `domain.Convoy` + HTTP create + dispatch-ready wave)
- [ ] `agent_health`
- [ ] `work_checkpoints`: history + restore (vs current `checkpoints` latest blob only)
- [ ] `agent_mailbox`
- [ ] `workspace_ports`
- [ ] `workspace_merges` / merge queue state
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
- [ ] Ed25519 **device** block on `connect` (MC `device-identity.ts` signing) — optional; token-only `connect` works for many gateways
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
- [x] Scheduling: `research_cadence_sec`, `ideation_cadence_sec`, `last_auto_*` on product; `autopilot.Service.TickScheduled`; `ARMS_AUTOPILOT_TICK_SEC` in-process ticker in `cmd/arms` (not cron/outbox). `auto_dispatch_enabled` + tier stored; **task auto-dispatch from tier not wired** (manual dispatch unchanged).
- [x] Background job — **minimal**: same in-process ticker as above; no separate worker port yet.
- [x] Preference stub: each swipe appends an event to `preference_model_json` (JSON array); not full MC preference_models / ML.
- [x] Maybe pool (baseline): `maybe_pool` table + `MaybePoolRepository`; swipe `maybe` adds; `GET /api/products/{id}/maybe-pool`; `POST /api/ideas/{id}/promote-maybe` → yes + pool remove + stage advance when in swipe. Resurface / batch re-eval: still open (§2).
- [x] Automation tiers: `automation_tier` enum `supervised` | `semi_auto` | `full_auto` on product + create/patch/JSON (behavioral differences beyond storage/TBD for dispatch).
- [ ] Post-execution chain: test → review → GitHub PR (orchestration after task done; **not** implemented end-to-end).
- [x] GitHub-shaped port: `ports.PullRequestPublisher` + `adapters/shipping.PullRequestNoop` stub (no real PR creation; optional hook from task completion still TBD).

---

## 6. Convoy mode

- [x] Persist baseline convoy + subtasks — SQLite + memory `ConvoyRepository` (graph + dispatch flags/refs); not yet “full MC” metadata
- [ ] Persist full convoy DAG metadata as in MC (beyond current domain)
- [ ] Convoy mail / inter-subtask messaging (port + persistence)
- [ ] Integrate convoy dispatch with agent health and retries
- [x] Minimal HTTP — `POST /api/convoys`, `GET /api/convoys/{id}`, `GET /api/products/{id}/convoys`, `POST /api/convoys/{id}/dispatch-ready`; `convoy.Service.Get`, `ListByProduct`; `ports.ConvoyRepository.ListByProduct`
- [ ] API parity with MC `/api/convoy/*` — mail, graph, richer status, naming (`/api/convoy` vs `/api/convoys`), etc.
- [ ] Richer subtask model (agent config, retries, nudges) if required for parity

---

## 7. Safety, cost, workspace

- [x] Baseline budget gate — `budget.Static` + `SumByProduct` on cost repo; enforced at task dispatch (`estimated_cost`)
- [ ] Budget policy: daily and monthly caps (not only single cumulative cap)
- [ ] Cost breakdown queries (by agent, model, time range)
- [ ] Workspace isolation port: git worktrees, sandbox paths
- [ ] Port allocation (e.g. 4200–4299) with persistence (`workspace_ports`)
- [ ] Serialized merge queue + conflict detection (`workspace_merges`)
- [ ] Product-scoped locks for concurrent completions
- [ ] Checkpoint history + restore API (not only latest checkpoint)
- [ ] Agent health monitor port + persistence
- [ ] Stalled/zombie detection + auto-nudge / reassign policy

---

## 8. Realtime and observability

- [ ] Domain events or outbox for orchestration occurrences
- [x] SSE transport shell — `GET /api/live/events` (hello + periodic ping; `SSEQueryToken` when auth on)
- [ ] SSE live activity feed wired to domain (filter by product, agent, type; meaningful event payloads)
- [ ] Operator chat: queued notes + direct messages (ports + storage)
- [ ] Per-task chat history
- [ ] Learner / knowledge base port + storage + injection into future dispatches

---

## 9. Agents domain

- [ ] Agent aggregate + repository
- [ ] Registration and discovery/import flows (gateway-backed)
- [ ] Agent APIs: health, mailbox, listing, configuration
- [x] Stub listing route — `GET /api/agents` returns empty `items` (placeholder until real agent domain)

---

## 10. Cross-cutting platform

- [x] Dedicated `internal/config` — `LoadFromEnv()`, env vars documented on `Config`; `httpapi.Config` is a type alias + `LoadConfig()` wrapper for the HTTP adapter
- [x] Dockerfile for `arms` service — `arms/Dockerfile` (Alpine runtime, static `CGO_ENABLED=0` build)
- [x] docker-compose — `arms/docker-compose.yml` (port 8080, `DATABASE_PATH=/data/arms.db`, named volume; OpenClaw/token env commented)
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
| Roadmap phases A→E | See **Suggested roadmap** above                           |
