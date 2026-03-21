import { ChevronRight, Search, Zap } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { Agent } from '../../domain/types';

export function AgentsPanel() {
  const { activeWorkspace, agents } = useMissionUi();
  const [query, setQuery] = useState('');

  const list = useMemo(() => {
    if (!activeWorkspace) return [];
    const scoped = agents.filter((a) => a.workspaceId === activeWorkspace.id);
    const q = query.trim().toLowerCase();
    if (!q) return scoped;
    return scoped.filter((a) => a.name.toLowerCase().includes(q) || a.id.toLowerCase().includes(q));
  }, [agents, activeWorkspace, query]);

  return (
    <aside className="ft-sidebar">
      <div className="ft-border-b" style={{ padding: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronRight size={16} className="ft-muted" />
          <span className="ft-upper-label">Agents</span>
        </div>
        <div style={{ marginTop: '0.65rem', position: 'relative' }}>
          <Search
            size={16}
            className="ft-muted"
            style={{ position: 'absolute', left: 10, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}
          />
          <input
            className="ft-input ft-input--sm ft-input--leading-icon"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter by task or status"
            aria-label="Search agents"
            style={{ width: '100%' }}
          />
        </div>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0.5rem' }}>
        {list.length === 0 ? (
          <p className="ft-muted" style={{ fontSize: '0.75rem', padding: '0.5rem' }}>
            No agent heartbeats for this product (or agent health is disabled on the server).
          </p>
        ) : (
          list.map((agent) => <AgentRow key={agent.id} agent={agent} />)
        )}
      </div>
    </aside>
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
          OpenClaw session (demo)
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
