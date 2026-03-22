# Fishtank UI — todo list

Checklist for the **`fishtank/`** React shell (Mission Control–style) against **`arms`** HTTP + SSE. Pair with [arms-mission-control-gap-todos.md](arms-mission-control-gap-todos.md) (backend parity), [api-ref.md](api-ref.md) (routes), and **[fishtank-ui-wiring-outstanding.md](fishtank-ui-wiring-outstanding.md)** (strategy, backend UI blockers, 48–72h sprint, MC borrow list).

**Baseline today:** product list + create product, open workspace, Kanban **read-only** (`GET /api/products`, `GET …/tasks`), agents/health summary, SSE via **`useArmsLiveFeed`** + **`MissionUiContext`** (append-only feed, coarse filters—not yet merged as authoritative board state). **`ArmsClient`** is thin (no task mutations, no autopilot/convoy/costs/chat).

The **Design reference** section is the YouTube summary. The **Checklist vs design reference** table and lettered blocks **[A]–[L]** map that summary to concrete **`arms`** + **`fishtank`** work items (with **`MissionControlSidebar`** called out where labels already match the video).

---

## Design reference: OpenClaw “Mission Control” (Alex Finn walkthrough)

The video **“OpenClaw is 100x better with this tool (Mission Control)”** by Alex Finn (uploaded March 3, 2026) is a walkthrough/demo of building a custom **Mission Control** dashboard as a frontend/central interface for an OpenClaw multi-agent AI setup. The UI is AI-generated (via prompts to OpenClaw), hosted locally (e.g. Next.js app), and aims for full **visibility**, **control**, and **extensibility** over agents and workflows—so operators rely less on constant chat-checking.

**Replication hint (from the demo narrative):** feed the video URL + transcript to OpenClaw with something like: *“Build a full Mission Control dashboard like in this video, but customized to my workflows: [describe your needs].”* Emphasize **hyper-personalization**: avoid blind copying; use reverse-prompting (*“What tools do I need based on my history/goals?”*) to adapt modules.

### 1. Overall dashboard structure and philosophy

- **Single-page, dark-mode dashboard** — clean, modern, Linear.app / Notion-like aesthetic.
- **Left fixed sidebar** — navigation to all custom tools/modules (dynamically buildable via AI prompts).
- **Main content area** — switches by selected module (Kanban, calendar, list views, etc.).
- **Right side / overlay or integrated panel** — live activity feed (real-time agent updates).
- **Top bar** — global stats overview, search, quick actions (new task, pause agents, ping a specific agent).
- **Responsive shell** — on **narrow / mobile** viewports, a **three-button tab strip** (e.g. **Tasks** · **Agents** · **Activity**) switches the main region between the Kanban, the agents list, and the live feed. On **desktop**, that tab strip **must not** appear: use the **left sidebar** for module navigation, the **center** for the active module, and the **right column** for live activity (no duplicate tab UI).
- **Core goal** — visibility + control + extensibility for autonomous agents without living in chat threads.

### 2. Key modules / sidebar navigation (each a separate view)

