import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { GitBranch, RefreshCw, Users } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiConvoy, ApiConvoySubtask } from '../api/armsTypes';
import { FeedEventRow, matchesFeedFilter } from '../components/workspace/feedDisplay';
import { isDevBuild } from '../config/armsEnv';
import { useMissionUi } from '../context/MissionUiContext';
import type { Task } from '../domain/types';

function subtaskStatusStyle(status: string): { border: string; dot: string } {
  switch (status) {
    case 'completed':
      return {
        border: 'color-mix(in srgb, var(--mc-accent) 22%, var(--mc-border))',
        dot: 'var(--mc-accent)',
      };
    case 'running':
      return {
        border: 'color-mix(in srgb, var(--mc-accent) 45%, var(--mc-border))',
        dot: 'var(--mc-accent)',
      };
    case 'ready':
      return { border: 'var(--mc-border)', dot: 'color-mix(in srgb, var(--mc-fg) 35%, var(--mc-border))' };
    case 'blocked':
      return {
        border: 'color-mix(in srgb, #f59e0b 35%, var(--mc-border))',
        dot: '#f59e0b',
      };
    default:
      return { border: 'var(--mc-border)', dot: 'var(--mc-muted)' };
  }
}

function convoyProgress(c: ApiConvoy): { done: number; total: number } {
  const total = c.subtasks?.length ?? 0;
  const done = c.subtasks?.filter((s) => s.completed).length ?? 0;
  return { done, total };
}

function TaskCouncilRow({ t, productId }: { t: Task; productId: string }) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'space-between',
        gap: '0.75rem',
        padding: '0.65rem 0.75rem',
        borderRadius: 'var(--ft-radius-sm)',
        border: '1px solid var(--mc-border)',
        background: 'var(--mc-bg-tertiary)',
      }}
    >
      <div style={{ minWidth: 0 }}>
        <div className="ft-mono" style={{ fontSize: '0.65rem', opacity: 0.65, marginBottom: '0.2rem' }}>
          {t.status.replace(/_/g, ' ')}
        </div>
        <div style={{ fontSize: '0.85rem', fontWeight: 600, lineHeight: 1.35 }}>{t.title}</div>
        {t.statusReason ? (
          <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.75rem', lineHeight: 1.4 }}>
            {t.statusReason}
          </p>
        ) : null}
      </div>
      <Link
        to={`/p/${encodeURIComponent(productId)}/tasks`}
        className="ft-btn-ghost"
        style={{ flexShrink: 0, fontSize: '0.72rem', textDecoration: 'none' }}
      >
        Board
      </Link>
    </div>
  );
}

