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

If you're planning to run it, start with the quickstart in the README, provide strong Anthropic keys, and probably stay in "Supervised" mode (manual merge) until you're confident in the agent's output quality. Very cool and forward-looking project! 🚀