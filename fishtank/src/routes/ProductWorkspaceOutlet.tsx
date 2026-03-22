import { useEffect } from 'react';
import { Outlet, useNavigate, useParams } from 'react-router-dom';
import { useMissionUi } from '../context/MissionUiContext';

function BoardSkeleton() {
  return (
    <div className="ft-screen-fixed">
      <div className="ft-header-bar ft-header-bar--mc" aria-hidden>
        <div className="ft-header-mc-left" style={{ gap: '0.75rem' }}>
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '2.75rem', height: '2.75rem' }} />
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '10rem', height: '1.25rem' }} />
        </div>
        <div className="ft-header-mc-center">
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '100%', height: '2.5rem' }} />
        </div>
        <div className="ft-header-mc-right">
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '8rem', height: '2.25rem' }} />
        </div>
      </div>
      <div className="ft-desktop-only" style={{ flex: 1, minHeight: 0 }}>
        <aside className="ft-mc-sidebar" style={{ padding: '0.75rem' }}>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem', marginBottom: '0.75rem' }}>
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="ft-skeleton ft-skeleton--text" style={{ width: '4.5rem', height: '2.5rem' }} />
            ))}
          </div>
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '100%', height: '12rem' }} />
        </aside>
        <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, padding: '0.75rem' }}>
          <div style={{ display: 'flex', gap: '0.75rem', overflow: 'hidden' }}>
            {[1, 2, 3].map((i) => (
              <div key={i} style={{ width: '200px', flexShrink: 0 }}>
                <div className="ft-skeleton ft-skeleton--text" style={{ height: '1.5rem', marginBottom: '0.5rem' }} />
                <div className="ft-skeleton" style={{ height: '4rem', marginBottom: '0.35rem' }} />
                <div className="ft-skeleton" style={{ height: '4rem' }} />
              </div>
            ))}
          </div>
        </div>
        <aside className="ft-feed ft-feed--mc" style={{ padding: '0.75rem' }}>
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '60%', height: '1rem', marginBottom: '0.75rem' }} />
          <div className="ft-skeleton ft-skeleton--text" style={{ width: '100%', height: '6rem' }} />
        </aside>
      </div>
      <div className="ft-mobile-only" style={{ flex: 1, padding: '1rem' }}>
        <div className="ft-skeleton ft-skeleton--text" style={{ width: '100%', height: '180px' }} />
      </div>
    </div>
  );
}

function UnknownProduct({ onBack }: { onBack: () => void }) {
  return (
    <div className="ft-screen" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '2rem' }}>
      <div style={{ textAlign: 'center', maxWidth: '24rem' }}>
        <h1 style={{ fontSize: '1.25rem', fontWeight: 700, marginBottom: '0.5rem' }}>Product not found</h1>
        <p className="ft-muted" style={{ marginBottom: '1.25rem', fontSize: '0.9rem' }}>
          No workspace matches this id in <code className="ft-mono">GET /api/products</code>. It may have been removed or the link is wrong.
        </p>
        <button type="button" className="ft-btn-primary" onClick={onBack}>
          Back to dashboard
        </button>
      </div>
    </div>
  );
}

/** Syncs `/p/:productId` with Mission UI state; child routes render `WorkspaceShellLayout` + module outlets. */
export function ProductWorkspaceOutlet() {
  const { productId } = useParams<{ productId: string }>();
  const navigate = useNavigate();
  const { workspaces, listLoading, openWorkspace, activeWorkspace } = useMissionUi();

  useEffect(() => {
    if (!productId || listLoading) return;
    const w = workspaces.find((x) => x.id === productId);
    if (!w) return;
    if (activeWorkspace?.id !== productId) void openWorkspace(w);
  }, [productId, listLoading, workspaces, openWorkspace, activeWorkspace?.id]);

  if (!productId) return null;

  if (listLoading && workspaces.length === 0) {
    return <BoardSkeleton />;
  }

  const match = workspaces.find((x) => x.id === productId);
  if (!listLoading && !match) {
    return <UnknownProduct onBack={() => navigate('/')} />;
  }

  if (!match || activeWorkspace?.id !== productId) {
    return <BoardSkeleton />;
  }

  return <Outlet />;
}
