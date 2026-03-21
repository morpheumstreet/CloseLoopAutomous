**Review of https://github.com/crshdn/mission-control**

This repository, **crshdn/mission-control**, hosts **Autensa** (also referred to as Mission Control in earlier versions), described as **the world's first Autonomous Product Engine (APE)**. It's an ambitious self-hosted AI system that runs a nearly fully autonomous product improvement loop for software projects:

- AI agents research your market/competitors  
- Generate scored feature ideas  
- Present them in a Tinder-style swipe interface (Pass / Maybe / Yes / Now!)  
- On approval → automatically plan, code, test, review, and open GitHub PRs  

Human input is ideally limited to swiping on ideas; everything else runs on AI agents. It evolved from an AI agent orchestration dashboard into this full autopilot product engine (major jump in v2.0.0 around March 2026).

**Quick summary – strengths & trade-offs**

- Very innovative concept if it works reliably  
- Strong focus on safety: checkpoints, crash recovery, cost caps, workspace isolation (git worktrees), serialized merges  
- Excellent observability: Kanban board, live SSE activity feed, per-task cost tracking, agent health monitoring, operator chat  
- 80+ API endpoints, real-time WebSocket integration, Docker-ready  

- Still early/experimental: heavy dependence on a separate project (OpenClaw Gateway), requires powerful LLM access (Anthropic recommended), and full autonomy carries high risk of bad PRs  
- SQLite + local git worktrees → great for solo/self-hosted, but scaling to large teams/repos would need careful thought  

**High-level design & architecture**

The system splits cleanly into two main parts:

1. **Autensa / Mission Control** (this repo)  
   → Next.js (TypeScript) full-stack app  
   → Dashboard, business logic, database, API, UI, autopilot orchestration  
   → Runs on port 4000 (default)

