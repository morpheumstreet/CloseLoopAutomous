# Fishtank UI — todo list

Checklist for the **`fishtank/`** React shell (Mission Control–style) against **`arms`** HTTP + SSE. Pair with [arms-mission-control-gap-todos.md](arms-mission-control-gap-todos.md) (backend parity), [api-ref.md](api-ref.md) (routes), and **[fishtank-ui-wiring-outstanding.md](fishtank-ui-wiring-outstanding.md)** (strategy, backend UI blockers, 48–72h sprint, MC borrow list).

**Baseline today:** product list + create product, open workspace, Kanban **read-only** (`GET /api/products`, `GET …/tasks`), agents/health summary, SSE via **`useArmsLiveFeed`** + **`MissionUiContext`** (append-only feed, coarse filters—not yet merged as authoritative board state). **`ArmsClient`** is thin (no task mutations, no autopilot/convoy/costs/chat).

---

## Shell, navigation, and env

- [ ] **URL routing** — deep links: `/` dashboard, `/p/:productId` (or slug) workspace; preserve reload/share.
- [ ] **404 / unknown product** — graceful empty state + back to dashboard.
- [ ] **Env & auth UX** — surface `VITE_ARMS_*` (or equivalent) in About or a small “Connection” panel: base URL, bearer/basic, copy-paste live URL with `?token=` for SSE.
- [ ] **Mobile shell parity** — today mobile hides Agents + Live Feed; add **tabs** or drawer for Agents / Feed / Queue.
- [ ] **Loading / empty states** — skeletons for board and feed; distinguish “no tasks” vs “failed to load”.

---

## Products (workspaces)

- [ ] **Product detail / settings** — `PATCH /api/products/{id}`: repo URL/branch, `automation_tier`, description, `program_document`, icons, soft-delete/restore if exposed.
- [ ] **Product schedule UI** — `GET`/`PATCH …/product-schedule` (cron, delay, enabled); show next-run hints from API metadata when available.
- [ ] **Merge queue visibility** — show `merge_queue_pending` + effective **`merge_policy`** gates from `GET /api/products/{id}` (read-only first).
- [ ] **Deleted products** — `GET /api/products?include_deleted=1` + restore/delete flows if operators need them.

---

## Tasks and Kanban

- [ ] **Extend `ArmsClient`** — `POST/PATCH` tasks, plan approve/reject, dispatch, complete, stall-nudge, checkpoints, PR open, merge-queue actions as needed (mirror OpenAPI).
- [ ] **New Task** — wire “New Task” to `POST /api/tasks` (or product-scoped create) + refresh board.
- [ ] **Task detail modal / drawer** — title, status, `status_reason`, planning JSON / clarifications, workspace paths, PR URL, execution agent.
- [ ] **Drag-and-drop Kanban** — `PATCH …/tasks/{id}` with `status` (+ validation for allowed transitions).
- [ ] **Planning gate UX** — edit `clarifications_json`, **Approve plan** / **Reject** (`POST …/plan/approve`, `…/plan/reject`).
- [ ] **Dispatch / complete** — buttons or actions with confirmation + error mapping (`ErrGateway`, budget caps).
- [ ] **Stalled tasks surfacing** — `GET …/stalled-tasks` list + **Stall nudge** (`POST …/stall-nudge`); optional badge when auto-nudge/reassign is server-side only (SSE).
- [ ] **Task images** — blocked on backend **#51**; UI upload + gallery once API exists.

---

## Autopilot (ideas, research, swipes)

- [ ] **Ideas list / filters** — `GET` product ideas (or equivalent catalog API); status, scores, category.
- [ ] **Swipe deck** — Pass / Maybe / Yes / Now → wire to existing swipe/promote/dispatch endpoints; optimistic UI + rollback.
- [ ] **Maybe pool** — `GET …/maybe-pool`, promote, **batch re-eval** UI.
- [ ] **Research cycles** — `GET …/research-cycles` timeline or table; trigger research if/when exposed.
- [ ] **Preference model** — `GET`/`PUT …/preference-model`, **Recompute** button (`POST …/preference-model/recompute`).
- [ ] **Product feedback** — `POST`/`GET …/feedback`, mark processed (`PATCH /api/product-feedback/{id}`).
- [ ] **Swipe history** — `GET …/swipe-history` for audit/debug.
- [ ] **Dashboard: Autopilot** — enable header button; entry point to schedules + ideas + swipe.

