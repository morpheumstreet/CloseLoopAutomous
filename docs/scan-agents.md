# Unified Agent Identity — design specification (arms + fishtank)

**Status:** **MVP shipped** — `domain.AgentIdentity`, SQLite **`agent_profiles`**, synthesis from **`gateway_endpoints`** + optional **`ARMS_GEOIP2_CITY`**, **`GET /api/agents`** field **`identities[]`**, **`/api/fleet/*`**, SSE **`agent_identity_updated`**, Fishtank Agents panel. **Not yet done:** per-driver live **`GetIdentity`** from WebSocket handshakes, richer metrics, **`GET /api/agents/{id}`** as identity (use **`/api/fleet/identities/{id}`**).

**Related:** `arms/internal/domain/gateway_endpoint.go` (driver strings), `arms/internal/ports/gateway.go` (`AgentGateway` today — dispatch only), [api-ref.md](api-ref.md), [fishtank-ui-todos.md](fishtank-ui-todos.md).

**Repo alignment (important):**

- Prefer **`arms/internal/domain/agent_identity.go`** for structs and **`arms/internal/ports/`** for new optional interfaces (e.g. `GatewayIdentityProvider`). Do **not** assume `arms/internal/driver/interface.go` exists; today execution uses **`AgentGateway`** + **`adapters/gateway`**. A unified `Driver` interface can be introduced only if the codebase is refactored to match.
- **`GET /api/agents` already exists** — returns `registry` (execution agents) + `items` (task heartbeats). Evolving it to **`[]AgentIdentity`** (or adding `identities[]` / versioned response) is a **breaking or additive** API change; plan migration and Fishtank updates.
- **Primary DB is SQLite** (`arms/internal/adapters/sqlite/migrations/`). Index expressions below use **SQLite JSON** (`json_extract`); PostgreSQL `JSONB` variants are noted as optional.

---

## 1. Core Go struct — single source of truth

**Location (proposed):** `arms/internal/domain/agent_identity.go`

```go
package domain

import "time"

// AgentIdentity is the canonical, unified representation of any agent across all drivers/claws.
// Every driver MUST map its native data into this exact shape (or return partial + fill Custom).
type AgentIdentity struct {
	// Core identifiers (arms + agent reported)
	ID         string `json:"id" db:"id"`                   // Primary: device_id / UUID from handshake or heartbeat
	GatewayURL string `json:"gateway_url" db:"gateway_url"` // Full endpoint used to connect (ws://, https://, …)

	// Descriptive / human-facing
	Name    string `json:"name" db:"name"`       // e.g. "Nemo-MacStudio", "Proxy-GPT4o-HK"
	Driver  string `json:"driver" db:"driver"`   // e.g. "openclaw_ws", "metaclaw_http", "copaw_http"
	Version string `json:"version" db:"version"` // Client / software version

	// Runtime state
	Status   AgentStatus `json:"status" db:"status"`       // online | offline | error | busy
	LastSeen time.Time   `json:"last_seen" db:"last_seen"` // Last heartbeat or status update

	// Capabilities & hardware
	Capabilities []string     `json:"capabilities" db:"capabilities"` // tool_calling, vision, browser, code_exec, …
	Platform     PlatformInfo `json:"platform" db:"platform_json"`
	Metrics      Metrics      `json:"metrics" db:"metrics_json"`

	// Multi-agent / hierarchical support
	SubAgents []SubAgentRef `json:"sub_agents,omitempty" db:"sub_agents_json"`

	// IP-derived geolocation (city-level, offline lookup)
	Geo *GeoLocation `json:"geo,omitempty" db:"geo_json"`

	// Escape hatch for framework-specific extras
	Custom map[string]any `json:"custom,omitempty" db:"custom_json"`
}

type AgentStatus string

const (
	StatusOnline  AgentStatus = "online"
	StatusOffline AgentStatus = "offline"
	StatusError   AgentStatus = "error"
	StatusBusy    AgentStatus = "busy"
)

type PlatformInfo struct {
	OS       string `json:"os"`   // darwin, linux, windows
	Arch     string `json:"arch"` // amd64, arm64
	Hostname string `json:"hostname"`
	GPU      string `json:"gpu,omitempty"` // Apple M3 Ultra, NVIDIA RTX 4090, …
}

type Metrics struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemUsedMB   int64   `json:"mem_used_mb"`
	MemTotalMB  int64   `json:"mem_total_mb"`
	DiskUsedGB  int64   `json:"disk_used_gb"`
	DiskTotalGB int64   `json:"disk_total_gb"`
	GPUMemMB    int64   `json:"gpu_mem_mb,omitempty"`
}

type SubAgentRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"` // reasoning, fast, worker-3, claude-3.5-sonnet, …
}

