import { Activity, Folder, Plus, Rocket } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { WorkspaceStats } from '../../domain/types';

export function WorkspaceDashboardView() {
  const { workspaces, openWorkspace } = useMissionUi();

  return (
    <div className="ft-screen">
      <header className="ft-border-b" style={{ background: 'var(--mc-bg-secondary)' }}>
        <div className="ft-container" style={{ paddingBlock: '1rem' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <span style={{ fontSize: '1.5rem' }} aria-hidden>
                🦞
              </span>
              <h1 style={{ fontSize: '1.25rem', fontWeight: 700 }}>Mission Control</h1>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
              <button type="button" className="ft-btn-ghost">
                <Rocket size={16} />
                Autopilot
              </button>
              <button type="button" className="ft-btn-ghost">
                <Activity size={16} />
                Activity Dashboard
              </button>
              <button type="button" className="ft-btn-primary">
                <Plus size={16} />
                New Workspace
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="ft-container" style={{ paddingBlock: '1.5rem' }}>
        <div style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '0.35rem' }}>All Workspaces</h2>
          <p className="ft-muted">Select a workspace to view its mission queue and agents</p>
        </div>

        {workspaces.length === 0 ? (
          <EmptyWorkspaces />
        ) : (
          <div className="ft-grid-ws ft-animate-slide-in">
            {workspaces.map((w) => (
              <WorkspaceCard key={w.id} workspace={w} onOpen={() => openWorkspace(w)} />
            ))}
            <button type="button" className="ft-add-card">
              <div className="ft-add-card-icon">
                <Plus size={24} className="ft-muted" />
              </div>
              <span style={{ fontWeight: 600 }}>Add Workspace</span>
            </button>
          </div>
        )}
      </main>
    </div>
  );
}

function EmptyWorkspaces() {
  return (
    <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
      <Folder size={64} className="ft-muted" style={{ marginInline: 'auto', marginBottom: '1rem' }} />
      <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.5rem' }}>No workspaces yet</h3>
      <p className="ft-muted" style={{ marginBottom: '1.5rem' }}>
        Create your first workspace to get started
      </p>
      <button type="button" className="ft-btn-primary">
        Create Workspace
      </button>
    </div>
  );
}

function WorkspaceCard({ workspace, onOpen }: { workspace: WorkspaceStats; onOpen: () => void }) {
  return (
    <button type="button" className="ft-ws-card" onClick={onOpen} style={{ textAlign: 'left', width: '100%' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: '1rem', gap: '0.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', minWidth: 0 }}>
          <span style={{ fontSize: '1.75rem' }} aria-hidden>
            {workspace.icon}
          </span>
          <div style={{ minWidth: 0 }}>
            <h3 className="ft-truncate" style={{ fontSize: '1.125rem', fontWeight: 600 }}>
              {workspace.name}
            </h3>
            <p className="ft-muted" style={{ fontSize: '0.875rem' }}>
              /{workspace.slug}
            </p>
          </div>
        </div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1.25rem', fontSize: '0.875rem' }} className="ft-muted">
        <span>{workspace.taskCounts.total} tasks</span>
        <span>
          {workspace.agentCounts.working}/{workspace.agentCounts.total} agents active
        </span>
      </div>
    </button>
  );
}
