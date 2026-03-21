import { ChevronRight, Search, Zap } from 'lucide-react';
import { useMemo } from 'react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { Agent } from '../../domain/types';

export function AgentsPanel() {
  const { activeWorkspace, agents } = useMissionUi();

  const list = useMemo(() => {
    if (!activeWorkspace) return [];
    return agents.filter((a) => a.workspaceId === activeWorkspace.id);
  }, [agents, activeWorkspace]);

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
            disabled
            placeholder="Search (shell)"
            aria-label="Search agents"
            style={{
              width: '100%',
              padding: '0.5rem 0.65rem 0.5rem 2.25rem',
              borderRadius: 'var(--ft-radius-sm)',
              border: '1px solid var(--mc-border)',
              background: 'var(--mc-bg)',
              color: 'var(--mc-text-secondary)',
              font: 'inherit',
              fontSize: '0.75rem',
            }}
          />
        </div>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0.5rem' }}>
        {list.map((agent) => (
          <AgentRow key={agent.id} agent={agent} />
        ))}
      </div>
    </aside>
  );
}

function AgentRow({ agent }: { agent: Agent }) {
  const badge = agentBadge(agent.status);
  return (
    <div className="ft-agent-row">
      <Zap size={16} color="var(--mc-accent-yellow)" aria-hidden />
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
