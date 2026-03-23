# Multi-claw gateway types in **arms**

**arms** (this repo) routes task dispatch to several external agent runtimes. Each runtime speaks a different wire protocol. The server does **not** assume a single “OpenClaw-only” stack: execution agents bind to **persisted gateway profiles** (driver + URL + auth), and [`RoutingGateway`](../arms/internal/adapters/gateway/routing_gateway.go) picks the right client.

This document summarizes **why** the protocols differ, **what arms implements today**, and **what may still be worth building** if you need deeper parity (streaming, pairing, health, and so on).

---

## Why NullClaw is not a drop-in OpenClaw client

NullClaw is **not** a byte-compatible replacement for OpenClaw on the wire. It can align on **config / identity** concepts (and upstream offers migration tooling), but the **default gateway path is different**:

| Aspect | OpenClaw-class (incl. ZeroClaw, Clawlet, IronClaw) | Stock NullClaw (HTTP A2A) |
|--------|------------------------------------------------------|----------------------------|
| Primary transport | WebSocket JSON-RPC (e.g. `connect`, `chat.send`) | JSON-RPC 2.0 **HTTP POST** to `…/a2a` (`message/send`, A2A-shaped) |
| Typical ports / paths | Custom WS URLs you configure | HTTP gateway origin; client appends `/a2a` unless the URL already ends with `/a2a` |
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
3. Non-stub remote drivers require a non-empty **`session_key`** on the execution agent (runtime-specific: OpenClaw-style session, MimiClaw `chat_id`, Nanobot `--session`, zclaw placeholder label, MisterMorph optional `topic_id`, etc.).

### Persisted profiles (no gateway URLs in env)

Gateway connection profiles are normally stored in SQLite (migration `030_gateway_endpoints.sql`) and exposed over HTTP:

- `GET /api/gateway-endpoints` — list profiles (`gateway_token` is always empty in JSON; `has_gateway_token` indicates a stored secret)
- `POST /api/gateway-endpoints` — create `{ driver, gateway_url, … }`
- `PATCH /api/gateway-endpoints/{id}` — partial update (omit `gateway_token` to leave unchanged)
- `DELETE /api/gateway-endpoints/{id}` — remove (409 if an execution agent still references it)

Execution agents: `POST /api/agents` with `gateway_endpoint_id` and `session_key` (see [`config/arms.toml`](../config/arms.toml)).

An in-memory [`GatewayEndpointStore`](../arms/internal/adapters/memory/gateway_endpoints.go) implements the same registry port for tests and lightweight setups.

Default RPC timeout when `timeout_sec` is `0` is controlled by config (e.g. `OPENCLAW_DISPATCH_TIMEOUT_SEC` in TOML / env).

### Canonical **driver** strings

Defined in [`domain/gateway_endpoint.go`](../arms/internal/domain/gateway_endpoint.go). [`NormalizeGatewayDriver`](../arms/internal/domain/gateway_endpoint.go) accepts aliases (e.g. `nullclaw_http` → `nullclaw_a2a`, `zclaw` → `zclaw_relay_http`).

