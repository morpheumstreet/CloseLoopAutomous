import { ChevronRight, MapPin, RefreshCw, Search, Server, Zap } from 'lucide-react';
import { useMemo, useState } from 'react';
import type { ApiAgentIdentity, ApiAgentRegistryRow } from '../../api/armsTypes';
import { useMissionUi } from '../../context/MissionUiContext';
import type { Agent } from '../../domain/types';

type AgentsPanelProps = { embedded?: boolean };

export function AgentsPanel({ embedded = false }: AgentsPanelProps) {
  const {
    activeWorkspace,
    agents,
    executionAgentRegistry,
    agentRegistryHealthStub,
    fleetAgentIdentities,
    refreshFleetIdentities,
    workspaces,
  } = useMissionUi();
  const [query, setQuery] = useState('');
  const [fleetRefreshing, setFleetRefreshing] = useState(false);

  const productLabel = useMemo(() => {
    const m = new Map<string, string>();
    for (const w of workspaces) {
      m.set(w.id, w.name.trim() || w.id);
    }
    return m;
  }, [workspaces]);

  const fleetIds = fleetAgentIdentities ?? [];

  const fleetList = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return fleetIds;
    return fleetIds.filter((r) => {
      const geo = r.geo;
      const geoHay = geo
        ? `${geo.city ?? ''} ${geo.country ?? ''} ${geo.country_iso ?? ''}`.toLowerCase()
        : '';
      const hay = `${r.name} ${r.id} ${r.driver} ${r.gateway_url} ${geoHay}`.toLowerCase();
      return hay.includes(q);
    });
  }, [fleetIds, query]);

  const registryList = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return executionAgentRegistry;
    return executionAgentRegistry.filter((r) => {
      const pid = r.product_id?.trim() ?? '';
      const pname = pid ? (productLabel.get(pid) ?? pid) : '';
      const hay = `${r.display_name} ${r.id} ${r.source} ${r.external_ref} ${pid} ${pname}`.toLowerCase();
      return hay.includes(q);
    });
  }, [executionAgentRegistry, productLabel, query]);

  const list = useMemo(() => {
    if (!activeWorkspace) return [];
    const scoped = agents.filter((a) => a.workspaceId === activeWorkspace.id);
    const q = query.trim().toLowerCase();
    if (!q) return scoped;
    return scoped.filter((a) => a.name.toLowerCase().includes(q) || a.id.toLowerCase().includes(q));
  }, [agents, activeWorkspace, query]);

  const shellClass = embedded ? 'ft-mc-agents-embed' : 'ft-sidebar';

  return (
    <aside className={shellClass}>
      <div className={embedded ? 'ft-mc-agents-embed-head' : 'ft-border-b'} style={{ padding: embedded ? '0.5rem 0.75rem' : '0.75rem' }}>
        {!embedded ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
            <ChevronRight size={16} className="ft-muted" />
            <span className="ft-upper-label">Agents</span>
          </div>
        ) : (
          <span className="ft-upper-label">Agents</span>
        )}
        <div style={{ marginTop: embedded ? '0.5rem' : '0.65rem', position: 'relative' }}>
          <Search
            size={16}
            className="ft-muted"
            style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}
          />
          <input
            className="ft-input ft-input--sm ft-input--leading-icon"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter agents"
            aria-label="Search agents"
            style={{ width: '100%' }}
          />
        </div>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0.5rem', minHeight: 0 }}>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '0.5rem',
            margin: '0.25rem 0.5rem 0.35rem',
          }}
        >
          <span className="ft-upper-label" style={{ fontSize: '0.65rem', opacity: 0.85 }}>
            Gateway identities
          </span>
          <button
            type="button"
            className="ft-btn ft-btn--ghost ft-btn--sm"
            disabled={fleetRefreshing}
            title="POST /api/fleet/refresh — re-synthesize from gateway_endpoints"
            onClick={() => {
              setFleetRefreshing(true);
              void refreshFleetIdentities().finally(() => setFleetRefreshing(false));
            }}
            style={{ display: 'inline-flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.65rem' }}
          >
            <RefreshCw size={12} className={fleetRefreshing ? 'ft-spin' : ''} aria-hidden />
            Refresh
          </button>
        </div>
        <p className="ft-muted" style={{ fontSize: '0.65rem', padding: '0 0.5rem 0.5rem', lineHeight: 1.45, margin: 0 }}>
          Unified <code className="ft-mono">AgentIdentity</code> from gateways (Geo via optional{' '}
          <code className="ft-mono">ARMS_GEOIP2_CITY</code>). SSE <code className="ft-mono">agent_identity_updated</code> updates this list.
        </p>
        {fleetList.length === 0 ? (
          <p className="ft-muted" style={{ fontSize: '0.75rem', padding: '0.5rem', lineHeight: 1.5 }}>
            No identity rows yet. Use Refresh after adding gateways, or open the app with a migrated SQLite DB.
          </p>
        ) : (
          fleetList.map((row) => <IdentityRow key={row.id} row={row} />)
        )}

        <div
          className="ft-upper-label"
          style={{ fontSize: '0.65rem', margin: '0.85rem 0.5rem 0.35rem', paddingTop: '0.5rem', borderTop: '1px solid var(--mc-border)', opacity: 0.85 }}
        >
          Registered execution agents
        </div>
        <p className="ft-muted" style={{ fontSize: '0.65rem', padding: '0 0.5rem 0.5rem', lineHeight: 1.45, margin: 0 }}>
          From <code className="ft-mono">GET /api/agents</code> — fleet-wide registry (independent of live gateway sockets).
          {agentRegistryHealthStub ? (
            <>
              {' '}
              Global heartbeat snapshot is stubbed (agent health not configured on the server).
            </>
          ) : null}
        </p>
        {registryList.length === 0 ? (
          <p className="ft-muted" style={{ fontSize: '0.75rem', padding: '0.5rem', lineHeight: 1.5 }}>
            No execution agents registered yet. Create a gateway profile, then{' '}
            <code className="ft-mono">POST /api/agents</code> with <code className="ft-mono">gateway_endpoint_id</code>.
          </p>
        ) : (
          registryList.map((row) => (
            <RegistryAgentRow key={row.id} row={row} productLabel={productLabel} activeProductId={activeWorkspace?.id ?? null} />
          ))
        )}

        <div
          className="ft-upper-label"
          style={{ fontSize: '0.65rem', margin: '0.85rem 0.5rem 0.35rem', paddingTop: '0.5rem', borderTop: '1px solid var(--mc-border)', opacity: 0.85 }}
        >
          Task liveness{activeWorkspace ? ` · ${activeWorkspace.name}` : ''}
        </div>
        {!activeWorkspace ? (
          <p className="ft-muted" style={{ fontSize: '0.75rem', padding: '0.5rem', lineHeight: 1.5 }}>
            Open a workspace to see task heartbeats for that product.
          </p>
        ) : list.length === 0 ? (
          <p className="ft-muted" style={{ fontSize: '0.75rem', padding: '0.5rem', lineHeight: 1.5 }}>
            No task heartbeats yet for this product. Rows appear when something reports liveness for a task (see{' '}
            <code className="ft-mono">PATCH /api/tasks/…/agent-health</code>, dispatch/complete flows, or your agent runtime).
            If arms returns 503 for agent-health, persistence may be off (e.g. in-memory server without SQLite).
          </p>
        ) : (
          list.map((agent) => <AgentRow key={agent.id} agent={agent} />)
        )}
      </div>
    </aside>
  );
}

