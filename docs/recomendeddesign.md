**Golang Architecture Outline for New Project: "GoAutensa" (Autensa / Mission Control reimplemented in Go)**

This is a **production-ready, clean-architecture redesign** of the original Autensa/Mission Control backend, fully in **Golang**.  
It keeps 100% functional compatibility with the existing **OpenClaw Gateway** (the Node.js AI runtime on port 18789), so you can start using it immediately without rewriting the agent execution layer.

You get:
- Single static binary (easy Docker/self-host)
- Superior concurrency & performance (goroutines + channels)
- Lower memory footprint than Next.js
- Same features: tasks, convoy DAGs, cost caps, checkpoints, swipe-to-ship, live activity feed, PR automation

### 1. High-Level System Split (same as original for easy migration)

```
Your Machine / Server
├── GoAutensa (new Go service – port 4000)
│   ├── REST API + SSE + WebSocket server
│   ├── Orchestration & safety logic
│   └── SQLite / Postgres
│
├── OpenClaw Gateway (keep original Node.js – port 18789)
│   └── Actual AI agents, LLM calls, tools, code writing
│
└── GitHub + Browser (UI)
```

Communication: **WebSocket** (bidirectional) + HTTP webhooks (for agent completion events).

Later you can replace OpenClaw with a pure-Go agent runtime (optional phase 2).

### 2. Recommended Tech Stack (modern, lightweight, battle-tested)

| Layer              | Technology                              | Why |
|--------------------|-----------------------------------------|-----|
| HTTP Router        | **Gin** (or Fiber)                      | Fast, middleware-rich, OpenAPI friendly |
| WebSocket          | **gorilla/websocket**                   | Mature, high performance |
| Real-time (SSE)    | Built-in `net/http` + channels          | Simple live activity feed |
| Database           | **SQLite** (modernc/sqlite) + **GORM** or **sqlx** | Drop-in replacement; optional Postgres support |
| Config / Env       | **viper** + **env**                     | 12-factor |
| Logging            | **zap** (structured)                    | Fast & production ready |
| Validation         | **validator.v10**                       | Same as original |
| Background jobs    | **Asynq** (Redis) or native goroutines  | For scheduled research cycles |
| Observability      | **Prometheus** + **OpenTelemetry**      | Metrics, traces, agent health |
| CLI & embedded UI  | **Templ** + **HTMX** + Tailwind (optional) | Zero-JS full dashboard in Go |
| Build              | Go 1.23+ → single binary                | `CGO_ENABLED=0` for static builds |

### 3. Project Directory Structure (Clean/Hexagonal Architecture)

```bash
goautensa/
├── cmd/
│   └── server/          # main.go + graceful shutdown
├── internal/
│   ├── api/             # Gin handlers + routes
│   │   ├── v1/
│   │   │   ├── tasks/
│   │   │   ├── products/
│   │   │   ├── convoy/
│   │   │   ├── costs/
│   │   │   ├── agents/
│   │   │   ├── webhooks/     # OpenClaw callbacks
│   │   │   └── live/         # SSE endpoint
│   │   └── middleware/       # auth, rate-limit, cost-guard
│   │
│   ├── core/            # Domain services (business logic)
│   │   ├── autopilot/
│   │   ├── convoy/           # DAG scheduler, dependency resolver
│   │   ├── costs/
│   │   ├── workspace/        # git worktree + port allocator
│   │   ├── checkpoint/
│   │   ├── agenthealth/
│   │   ├── mailbox/          # inter-agent chat
│   │   └── learner/
│   │
│   ├── ports/           # Interfaces (hexagonal)
│   │   ├── repository.go
│   │   ├── openclaw.go       # WS client interface
│   │   └── notifier.go
│   │
│   ├── adapter/
│   │   ├── repository/       # GORM/SQLite impl
│   │   ├── openclaw/         # WebSocket client to port 18789
│   │   └── notifier/         # SSE + WebSocket broadcaster
│   │
│   ├── domain/          # Entities & value objects
│   │   ├── task.go
│   │   ├── convoy.go
│   │   ├── idea.go
│   │   └── product.go
│   │
│   ├── config/
│   └── util/            # helpers (cost calc, git, etc.)
│
├── pkg/                 # Reusable packages (optional)
├── migrations/          # SQLite schema + GORM auto-migrate
├── web/                 # Optional embedded HTMX/Templ frontend
├── docker-compose.yml
└── go.mod
```

### 4. Core Backend Layers & Responsibilities

```
GoAutensa (port 4000)
├── API Layer (Gin)
│   └── 80+ endpoints (tasks, products, convoy, costs, webhooks, agents…)
│
├── Real-time Layer
│   ├── SSE (/live) → activity feed
│   ├── WebSocket server → operator chat relay
│   └── OpenClaw WS Client (goroutine per connection)
│
├── Domain Services (core/)
│   ├── Autopilot pipeline (research → ideation → planning)
│   ├── Convoy orchestrator (DAG scheduler using goroutines + channels)
│   ├── Cost guard + budget enforcement
│   ├── Workspace isolation service (git worktrees + port pool 4200-4299)
│   ├── Checkpoint & crash-recovery manager
│   └── Agent watchdog (detect stalled → nudge/restart)
│
├── Persistence (adapter/repository)
│   └── SQLite (same schema as original: products, tasks, convoys, costs, checkpoints…)
│
└── OpenClaw Adapter
    └── WebSocket client (reconnects automatically, heartbeat, JSON messages)
```

### 5. Key Data & Control Flow (same as original but in Go)

1. Product → Research/Ideation (background goroutine)  
2. User swipes → approve → Planning service  
3. Convoy service builds DAG → spawns subtasks  
4. Each subtask dispatched via OpenClaw WS client  
5. OpenClaw executes → sends progress + checkpoints via WS  
6. Webhook receiver (`/webhooks/agent-completion`) → updates DB + triggers next stage  
7. Reviewer agent → serialized merge queue → GitHub PR

All heavy work runs in **goroutines** with **context cancellation** and **errgroup** for clean shutdown.

### 6. Safety & Resilience (even stronger in Go)

- Real-time cost tracking with hard caps (atomic counters)
- Per-task git worktree isolation + port allocator (sync.Pool + mutex)
- Periodic checkpoints saved to DB → resume on restart
- Agent watchdog goroutine (every 30s health check)
- Merge queue with product-level locking
- Graceful shutdown + panic recovery middleware
- Configurable daily/monthly budget enforcement at service level

### 7. Deployment Options

- Single binary: `./goautensa --port 4000 --openclaw ws://127.0.0.1:18789`
- Docker + docker-compose (includes OpenClaw)
- Kubernetes-ready (horizontal scaling with Redis for Asynq queues)
- PM2 alternative: systemd service or `supervisor`

### 8. Migration Path from Original Autensa

1. Start with this Go backend + existing OpenClaw + existing SQLite file (GORM can auto-migrate safely)
2. Keep using the original Next.js frontend temporarily (just change base URL)
3. Phase 2: Replace frontend with embedded Templ/HTMX version (zero Node.js)
4. Phase 3 (optional): Write pure-Go agent runtime and remove OpenClaw dependency

This design gives you **exactly** the same product experience as Autensa but with Go’s performance, simplicity, and single-binary deployment.

Would you like me to:
- Generate the full `go.mod` + folder scaffolding commands?
- Provide sample code for the OpenClaw WebSocket client?
- Or the Convoy DAG scheduler implementation sketch?

Just say the word and I’ll drop the next layer of detail. 🚀