type GeoLocation struct {
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	City        string    `json:"city,omitempty"`
	Region      string    `json:"region,omitempty"`
	Country     string    `json:"country"`
	CountryISO  string    `json:"country_iso"`
	AccuracyKM  int       `json:"accuracy_km"`
	Source      string    `json:"source"` // maxmind_geoip2, manual, none
	LastUpdated time.Time `json:"last_updated"`
}
```

**OpenClaw-class reference:** WS drivers should treat OpenClaw handshake + heartbeat JSON as the **reference mapping** into `AgentIdentity` (device id, name, host metrics, capabilities → `Platform` / `Metrics` / `Capabilities`).

---

## 2. Driver / gateway execution extension

**Ideal contract (pseudocode — wire into `ports` + `adapters/gateway`, not a fictional path):**

```go
// Example: arms/internal/ports/gateway_identity.go (new)
type GatewayIdentityProvider interface {
	GetIdentity(ctx context.Context, endpoint *domain.GatewayEndpoint) (*domain.AgentIdentity, error)
	ListIdentities(ctx context.Context, endpoint *domain.GatewayEndpoint) ([]*domain.AgentIdentity, error)
}
```

**Semantics:**

- **`GetIdentity`** — On connect and on heartbeat tick; single primary row for that endpoint.
- **`ListIdentities`** — Default implementation returns **one** element (`GetIdentity`); relay / multi-workspace drivers return **many** (e.g. CoPaw workspaces, zclaw downstream agents).

**Heartbeat loop:** Optional `HeartbeatLoop(ctx, endpointID, ch chan<- *domain.AgentIdentity)` or reuse existing pool + periodic `GetIdentity` — avoid duplicating connection ownership.

---

## 3. Storage (database)

**New table** (SQLite-oriented). Store full document as JSON for flexibility; **denormalize** `status` for cheap indexing if needed.

```sql
-- SQLite (example; embed as new numbered migration under arms/internal/adapters/sqlite/migrations/)
CREATE TABLE IF NOT EXISTS agent_profiles (
  id TEXT PRIMARY KEY,
  gateway_id TEXT NOT NULL REFERENCES gateway_endpoints(id),
  gateway_url TEXT NOT NULL,
  identity_json TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'offline',
  last_updated TEXT NOT NULL DEFAULT (datetime('now')),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_gateway_id ON agent_profiles(gateway_id);
CREATE INDEX IF NOT EXISTS idx_agent_profiles_status ON agent_profiles(status);
-- Optional: generated column + index on json_extract(identity_json, '$.status') if you drop denormalized status
```

**PostgreSQL (if ever used):** equivalent with `JSONB` and `identity_json->>'status'` index as in the original sketch.

**Bump:** `ExpectedSchemaVersion` in `migrate.go` when adding the file.

---

## 4. Population flow

1. Operator registers a gateway (`gateway_endpoints` row — UI / API / config).
2. Driver connects (existing pool / dial path).
3. **`GetIdentity` / `ListIdentities`** run → build `AgentIdentity` (driver-specific mapping).
4. **Geo:** Resolve host from `GatewayURL` (DNS → IP); **offline** MaxMind GeoLite2 lookup → fill `Geo` (`Source: maxmind_geoip2` or `none` if DB missing).
5. **Upsert** `agent_profiles` (by `AgentIdentity.ID` + `gateway_id` as needed for multi-row gateways).
6. **SSE:** publish `agent_identity_updated` with payload (subset or full identity JSON).
7. **Heartbeat:** refresh `Metrics`, `Status`, `LastSeen`; re-run Geo only if egress IP / host changed (rate-limit).

**Geolocation notes:** MaxMind requires **license + on-disk DB** (e.g. GeoLite2-City.mmdb); document env var for path; **never** require network for lookup in offline-first mode.

---

## 5. API endpoints (arms) — implemented (MVP)

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/agents` | Returns **`identities[]`** (cached `AgentIdentity` rows) alongside **`registry`** and **`items`**. |
| `GET` | `/api/fleet/identities` | List identities (`?limit=`). |
| `GET` | `/api/fleet/identities/{id}` | Single identity by profile id. |
| `POST` | `/api/fleet/refresh` | Re-synthesize from **`gateway_endpoints`** (registry-based MVP, not live driver probe). |
| `GET` | `/api/fleet/geo-summary` | Country counts from identities with GeoIP data. |
| `GET` | `/api/agents/{id}` | *Not* identity — use **`/api/fleet/identities/{id}`** (avoids clash with execution agent + mailbox routes). |

Auth: same as existing operator routes (`MC_API_TOKEN`, `ARMS_ACL`, …).

---

## 6. Fishtank integration

- **Agents / Fleet** — Table: ID, Name, Driver, Status (dot), Gateway URL, Geo (city + pin), CPU/Mem, Last seen; **expandable** `SubAgents`.
- **Map** — Optional Leaflet (or similar) from `Geo` when lat/lon present.
- **After gateway create/edit** — Call `POST /api/fleet/refresh` (or single-gateway refresh) then refetch identities.
- **SSE** — Subscribe to `agent_identity_updated` (and merge into local store without full page reload).

**TypeScript:** Mirror `AgentIdentity` in `fishtank/src/api/armsTypes.ts` (or `domain/`) and extend `ArmsClient`.

---

## 7. Driver mapping quick reference (implementation)

| Driver | ID source | Name source | Geo source | Multi-agent? | Notes |
|--------|-----------|-------------|------------|--------------|-------|
| `openclaw_ws` | `device_id` from handshake | heartbeat / connect | MaxMind from URL host | Optional subs | Reference standard |
| `nemoclaw_ws` | `device_id` | heartbeat | MaxMind | Sandbox subs | NVIDIA extras → `Custom` |
| `metaclaw_http` | model / proxy id | model name | MaxMind | Single | `/v1/models` fallback if available |
| `copaw_http` | workspace ids | workspace names | MaxMind | Yes | One identity per workspace |
| `zclaw_relay_http` | relay-provided ids | relay | MaxMind | Yes | Split relay payload |
| `nanobot_cli` | hostname + pid | CLI / stdout | MaxMind (`127.0.0.1` → none) | Single | Local |
| `stub` | mock uuid | `"Stub Agent"` | `none` | Single | Tests |

**All 16 drivers** remain as in `domain/gateway_endpoint.go`; unlisted rows follow the same pattern: **one logical agent per endpoint** unless the wire format exposes multiple identities — then implement **`ListIdentities`**.

---

## 8. Implementation checklist (suggested order)

1. **Migration** — `agent_profiles` + repository in `adapters/sqlite`.
2. **`domain/agent_identity.go`** — types as above.
3. **`internal/geo` (or `platform/geoip`)** — MaxMind reader + URL-host → IP helper + tests (no network in unit tests).
4. **`ports` + first drivers** — `openclaw_ws` + `metaclaw_http` identity mapping behind feature flag.
5. **HTTP handlers** — extend or version `/api/agents`; add `GET …/{id}`, `POST /api/fleet/refresh`, optional `geo-summary`.
6. **SSE** — `agent_identity_updated` via existing outbox/hub.
7. **Fishtank** — TS types, `ArmsClient`, Fleet table + SSE handler.

---

## 9. Non-goals (this design)

- **Network scanning** — No mDNS / subnet discovery; registration stays manual.
- **Replacing `execution_agents` / `gateway_endpoints`** — `AgentIdentity` is a **runtime / presentation** layer; registry rows remain authoritative for dispatch targeting until product merges them.

---

## 10. Narrative guardrails

- **`GET /api/ops/host-metrics`** remains **arms server host** only — not a substitute for `AgentIdentity.Metrics` from agents.
- Identity **`Metrics` / `Version` / live `Status`** are still mostly placeholders until drivers report them; **`stub`** gateway is **`online`**, others default **`offline`** in the MVP synthesizer.
