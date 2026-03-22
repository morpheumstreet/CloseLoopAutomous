# Fishtank / UI wiring — outstanding modules

_Extracted from [arms-mission-control-gap-todos.md](arms-mission-control-gap-todos.md) and the actual **`fishtank/`** tree (March 2026)._

The gap doc is **backend-parity first**; the backlog header counts **~7** still-open checklist rows (master index + §3–§9). **None are “pure UI” tickets**, but several **block or shape** how Fishtank can wire to `arms`. The React + TypeScript + Vite app is the **operational surface** (the “tank”); most “swimming fish” visuals are **not implemented yet**—today you have `MissionQueuePanel`, `AgentsPanel`, `LiveFeedPanel`, `TaskCard`, and dashboard chrome.

**Granular UI checklist:** [fishtank-ui-todos.md](fishtank-ui-todos.md).

---

## From the official gap list (UI-impacting)

| Item | Why it matters for Fishtank |
|------|-----------------------------|
| **#45** — Ed25519 device signing on OpenClaw `connect` | Stricter **UI → gateway** auth story if the browser ever proxies or embeds gateway calls; until then, header token/env is enough. |
| **#46** — Optional **`/api/openclaw/*`** HTTP proxy | Lets the UI call gateway-shaped APIs **same-origin** and avoids **CORS** pain for advanced operator tools. |
| **#51** — Task images / attachments + API | Rich task cards, mockups, screenshots in the tank. |
| **#52** — Distinguish manual vs autopilot-derived tasks | Different **icons/colors/badges** in Kanban and “task stream” so operators read the loop at a glance. |
| **#93** — MC-style agent registration / discovery / import | **Deferred** in gap doc; “new agents auto-appear as fish” is a later automation. |
| **#94** — Agent config depth + aggregate health | **Deferred**; powers a serious **monitoring** dashboard later. |

**Also open in §5 (mostly backend):** post-execution **PR + merge** polish (CI signals, dedupe keys)—affects what the **live feed** and task headers can truthfully show, less about layout widgets.

---

## Gaps in `fishtank/` today (actual wiring)

**Already present**

- **`useArmsLiveFeed`** + **`buildLiveEventsUrl`** — `EventSource` on `GET /api/live/events` with `product_id` (+ token query param when configured).
- **`MissionUiContext`** — holds workspaces, tasks (from REST), agents/health summary, live events (append-only cap), create product, open workspace.
- **Read-only Kanban** — tasks from `GET /api/products/{id}/tasks`; no drag, no “New Task” POST, no task modal.

**Still missing or thin**

- **Authoritative state vs SSE** — Events augment the feed but **do not** yet merge into a single “live board” model (e.g. task status after `task_completed` without refetch). Refetch-on-key-events or patch reducers is TBD.
- **Event coverage & UX** — Mapper/filters lag the full SSE catalog (convoy subtasks, merge ship, stall nudge, execution reassigned, chat, etc.); little **disconnect / error** surfacing beyond browser defaults.
- **`ArmsClient`** — Only health, version, products, product tasks, agent-health list, create product. No dispatch, nudge, reassign, convoy, costs, chat, merge queue.
- **Named “tank modules”** — There are **no** `ConvoyView`, `AgentPool`, `TaskStream`, or `LiveCostTracker` components yet; treat those as **planned modules**, not half-finished files.
- **Visualization layer** — No fish motion, convoy grouping animation, or graph view; **Mission Control** references below are **patterns to borrow**, not copied code in-tree.

**Stack note:** Fishtank uses **plain CSS** (`ft-*` tokens in `src/styles/`), not Tailwind, and **no React Query**—only React context/hooks. You can add TanStack Query later if fetch caching becomes painful. **Tooling:** **Bun** only for install/dev/build (`bun install`, `bun run dev`, `bun run build`)—**not npm or Node** for this UI (see [setup-guide.md](setup-guide.md)).

---

## Elon’s 5-step process (applied to Fishtank wiring)

Treat **wiring + minimal visualization** as one feature: wiring without feedback is blind; animation without data is decoration.

1. **Make requirements less dumb**  
   Implicit “MC parity + swimming fish + every API” is too large for today. **Better requirement:** in **&lt; 5 seconds**, a human sees **what the loop is doing** and can **intervene** (nudge, reassign, open task). Fish physics can wait.