export function MissionCouncilPage() {
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const { client, tasks, stalledTasks, events, refreshActiveBoard, boardLoading } = useMissionUi();
  const showDevTools = isDevBuild();
  const [feedExpanded, setFeedExpanded] = useState<Record<string, boolean>>({});

  const [convoys, setConvoys] = useState<ApiConvoy[]>([]);
  const [convoyLoading, setConvoyLoading] = useState(true);
  const [convoyError, setConvoyError] = useState<string | null>(null);

  const scopedTasks = useMemo(() => tasks.filter((t) => t.workspaceId === pid), [tasks, pid]);

  const taskById = useMemo(() => {
    const m = new Map<string, Task>();
    for (const t of scopedTasks) m.set(t.id, t);
    return m;
  }, [scopedTasks]);

  const loadConvoys = useCallback(async () => {
    if (!pid) return;
    setConvoyError(null);
    setConvoyLoading(true);
    try {
      const list = await client.listProductConvoys(pid);
      list.sort((a, b) => (a.created_at < b.created_at ? 1 : -1));
      setConvoys(list);
    } catch (e) {
      setConvoys([]);
      if (e instanceof ArmsHttpError) {
        setConvoyError(e.message);
      } else {
        setConvoyError(e instanceof Error ? e.message : 'Could not load convoys');
      }
    } finally {
      setConvoyLoading(false);
    }
  }, [client, pid]);

  useEffect(() => {
    void loadConvoys();
  }, [loadConvoys]);

  const refreshAll = useCallback(async () => {
    await Promise.all([loadConvoys(), refreshActiveBoard({ silent: true })]);
  }, [loadConvoys, refreshActiveBoard]);

  const inHumanReview = useMemo(
    () => scopedTasks.filter((t) => t.status === 'testing' || t.status === 'review' || t.status === 'convoy_active'),
    [scopedTasks],
  );

  const planAttention = useMemo(
    () =>
      scopedTasks.filter(
        (t) =>
          t.planApproved !== true &&
          t.status !== 'done' &&
          t.status !== 'failed' &&
          (t.status === 'planning' || t.status === 'inbox' || t.status === 'assigned'),
      ),
    [scopedTasks],
  );

  const convoyEvents = useMemo(() => events.filter((e) => matchesFeedFilter(e, 'convoy')).slice(0, 10), [events]);

  const activeAgents = useMemo(
    () => scopedTasks.filter((t) => t.status === 'in_progress' || t.status === 'convoy_active').length,
    [scopedTasks],
  );

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'auto', padding: '1rem 1.25rem' }}>
      <div style={{ maxWidth: '56rem', margin: '0 auto', width: '100%', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
        <header>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
              <span className="ft-muted" aria-hidden>
                <Users size={22} />
              </span>
              <div>
                <h1 style={{ fontSize: '1.2rem', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>Council</h1>
                <p className="ft-muted" style={{ margin: '0.25rem 0 0', fontSize: '0.8rem', lineHeight: 1.45, maxWidth: '38rem' }}>
                  Multi-agent workload for this workspace: convoy DAGs on parent tasks, items waiting on humans, and plan gates. Data comes from{' '}
                  <code className="ft-mono">GET /api/products/…/convoys</code> and the live task board.
                </p>
              </div>
            </div>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
              disabled={convoyLoading || boardLoading}
              onClick={() => void refreshAll()}
            >
              <RefreshCw size={14} className={convoyLoading || boardLoading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
          </div>
        </header>

        <section
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '0.5rem',
            alignItems: 'center',
          }}
          aria-label="Council summary"
        >
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            <GitBranch size={14} aria-hidden style={{ opacity: 0.85 }} />
            {convoyLoading ? '…' : `${convoys.length} convoy${convoys.length === 1 ? '' : 's'}`}
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {inHumanReview.length} in review / testing
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {planAttention.length} plan gate
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {stalledTasks.length} stalled
          </span>
          <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
            {activeAgents} active lanes
          </span>
        </section>

        {convoyError ? (
          <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem' }}>
            Convoys: {convoyError}
          </p>
        ) : null}

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Convoys
          </h2>
          {convoyLoading && !convoys.length ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>Loading convoys…</p>
          ) : convoys.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0, lineHeight: 1.5 }}>
              No convoys yet. When a parent task runs in convoy mode, its subtasks appear here as dependent roles (builder, reviewer, etc.).
            </p>
          ) : (
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.85rem' }}>
              {convoys.map((c) => {
                const parent = taskById.get(c.parent_id);
                const { done, total } = convoyProgress(c);
                const subSorted = [...(c.subtasks ?? [])].sort((a, b) => a.dag_layer - b.dag_layer || a.id.localeCompare(b.id));
                return (
                  <li
                    key={c.id}
                    style={{
                      borderRadius: 'var(--ft-radius-sm)',
                      border: '1px solid var(--mc-border)',
                      background: 'var(--mc-bg-secondary)',
                      padding: '0.85rem 1rem',
                    }}
                  >
                    <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'baseline', justifyContent: 'space-between', gap: '0.5rem' }}>
                      <div style={{ minWidth: 0 }}>
                        <div className="ft-mono" style={{ fontSize: '0.65rem', opacity: 0.65, marginBottom: '0.15rem' }}>
                          {c.id} · parent {c.parent_id}
                        </div>
                        <div style={{ fontWeight: 700, fontSize: '0.95rem' }}>{parent?.title ?? '(parent task not in board cache)'}</div>
                      </div>
                      <span className="ft-chip" style={{ fontSize: '0.72rem' }}>
                        {done}/{total} complete · {c.graph?.edge_count ?? 0} deps · depth {c.graph?.max_depth ?? 0}
                      </span>
                    </div>
                    <div
                      style={{
                        marginTop: '0.75rem',
                        display: 'flex',
                        flexWrap: 'wrap',
                        gap: '0.45rem',
                      }}
                    >
                      {subSorted.map((s: ApiConvoySubtask) => {
                        const st = subtaskStatusStyle(s.status);
                        const label = (s.title || s.agent_role || s.id).trim();
                        return (
                          <div
                            key={s.id}
                            title={`${s.status}${s.dispatch_attempts > 0 ? ` · attempts ${s.dispatch_attempts}` : ''}`}
                            style={{
                              border: `1px solid ${st.border}`,
                              borderRadius: 'var(--ft-radius-sm)',
                              padding: '0.35rem 0.5rem',
                              maxWidth: '14rem',
                              background: 'var(--mc-bg-tertiary)',
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', marginBottom: '0.2rem' }}>
                              <span
                                style={{
                                  width: 6,
                                  height: 6,
                                  borderRadius: 999,
                                  background: st.dot,
                                  flexShrink: 0,
                                }}
                                aria-hidden
                              />
                              <span className="ft-mono" style={{ fontSize: '0.62rem', opacity: 0.75, textTransform: 'uppercase' }}>
                                {s.agent_role || 'role'}
                              </span>
                            </div>
                            <div style={{ fontSize: '0.78rem', lineHeight: 1.35, overflow: 'hidden', textOverflow: 'ellipsis' }}>{label}</div>
                          </div>
                        );
                      })}
                    </div>
                    <div style={{ marginTop: '0.65rem' }}>
                      <Link
                        to={`/p/${encodeURIComponent(pid)}/tasks`}
                        className="ft-btn-ghost"
                        style={{ fontSize: '0.72rem', textDecoration: 'none' }}
                      >
                        Open task board
                      </Link>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Human review queue
          </h2>
          {inHumanReview.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>Nothing in testing, review, or convoy-active right now.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {inHumanReview.map((t) => (
                <TaskCouncilRow key={t.id} t={t} productId={pid} />
              ))}
            </div>
          )}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Plan &amp; approval attention
          </h2>
          {planAttention.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>All open tasks have an approved plan, or none are waiting.</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
              {planAttention.map((t) => (
                <TaskCouncilRow key={t.id} t={t} productId={pid} />
              ))}
            </div>
          )}
        </section>

        <section>
          <h2 className="ft-field-label" style={{ margin: '0 0 0.6rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Recent convoy activity
          </h2>
          {convoyEvents.length === 0 ? (
            <p className="ft-muted" style={{ fontSize: '0.85rem', margin: 0 }}>
              No convoy events in the live buffer yet. Subtask dispatch and completion will show here via SSE.
            </p>
          ) : (
            <ul style={{ listStyle: 'none', margin: 0, padding: 0, display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
              {convoyEvents.map((e) => (
                <li key={e.id}>
                  <FeedEventRow
                    event={e}
                    showRaw={showDevTools}
                    expanded={!!feedExpanded[e.id]}
                    onToggleRaw={() => setFeedExpanded((prev) => ({ ...prev, [e.id]: !prev[e.id] }))}
                  />
                </li>
              ))}
            </ul>
          )}
        </section>

        <p className="ft-muted" style={{ fontSize: '0.72rem', margin: 0, paddingBottom: '0.5rem' }}>
          Use the <Link to={`/p/${encodeURIComponent(pid)}/approvals`}>Approvals</Link> module for swipe, maybe pool, and plan approve actions. Use Refresh to sync convoys with the task board.
        </p>
      </div>
    </div>
  );
}
