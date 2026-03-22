import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronLeft, RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiOperationLogEntry } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import { formatRelativeTime } from '../lib/time';

export function ActivityLogPage() {
  const navigate = useNavigate();
  const { fetchOperationsLog } = useMissionUi();
  const [entries, setEntries] = useState<ApiOperationLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const list = await fetchOperationsLog({ limit: 150 });
      setEntries(list);
    } catch (e) {
      setEntries([]);
      setError(e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not load operations log.');
    } finally {
      setLoading(false);
    }
  }, [fetchOperationsLog]);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="ft-screen">
      <header className="ft-border-b" style={{ padding: '1rem', background: 'var(--mc-bg-secondary)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap' }}>
        <button type="button" className="ft-btn-ghost" onClick={() => navigate(-1)} style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronLeft size={18} />
          Back
        </button>
        <button type="button" className="ft-btn-ghost" onClick={() => void load()} disabled={loading}>
          <RefreshCw size={16} className={loading ? 'ft-spin' : ''} />
          Refresh
        </button>
      </header>
      <main className="ft-container" style={{ paddingBlock: '1.5rem' }}>
        <h1 style={{ fontSize: '1.35rem', fontWeight: 700, marginBottom: '0.35rem' }}>Operations log</h1>
        <p className="ft-muted" style={{ marginBottom: '1rem', fontSize: '0.875rem' }}>
          <code className="ft-mono">GET /api/operations-log</code> — newest first.
        </p>
        {error ? (
          <div className="ft-banner ft-banner--error" role="alert" style={{ marginBottom: '1rem' }}>
            {error}
          </div>
        ) : null}
        {loading && entries.length === 0 ? (
          <p className="ft-muted">Loading…</p>
        ) : entries.length === 0 ? (
          <p className="ft-muted">No entries returned.</p>
        ) : (
          <div style={{ overflowX: 'auto' }}>
            <table className="ft-table">
              <thead>
                <tr>
                  <th>When</th>
                  <th>Action</th>
                  <th>Resource</th>
                  <th>Id</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e, i) => (
                  <tr key={e.id ?? `${e.created_at}-${i}`}>
                    <td className="ft-muted" style={{ fontSize: '0.8rem', whiteSpace: 'nowrap' }}>
                      {e.created_at ? formatRelativeTime(e.created_at) : '—'}
                    </td>
                    <td>
                      <code className="ft-mono" style={{ fontSize: '0.75rem' }}>
                        {e.action ?? '—'}
                      </code>
                    </td>
                    <td className="ft-muted" style={{ fontSize: '0.8rem' }}>
                      {e.resource_type ?? '—'}
                    </td>
                    <td>
                      <code className="ft-mono" style={{ fontSize: '0.72rem', wordBreak: 'break-all' }}>
                        {e.resource_id ?? '—'}
                      </code>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  );
}
