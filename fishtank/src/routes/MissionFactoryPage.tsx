import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Factory, Layers, RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiConvoy, ApiMaybePoolIdea } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import { formatRelativeTime } from '../lib/time';

function formatUpcomingOrPast(iso: string): string {
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '—';
  const diffMs = t - Date.now();
  if (diffMs <= 0) return formatRelativeTime(iso);
  const mins = Math.floor(diffMs / 60_000);
  if (mins < 1) return 'soon';
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 48) return `in ${hrs}h`;
  const days = Math.floor(hrs / 24);
  return `in ${days}d`;
}

function convoyProgress(c: ApiConvoy): { completed: number; total: number; ready: number } {
  const subs = c.subtasks ?? [];
  const total = subs.length;
  const completed = subs.filter((s) => s.completed).length;
  const ready = subs.filter((s) => s.status === 'ready').length;
  return { completed, total, ready };
}

function shortId(id: string): string {
  const t = id.trim();
  if (t.length <= 10) return t;
  return `${t.slice(0, 6)}…${t.slice(-4)}`;
}

export function MissionFactoryPage() {
  const { productId } = useParams<{ productId: string }>();
  const { client, tasks, refreshActiveBoard } = useMissionUi();

  const [convoys, setConvoys] = useState<ApiConvoy[]>([]);
  const [convoysLoading, setConvoysLoading] = useState(true);
  const [convoysError, setConvoysError] = useState<string | null>(null);

  const [selectedParents, setSelectedParents] = useState<Set<string>>(() => new Set());
  const [waveCost, setWaveCost] = useState('0');
  const [rowBusy, setRowBusy] = useState<string | null>(null);
  const [batchBusy, setBatchBusy] = useState(false);
  const [waveMessage, setWaveMessage] = useState<string | null>(null);

  const [maybeIdeas, setMaybeIdeas] = useState<ApiMaybePoolIdea[]>([]);
  const [maybeConfigured, setMaybeConfigured] = useState(true);
  const [maybeLoading, setMaybeLoading] = useState(true);
  const [maybeError, setMaybeError] = useState<string | null>(null);
  const [reevalNote, setReevalNote] = useState('');
  const [reevalDelaySec, setReevalDelaySec] = useState('');
  const [reevalBusy, setReevalBusy] = useState(false);

  const taskTitle = useMemo(() => {
    const m = new Map<string, string>();
    for (const t of tasks) m.set(t.id, t.title);
    return (id: string) => m.get(id) ?? shortId(id);
  }, [tasks]);

  const loadConvoys = useCallback(async () => {
    if (!productId) return;
    setConvoysError(null);
    setConvoysLoading(true);
    try {
      const list = await client.listProductConvoys(productId);
      setConvoys(list);
      setSelectedParents((prev) => {
        const next = new Set<string>();
        const ids = new Set(list.map((c) => c.parent_id));
        for (const id of prev) {
          if (ids.has(id)) next.add(id);
        }
        return next;
      });
    } catch (e) {
      setConvoys([]);
      setConvoysError(e instanceof ArmsHttpError ? e.message : 'Could not load convoys.');
    } finally {
      setConvoysLoading(false);
    }
  }, [client, productId]);

  const loadMaybePool = useCallback(async () => {
    if (!productId) return;
    setMaybeError(null);
    setMaybeLoading(true);
    try {
      const { ideas, configured } = await client.listProductMaybePool(productId);
      setMaybeIdeas(ideas);
      setMaybeConfigured(configured);
    } catch (e) {
      setMaybeIdeas([]);
      setMaybeError(e instanceof ArmsHttpError ? e.message : 'Could not load maybe pool.');
    } finally {
      setMaybeLoading(false);
    }
  }, [client, productId]);

  useEffect(() => {
    void loadConvoys();
    void loadMaybePool();
  }, [loadConvoys, loadMaybePool]);

  const estimatedCost = Number(waveCost);
  const costArg = Number.isFinite(estimatedCost) ? estimatedCost : 0;

  const toggleParent = (parentId: string) => {
    setSelectedParents((prev) => {
      const next = new Set(prev);
      if (next.has(parentId)) next.delete(parentId);
      else next.add(parentId);
      return next;
    });
  };

  const selectAllVisible = () => {
    setSelectedParents(new Set(convoys.map((c) => c.parent_id)));
  };

  const clearSelection = () => setSelectedParents(new Set());

  const onDispatchOne = async (parentTaskId: string) => {
    setRowBusy(parentTaskId);
    setWaveMessage(null);
    try {
      const r = await client.dispatchTaskConvoyWave(parentTaskId, costArg);
      setWaveMessage(`Dispatched ${r.dispatched} of ${r.total} ready subtasks for ${taskTitle(parentTaskId)}.`);
      await loadConvoys();
      await refreshActiveBoard({ silent: true });
    } catch (e) {
      setWaveMessage(e instanceof ArmsHttpError ? e.message : 'Dispatch failed.');
    } finally {
      setRowBusy(null);
    }
  };

  const onDispatchSelected = async () => {
    const ids = [...selectedParents];
    if (ids.length === 0) return;
    setBatchBusy(true);
    setWaveMessage(null);
    const errors: string[] = [];
    try {
      for (const id of ids) {
        try {
          await client.dispatchTaskConvoyWave(id, costArg);
        } catch (e) {
          const msg = e instanceof ArmsHttpError ? e.message : 'Dispatch failed';
          errors.push(`${taskTitle(id)}: ${msg}`);
        }
      }
      await loadConvoys();
      await refreshActiveBoard({ silent: true });
      if (errors.length) setWaveMessage(errors.join(' · '));
      else setWaveMessage(`Ran convoy waves for ${ids.length} parent task(s).`);
    } finally {
      setBatchBusy(false);
    }
  };

  const onBatchReeval = async () => {
    if (!productId) return;
    setReevalBusy(true);
    setMaybeError(null);
    try {
      const delayRaw = reevalDelaySec.trim();
      let next_evaluate_delay_sec: number | undefined;
      if (delayRaw !== '') {
        const n = Math.floor(Number(delayRaw));
        if (Number.isFinite(n) && n >= 0) next_evaluate_delay_sec = n;
      }
      const body: { note?: string; next_evaluate_delay_sec?: number } = {};
      const note = reevalNote.trim();
      if (note) body.note = note;
      if (next_evaluate_delay_sec !== undefined) body.next_evaluate_delay_sec = next_evaluate_delay_sec;
      const ideas = await client.batchReevalProductMaybePool(productId, body);
      setMaybeIdeas(ideas);
    } catch (e) {
      setMaybeError(e instanceof ArmsHttpError ? e.message : 'Batch re-eval failed.');
    } finally {
      setReevalBusy(false);
    }
  };

  const allSelected = convoys.length > 0 && convoys.every((c) => selectedParents.has(c.parent_id));

  if (!productId) {
    return (
      <div className="ft-queue-flex ft-muted" style={{ flex: 1, padding: '1rem' }}>
        Missing workspace.
      </div>
    );
  }

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <Factory size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">Factory</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Batch convoy waves and autopilot maybe-pool re-evaluation for this workspace. Convoy dispatch uses{' '}
              <code className="ft-mono">POST /api/tasks/{'{id}'}/convoy/dispatch</code>; maybe pool uses{' '}
              <code className="ft-mono">POST …/maybe-pool/batch-reeval</code>.
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <Link
              to={`/p/${productId}/tasks`}
              className="ft-btn-ghost"
              style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
            >
              <Layers size={16} aria-hidden />
              Tasks board
            </Link>
          </div>
        </div>

        <section
          style={{
            marginTop: '0.5rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem' }}>
            <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
              Convoy waves
            </h2>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
              <label className="ft-muted" style={{ fontSize: '0.75rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                Est. cost
                <input
                  type="text"
                  inputMode="decimal"
                  className="ft-input"
                  style={{ width: '5rem', fontSize: '0.8rem' }}
                  value={waveCost}
                  onChange={(ev) => setWaveCost(ev.target.value)}
                  aria-label="Estimated cost per convoy dispatch"
                />
              </label>
              <button
                type="button"
                className="ft-btn-ghost"
                style={{ fontSize: '0.75rem' }}
                disabled={convoysLoading}
                onClick={() => void loadConvoys()}
              >
                <RefreshCw size={14} className={convoysLoading ? 'ft-spin' : ''} aria-hidden />
                Refresh convoys
              </button>
              <button
                type="button"
                className="ft-btn-ghost"
                style={{ fontSize: '0.75rem' }}
                disabled={!convoys.length || batchBusy}
                onClick={() => (allSelected ? clearSelection() : selectAllVisible())}
              >
                {allSelected ? 'Clear selection' : 'Select all'}
              </button>
              <button
                type="button"
                className="ft-btn-primary"
                style={{ fontSize: '0.75rem' }}
                disabled={selectedParents.size === 0 || batchBusy || convoysLoading}
                onClick={() => void onDispatchSelected()}
              >
                {batchBusy ? 'Dispatching…' : `Dispatch selected (${selectedParents.size})`}
              </button>
            </div>
          </div>

          {convoysError ? (
            <p className="ft-muted" style={{ margin: '0.75rem 0 0', fontSize: '0.85rem' }}>
              {convoysError}
            </p>
          ) : null}
          {waveMessage ? (
            <p className="ft-muted" style={{ margin: '0.65rem 0 0', fontSize: '0.8rem', lineHeight: 1.45 }}>
              {waveMessage}
            </p>
          ) : null}

          {convoysLoading && convoys.length === 0 ? (
            <p className="ft-muted" style={{ margin: '0.75rem 0 0', fontSize: '0.85rem' }}>
              Loading convoys…
            </p>
          ) : null}

          {!convoysLoading && convoys.length === 0 && !convoysError ? (
            <p className="ft-muted" style={{ margin: '0.75rem 0 0', fontSize: '0.85rem', lineHeight: 1.5 }}>
              No convoys for this product yet. Create a convoy from a task (Mission Control path) or wait until parent tasks enter convoy flow.
            </p>
          ) : null}

          {convoys.length > 0 ? (
            <div style={{ marginTop: '0.75rem', overflowX: 'auto' }}>
              <table className="ft-table" style={{ width: '100%', fontSize: '0.8rem', borderCollapse: 'collapse' }}>
                <thead>
                  <tr style={{ textAlign: 'left', borderBottom: '1px solid var(--mc-border)' }}>
                    <th style={{ padding: '0.35rem 0.5rem 0.5rem 0', width: '2rem' }} aria-label="Select" />
                    <th style={{ padding: '0.35rem 0.5rem' }}>Parent task</th>
                    <th style={{ padding: '0.35rem 0.5rem' }}>Progress</th>
                    <th style={{ padding: '0.35rem 0.5rem' }}>Ready</th>
                    <th style={{ padding: '0.35rem 0.5rem' }}>Convoy</th>
                    <th style={{ padding: '0.35rem 0.5rem' }} />
                  </tr>
                </thead>
                <tbody>
                  {convoys.map((c) => {
                    const { completed, total, ready } = convoyProgress(c);
                    const pid = c.parent_id;
                    const busy = rowBusy === pid;
                    return (
                      <tr key={c.id} style={{ borderBottom: '1px solid color-mix(in srgb, var(--mc-border) 65%, transparent)' }}>
                        <td style={{ padding: '0.45rem 0.5rem 0.45rem 0', verticalAlign: 'middle' }}>
                          <input
                            type="checkbox"
                            checked={selectedParents.has(pid)}
                            onChange={() => toggleParent(pid)}
                            aria-label={`Select ${taskTitle(pid)}`}
                          />
                        </td>
                        <td style={{ padding: '0.45rem 0.5rem', verticalAlign: 'middle', fontWeight: 600 }}>{taskTitle(pid)}</td>
                        <td style={{ padding: '0.45rem 0.5rem', verticalAlign: 'middle' }} className="ft-muted">
                          {completed}/{total} subtasks done
                        </td>
                        <td style={{ padding: '0.45rem 0.5rem', verticalAlign: 'middle' }} className="ft-muted">
                          {ready} ready
                        </td>
                        <td style={{ padding: '0.45rem 0.5rem', verticalAlign: 'middle' }} className="ft-mono ft-muted">
                          {shortId(c.id)}
                        </td>
                        <td style={{ padding: '0.45rem 0.5rem', verticalAlign: 'middle', textAlign: 'right' }}>
                          <button
                            type="button"
                            className="ft-btn-primary"
                            style={{ fontSize: '0.72rem' }}
                            disabled={busy || batchBusy}
                            onClick={() => void onDispatchOne(pid)}
                          >
                            {busy ? '…' : 'Dispatch wave'}
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : null}
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Maybe pool (batch re-eval)
          </h2>
          {!maybeConfigured ? (
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem', lineHeight: 1.5 }}>
              Maybe pool is not configured on this arms instance (503). Enable the maybe pool store to use batch re-evaluation.
            </p>
          ) : (
            <>
              <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
                Records a batch evaluation pass for every idea still in the pool and refreshes preference hints server-side.
              </p>
              {maybeError ? (
                <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.85rem' }}>
                  {maybeError}
                </p>
              ) : null}
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.65rem', alignItems: 'flex-end', marginBottom: '0.65rem' }}>
                <label style={{ flex: '1 1 12rem', display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                  <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                    Note (optional)
                  </span>
                  <input
                    type="text"
                    className="ft-input"
                    style={{ fontSize: '0.85rem' }}
                    value={reevalNote}
                    onChange={(ev) => setReevalNote(ev.target.value)}
                    placeholder="e.g. Weekly factory pass"
                  />
                </label>
                <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
                  <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                    Next eval delay (sec)
                  </span>
                  <input
                    type="text"
                    inputMode="numeric"
                    className="ft-input"
                    style={{ width: '7rem', fontSize: '0.85rem' }}
                    value={reevalDelaySec}
                    onChange={(ev) => setReevalDelaySec(ev.target.value)}
                    placeholder="optional"
                  />
                </label>
                <button
                  type="button"
                  className="ft-btn-primary"
                  style={{ fontSize: '0.8rem' }}
                  disabled={reevalBusy || maybeLoading}
                  onClick={() => void onBatchReeval()}
                >
                  {reevalBusy ? 'Running…' : 'Run batch re-eval'}
                </button>
                <button
                  type="button"
                  className="ft-btn-ghost"
                  style={{ fontSize: '0.8rem' }}
                  disabled={maybeLoading}
                  onClick={() => void loadMaybePool()}
                >
                  <RefreshCw size={14} className={maybeLoading ? 'ft-spin' : ''} aria-hidden />
                  Refresh list
                </button>
              </div>
              {maybeLoading && maybeIdeas.length === 0 ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
                  Loading maybe pool…
                </p>
              ) : null}
              {!maybeLoading && maybeIdeas.length === 0 ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
                  Pool is empty.
                </p>
              ) : null}
              {maybeIdeas.length > 0 ? (
                <ul style={{ margin: 0, paddingLeft: '1.1rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
                  {maybeIdeas.slice(0, 24).map((idea, i) => {
                    const id = typeof idea.id === 'string' ? idea.id : `idea-${i}`;
                    const title = typeof idea.title === 'string' && idea.title.trim() ? idea.title : shortId(id);
                    const nextAt = typeof idea.maybe_next_evaluate_at === 'string' ? idea.maybe_next_evaluate_at : '';
                    const rel = nextAt ? formatUpcomingOrPast(nextAt) : null;
                    return (
                      <li key={id} style={{ marginBottom: '0.35rem' }}>
                        <span style={{ fontWeight: 600 }}>{title}</span>
                        {rel ? (
                          <span className="ft-muted">
                            {' '}
                            — next eval {rel}
                          </span>
                        ) : null}
                      </li>
                    );
                  })}
                </ul>
              ) : null}
              {maybeIdeas.length > 24 ? (
                <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.75rem' }}>
                  Showing 24 of {maybeIdeas.length} ideas.
                </p>
              ) : null}
            </>
          )}
        </section>
      </div>
    </div>
  );
}
