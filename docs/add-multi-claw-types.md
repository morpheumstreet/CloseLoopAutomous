# Multi-claw gateway types in **arms**

**arms** (this repo) routes task dispatch to several external agent runtimes. Each runtime speaks a different wire protocol. The server does **not** assume a single “OpenClaw-only” stack: execution agents bind to **persisted gateway profiles** (driver + URL + auth), and [`RoutingGateway`](../arms/internal/adapters/gateway/routing_gateway.go) picks the right client.

This document summarizes **why** the protocols differ, **what arms implements today**, and **what may still be worth building** if you need deeper parity (streaming, pairing, health, and so on).

---

## Why NullClaw is not a drop-in OpenClaw client

NullClaw is **not** a byte-compatible replacement for OpenClaw on the wire. It can align on **config / identity** concepts (and upstream offers migration tooling), but the **default gateway path is different**:

| Aspect | OpenClaw-class (incl. ZeroClaw) | Stock NullClaw (HTTP A2A) |
|--------|----------------------------------|----------------------------|
| Primary transport | WebSocket JSON-RPC (e.g. `connect`, `chat.send`) | JSON-RPC 2.0 **HTTP POST** to `…/a2a` (`message/send`, A2A-shaped) |
| Typical ports / paths | Custom WS (e.g. 18789-style URLs you configure) | HTTP gateway origin; client appends `/a2a` unless the URL already ends with `/a2a` |
| Extra channels | WS events for progress / chat | Optional WebChannel WS, `/health`, pairing flows — **not all are used by arms today** |

So “add NullClaw” in arms means **a dedicated HTTP client** (and separate driver string), not reusing the OpenClaw WebSocket stack.

---

## What arms implements (current shape)

### Port: `AgentGateway`

[`ports.AgentGateway`](../arms/internal/ports/gateway.go) is the execution-plane abstraction:

- `DispatchTask(ctx, task) (externalRef, error)`
- `DispatchSubtask(ctx, parent, sub) (externalRef, error)`

Implementations are wired in [`platform/wiring.go`](../arms/internal/platform/wiring.go) (high level): remote drivers go through **`RoutingGateway`** + a pooled client; the in-process stub handles `stub`.

### Routing: agent → endpoint → driver

1. Each **task** has a `CurrentExecutionAgentID`.
2. [`TargetResolver`](../arms/internal/adapters/gateway/target_resolver.go) loads the **execution agent**, then its **`gateway_endpoint_id`**, then the **gateway endpoint** row (driver, URL, token, device id, timeout).
3. Non-stub remote drivers require a non-empty **`session_key`** on the execution agent (same idea as “which session to send to” on the remote gateway).

### Persisted profiles (no gateway URLs in env)

Gateway connection profiles live in SQLite (migration `030_gateway_endpoints.sql`) and are managed over HTTP:

- `GET /api/gateway-endpoints` — list profiles  
- `POST /api/gateway-endpoints` — create `{ driver, gateway_url, … }`  
- Execution agents: `POST /api/agents` with `gateway_endpoint_id` and `session_key` (see comments in [`config/arms.toml`](../config/arms.toml)).

Default RPC timeout when `timeout_sec` is `0` is controlled by config (e.g. `OPENCLAW_DISPATCH_TIMEOUT_SEC` in TOML / env).

### Canonical **driver** strings

Defined in [`domain/gateway_endpoint.go`](../arms/internal/domain/gateway_endpoint.go). [`NormalizeGatewayDriver`](../arms/internal/domain/gateway_endpoint.go) accepts several aliases (e.g. `nullclaw_http` → `nullclaw_a2a`).

