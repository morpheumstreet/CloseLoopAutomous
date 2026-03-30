import { Outlet } from 'react-router-dom';
import { Settings } from 'lucide-react';

export function MissionSystemLayout() {
  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <Settings size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">System</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Workspace connection, appearance, and operator references. Values come from Vite env (<code className="ft-mono">VITE_ARMS_*</code>),
              <code className="ft-mono"> GET /api/version</code>, and <code className="ft-mono">GET /api/ops/host-metrics</code> (CPU / RAM / disk for the
              machine running arms). SSE URLs include <code className="ft-mono">?token=</code> when a bearer token is configured.
            </p>
          </div>
        </div>
        <Outlet />
      </div>
    </div>
  );
}
