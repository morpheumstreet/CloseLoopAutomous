import { useState } from 'react';
import { Activity, Folder, Plus, RefreshCw, Rocket } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { WorkspaceStats } from '../../domain/types';
import { CreateProductModal } from './CreateProductModal';

export function WorkspaceDashboardView() {
  const {
    workspaces,
    openWorkspace,
    refreshWorkspaces,
    registerProduct,
    listLoading,
    apiError,
    dismissError,
  } = useMissionUi();
  const [modalOpen, setModalOpen] = useState(false);

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
              <button
                type="button"
                className="ft-btn-ghost"
                onClick={() => void refreshWorkspaces()}
                disabled={listLoading}
                title="Reload products from arms"
              >
                <RefreshCw size={16} className={listLoading ? 'ft-spin' : ''} />
                Refresh
              </button>
              <button type="button" className="ft-btn-ghost" disabled title="Autopilot UI (next)">
                <Rocket size={16} />
                Autopilot
              </button>
              <button type="button" className="ft-btn-ghost" disabled title="Activity dashboard (next)">
                <Activity size={16} />
                Activity Dashboard
              </button>
              <button type="button" className="ft-btn-primary" onClick={() => setModalOpen(true)}>
                <Plus size={16} />
                New product
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="ft-container" style={{ paddingBlock: '1.5rem' }}>
        {apiError ? (
          <div className="ft-banner ft-banner--error" style={{ marginBottom: '1rem' }} role="alert">
            <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '0.75rem' }}>
              <span>{apiError}</span>
              <button type="button" className="ft-btn-ghost" onClick={dismissError}>
                Dismiss
              </button>
            </div>
          </div>
        ) : null}

        <div style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '0.35rem' }}>Products</h2>
          <p className="ft-muted">
            Data from arms — each card is a <code style={{ fontSize: '0.85em' }}>GET /api/products</code> row with live
            task counts
          </p>
        </div>

        {listLoading && workspaces.length === 0 ? (
          <p className="ft-muted" style={{ padding: '2rem 0' }}>
            Loading products…
          </p>
        ) : workspaces.length === 0 ? (
          <EmptyWorkspaces onCreate={() => setModalOpen(true)} />
        ) : (
          <div className="ft-grid-ws ft-animate-slide-in">
            {workspaces.map((w) => (
              <WorkspaceCard key={w.id} workspace={w} onOpen={() => void openWorkspace(w)} />
            ))}
            <button type="button" className="ft-add-card" onClick={() => setModalOpen(true)}>
              <div className="ft-add-card-icon">
                <Plus size={24} className="ft-muted" />
              </div>
              <span style={{ fontWeight: 600 }}>Add product</span>
            </button>
          </div>
        )}
      </main>

      <CreateProductModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onCreate={(name, workspaceId) => registerProduct(name, workspaceId)}
      />
    </div>
  );
}

function EmptyWorkspaces({ onCreate }: { onCreate: () => void }) {
  return (
    <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
      <Folder size={64} className="ft-muted" style={{ marginInline: 'auto', marginBottom: '1rem' }} />
      <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.5rem' }}>No products yet</h3>
      <p className="ft-muted" style={{ marginBottom: '1.5rem' }}>
        Create a product in arms to see it here, or use the button below.
      </p>
      <button type="button" className="ft-btn-primary" onClick={onCreate}>
        Create product
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