---

## Convoy

- [ ] **Convoy panel per task** — `GET/POST/PATCH …/tasks/{id}/convoy` (create, metadata, subtasks).
- [ ] **DAG visualization** — consume `GET /api/convoys/{id}/graph` (or task-scoped convoy payload): layers/edges, status per subtask.
- [ ] **Dispatch wave** — `POST …/convoy/dispatch` or `POST /api/convoys/{id}/dispatch-ready` with clear UX + errors.
- [ ] **Convoy mail** — thread UI for `GET`/`POST …/convoy/.../mail` (notes, blockers).

---

## Costs and budgets

- [ ] **Cost breakdown view** — `GET …/costs/breakdown` with date range; charts or tables by agent/model.
- [ ] **Caps editor** — if API supports creating/updating `cost_caps`, simple form per product; otherwise read-only summary.

---

## Merge queue and shipping

- [ ] **Per-task merge queue** — `GET …/merge-queue`, head vs pending, **complete** / **resolve** / dequeue actions (respect 409 `ErrNotMergeQueueHead`).
- [ ] **PR flow** — `POST …/pull-request` from UI when head branch set; show PR link from task + SSE `pull_request_opened`.

---

## Chat, knowledge, operations

- [ ] **Per-task chat** — `GET`/`POST …/tasks/{id}/chat`; show SSE `task_chat_message` in-thread or merge with poll-on-open.
- [ ] **Operator queue** — `GET …/chat-queue`, ack messages (`POST …/chat-queue/{id}/ack`).
- [ ] **Knowledge** — `GET/POST/PATCH/DELETE …/knowledge` minimal CRUD for snippets used at dispatch.
- [ ] **Operations log** — `GET /api/operations-log` with filters (action, resource, since); link from “Activity Dashboard” header button.

---

## Agents

- [ ] **Registry management** — `POST /api/agents` + list from `GET /api/agents` (`registry[]`).
- [ ] **Agent mailbox (optional)** — `GET`/`POST …/agents/{id}/mailbox` for power users.
- [ ] **Richer health** — expand Agents panel beyond summaries: heartbeat age, stall nudges, reassign history when present in API.

---

## Live feed

- [ ] **Event type coverage** — align filters with SSE catalog: convoy subtask events, merge ship, stall nudge, execution reassigned, chat, etc.
- [ ] **Payload inspector** — expand/collapse JSON for debugging (dev toggle).
- [ ] **Reconnect UX** — visible “disconnected” when `EventSource` errors; backoff or manual reconnect.

---

## Settings, docs, and polish

- [ ] **Settings page** — `GET /api/settings` today is stub; show placeholder or hide until backend fills in.
- [ ] **OpenAPI / docs link** — in About: link to `docs/openapi/arms-openapi.yaml` or hosted Swagger for operators.
- [ ] **Accessibility** — keyboard DnD or alternative status moves; focus traps in modals; live region for feed errors.
- [ ] **Theming** — audit contrast and `prefers-reduced-motion` for existing theme cycle.

---

## Optional / later

- [ ] **CI webhook testing UI** — only if you need to simulate `POST /api/webhooks/ci-completion` from the browser (usually server-side).
- [ ] **`/api/openclaw/proxy`** — if added (**gap #46**), optional debug panel for gateway calls from the UI.
- [ ] **E2E tests** — Playwright against `arms` + `fishtank` dev servers for create product → task → status change.

---

_Last aligned with `fishtank/src` and `docs/arms-mission-control-gap-todos.md` (2026-03)._
