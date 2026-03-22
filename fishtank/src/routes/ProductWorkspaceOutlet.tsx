import { useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useMissionUi } from '../context/MissionUiContext';
import { MissionWorkspacePage } from '../components/workspace/MissionWorkspacePage';

function BoardSkeleton() {
  return (
    <div className="ft-screen-fixed">
      <div className="ft-header-bar" style={{ padding: '0.75rem 1rem' }}>
        <div className="ft-skeleton ft-skeleton--text" style={{ width: '12rem', height: '1.25rem' }} />
      </div>
      <div className="ft-container" style={{ padding: '2rem 1rem' }}>
        <div className="ft-skeleton ft-skeleton--text" style={{ width: '100%', height: '200px', maxWidth: '48rem' }} />
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

/** Syncs `/p/:productId` with Mission UI state and renders the workspace shell. */
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

  return <MissionWorkspacePage />;
}