| Module | Purpose / core functionality | Main UI elements (shown or described) | Key user requirements |
|--------|-------------------------------|----------------------------------------|------------------------|
| **Tasks** | Central Kanban for user- and agent-created work | Columns: Backlog, In Progress, Review (possibly Done). Cards: title, status dot, assignee (avatar/initial), tags (e.g. YouTube, Council), relative time. + New task. Drag-and-drop. Live feed integration. | Task visibility and progress; auto-heartbeat polling by agents; assign to self or agents; quick add + auto-categorization. |
| **Calendar** | Scheduling and cron/recurring tasks for proactive agents | Calendar view (monthly/daily as needed); marked scheduled jobs; execution confirmation. | Single record of truth for scheduled/recurring work; verify proactivity; easy add/edit of recurring items. |
| **Projects** | High-level initiatives linked to tasks, memories, docs | Project cards/lists; progress indicators; reverse-prompt hook (“What tasks advance this project?”). | Organize big initiatives; auto-generate advancing tasks; categorize historical tasks into projects. |
| **Memories** | Long-term memory / journal of conversations and insights | Chronological (by day) entries; searchable; aggregated long-term memory docs. | Replace ad-hoc markdown sprawl; fast context recall; search past interactions. |
| **Docs** | Searchable repo for generated documents (plans, newsletters, etc.) | Categorized list; metadata (type, tags); search; preview/formatting. | Centralized, readable docs; recurring drafts (e.g. newsletter); find by keyword/topic. |
| **Team** | Org structure of AI agents | Hierarchy (main agent → sub-agents); cards: name, role, model/device, avatar; top mission statement; reverse-prompt for delegation. | Clear delegation; record of who does what; mission-aligned task generation. |
| **Office** | Light 2D visual tracker of agent activity (pixel-art style) | Office layout with desks; agents moving/working; real-time activity flavor (e.g. water cooler). | Visual confirmation + entertainment; low-priority “fun” layer. |
| **Other / extensible** | Prompt-built extras | Examples in the narrative: Agents, Content, Approvals, Council, Memory, People, System, Radar, Factory, Pipeline, Feedback—activity, stats, approvals, etc. | Infinite extensibility via natural-language prompts; no hard-coded ceiling on modules. |

### 3. Cross-cutting / global UI features

- **Live activity feed** (right or persistent): chronological log; agent icons + short messages; relative timestamps (“less than a minute ago”, “23hrs ago”); color/status coding (e.g. green for completed); scrollable, newest on top.
- **Task cards (shared style)**: status dot; bold title + short description; assignee chip/avatar; tags (project, type: YouTube, MacStudio, …); timestamp / last activity.
- **Stats overview (top)**: e.g. this week / in progress / total tasks; completion % with progress bar.
- **Quick actions**: prominent + New task; assignee filter chips; project dropdown; pause/resume agents; ping specific agent.
- **Interactivity**: drag-and-drop on Kanban; click card → detail/modal; search across modules; reverse-prompt (“what to build next”).

### 4. Non-functional / setup notes from the demo

- Locally hosted (e.g. `localhost:3000` via Next.js or similar).
- Frontend generated through OpenClaw prompts (video/transcript as input).
- “No-code” for the end user assumes the agent stack can emit a full frontend.
- Again: **personalize** via prompts rather than cloning the demo pixel-for-pixel.

**Product framing:** this pattern turns a chat-centric agent swarm into an **operations center**: visibility (activity + tasks), control (assign, schedule, delegate), extensibility (prompt new modules).

---

## Checklist vs design reference

Use this table to trace **YouTube summary §** → **work items**. Implementation still goes through **`arms`** HTTP + SSE unless noted.

| Design reference | Checklist block |
|------------------|-----------------|
| §1 Shell (sidebar, main, right feed, top bar) | [A] Shell & layout |
| §3 Global (stats, search, quick actions, card + feed patterns) | [B] Global controls & UI patterns |
| §2 **Tasks** | [C] Module: Tasks |
| §2 **Calendar** | [D] Module: Calendar |
| §2 **Projects** | [E] Module: Projects (arms: products / workspace) |
| §2 **Memories** | [F] Module: Memories |
| §2 **Docs** | [G] Module: Docs |
| §2 **Team** | [H] Module: Team |
| §2 **Office** | [I] Module: Office (optional) |
| §2 **Other / extensible** (+ Convoy, costs, merge, chat, …) | [J] Arms-native extensions |
| §4 NFR + quality | [K] Settings, accessibility, polish |
| — | [L] Optional / later |

