# Arms production hardening

Operational checklist for running `arms` (`cmd/arms`) beyond local development. For HTTP routes and payloads, see [api-ref.md](api-ref.md). For environment variables, see `internal/config/config.go`.

---

## Secrets and configuration

- **`MC_API_TOKEN`** — Treat as a long-lived API secret. Generate with a CSPRNG (for example 32+ random bytes, hex or base64). Never commit it to git or bake it into image layers; inject at runtime (Kubernetes secret, Docker `--env-file`, hosted platform env). For **`GET /api/live/events`**, browsers using native `EventSource` typically pass the same secret as **`?token=`** (visible in query logs and referrers); prefer **`Authorization: Bearer`** when using fetch/readable-stream SSE clients, or terminate the stream behind same-origin / internal-only URLs.
- **`WEBHOOK_SECRET`** — HMAC key for `POST /api/webhooks/agent-completion`. Must match what the agent or bridge uses to sign the raw JSON body. Rotate by updating both sides; expect brief mismatch during rollout.
- **`OPENCLAW_GATEWAY_TOKEN`** and **`ARMS_OPENCLAW_SESSION_KEY`** — Gateway credentials and session routing for dispatch. Scope tokens per environment; restrict who can read deployment manifests.

Prefer **separate values per environment** (dev/staging/prod). If a secret leaks, rotate immediately and assume the old value is compromised.

---

## TLS and network placement

- **HTTP API** — `arms` listens on plain HTTP (default `ARMS_LISTEN`, e.g. `:8080`). In production, terminate **TLS at a reverse proxy** or cloud load balancer (Caddy, nginx, Envoy, AWS ALB, etc.) and forward HTTP to the container or process. Enforce HTTPS on the public URL only at the edge.
- **OpenClaw WebSocket** — Set **`OPENCLAW_GATEWAY_URL`** to a **`wss://`** URL when the gateway requires TLS. The client uses the [coder/websocket](https://github.com/coder/websocket) library; certificate validation follows normal Go TLS rules (system roots unless you customize the transport later).
- **Internal-only APIs** — If the server must not be reachable from the internet, bind to a private interface or loopback and rely on mesh/VPC rules; keep `MC_API_TOKEN` set even on internal networks if multiple tenants or teams share the mesh.

---

## Webhooks, proxies, and `NO_PROXY`

Outbound **callers** that POST to `arms` (for example an agent runtime notifying completion) often run behind an **HTTP(S) proxy** in corporate or cluster setups.

- If the proxy intercepts traffic to your `arms` URL and breaks connectivity or TLS, configure the caller to **bypass the proxy** for that host. Common patterns:
  - Environment variable **`NO_PROXY`** / **`no_proxy`** — comma-separated hostnames or CIDRs that should not use the proxy (for example `arms.internal`, `10.0.0.0/8`).
  - Ensure the webhook URL you configure points at a hostname that resolves to the real `arms` instance (internal DNS, service name in Kubernetes, etc.).
- **`X-Arms-Signature`** must be computed over the **exact raw request body** bytes the server receives; avoid middleware that rewrites the body before HMAC verification.

---

## Persistence and backups

- Set **`DATABASE_PATH`** to a **persistent volume** path in containers (see `arms/docker-compose.yml`).
- Enable **`ARMS_DB_BACKUP=1`** (or `true`) so migrations run **`VACUUM INTO`** a timestamped `*.pre-migrate-*.bak` next to the DB file before schema upgrades. This does not replace a full backup strategy; copy volume snapshots or files on a schedule for disaster recovery.

---

## Background worker (`cmd/arms-worker`) and Redis

When **`ARMS_REDIS_ADDR`** is set, **`cmd/arms`** reconciles **per-product** Asynq tasks (**`arms:product_autopilot_tick`**) on startup, **every 5 minutes**, and after product / product-schedule mutations; it also resyncs **`product_schedules`** Asynq rows on the same **5m** tick. **`ARMS_AUTOPILOT_TICK_SEC`** is **deprecated** (ignored; a warning is logged if set). **`cmd/arms-worker`** must run separately with the **same** **`DATABASE_PATH`** (and other DB-related env) so each task runs **`TickProduct`** against the same SQLite file as the API. The legacy **`arms:autopilot_tick`** task type (full **`TickScheduled`** sweep) remains available for manual enqueue.

- Treat Redis as **non-durable** scheduling metadata only; the source of truth remains SQLite (or in-memory when **`DATABASE_PATH`** is empty—in that case the worker is still useful for integration tests that point at a file DB).
- Run **one worker process** per Redis + DB pair unless you intentionally scale out (duplicate tasks for the same product are suppressed via Asynq **Unique**; **`TickProduct`** / **`TickScheduled`** should be safe to repeat; verify cadence idempotency for your workload).
- Lock down Redis (password, VPC) like any job broker; it carries task payloads metadata only for the current queue implementation.

---

## Container notes

- The **`arms/Dockerfile`** runs as **`nobody`** and ships **CA certificates** for outbound TLS (OpenClaw). Keep the image updated for base image security patches.
- Do not mount the SQLite file from a **network filesystem** that lacks proper locking if you observe corruption; prefer local disk or a documented-safe shared store.

---

## Logging and observability

- Use **`ARMS_LOG_JSON=1`** when shipping logs to a collector (Loki, CloudWatch, Datadog, etc.).
- **`X-Request-ID`** on responses supports correlating access logs with client or upstream errors.
- **`GET /api/health`** is suitable for load balancer health checks (no Bearer required).

---

## What arms does not provide yet

Rate limiting, IP allowlists, mTLS for the REST API, and automated secret rotation are **not** built into the binary. Implement those at the edge (API gateway, service mesh, or WAF) or extend the codebase when you need them.

---

## Related

- [api-ref.md](api-ref.md) — HTTP API, auth, env vars, worker binary notes.
- [arms-mission-control-gap-todos.md](arms-mission-control-gap-todos.md) — parity backlog with Mission Control (schema version, Asynq slices).
