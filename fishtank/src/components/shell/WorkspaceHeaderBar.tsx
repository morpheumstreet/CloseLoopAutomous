import { useEffect, useMemo, useState } from 'react';
import { ChevronLeft, LayoutGrid, Rocket, Settings, Zap } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import { formatClock } from '../../lib/time';

export function WorkspaceHeaderBar() {
  const { activeWorkspace, isOnline, goHome, tasks, agents } = useMissionUi();
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const scopedTasks = useMemo(() => {
    if (!activeWorkspace) return tasks;
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  const scopedAgents = useMemo(() => {
    if (!activeWorkspace) return agents;
    return agents.filter((a) => a.workspaceId === activeWorkspace.id);
  }, [agents, activeWorkspace]);

  const workingAgents = scopedAgents.filter((a) => a.status === 'working').length;
  const tasksInQueue = scopedTasks.filter((t) => t.status !== 'done' && t.status !== 'review').length;

  return (
    <header className="ft-header-bar">
      <div className="ft-header-left">
        <div className="ft-logo-row">
          <Zap aria-hidden size={20} color="var(--mc-accent-cyan)" />
          <span className="ft-upper-label" style={{ color: 'var(--mc-text)', fontWeight: 600 }}>
            Mission Control
          </span>
        </div>

        {activeWorkspace ? (
          <>
            <button type="button" className="ft-btn-icon" onClick={goHome} title="All workspaces">
              <ChevronLeft size={16} />
              <LayoutGrid size={16} />
            </button>
            <div className="ft-chip ft-truncate" style={{ maxWidth: 'min(40vw, 280px)' }}>
              <span aria-hidden>{activeWorkspace.icon}</span>
              <span className="ft-truncate" style={{ fontWeight: 600 }}>
                {activeWorkspace.name}
              </span>
            </div>
          </>
        ) : (
          <button type="button" className="ft-chip" onClick={goHome} title="Home">
            <LayoutGrid size={16} />
            <span style={{ fontSize: '0.875rem' }}>All Workspaces</span>
          </button>
        )}
      </div>

      {activeWorkspace ? (
        <div className="ft-show-lg">
          <div className="ft-stat-lg">
            <div className="ft-stat-lg-value" style={{ color: 'var(--mc-accent-cyan)' }}>
              {workingAgents}
            </div>
            <div className="ft-stat-lg-label">Agents active</div>
          </div>
          <div className="ft-stat-lg">
            <div className="ft-stat-lg-value" style={{ color: 'var(--mc-accent-purple)' }}>
              {tasksInQueue}
            </div>
            <div className="ft-stat-lg-label">Tasks in queue</div>
          </div>
        </div>
      ) : null}

      <div className="ft-header-right">
        <span className="ft-time" style={{ fontVariantNumeric: 'tabular-nums' }}>
          {formatClock(now)}
        </span>
        <div className={`ft-online-pill ${isOnline ? 'ft-online-pill--on' : 'ft-online-pill--off'}`}>
          <span
            className={`ft-dot ${isOnline ? 'ft-dot--pulse' : ''}`}
            style={{ background: isOnline ? 'var(--mc-accent-green)' : 'var(--mc-accent-red)' }}
          />
          {isOnline ? 'ONLINE' : 'OFFLINE'}
        </div>
        <button type="button" className="ft-btn-icon" title="Autopilot (UI shell)">
          <Rocket size={20} />
        </button>
        <button type="button" className="ft-btn-icon" title="Settings (UI shell)">
          <Settings size={20} />
        </button>
      </div>
    </header>
  );
}