| Driver constant | Meaning |
|-----------------|--------|
| `stub` | In-process [`SimulationMockClaw`](../arms/internal/adapters/gateway/simulation_mock_claw.go) |
| `openclaw_ws` | OpenClaw WebSocket client ([`openclaw`](../arms/internal/adapters/gateway/openclaw/)) |
| `nullclaw_ws` | **Legacy** OpenClaw-shaped WebSocket RPC (not stock NullClaw HTTP); kept for compatibility |
| `nullclaw_a2a` | NullClaw HTTP **POST /a2a** ([`nullclaw.Client`](../arms/internal/adapters/gateway/nullclaw/client.go)) |
| `picoclaw_ws` | Pico Protocol WebSocket `message.send` + `session_id` ([`picoclaw`](../arms/internal/adapters/gateway/picoclaw/)) |
| `zeroclaw_ws` | ZeroClaw: OpenClaw-compatible WS sequence via [`zeroclaw`](../arms/internal/adapters/gateway/zeroclaw/) (wraps shared OpenClaw wire helpers) |
| `zclaw_relay_http` | [zclaw](https://github.com/tnm/zclaw) web relay: HTTP `POST …/api/chat` via [`zclaw`](../arms/internal/adapters/gateway/zclaw/) (`gateway_token` → `X-Zclaw-Key` when the relay requires `ZCLAW_WEB_API_KEY`) |

[`clientPool`](../arms/internal/adapters/gateway/pool.go) reuses clients per `(driver, url, token, device_id, timeout)` and dispatches to the matching implementation.

### Knowledge hook

OpenClaw, ZeroClaw, PicoClaw, and NullClaw HTTP clients can all attach **knowledge snippets** to dispatch payloads when a `KnowledgeForDispatch` callback is configured (same hook path as OpenClaw dispatch).

---

## Adapter packages (where to look in code)

| Package | Role |
|---------|------|
| [`arms/internal/adapters/gateway/routing_gateway.go`](../arms/internal/adapters/gateway/routing_gateway.go) | `AgentGateway` facade: resolve target, route to stub or pool |
| [`arms/internal/adapters/gateway/pool.go`](../arms/internal/adapters/gateway/pool.go) | Client pooling + driver switch for task/subtask dispatch |
| [`arms/internal/adapters/gateway/openclaw/`](../arms/internal/adapters/gateway/openclaw/) | OpenClaw WS RPC + markdown/query helpers |
| [`arms/internal/adapters/gateway/nullclaw/`](../arms/internal/adapters/gateway/nullclaw/) | NullClaw A2A HTTP JSON-RPC |
| [`arms/internal/adapters/gateway/picoclaw/`](../arms/internal/adapters/gateway/picoclaw/) | PicoClaw Pico Protocol WS |
| [`arms/internal/adapters/gateway/zeroclaw/`](../arms/internal/adapters/gateway/zeroclaw/) | ZeroClaw WS (OpenClaw-class) |
| [`arms/internal/adapters/gateway/zclaw/`](../arms/internal/adapters/gateway/zclaw/) | zclaw web relay HTTP (`/api/chat`) |
| [`arms/internal/adapters/sqlite/gateway_endpoints.go`](../arms/internal/adapters/sqlite/gateway_endpoints.go) | Persistence for profiles |

---

## Possible follow-ups (not a commitment)

The older “Mission Control / TypeScript runtime adapter” checklist mapped many **UI and product** concerns. For **arms** specifically, partial or future enhancements might include:

- **Richer NullClaw integration**: WebChannel streaming, pairing (`/pair`), stricter upstream defaults, or health polling — today dispatch centers on **HTTP `/a2a`**.
- **Per-driver capabilities**: feature flags if a driver lacks checkpoints, convoy nudges, or other OpenClaw-only behaviors.
- **Observability**: passthrough metrics/traces if you standardize on OTel across runtimes.

Effort for those items depends on whether arms must **observe** live progress on the gateway vs **only** dispatch + rely on **webhooks** and DB state (which is already the common pattern for completion).

---

## Quick operational recipe

1. Create a gateway profile: `POST /api/gateway-endpoints` with the right `driver` and `gateway_url` (for `nullclaw_a2a`, use the HTTP **origin**; the client adds `/a2a` when needed).  
2. Create an execution agent: `POST /api/agents` with `gateway_endpoint_id` and `session_key` appropriate for that runtime.  
3. Assign work so tasks use that execution agent; **arms** resolves the endpoint and uses the matching client.

For local defaults and examples, see [`config/arms.toml`](../config/arms.toml).