2. **OpenClaw Gateway** (separate repo: https://github.com/openclaw/openclaw)  
   → AI agent runtime & execution engine  
   → Handles actual LLM calls, tool usage, long-running agent tasks  
   → Runs on port 18789 (default)  
   → Communicates with Autensa via **WebSocket**

**Core architecture diagram** (adapted from README ASCII art):

```
┌──────────────────────────────────────────────────────────────────────┐
│                          YOUR MACHINE / SERVER                       │
│                                                                      │
│  ┌─────────────────────┐                ┌─────────────────────────┐  │
│  │ Autensa             │  WebSocket     │ OpenClaw Gateway        │  │
│  │ (Next.js App)       │◄─────────────►│ (AI Agent Runtime)      │  │
│  │ Port 4000           │                │ Port 18789              │  │
│  └──────────┬──────────┘                └─────────────┬───────────┘  │
│             │                                         │              │
│             ▼                                         ▼              │
│  ┌─────────────────────┐                ┌─────────────────────────┐  │
│  │ SQLite Database     │                │ LLM Providers           │  │
│  │ (tasks, ideas,      │                │ (Anthropic, OpenAI,     │  │
│  │  costs, products…)  │                │  Google, … via gateway) │  │
│  └─────────────────────┘                └─────────────────────────┘  │
│                                                                      │
│                   ┌───────────────────────────────┐                 │
│                   │ GitHub (receives PRs)         │                 │
│                   └───────────────────────────────┘                 │
└──────────────────────────────────────────────────────────────────────┘
```

**Main data & control flow (Autopilot pipeline)**

1. **Research phase** → agents use LLMs to analyze market/competitors → results saved to DB  
2. **Ideation phase** → generate feature ideas with impact/feasibility scores + reasoning  
3. **Swipe UI** → user approves/rejects (trains per-product preference model)  
4. **Planning** → approved idea → clarifying Q&A → detailed spec  
5. **Convoy Mode** (for complex features)  
   - Creates a DAG of parallel + dependent sub-tasks  
   - Multiple specialized agents: Builder, Tester, Reviewer, Learner…  
   - Checkpoints + crash recovery (resume from last good state)  
   - Auto-nudge stalled agents  
   - Inter-agent mailbox / chat  
6. **Execution** → code written in isolated git worktree/sandbox  
7. **Review & ship** → AI reviewer → opens PR (or auto-merge in higher automation tiers)  

**Key supporting systems**

- **Workspace isolation** — git worktrees + port ranges (4200–4299) prevent conflicts  
- **Cost & safety** — real-time cost tracking, daily/monthly caps, merge queue  
- **Observability** — Kanban (7 stages), live activity feed (SSE), agent health sidebar  
- **Persistence** — SQLite (auto-migrations)  
- **Deployment** — Docker + docker-compose, PM2, Tailscale-friendly for multi-machine  

Overall, it's a clean **frontend-heavy orchestration layer** sitting on top of a powerful **WebSocket-based agent executor** (OpenClaw). The architecture prioritizes safety, observability, and recoverability — which is smart given how error-prone full AI autonomy can be.


**In https://github.com/crshdn/mission-control**, there is **no separate traditional backend** (no dedicated Express/FastAPI/Rails/Go service, no `/backend` or `/server` folder outside of Next.js conventions).

The project is built as a **full-stack Next.js application** (TypeScript), and the "backend" is implemented using **Next.js API routes**.

### Where is the backend code located?

- **Main backend location**: `src/app/api/`  
  This folder contains all the server-side API endpoints (Route Handlers in the App Router).

Key subfolders / endpoint groups include:

- `src/app/api/tasks/` → Task CRUD, planning, dispatch to agents, status updates  
- `src/app/api/agents/` → Agent management, registration, health  
- `src/app/api/openclaw/` → Proxy / wrapper endpoints that talk to the OpenClaw Gateway  
- `src/app/api/webhooks/` → Receive callbacks when agents finish / produce events  
- Plus various others (products, costs, convoy mode, events/SSE, settings, etc.)

There are reportedly **80+ API endpoints** in total, all living inside this Next.js structure.

### How does the actual "heavy backend" work?

While the Next.js app (running on port 4000) handles:

- The dashboard UI  
- Business logic (task queuing, Kanban state, cost tracking, user preferences)  
- SQLite database access (`src/lib/db/`)  
- API authentication & validation  

The **real execution backend** — i.e. where the AI agents actually run, call LLMs, use tools, write code, etc. — is **not** in this repository.

That part lives in the separate project:

**OpenClaw Gateway**  
→ https://github.com/openclaw/openclaw  
→ Runs on its own port (default: **18789**)  
→ Mission Control connects to it via **WebSocket** (`ws://127.0.0.1:18789` or remote)  
→ This is the AI runtime / agent executor

### Simplified architecture recap (backend perspective)

```
User browser
     ↕ HTTP + WebSocket + SSE
Mission Control / Autensa
  (Next.js full-stack app – port 4000)
  ├── Frontend (React components, Kanban, swipe UI)
  ├── Backend = Next.js API routes (src/app/api/…) 
  │     ├── Manages tasks/products/ideas/costs in SQLite
  │     └── Proxies & orchestrates via WebSocket → OpenClaw
  └── 
       ↕ WebSocket (bidirectional, real-time)
OpenClaw Gateway (separate repo – port 18789)
  └── Actual AI agent runtime:
        • LLM calls (Anthropic, OpenAI, …)
        • Tool execution
        • Multi-agent convoy / DAG
        • Code writing in git worktrees
        • Checkpoints & recovery
```

**Short answer to your question:**

- The **dashboard's backend** = Next.js API routes in `src/app/api/` (this repo)  
- The **AI execution backend** = OpenClaw Gateway (separate repo, separate process)

You run **both** services: the Next.js app + the OpenClaw Gateway. The dashboard essentially acts as a very sophisticated client / orchestrator to the real agent-running backend (OpenClaw).


Here is a clean, structured **backend architecture outline** for **Autensa / Mission Control** (crshdn/mission-control, v2.0.1 as of March 2025), based on its current design as a full-stack Next.js application with heavy orchestration responsibilities.

### 1. Overall System Split (Logical Backend View)
- **Orchestration & Control Plane**  
  → Lives in **this repo** (Next.js)  
  → Port 4000  
  → Next.js API Routes = the effective "backend"

- **Execution Plane (AI Runtime)**  
  → Separate repo: **OpenClaw Gateway**  
  → Port 18789  
  → Runs actual LLM calls, tool usage, agent logic, code writing

- **Communication**  
  Bidirectional **WebSocket** (`ws://…:18789`)  
  + HTTP callbacks via **webhooks** from OpenClaw → Autensa

### 2. Backend Layers inside Mission Control (Next.js)

```
Mission Control Backend (Next.js API Routes + Libraries)
├── HTTP/REST API Layer
│   └── src/app/api/                         # 80+ endpoints
│       ├── tasks/                           # Core task lifecycle (CRUD, dispatch, status, chat)
│       ├── products/                        # Product CRUD, research/ideation cycles, swipe decisions
│       ├── agents/                          # Agent registration, health, mailbox (inter-agent messages)
│       ├── costs/                           # Cost events, breakdowns, budget caps enforcement
│       ├── convoy/                          # Convoy-specific: subtask mail, dependency coordination
│       ├── openclaw/                        # Proxy & helper routes to talk to Gateway
│       ├── webhooks/                        # agent-completion webhook receiver (HMAC protected)
│       ├── events/                          # Possibly internal event pub/sub helpers
│       ├── workspaces/                      # Workspace/port management
│       └── others (demo, files, task-images…) 
│
├── Real-time Layer
│   ├── SSE → /api/live                      # Live activity feed (research → build → PR events)
│   └── WebSocket client → OpenClaw Gateway  # Task dispatch, progress, health, chat relay
│
├── Domain Logic / Services (src/lib/)
│   ├── autopilot/          # Research → ideation → swipe → planning pipeline
│   ├── convoy.ts           # Multi-agent DAG orchestration, scheduling, dependency resolution
│   ├── costs/              # Tracking, capping, reporting logic
│   ├── db/                 # SQLite client + schema + migration runner
│   ├── openclaw/           # Gateway client (WS connection, dispatch, device identity)
│   ├── agent-health.ts     # Stalled/zombie detection, auto-nudge/restart
│   ├── checkpoint.ts       # Periodic save + crash recovery logic
│   ├── workspace-isolation.ts  # Git worktree creation, port allocation (4200–4299), merge queue
│   ├── mailbox.ts          # Inter-agent message store & delivery
│   ├── chat-listener.ts    # Operator ↔ Agent chat queuing & relay
│   └── learner.ts          # Failure/success lesson capture for future prompts
│
└── Persistence
    └── SQLite (mission-control.db)
        ├── Tables: products, ideas, tasks, convoys, cost_events, agent_health, checkpoints, mailbox, …
        └── Features: auto-migrations, pre-migration backups, cascade deletes, safety guards
```

### 3. Core Backend Flows (Simplified)

**Idea → Code → PR (Autopilot path)**

1. Product triggers research/ideation cycle  
   → `/api/products` + autopilot/ logic  
   → Ideas stored → user swipes via UI

2. Approved idea → planning phase  
   → `/api/tasks` creates parent task  
   → Clarifying questions → spec generation

3. If simple → single agent task dispatched  
   If complex → Convoy Mode  
   → `/api/convoy` + convoy.ts creates DAG of subtasks  
   → Multiple agents dispatched in parallel (with deps)

4. Dispatch to OpenClaw Gateway  
   → openclaw/ client → WebSocket  
   → Gateway executes → calls LLMs + tools  
   → Sends progress/checkpoints via WS  
   → On finish → webhook to `/api/webhooks/agent-completion`

5. Safety & observability during execution  
   - Checkpoints saved periodically → crash → resume  
   - Cost tracking → pause if cap hit  
   - Agent health monitor → nudge/reassign stalled agents  
   - Workspace isolation → git worktree + port range

6. Completion  
   → Reviewer agent → diff check  
   → Serialized merge queue (workspace-isolation.ts)  
   → GitHub PR created (or auto-merged in higher tiers)

### 4. Safety & Resilience Mechanisms (Backend-enforced)

- **Cost guardrails** — real-time accumulation → hard pause  
- **Workspace isolation** — per-task git worktree, unique ports, serialized merges  
- **Crash recovery** — checkpoints + resume API  
- **Agent watchdog** — detects zombie/stuck agents → auto-retry  
- **Database safety** — backups before migrations, skip destructive ops on populated DB  
- **Concurrency control** — product-scoped locks, merge queue  
- **Preference learning** — swipe history → per-product model tuning

### Summary Table – Backend Responsibilities

| Layer                  | Owned by              | Main Responsibilities                              | Tech / Location                  |
|------------------------|-----------------------|-----------------------------------------------------|----------------------------------|
| API / REST             | Mission Control       | CRUD, orchestration, proxying                       | `src/app/api/`                   |
| Real-time updates      | Mission Control       | Live feed (SSE), agent chat, progress               | `/api/live` + WS client          |
| Task & Convoy Logic    | Mission Control       | DAG planning, dispatch sequencing, recovery         | `src/lib/convoy.ts`, `autopilot/`|
| Cost & Safety          | Mission Control       | Budget enforcement, workspace isolation, checkpoints| `src/lib/costs/`, `workspace-isolation.ts` |
| AI Execution           | OpenClaw Gateway      | LLM calls, tool usage, code gen, testing            | Separate repo (WS API)           |
| Persistence            | Mission Control       | All state (tasks, costs, agents, ideas, checkpoints)| SQLite + `src/lib/db/`           |

This architecture keeps the **heavy AI execution** decoupled (OpenClaw) while **Mission Control** acts as a very capable orchestrator, safety net, and observability hub — all within a single Next.js process for simplicity of self-hosting.