| Driver constant | Meaning |
|-----------------|--------|
| `stub` | In-process [`SimulationMockClaw`](../arms/internal/adapters/gateway/simulation_mock_claw.go) |
| `openclaw_ws` | OpenClaw WebSocket client ([`openclaw`](../arms/internal/adapters/gateway/openclaw/)); also the **default** pool branch for `nullclaw_ws` (legacy OpenClaw-shaped WS) |
| `nullclaw_ws` | **Legacy** OpenClaw-shaped WebSocket RPC (not stock NullClaw HTTP) |
| `nullclaw_a2a` | NullClaw HTTP **POST /a2a** ([`nullclaw.Client`](../arms/internal/adapters/gateway/nullclaw/client.go)) |
| `picoclaw_ws` | Pico Protocol WebSocket `message.send` + `session_id` ([`picoclaw`](../arms/internal/adapters/gateway/picoclaw/)) |
| `zeroclaw_ws` | ZeroClaw: OpenClaw-compatible WS ([`zeroclaw`](../arms/internal/adapters/gateway/zeroclaw/)) |
| `clawlet_ws` | [Clawlet](https://github.com/mosaxiv/clawlet) OpenClaw-class WS ([`clawlet`](../arms/internal/adapters/gateway/clawlet/)) |
| `ironclaw_ws` | IronClaw (Rust OpenClaw-class gateway; same WS flow as OpenClaw) — [`ironclaw`](../arms/internal/adapters/gateway/ironclaw/) |
| `mimiclaw_ws` | MimiClaw JSON WebSocket ([`mimiclaw`](../arms/internal/adapters/gateway/mimiclaw/)); `session_key` = `chat_id` |
| `nanobot_cli` | [HKUDS nanobot](https://github.com/HKUDS/nanobot) via subprocess `nanobot agent -m` ([`nanobotcli`](../arms/internal/adapters/gateway/nanobotcli/)); **not** a WebSocket gateway — see [Non-URL drivers](#non-url-and-overloaded-fields) |
| `zclaw_relay_http` | [zclaw](https://github.com/tnm/zclaw) web relay: HTTP `POST …/api/chat` ([`zclaw`](../arms/internal/adapters/gateway/zclaw/)); `gateway_token` → `X-Zclaw-Key` when the relay requires `ZCLAW_WEB_API_KEY` |
| `mistermorph_http` | [MisterMorph](https://github.com/quailyquaily/mistermorph) daemon HTTP API: `POST …/tasks` + poll ([`mistermorph`](../arms/internal/adapters/gateway/mistermorph/)); optional model via `device_id`; optional `topic_id` via execution agent `session_key` |

[`clientPool`](../arms/internal/adapters/gateway/pool.go) reuses clients per `(driver, url, token, device_id, timeout)` and dispatches to the matching implementation.

### Non-URL and overloaded fields

Some drivers reuse `gateway_url` / `gateway_token` / `device_id` for non-network settings:

- **`nanobot_cli`**: [`RoutingGateway`](../arms/internal/adapters/gateway/routing_gateway.go) allows an empty `gateway_url`. Mapping: `gateway_token` → optional path to the `nanobot` binary; `gateway_url` → optional config path (`-c`); `device_id` → optional workspace (`-w`); execution agent `session_key` → `--session` (e.g. `cli:direct`). See comments in [`config/arms.toml`](../config/arms.toml).
- **`zclaw_relay_http`**: `session_key` must be non-empty for validation but is **not** sent on the wire (single serial bridge per relay).
- **`mistermorph_http`**: `gateway_url` = runtime base URL; `gateway_token` = Bearer (`server.auth_token`); `device_id` = optional JSON `model` override.

### Knowledge hook

Remote clients in the pool that support it pass the same **`KnowledgeForDispatch`** callback into OpenClaw-class adapters, PicoClaw, MimiClaw, NullClaw HTTP, zclaw relay, MisterMorph, Clawlet, IronClaw, and Nanobot CLI (where applicable).

---

## Adapter packages (where to look in code)

| Package | Role |
|---------|------|
| [`routing_gateway.go`](../arms/internal/adapters/gateway/routing_gateway.go) | `AgentGateway` facade: resolve target, route to stub or pool |
| [`pool.go`](../arms/internal/adapters/gateway/pool.go) | Client pooling + per-driver dispatch for task/subtask |
| [`openclaw/`](../arms/internal/adapters/gateway/openclaw/) | OpenClaw WS RPC + markdown/query helpers |
| [`nullclaw/`](../arms/internal/adapters/gateway/nullclaw/) | NullClaw A2A HTTP JSON-RPC |
| [`picoclaw/`](../arms/internal/adapters/gateway/picoclaw/) | PicoClaw Pico Protocol WS |
| [`zeroclaw/`](../arms/internal/adapters/gateway/zeroclaw/) | ZeroClaw WS (OpenClaw-class) |
| [`clawlet/`](../arms/internal/adapters/gateway/clawlet/) | Clawlet WS (OpenClaw-class) |
| [`ironclaw/`](../arms/internal/adapters/gateway/ironclaw/) | IronClaw WS (OpenClaw-class) |
| [`mimiclaw/`](../arms/internal/adapters/gateway/mimiclaw/) | MimiClaw JSON WS |
| [`nanobotcli/`](../arms/internal/adapters/gateway/nanobotcli/) | Nanobot subprocess dispatch |
| [`zclaw/`](../arms/internal/adapters/gateway/zclaw/) | zclaw web relay HTTP (`/api/chat`) |
| [`mistermorph/`](../arms/internal/adapters/gateway/mistermorph/) | MisterMorph HTTP task API |
| [`sqlite/gateway_endpoints.go`](../arms/internal/adapters/sqlite/gateway_endpoints.go) | SQLite persistence for profiles |
| [`memory/gateway_endpoints.go`](../arms/internal/adapters/memory/gateway_endpoints.go) | In-memory registry (tests / demos) |

---

## Possible follow-ups (not a commitment)

The older “Mission Control / TypeScript runtime adapter” checklist mapped many **UI and product** concerns. For **arms** specifically, partial or future enhancements might include:

- **Richer NullClaw integration**: WebChannel streaming, pairing (`/pair`), stricter upstream defaults, or health polling — today dispatch centers on **HTTP `/a2a`**.
- **Per-driver capabilities**: feature flags if a driver lacks checkpoints, convoy nudges, or other OpenClaw-only behaviors.
- **Observability**: passthrough metrics/traces if you standardize on OTel across runtimes.

Effort for those items depends on whether arms must **observe** live progress on the gateway vs **only** dispatch + rely on **webhooks** and DB state (which is already the common pattern for completion).

---

## Quick operational recipe

1. Create a gateway profile: `POST /api/gateway-endpoints` with the right `driver` and fields for that driver (for `nullclaw_a2a`, use the HTTP **origin**; the client adds `/a2a` when needed).  
2. Create an execution agent: `POST /api/agents` with `gateway_endpoint_id` and a `session_key` valid for that driver.  
3. Assign work so tasks use that execution agent; **arms** resolves the endpoint and uses the matching client.

For per-driver examples (Clawlet, IronClaw, Nanobot, MimiClaw, zclaw, etc.), see [`config/arms.toml`](../config/arms.toml) and [`routes_catalog.go`](../arms/internal/adapters/httpapi/routes_catalog.go) (`POST /api/gateway-endpoints` line).
