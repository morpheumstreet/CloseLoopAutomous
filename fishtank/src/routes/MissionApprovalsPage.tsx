import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ClipboardCheck, RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiIdea, ApiMaybePoolIdea } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import type { Task } from '../domain/types';

function ideaNeedsSwipe(i: ApiIdea): boolean {
  if (i.decided === true) return false;
  if ((i.task_id ?? '').trim() !== '') return false;
  return true;
}

function maybePoolRowId(row: ApiMaybePoolIdea): string {
  const id = row.id;
  return typeof id === 'string' ? id : '';
}

export function MissionApprovalsPage() {
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const { client, tasks, productDetail, approveTaskPlan, refreshActiveBoard, boardLoading } = useMissionUi();

  const [ideas, setIdeas] = useState<ApiIdea[]>([]);
  const [ideasLoading, setIdeasLoading] = useState(true);
  const [ideasError, setIdeasError] = useState<string | null>(null);

  const [maybeIdeas, setMaybeIdeas] = useState<ApiMaybePoolIdea[]>([]);
  const [maybeConfigured, setMaybeConfigured] = useState(true);
  const [maybeLoading, setMaybeLoading] = useState(true);
  const [maybeError, setMaybeError] = useState<string | null>(null);

  const [ideaBusy, setIdeaBusy] = useState<string | null>(null);
  const [planBusy, setPlanBusy] = useState<string | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);

  const scopedTasks = useMemo(() => tasks.filter((t) => t.workspaceId === pid), [tasks, pid]);

  const planQueue = useMemo(
    () =>
      scopedTasks.filter((t) => t.status === 'planning' && t.planApproved !== true),
    [scopedTasks],
  );

  const planAttentionOther = useMemo(
    () =>
      scopedTasks.filter(
        (t) =>
          t.planApproved !== true &&
          t.status !== 'done' &&
          t.status !== 'failed' &&
          t.status !== 'planning' &&
          (t.status === 'inbox' || t.status === 'assigned'),
      ),
    [scopedTasks],
  );

  const swipeQueue = useMemo(() => ideas.filter(ideaNeedsSwipe), [ideas]);

  const loadIdeas = useCallback(async () => {
    if (!pid.trim()) return;
    setIdeasError(null);
    setIdeasLoading(true);
    try {
      const list = await client.listProductIdeas(pid);
      list.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
      setIdeas(list);
    } catch (e) {
      setIdeas([]);
      setIdeasError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Could not load ideas.');
    } finally {
      setIdeasLoading(false);
    }
  }, [client, pid]);

  const loadMaybePool = useCallback(async () => {
    if (!pid.trim()) return;
    setMaybeError(null);
    setMaybeLoading(true);
    try {
      const { ideas: rows, configured } = await client.listProductMaybePool(pid);
      setMaybeConfigured(configured);
      setMaybeIdeas(rows);
    } catch (e) {
      setMaybeIdeas([]);
      setMaybeConfigured(true);
      setMaybeError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Could not load maybe pool.');
    } finally {
      setMaybeLoading(false);
    }
  }, [client, pid]);

  useEffect(() => {
    void loadIdeas();
    void loadMaybePool();
  }, [loadIdeas, loadMaybePool]);

  const refreshAll = useCallback(async () => {
    setPageError(null);
    await Promise.all([loadIdeas(), loadMaybePool(), refreshActiveBoard({ silent: true })]);
  }, [loadIdeas, loadMaybePool, refreshActiveBoard]);

  async function runSwipe(ideaId: string, decision: 'pass' | 'maybe' | 'yes' | 'now') {
    setPageError(null);
    setIdeaBusy(ideaId);
    try {
      await client.swipeIdea(ideaId, decision);
      await Promise.all([loadIdeas(), loadMaybePool()]);
      await refreshActiveBoard({ silent: true });
    } catch (e) {
      setPageError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Swipe failed.');
    } finally {
      setIdeaBusy(null);
    }
  }

  async function runPromote(ideaId: string) {
    setPageError(null);
    setIdeaBusy(ideaId);
    try {
      await client.promoteMaybeIdea(ideaId);
      await Promise.all([loadIdeas(), loadMaybePool()]);
      await refreshActiveBoard({ silent: true });
    } catch (e) {
      setPageError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Promote failed.');
    } finally {
      setIdeaBusy(null);
    }
  }

  async function runApprovePlan(taskId: string) {
    setPageError(null);
    setPlanBusy(taskId);
    try {
      await approveTaskPlan(taskId);
    } catch (e) {
      setPageError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Approve failed.');
    } finally {
      setPlanBusy(null);
    }
  }

  const mergePending = productDetail?.merge_queue_pending ?? 0;
  const loading = ideasLoading || maybeLoading || boardLoading;

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'auto', padding: '1rem 1.25rem' }}>
      <div style={{ maxWidth: '56rem', margin: '0 auto', width: '100%', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
        <header>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
              <span className="ft-muted" aria-hidden>
                <ClipboardCheck size={22} />
              </span>
              <div>
                <h1 style={{ fontSize: '1.2rem', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>Approvals</h1>
                <p className="ft-muted" style={{ margin: '0.25rem 0 0', fontSize: '0.8rem', lineHeight: 1.45, maxWidth: '40rem' }}>
                  Swipe ideas, clear the maybe pool, and approve task plans for this product. Actions call{' '}
                  <code className="ft-mono">POST /api/ideas/…/swipe</code>,{' '}
                  <code className="ft-mono">POST /api/ideas/…/promote-maybe</code>, and{' '}
                  <code className="ft-mono">POST /api/tasks/…/plan/approve</code>.
                </p>
              </div>
            </div>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
              disabled={loading}
              onClick={() => void refreshAll()}
            >
              <RefreshCw size={14} className={loading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
          </div>
        </header>

        {pageError ? (
          <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--mc-danger, #dc2626)' }} role="alert">
            {pageError}
          </p>
        ) : null}

        <section style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }} aria-label="Approvals summary">
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {ideasLoading ? '…' : `${swipeQueue.length} idea${swipeQueue.length === 1 ? '' : 's'} to swipe`}
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {maybeLoading ? '…' : !maybeConfigured ? 'Maybe pool off' : `${maybeIdeas.length} in maybe pool`}
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {boardLoading ? '…' : `${planQueue.length} plan${planQueue.length === 1 ? '' : 's'} to approve`}
          </span>
          {mergePending > 0 ? (
            <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
              {mergePending} merge queue
            </span>
          ) : null}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Idea swipe queue
          </h2>
          {ideasError ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }} role="alert">
              {ideasError}
            </p>
          ) : ideasLoading ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              Loading ideas…
            </p>
          ) : swipeQueue.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              No undecided ideas without a linked task. New ideas from research or ideation appear here until you swipe.
            </p>
          ) : (
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {swipeQueue.map((i) => (
                <li
                  key={i.id}
                  style={{
                    padding: '0.65rem 0.75rem',
                    borderRadius: 'var(--ft-radius-sm)',
                    border: '1px solid var(--mc-border)',
                    background: 'var(--mc-bg-tertiary)',
                  }}
                >
                  <div className="ft-mono" style={{ fontSize: '0.65rem', opacity: 0.7, marginBottom: '0.25rem' }}>
                    {i.id}
                  </div>
                  <div style={{ fontSize: '0.88rem', fontWeight: 600, lineHeight: 1.35 }}>{i.title?.trim() || '(no title)'}</div>
                  {i.description?.trim() ? (
                    <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.78rem', lineHeight: 1.45 }}>
                      {i.description.length > 280 ? `${i.description.slice(0, 277)}…` : i.description}
                    </p>
                  ) : null}
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem', marginTop: '0.55rem' }}>
                    {(['pass', 'maybe', 'yes', 'now'] as const).map((d) => (
                      <button
                        key={d}
                        type="button"
                        className={d === 'yes' || d === 'now' ? 'ft-btn-primary' : 'ft-btn-ghost'}
                        style={{ fontSize: '0.72rem', textTransform: 'capitalize' }}
                        disabled={ideaBusy === i.id}
                        onClick={() => void runSwipe(i.id, d)}
                      >
                        {d}
                      </button>
                    ))}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Maybe pool
          </h2>
          {!maybeConfigured ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              Maybe pool storage is not configured on this arms instance (503 from{' '}
              <code className="ft-mono">GET /api/products/…/maybe-pool</code>).
            </p>
          ) : maybeError ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }} role="alert">
              {maybeError}
            </p>
          ) : maybeLoading ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              Loading maybe pool…
            </p>
          ) : maybeIdeas.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              No ideas in the maybe pool.
            </p>
          ) : (
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {maybeIdeas.filter((row) => maybePoolRowId(row) !== '').map((row) => {
                const iid = maybePoolRowId(row);
                const title = typeof row.title === 'string' && row.title.trim() ? row.title : iid || '(idea)';
                return (
                  <li
                    key={iid}
                    style={{
                      display: 'flex',
                      alignItems: 'flex-start',
                      justifyContent: 'space-between',
                      gap: '0.75rem',
                      flexWrap: 'wrap',
                      padding: '0.65rem 0.75rem',
                      borderRadius: 'var(--ft-radius-sm)',
                      border: '1px solid var(--mc-border)',
                      background: 'var(--mc-bg-tertiary)',
                    }}
                  >
                    <div style={{ minWidth: 0 }}>
                      <div className="ft-mono" style={{ fontSize: '0.65rem', opacity: 0.7 }}>
                        {iid}
                      </div>
                      <div style={{ fontSize: '0.85rem', fontWeight: 600, marginTop: '0.2rem' }}>{title}</div>
                      {typeof row.maybe_next_evaluate_at === 'string' && row.maybe_next_evaluate_at ? (
                        <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.72rem' }}>
                          Next eval: {row.maybe_next_evaluate_at}
                        </p>
                      ) : null}
                    </div>
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem', flexShrink: 0 }}>
                      <button
                        type="button"
                        className="ft-btn-primary"
                        style={{ fontSize: '0.72rem' }}
                        disabled={ideaBusy === iid}
                        onClick={() => void runPromote(iid)}
                      >
                        Promote to yes
                      </button>
                      <button
                        type="button"
                        className="ft-btn-ghost"
                        style={{ fontSize: '0.72rem' }}
                        disabled={ideaBusy === iid}
                        onClick={() => void runSwipe(iid, 'pass')}
                      >
                        Pass
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Task plans awaiting approval
          </h2>
          {planQueue.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              No tasks in <strong>planning</strong> with a pending plan. Approve moves them to the inbox so you can assign and dispatch.
            </p>
          ) : (
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {planQueue.map((t) => (
                <PlanRow key={t.id} t={t} productId={pid} busy={planBusy === t.id} onApprove={() => void runApprovePlan(t.id)} />
              ))}
            </ul>
          )}
        </section>

        {planAttentionOther.length > 0 ? (
          <section>
            <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
              Other plan attention
            </h2>
            <p className="ft-muted" style={{ fontSize: '0.8rem', margin: '0 0 0.5rem', lineHeight: 1.45 }}>
              These tasks are not in planning but still show an unapproved plan flag. Open the task board to inspect or use the task detail drawer.
            </p>
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
              {planAttentionOther.map((t) => (
                <li key={t.id} className="ft-mono" style={{ fontSize: '0.75rem', opacity: 0.85 }}>
                  {t.title} · {t.status.replace(/_/g, ' ')} ·{' '}
                  <Link to={`/p/${encodeURIComponent(pid)}/tasks`} className="ft-btn-ghost" style={{ fontSize: '0.7rem', textDecoration: 'none', padding: '0.15rem 0.4rem' }}>
                    Board
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <p className="ft-muted" style={{ fontSize: '0.72rem', margin: 0, paddingBottom: '0.5rem' }}>
          Use <Link to={`/p/${encodeURIComponent(pid)}/tasks`}>Tasks</Link> for the full Kanban board and task detail actions (reject plan, dispatch, and more).
        </p>
      </div>
    </div>
  );
}

function PlanRow({
  t,
  productId,
  busy,
  onApprove,
}: {
  t: Task;
  productId: string;
  busy: boolean;
  onApprove: () => void;
}) {
  return (
    <li
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: '0.75rem',
        flexWrap: 'wrap',
        padding: '0.65rem 0.75rem',
        borderRadius: 'var(--ft-radius-sm)',
        border: '1px solid var(--mc-border)',
        background: 'var(--mc-bg-tertiary)',
      }}
    >
      <div style={{ minWidth: 0 }}>
        <div className="ft-mono" style={{ fontSize: '0.65rem', opacity: 0.65, marginBottom: '0.2rem' }}>
          {t.ideaId} · planning
        </div>
        <div style={{ fontSize: '0.85rem', fontWeight: 600, lineHeight: 1.35 }}>{t.title}</div>
      </div>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem', flexShrink: 0 }}>
        <button type="button" className="ft-btn-primary" style={{ fontSize: '0.72rem' }} disabled={busy} onClick={onApprove}>
          {busy ? 'Approving…' : 'Approve plan'}
        </button>
        <Link
          to={`/p/${encodeURIComponent(productId)}/tasks`}
          className="ft-btn-ghost"
          style={{ fontSize: '0.72rem', textDecoration: 'none', display: 'inline-flex', alignItems: 'center' }}
        >
          Board
        </Link>
      </div>
    </li>
  );
}