`MissionControlSidebar` already lists many §2 labels (Content, Approvals, Council, Calendar, Projects, Memory, Docs, Office, Team, Radar, Factory, Pipeline, Feedback, …); **[A]**/**[C–I]** todos include turning **disabled** entries into real routes or stub views as APIs exist.

---

## [A] Shell & layout (reference §1)

- [ ] **URL routing** — deep links: `/` dashboard, `/p/:productId` (or slug) workspace; preserve reload/share.
- [ ] **404 / unknown product** — graceful empty state + back to dashboard.
- [ ] **Left sidebar = module nav** — fixed nav; **main column swaps** by module (not only Tasks). Add routes or in-workspace views for Calendar, Projects, Memories, Docs, Team, etc., reusing the sidebar labels from the reference (enable items currently `disabled` in `MissionControlSidebar` when backed by UI).
- [ ] **Right / persistent activity column** — desktop: live feed column (already present); ensure it matches reference §3 feed behavior in **[B]**.
- [ ] **Top bar** — global chrome: search, quick actions, and stats **or** clear split with sidebar stats (today stats live in sidebar + header bar; align with “top bar overview” from the video).
- [ ] **Env & auth UX** — surface `VITE_ARMS_*` (or equivalent) in About or a small “Connection” panel: base URL, bearer/basic, copy-paste live URL with `?token=` for SSE.
- [ ] **Mobile shell parity** — the **three tab buttons** (Tasks / Agents / Activity) are **mobile-only** (`ft-mobile-tab-bar` in **`WorkspaceShellLayout`**): they route between `/p/:id/tasks`, `/agents`, `/feed` on small screens. **Do not show this tab strip on desktop** — desktop keeps sidebar + main + persistent live feed column per the design reference §1 responsive rule.
- [ ] **Loading / empty states** — skeletons for board and feed; distinguish “no tasks” vs “failed to load”.
- [ ] **Single-page dark dashboard polish** — Linear.app / Notion-like density, type, and contrast (see **[K]** theming).

---

## [B] Global controls & UI patterns (reference §3)

- [ ] **Stats overview** — this week / in progress / total tasks + completion **%** (progress bar); wire to real task data per workspace (baseline logic exists in workspace page; extend if header/top bar gains stats).
- [ ] **Search** — board/task search; plan **cross-module** search once Memories/Docs APIs expose content (reference: “search across modules”).
- [ ] **Quick actions** — prominent **+ New task** (wire to `POST` in **[C]**); **assignee** filter chips; **project / workspace** scope in header when multi-context views exist.
- [ ] **Pause / resume agents** — if **`arms`** exposes control, wire header toggle; otherwise document as OpenClaw-side or stub until API exists (reference §3).
- [ ] **Ping specific agent** — quick action or command surface when gateway/OpenClaw path is available (reference §3).
- [ ] **Reverse-prompt entry (optional)** — lightweight affordance or doc link: “suggest next module / tasks from context” when OpenClaw integration exists (reference §3 interactivity).
- [ ] **Task cards (shared style)** — status **dot**; bold **title** + short description; **assignee** avatar or initial chip; **tags/labels** (project, type, …); **relative** last-activity time.
- [ ] **Live activity feed** — chronological, **newest on top**; agent **icon** + short line; **relative** timestamps; **color/status** for completed vs in-flight; scrollable; align filters with SSE types (**[J]**).
- [ ] **Interactivity** — DnD Kanban (**[C]**); click card → detail/modal (**[C]**); reconnect/disconnect UX for feed stream.

---

## [C] Module: Tasks — Kanban (reference §2 Tasks)

- [ ] **Extend `ArmsClient`** — `POST/PATCH` tasks, plan approve/reject, dispatch, complete, stall-nudge, checkpoints, PR open, merge-queue actions as needed (mirror OpenAPI).
- [ ] **New Task** — wire “New Task” to `POST /api/tasks` (or product-scoped create) + refresh board.
- [ ] **Column semantics** — map **`arms`** statuses to Backlog / In Progress / Review / (Done) **or** document intentional differences vs the video.
- [ ] **Task detail modal / drawer** — title, status, `status_reason`, planning JSON / clarifications, workspace paths, PR URL, execution agent.
- [ ] **Drag-and-drop Kanban** — `PATCH …/tasks/{id}` with `status` (+ validation for allowed transitions).
- [ ] **Planning gate UX** — edit `clarifications_json`, **Approve plan** / **Reject** (`POST …/plan/approve`, `…/plan/reject`).
- [ ] **Dispatch / complete** — buttons or actions with confirmation + error mapping (`ErrGateway`, budget caps).
- [ ] **Assignment** — assign to user vs execution **agent** in UI where API supports it (reference: assignee on cards).
- [ ] **Tags / labels on cards** — surface task metadata or labels when API/model provides them (reference: YouTube, Council, …).
- [ ] **Agent heartbeat / polling** — surface last agent touch or SSE-driven updates so the board feels “live” without manual refresh (reference §2 Tasks).
- [ ] **Stalled tasks surfacing** — `GET …/stalled-tasks` list + **Stall nudge** (`POST …/stall-nudge`); optional badge when auto-nudge/reassign is server-side only (SSE).
- [ ] **Task images** — blocked on backend **#51**; UI upload + gallery once API exists.

---

## [D] Module: Calendar (reference §2 Calendar)

- [ ] **Product schedule UI** — `GET`/`PATCH …/product-schedule` (cron, delay, enabled); show next-run hints from API metadata when available.
- [ ] **Calendar grid view** — month/week (or day) grid showing scheduled jobs and execution confirmation, fed from schedule + relevant events when available (reference: “record of truth” for recurring/proactive work).

---

## [E] Module: Projects — workspaces (reference §2 Projects)

In **`arms`**, a **product** is the natural stand-in for a **project** (workspace container).

- [ ] **Product detail / settings** — `PATCH /api/products/{id}`: repo URL/branch, `automation_tier`, description, `program_document`, icons, soft-delete/restore if exposed.
- [ ] **Dashboard / project list UX** — `WorkspaceDashboardView`: project cards, **progress** indicators, clear drill-in to tasks/docs/memories (reference §2 Projects).
- [ ] **Merge queue visibility** — show `merge_queue_pending` + effective **`merge_policy`** gates from `GET /api/products/{id}` (read-only first).
- [ ] **Deleted products** — `GET /api/products?include_deleted=1` + restore/delete flows if operators need them.
- [ ] **Reverse-prompt hook (optional)** — e.g. “what tasks advance this project?” when OpenClaw/gateway is available (reference §2 Projects).

---

## [F] Module: Memories (reference §2 Memories)

- [ ] **Chronological journal** — group by **day** for operator-relevant history (task chat, ops log, agent notes) as data sources allow.
- [ ] **Searchable memory** — unified find across stored conversation/insight content when backend supports it (reference §2 Memories).
- [ ] **Replace ad-hoc markdown** — prefer in-app views over loose files where **`arms`** is source of truth.

---

## [G] Module: Docs (reference §2 Docs)

- [ ] **Knowledge as docs** — `GET/POST/PATCH/DELETE …/knowledge`: list with **category**, **type/tags**, search, readable **preview** (reference §2 Docs).
- [ ] **OpenAPI / operator docs link** — in About or Docs area: `docs/openapi/arms-openapi.yaml` or hosted Swagger.

---

## [H] Module: Team (reference §2 Team)

- [ ] **Registry management** — `POST /api/agents` + list from `GET /api/agents` (`registry[]`).
- [ ] **Agent cards** — name, **role**, model/device, avatar/initial; optional **hierarchy** (lead vs sub-agents) when data model allows.
- [ ] **Mission / charter** — show product or team **mission** line from `program_document` or settings when available (reference §2 Team).
- [ ] **Delegation UX** — tie tasks and dispatch to visible agent roles (reference §2 Team).
- [ ] **Agent mailbox (optional)** — `GET`/`POST …/agents/{id}/mailbox` for power users.
- [ ] **Richer health** — heartbeat age, stall nudges, reassign history when present in API.

---

## [I] Module: Office — optional (reference §2 Office)

- [ ] **2D / pixel-style activity view** — fun layer: desks, agents “at work”, optional water-cooler-style hints driven from feed or agent state (low priority; reference §2 Office).

---

## [J] Arms-native extensions (reference §2 Other / extensible)

These map to the video’s “Factory, Pipeline, Radar, Approvals, Content, Feedback, …” style modules: **prompt-extensible** in OpenClaw; here wired to **`arms`**.

### Autopilot (ideas, research, swipes)

- [ ] **Ideas list / filters** — `GET` product ideas (or equivalent catalog API); status, scores, category.
- [ ] **Swipe deck** — Pass / Maybe / Yes / Now → wire to existing swipe/promote/dispatch endpoints; optimistic UI + rollback.
- [ ] **Maybe pool** — `GET …/maybe-pool`, promote, **batch re-eval** UI.
- [ ] **Research cycles** — `GET …/research-cycles` timeline or table; trigger research if/when exposed.
- [ ] **Preference model** — `GET`/`PUT …/preference-model`, **Recompute** button (`POST …/preference-model/recompute`).
- [ ] **Product feedback** — `POST`/`GET …/feedback`, mark processed (`PATCH /api/product-feedback/{id}`).
- [ ] **Swipe history** — `GET …/swipe-history` for audit/debug.
- [ ] **Dashboard: Autopilot** — enable header/sidebar entry; schedules + ideas + swipe.

### Convoy

- [ ] **Convoy panel per task** — `GET/POST/PATCH …/tasks/{id}/convoy` (create, metadata, subtasks).
- [ ] **DAG visualization** — consume `GET /api/convoys/{id}/graph` (or task-scoped convoy payload): layers/edges, status per subtask.
- [ ] **Dispatch wave** — `POST …/convoy/dispatch` or `POST /api/convoys/{id}/dispatch-ready` with clear UX + errors.
- [ ] **Convoy mail** — thread UI for `GET`/`POST …/convoy/.../mail` (notes, blockers).

### Costs and budgets

- [ ] **Cost breakdown view** — `GET …/costs/breakdown` with date range; charts or tables by agent/model.
- [ ] **Caps editor** — if API supports creating/updating `cost_caps`, simple form per product; otherwise read-only summary.

### Merge queue and shipping

- [ ] **Per-task merge queue** — `GET …/merge-queue`, head vs pending, **complete** / **resolve** / dequeue actions (respect 409 `ErrNotMergeQueueHead`).
- [ ] **PR flow** — `POST …/pull-request` from UI when head branch set; show PR link from task + SSE `pull_request_opened`.

### Chat, operator queue, operations

- [ ] **Per-task chat** — `GET`/`POST …/tasks/{id}/chat`; show SSE `task_chat_message` in-thread or merge with poll-on-open.
- [ ] **Operator queue** — `GET …/chat-queue`, ack messages (`POST …/chat-queue/{id}/ack`).
- [ ] **Operations log** — `GET /api/operations-log` with filters (action, resource, since); link from “Activity Dashboard” header button.

### Live feed (engineering depth)

- [ ] **Event type coverage** — align filters with SSE catalog: convoy subtask events, merge ship, stall nudge, execution reassigned, chat, etc.
- [ ] **Payload inspector** — expand/collapse JSON for debugging (dev toggle).
- [ ] **Reconnect UX** — visible “disconnected” when `EventSource` errors; backoff or manual reconnect.

---

## [K] Settings, accessibility, polish (reference §4 + quality)

- [ ] **Local / dev hosting** — document and keep **Bun** dev UX (`bun run dev`) comparable to “localhost app” in reference §4; **no npm/Node** for Fishtank in this repo.
- [ ] **Settings page** — `GET /api/settings` today is stub; show placeholder or hide until backend fills in.
- [ ] **Accessibility** — keyboard DnD or alternative status moves; focus traps in modals; live region for feed errors.
- [ ] **Theming** — audit contrast and `prefers-reduced-motion` for existing theme cycle; dark-first polish per §1.

---

## [L] Optional / later

- [ ] **CI webhook testing UI** — only if you need to simulate `POST /api/webhooks/ci-completion` from the browser (usually server-side).
- [ ] **`/api/openclaw/proxy`** — if added (**gap #46**), optional debug panel for gateway calls from the UI.
- [ ] **E2E tests** — Playwright against `arms` + `fishtank` dev servers for create product → task → status change.

---

_Checklist structure aligned with YouTube Mission Control summary (§1–§4); items still tracked against `fishtank/src`, `docs/arms-mission-control-gap-todos.md`, and Alex Finn walkthrough (2026-03)._