2. **Delete**  
   Skip full auth pages (#45 as env/header only), skip import wizard (#93/#94 deferred), skip image gallery (#51), skip physics. Remove or don’t add components that are not on the path of **one end-to-end product run**.

3. **Simplify**  
   One **SSE + REST** story: context or small store, existing **mappers**, agents as **simple cards** (emoji + status), convoy as **accordion/list**, cost as **one number** (then a sparkline). No Redux, no WebGL, no D3 physics on day one.

4. **Accelerate cycle time**  
   Ship a **minimum tank**: SSE hooked, three **logical** views (convoy summary, agent pool, task stream/Kanban), **one** mutation (e.g. stall nudge or reassign) wired to a real `POST`. Run **one** idea → PR cycle while watching the UI.

5. **Automate (later)**  
   Auto-discover agents (#93), auto-layout on health (#94), smarter feed classification, optional layout prefs.

**Verdict:** You are not over-engineering yet; the risk is **weeks on animation** before **one** live autonomous cycle is watchable in the UI.

---

## Immediate prioritization

- **UI-wiring blockers (order of attack):** **#46** (if CORS/proxy pain appears) → **#52** (task provenance in API + UI) → **#51** (when visuals matter) → **#45** (when gateway identity hardening is required).  
- **Keep deferred:** **#93**, **#94** until agent count and ops needs justify them.

---

## Expanded 48–72 hour sprint (lean)

_Goal: “holy shit it’s alive”—**real-time visibility** into the loop, not pixel-perfect fish._

### 1. Harden the live pipe (highest leverage)

- Keep **`useArmsLiveFeed`**, extend **`ssePayloadToFeedEvent`** for more **arms** event types and stable IDs.
- Optionally split **`useLiveEvents`** from context for reuse; add **filter** hooks (product — already; later convoy/agent).
- **Reconnect / error** — `onerror` state, banner, or backoff; don’t rely on silent `EventSource` behavior alone.
- **Reference:** [crshdn/mission-control](https://github.com/crshdn/mission-control) — `LiveFeed.tsx`: append-only list, filters, disconnect handling (adapt to Vite + `arms` payloads).

### 2. Three core views (no physics)

- **Convoy overview** — List active convoys for the product (REST: product convoys + task-scoped convoy when present). Status colors; later **DAG** from `GET /api/convoys/{id}/graph`. Buttons only for **real** endpoints (dispatch-ready, mail)—no fake pause/kill until API exists.
- **Agent pool** — Registry + heartbeats: extend **`GET /api/agents`** usage if not already; show load and health strings. **Reassign** only when wired to **`POST`** paths that match **#107** auto-reassign flows (operator may use stall-nudge first).
- **Task stream** — Kanban or dense list: show **title, status, agent, cost hints** from REST + SSE. Add **#52** styling when API exposes a flag/source.

### 3. Live cost + quick wins

- Header or corner: **aggregate cost** for product (poll breakdown or sum from SSE `cost_recorded` if mapped). Cap warning when **`cost_caps`** or API supports it.
- **Reference:** MC `components/costs/*` for layout ideas—start with one number + simple trend.

### 4. Polish & E2E

- Theme consistency (existing theme cycle).  
- Mobile: tabs/drawer for feed + agents (queue-only mobile is limiting).  
- Run **one** full cycle with **no manual refresh**; fix gaps with refetch or state merge.

---

## What to borrow from Mission Control (patterns, not a port)

Upstream: [crshdn/mission-control](https://github.com/crshdn/mission-control) (Next.js + SSE + orchestration UI).

| Area | MC pointers | Fishtank note |
|------|-------------|----------------|
| Real-time | `LiveFeed.tsx`, SSE hook patterns | Same **EventSource** mental model; align event shapes with **arms** outbox. |
| Convoy | `ConvoyTab.tsx`, `DependencyGraph.tsx` | List → **react-flow** or nested list first. |
| Agents | `AgentsSidebar.tsx`, `HealthIndicator.tsx` | Badges + stale heartbeat; match **arms** agent-health fields. |
| Tasks | `MissionQueue.tsx`, `TaskModal.tsx`, `TaskChatTab.tsx` | Card chrome, columns, modal sections—wire to **arms** routes when ready. |
| Costs | `components/costs/*` | Breakdowns later; single aggregate first. |

**Defer copying:** full swipe deck, heavy Next.js routing, workspace isolation UI unless you need worktrees in-app.

---

## Verdict

Mission Control is a **battle-tested reference** for SSE + orchestration layout. Fishtank should **ship the pipe + three views + one intervention** first, then iterate on motion and parity. After that sprint, re-run the 5 steps: delete unused views, simplify animations, add SSE event types and API methods as needed.

---

_Last updated: 2026-03-22 · Cross-check gap doc for current open-row count._
