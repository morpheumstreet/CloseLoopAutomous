import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Activity, Folder, Info, Plus, RefreshCw, Rocket, Unplug } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import { BackendConnectionPill } from '../shell/BackendConnectionPill';
import { ThemeCycleButton } from '../shell/ThemeCycleButton';
import { AboutModal } from '../shell/AboutModal';
import type { WorkspaceStats } from '../../domain/types';
import { CreateProductModal } from './CreateProductModal';

export function WorkspaceDashboardView() {
  const navigate = useNavigate();
  const {
    workspaces,
    refreshWorkspaces,
    registerProduct,
    listLoading,
    apiError,
    dismissError,
    isOnline,
    fetchVersion,
    goHome,
    armsEnv,
  } = useMissionUi();
  const [modalOpen, setModalOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);

  useEffect(() => {
    goHome();
  }, [goHome]);

  return (
    <div className="ft-screen">
      <header className="ft-border-b ft-dashboard-header" style={{ background: 'var(--mc-bg-secondary)' }}>
        <div className="ft-container" style={{ paddingBlock: '1rem' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
              <span style={{ fontSize: '1.5rem' }} aria-hidden>
                🦞
              </span>
              <h1 style={{ fontSize: '1.25rem', fontWeight: 700, letterSpacing: '-0.02em' }}>Mission Control</h1>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
              <BackendConnectionPill isOnline={isOnline} />
              <ThemeCycleButton />
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
              <button type="button" className="ft-btn-ghost" onClick={() => navigate('/autopilot')} title="Autopilot">
                <Rocket size={16} />
                Autopilot
              </button>
              <button type="button" className="ft-btn-ghost" onClick={() => navigate('/activity')} title="Operations log">
                <Activity size={16} />
                Activity Dashboard
              </button>
              <button type="button" className="ft-btn-ghost" onClick={() => setAboutOpen(true)} title="About & arms version">
                <Info size={16} />
                About
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
          <h2 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '0.35rem', letterSpacing: '-0.025em' }}>
            Products
          </h2>
          <p className="ft-muted">
            Data from arms — each card is a <code className="ft-mono">GET /api/products</code> row with live task counts. Open a product for{' '}
            <code className="ft-mono">/p/&lt;id&gt;/tasks</code> (and other modules) deep links.
          </p>
        </div>

        {listLoading && workspaces.length === 0 ? (
          <div style={{ display: 'grid', gap: '0.75rem' }} aria-busy="true" aria-label="Loading products">
            {[1, 2, 3].map((i) => (
              <div key={i} className="ft-skeleton" style={{ height: '5.5rem', borderRadius: 'var(--ft-radius-sm)' }} />
            ))}
          </div>
        ) : workspaces.length === 0 && !isOnline ? (
          <BackendOfflineCallout onRetry={() => void refreshWorkspaces()} retrying={listLoading} />
        ) : workspaces.length === 0 ? (
          <EmptyWorkspaces onCreate={() => setModalOpen(true)} />
        ) : (
          <div className="ft-grid-ws ft-animate-slide-in">
            {workspaces.map((w) => (
              <WorkspaceCard key={w.id} workspace={w} onOpen={() => navigate(`/p/${encodeURIComponent(w.id)}/tasks`)} />
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
        onCreate={async (name, workspaceId) => {
          const id = await registerProduct(name, workspaceId);
          navigate(`/p/${encodeURIComponent(id)}/tasks`);
        }}
      />
      <AboutModal open={aboutOpen} onClose={() => setAboutOpen(false)} fetchVersion={fetchVersion} armsEnv={armsEnv} productIdForSse={null} />
    </div>
  );
}

function BackendOfflineCallout({ onRetry, retrying }: { onRetry: () => void; retrying: boolean }) {
  return (
    <div style={{ textAlign: 'center', padding: '4rem 1rem' }}>
      <Unplug size={64} className="ft-muted" style={{ marginInline: 'auto', marginBottom: '1rem' }} aria-hidden />
      <h3 style={{ fontSize: '1.125rem', fontWeight: 600, marginBottom: '0.5rem' }}>Not connected to arms</h3>
      <p className="ft-muted" style={{ marginBottom: '1.5rem', maxWidth: '28rem', marginInline: 'auto' }}>
        The UI could not reach the backend health endpoint. Confirm arms is running, <code className="ft-mono">VITE_ARMS_URL</code>{' '}
        points at it, and CORS allows this origin if needed.
      </p>
      <button type="button" className="ft-btn-primary" onClick={onRetry} disabled={retrying}>
        <RefreshCw size={16} className={retrying ? 'ft-spin' : ''} />
        Try again
      </button>
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