function IdentityRow({ row }: { row: ApiAgentIdentity }) {
  const [open, setOpen] = useState(false);
  const subs = row.sub_agents?.length ? row.sub_agents : [];
  const badge = identityStatusBadge(row.status);
  const geoLine =
    row.geo && row.geo.source && row.geo.source !== 'none'
      ? [row.geo.city, row.geo.region, row.geo.country].filter(Boolean).join(', ') || row.geo.country_iso
      : '';
  return (
    <div className="ft-agent-row" style={{ flexWrap: 'wrap' }}>
      <MapPin size={16} color="var(--mc-accent)" aria-hidden />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div className="ft-truncate" style={{ fontWeight: 600, fontSize: '0.8rem' }}>
          {row.name?.trim() || row.id}
        </div>
        <div style={{ fontSize: '0.65rem' }} className="ft-muted">
          <span className="ft-mono">{row.driver}</span>
          {geoLine ? ` · ${geoLine}` : ''}
        </div>
        {row.gateway_url ? (
          <div className="ft-truncate ft-mono" style={{ fontSize: '0.6rem', opacity: 0.75, marginTop: 2 }}>
            {row.gateway_url}
          </div>
        ) : null}
        {subs.length > 0 ? (
          <button
            type="button"
            className="ft-btn ft-btn--ghost ft-btn--sm"
            onClick={() => setOpen((o) => !o)}
            style={{ marginTop: 4, fontSize: '0.6rem', padding: '0.1rem 0.35rem' }}
          >
            {open ? 'Hide' : 'Show'} sub-agents ({subs.length})
          </button>
        ) : null}
        {open && subs.length > 0 ? (
          <ul style={{ margin: '0.35rem 0 0', paddingLeft: '1rem', fontSize: '0.65rem' }} className="ft-muted">
            {subs.map((s) => (
              <li key={s.id}>
                {s.name} <span className="ft-mono">({s.id})</span>
                {s.role ? ` · ${s.role}` : ''}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
      <span className={badge.className}>{badge.label}</span>
    </div>
  );
}

function identityStatusBadge(status: string): { label: string; className: string } {
  switch (status.toLowerCase()) {
    case 'online':
      return { label: 'ONLINE', className: 'ft-agent-badge ft-agent-badge--working' };
    case 'busy':
      return { label: 'BUSY', className: 'ft-agent-badge ft-agent-badge--working' };
    case 'error':
      return { label: 'ERROR', className: 'ft-agent-badge ft-agent-badge--offline' };
    case 'offline':
    default:
      return { label: 'OFFLINE', className: 'ft-agent-badge ft-agent-badge--offline' };
  }
}

function RegistryAgentRow({
  row,
  productLabel,
  activeProductId,
}: {
  row: ApiAgentRegistryRow;
  productLabel: Map<string, string>;
  activeProductId: string | null;
}) {
  const pid = row.product_id?.trim() ?? '';
  const pname = pid ? (productLabel.get(pid) ?? pid) : 'any product';
  const inScope = Boolean(activeProductId && (!pid || pid === activeProductId));
  const badge =
    activeProductId == null
      ? { label: 'FLEET', title: 'All registered agents', className: 'ft-agent-badge ft-agent-badge--standby' as const }
      : inScope
        ? { label: 'HERE', title: 'Tied to this workspace or global', className: 'ft-agent-badge ft-agent-badge--standby' as const }
        : { label: 'OTHER', title: 'Different product', className: 'ft-agent-badge ft-agent-badge--standby' as const };
  return (
    <div className="ft-agent-row">
      <Server size={16} color="var(--mc-accent)" aria-hidden />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div className="ft-truncate" style={{ fontWeight: 600, fontSize: '0.8rem' }}>
          {row.display_name?.trim() || row.id}
        </div>
        <div style={{ fontSize: '0.65rem' }} className="ft-muted">
          <span className="ft-mono">{row.id.slice(0, 12)}</span>
          {row.source ? ` · ${row.source}` : ''} · {pname}
        </div>
      </div>
      <span className={badge.className} title={badge.title} style={badge.label === 'OTHER' ? { opacity: 0.65 } : undefined}>
        {badge.label}
      </span>
    </div>
  );
}

function AgentRow({ agent }: { agent: Agent }) {
  const badge = agentBadge(agent.status);
  return (
    <div className="ft-agent-row">
      <Zap size={16} color="var(--mc-accent)" aria-hidden />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div className="ft-truncate" style={{ fontWeight: 600, fontSize: '0.8rem' }}>
          {agent.name}
        </div>
        <div style={{ fontSize: '0.65rem' }} className="ft-muted">
          From task agent-health (liveness)
        </div>
      </div>
      <span className={badge.className}>{badge.label}</span>
    </div>
  );
}

function agentBadge(status: Agent['status']): { label: string; className: string } {
  switch (status) {
    case 'working':
      return { label: 'WORKING', className: 'ft-agent-badge ft-agent-badge--working' };
    case 'offline':
      return { label: 'OFFLINE', className: 'ft-agent-badge ft-agent-badge--offline' };
    default:
      return { label: 'STANDBY', className: 'ft-agent-badge ft-agent-badge--standby' };
  }
}
