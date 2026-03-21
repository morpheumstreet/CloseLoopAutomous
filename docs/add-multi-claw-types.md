**Supports Needed to Add NullClaw Compatibility to Autensa / Mission Control**

NullClaw is **not** a drop-in replacement for OpenClaw.  
It uses the **same config schema + identity format** (and has a built-in `nullclaw migrate openclaw` command), but its **communication protocol is completely different**:

- OpenClaw → custom WebSocket on **port 18789** with proprietary messages (`node.invoke`, `sessions_send`, etc.)
- NullClaw → **Google A2A v0.3.0** (JSON-RPC 2.0 over HTTP `/a2a`) + **WebChannel WebSocket** on **port 32123** (`/ws`) with `webchannel_v1.json` envelope

This is the **core incompatibility**.  
Mission Control was built exclusively for OpenClaw’s protocol, so you must add the items below.

### 1. Runtime Abstraction Layer (MUST-HAVE — Foundation)
- Create `src/lib/runtimes/` folder
- Define interface `RuntimeAdapter` with methods:
  - `connect()`
  - `dispatchTask(task, options)`
  - `streamProgress(taskId)`
  - `sendChatMessage(...)`
  - `getAgentHealth()`
  - `sendWebhookAck(...)`
  - `disconnect()`
- Refactor existing OpenClaw code into `OpenClawAdapter`
- Add new `NullClawAdapter`
- Add UI setting: `runtime_type: "openclaw" | "nullclaw"` (with auto-detection)

### 2. Configuration & Discovery Changes
- Add new environment / settings variables:
  - `NULLCLAW_HTTP_URL` (default: `http://127.0.0.1:3000`)
  - `NULLCLAW_WS_URL` (default: `ws://127.0.0.1:32123/ws`)
  - `NULLCLAW_A2A_ENABLED` (boolean)
  - `NULLCLAW_AUTH_MODE` (`token` | `pairing`)
- Add migration button in Settings → “Migrate from OpenClaw” that:
  - Runs `nullclaw migrate openclaw` via shell
  - Imports memory/config
  - Switches runtime flag automatically
- Support NullClaw’s stricter defaults (`workspace_only: true`, no `0.0.0.0` bind without flag)

### 3. Protocol Translation / Adapter Logic (Biggest Code Work)
Implement mapping in `NullClawAdapter`:

| Mission Control Action          | OpenClaw (current)       | NullClaw Equivalent (new)                     | Notes |
|---------------------------------|--------------------------|-----------------------------------------------|-------|
| Task dispatch                   | WS `node.invoke`         | A2A `message/send` or `tasks/*`               | JSON-RPC 2.0 |
| Progress / streaming            | WS events                | A2A `message/stream` + WebChannel WS          | Use `/ws` + webchannel_v1 envelope |
| Agent health / presence         | WS heartbeat             | A2A `tasks/get` + `/health` endpoint          | Partial |
| Inter-agent mailbox / chat      | WS `sessions_send`       | A2A `message/send`                            | Full support |
| Checkpoints & recovery          | Custom WS checkpoints    | Memory snapshots (via A2A or local)           | May need custom extension |
| Completion webhook              | POST to Mission Control  | Use existing `/webhook` endpoint (already supported) | Good match |
| Convoy / DAG coordination       | Custom WS                | A2A `tasks/*` + resubscribe                   | Partial — may lose full auto-nudge |

### 4. Authentication & Security Adjustments
- Support NullClaw’s **pairing flow** (default) + static token
- Add `/pair` endpoint handling (one-time code exchange)
- Handle `access_token` in WebChannel payloads
- Optional: E2E encryption support (X25519) if used

### 5. Webhook & Real-Time Layer Updates
- Keep existing `/api/webhooks/agent-completion` (NullClaw already sends to `/webhook`)
- Add fallback SSE bridge for A2A streaming → Mission Control live feed
- Update agent-health sidebar to poll `/health` + A2A `tasks/list`

### 6. Optional / Nice-to-Have Enhancements
- Prometheus / OpenTelemetry exporter passthrough (NullClaw supports OTel natively)
- Convoy Mode compatibility layer (if A2A `tasks/resubscribe` is not enough)
- Per-runtime feature flags (e.g. disable advanced checkpoints if NullClaw doesn’t expose them)
- UI badge “Running on NullClaw (Zig)” with binary size / RAM stats

### Effort Estimate
- **Minimal viable support** (basic task dispatch + progress): ~2–4 days for an experienced dev
- **Full feature parity** (Convoy, checkpoints, health, cost tracking): 1–2 weeks
- **Production-ready abstraction**: 2–3 weeks (recommended if you plan to support more claws later)

### Recommended First Steps
1. Add the `RuntimeAdapter` interface + config toggle (do this first)
2. Implement `NullClawAdapter` skeleton using A2A JSON-RPC
3. Test with `nullclaw migrate openclaw` + simple “Hello World” task
4. Gradually map the rest

Would you like me to sketch the actual TypeScript code for the `RuntimeAdapter` interface + `NullClawAdapter` skeleton right now? Or focus on one specific part (e.g. just the WebSocket translation)?