import { useLayoutEffect, useMemo, useState } from 'react';
import { ExternalLink, RefreshCw, X } from 'lucide-react';
import { ArmsHttpError } from '../../api/armsClient';
import { useMissionUi } from '../../context/MissionUiContext';
import { KANBAN_COLUMNS } from '../../domain/kanban';
import { allowedKanbanTransition } from '../../domain/kanbanTransitions';
import type { Task, TaskStatus } from '../../domain/types';
import { apiTaskToTask } from '../../mappers/missionMappers';

type Props = {
  task: Task | null;
  onClose: () => void;
};

function errMsg(e: unknown): string {
  if (e instanceof ArmsHttpError) {
    return `${e.message}${e.code ? ` (${e.code})` : ''} [${e.status}]`;
  }
  return e instanceof Error ? e.message : 'Request failed.';
}

export function TaskDetailModal({ task, onClose }: Props) {
  const {
    client,
    patchTaskStatus,
    approveTaskPlan,
    rejectTaskPlan,
    saveTaskClarifications,
    dispatchTaskById,
    completeTaskById,
    stallNudgeTask,
    openTaskPullRequest,
  } = useMissionUi();

  const [row, setRow] = useState<Task | null>(task);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [clarificationsDraft, setClarificationsDraft] = useState('');
  const [rejectReason, setRejectReason] = useState('');
  const [dispatchCost, setDispatchCost] = useState('0');
  const [nudgeNote, setNudgeNote] = useState('');
  const [prHead, setPrHead] = useState('');
  const [prTitle, setPrTitle] = useState('');

  useLayoutEffect(() => {
    setRow(task);
    setClarificationsDraft(task?.clarificationsJson ?? '');
    setActionError(null);
    setRejectReason('');
    setPrHead(task?.pullRequestHeadBranch ?? '');
    setPrTitle('');
    setNudgeNote('');
    setDispatchCost('0');
  }, [task]);

  async function refreshFromApi() {
    if (!task) return;
    setBusy(true);
    setActionError(null);
    try {
      const t = await client.getTask(task.id);
      setRow(apiTaskToTask(t));
      setClarificationsDraft(t.clarifications_json ?? '');
      setPrHead(t.pull_request_head_branch ?? '');
    } catch (e) {
      setActionError(errMsg(e));
    } finally {
      setBusy(false);
    }
  }

  const moveTargets = useMemo(() => {
    if (!row) return [];
    return KANBAN_COLUMNS.map((c) => c.id).filter(
      (to) => to !== row.status && allowedKanbanTransition(row.status, to),
    );
  }, [row]);

  if (!task || !row) return null;

  const canEditClarifications = row.status === 'planning';
  const canApprove = row.status === 'planning' && !row.planApproved;
  const canRejectPlan =
    (row.status === 'inbox' || row.status === 'assigned') && !String(row.externalRef ?? '').trim();
  const canDispatch = row.status === 'assigned' && row.planApproved;
  const canComplete = row.status === 'in_progress' || row.status === 'testing' || row.status === 'review';
  const canNudge =
    row.status === 'in_progress' ||
    row.status === 'testing' ||
    row.status === 'review' ||
    row.status === 'convoy_active';

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setActionError(null);
    try {
      await fn();
      await refreshFromApi();
    } catch (e) {
      setActionError(errMsg(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-task-detail-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ width: 'min(100%, 520px)', maxHeight: 'min(92vh, 720px)', display: 'flex', flexDirection: 'column' }}>
        <div className="ft-modal-head" style={{ flexShrink: 0 }}>
          <h2 id="ft-task-detail-title" className="ft-truncate" style={{ margin: 0, fontSize: '1.05rem', fontWeight: 600, flex: 1, minWidth: 0 }}>
            {row.title}
          </h2>
          <button type="button" className="ft-btn-icon" onClick={() => void refreshFromApi()} disabled={busy} title="Reload from API" aria-label="Reload task">
            <RefreshCw size={18} className={busy ? 'ft-spin' : ''} />
          </button>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog">
            <X size={18} />
          </button>
        </div>
        <div className="ft-modal-body" style={{ overflowY: 'auto', flex: 1 }}>
          {actionError ? (
            <p className="ft-banner ft-banner--error" role="alert">
              {actionError}
            </p>
          ) : null}

          <dl style={{ display: 'grid', gap: '0.65rem', fontSize: '0.85rem', marginBottom: '1rem' }}>
            <div>
              <dt className="ft-field-label">Status</dt>
              <dd style={{ margin: 0 }}>
                <code className="ft-mono">{row.status}</code>
              </dd>
            </div>
            {row.statusReason ? (
              <div>
                <dt className="ft-field-label">Status reason</dt>
                <dd style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{row.statusReason}</dd>
              </div>
            ) : null}
            <div>
              <dt className="ft-field-label">Task id</dt>
              <dd style={{ margin: 0 }} className="ft-mono">
                {row.id}
              </dd>
            </div>
            <div>
              <dt className="ft-field-label">Idea id</dt>
              <dd style={{ margin: 0 }} className="ft-mono">
                {row.ideaId}
              </dd>
            </div>
            {row.currentExecutionAgentId ? (
              <div>
                <dt className="ft-field-label">Execution agent</dt>
                <dd style={{ margin: 0 }} className="ft-mono">
                  {row.currentExecutionAgentId}
                </dd>
              </div>
            ) : null}
            {row.pullRequestUrl ? (
              <div>
                <dt className="ft-field-label">Pull request</dt>
                <dd style={{ margin: 0 }}>
                  <a href={row.pullRequestUrl} target="_blank" rel="noreferrer" className="ft-btn-ghost" style={{ display: 'inline-flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.8rem' }}>
                    Open PR
                    <ExternalLink size={14} />
                  </a>
                </dd>
              </div>
            ) : null}
            {(row.sandboxPath || row.worktreePath) && (
              <div>
                <dt className="ft-field-label">Paths</dt>
                <dd style={{ margin: 0, display: 'grid', gap: '0.25rem' }}>
                  {row.sandboxPath ? (
                    <span className="ft-mono" style={{ fontSize: '0.72rem', wordBreak: 'break-all' }}>
                      sandbox: {row.sandboxPath}
                    </span>
                  ) : null}
                  {row.worktreePath ? (
                    <span className="ft-mono" style={{ fontSize: '0.72rem', wordBreak: 'break-all' }}>
                      worktree: {row.worktreePath}
                    </span>
                  ) : null}
                </dd>
              </div>
            )}
          </dl>

          <div style={{ marginBottom: '1rem' }}>
            <span className="ft-field-label">Spec</span>
            <pre
              style={{
                marginTop: '0.35rem',
                padding: '0.5rem',
                background: 'var(--mc-bg-tertiary)',
                border: '1px solid var(--mc-border)',
                fontSize: '0.75rem',
                overflow: 'auto',
                maxHeight: '10rem',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {row.spec || '—'}
            </pre>
          </div>

          <div style={{ marginBottom: '1rem' }}>
            <span className="ft-field-label">Move status (keyboard-friendly)</span>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem', marginTop: '0.35rem' }}>
              {moveTargets.length === 0 ? (
                <span className="ft-muted" style={{ fontSize: '0.8rem' }}>
                  No allowed moves from this column.
                </span>
              ) : (
                moveTargets.map((st) => (
                  <button
                    key={st}
                    type="button"
                    className="ft-btn-ghost"
                    style={{ fontSize: '0.75rem' }}
                    disabled={busy}
                    onClick={() =>
                      void run(async () => {
                        await patchTaskStatus(row.id, st as TaskStatus);
                      })
                    }
                  >
                    → {st}
                  </button>
                ))
              )}
            </div>
          </div>

          {canEditClarifications ? (
            <label className="ft-field" style={{ marginBottom: '1rem' }}>
              <span className="ft-field-label">Clarifications JSON</span>
              <textarea
                className="ft-input"
                rows={4}
                value={clarificationsDraft}
                onChange={(e) => setClarificationsDraft(e.target.value)}
                disabled={busy}
                style={{ resize: 'vertical', fontFamily: 'var(--ft-mono, ui-monospace, monospace)', fontSize: '0.75rem' }}
              />
              <button
                type="button"
                className="ft-btn-accent-pink"
                style={{ marginTop: '0.5rem' }}
                disabled={busy}
                onClick={() =>
                  void run(async () => {
                    await saveTaskClarifications(row.id, clarificationsDraft);
                  })
                }
              >
                Save clarifications
              </button>
            </label>
          ) : null}

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginBottom: '1rem' }}>
            {canApprove ? (
              <button type="button" className="ft-btn-primary" disabled={busy} onClick={() => void run(async () => approveTaskPlan(row.id))}>
                Approve plan
              </button>
            ) : null}
            {canRejectPlan ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem', flex: '1 1 12rem' }}>
                <input
                  className="ft-input ft-input--sm"
                  placeholder="Reject reason (optional)"
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                  disabled={busy}
                />
                <button type="button" className="ft-btn-ghost" disabled={busy} onClick={() => void run(async () => rejectTaskPlan(row.id, rejectReason))}>
                  Reject plan
                </button>
              </div>
            ) : null}
          </div>

          {canDispatch ? (
            <div style={{ marginBottom: '1rem', padding: '0.65rem', border: '1px solid var(--mc-border)', background: 'var(--mc-bg-secondary)' }}>
              <span className="ft-field-label">Dispatch</span>
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.35rem', flexWrap: 'wrap', alignItems: 'center' }}>
                <input
                  className="ft-input ft-input--sm"
                  style={{ width: '6rem' }}
                  type="number"
                  min={0}
                  step={0.01}
                  value={dispatchCost}
                  onChange={(e) => setDispatchCost(e.target.value)}
                  disabled={busy}
                  aria-label="Estimated cost"
                />
                <button type="button" className="ft-btn-primary" disabled={busy} onClick={() => void run(async () => dispatchTaskById(row.id, Number(dispatchCost) || 0))}>
                  Dispatch
                </button>
              </div>
              <p className="ft-muted" style={{ fontSize: '0.7rem', marginTop: '0.35rem', marginBottom: 0 }}>
                Budget caps may return <code className="ft-mono">402 budget_exceeded</code>.
              </p>
            </div>
          ) : null}

          {canComplete ? (
            <button type="button" className="ft-btn-primary" style={{ marginBottom: '1rem' }} disabled={busy} onClick={() => void run(async () => completeTaskById(row.id))}>
              Mark complete
            </button>
          ) : null}

          {canNudge ? (
            <div style={{ marginBottom: '1rem' }}>
              <span className="ft-field-label">Stall nudge</span>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem', marginTop: '0.35rem' }}>
                <input className="ft-input ft-input--sm" placeholder="Note (optional)" value={nudgeNote} onChange={(e) => setNudgeNote(e.target.value)} disabled={busy} />
                <button type="button" className="ft-btn-ghost" disabled={busy} onClick={() => void run(async () => stallNudgeTask(row.id, nudgeNote))}>
                  Send stall nudge
                </button>
              </div>
            </div>
          ) : null}

          <div style={{ marginBottom: '0.5rem', padding: '0.65rem', border: '1px solid var(--mc-border)' }}>
            <span className="ft-field-label">Open pull request</span>
            <input className="ft-input ft-input--sm" style={{ marginTop: '0.35rem', width: '100%' }} placeholder="head_branch" value={prHead} onChange={(e) => setPrHead(e.target.value)} disabled={busy} />
            <input className="ft-input ft-input--sm" style={{ marginTop: '0.35rem', width: '100%' }} placeholder="PR title (optional)" value={prTitle} onChange={(e) => setPrTitle(e.target.value)} disabled={busy} />
            <button
              type="button"
              className="ft-btn-accent-pink"
              style={{ marginTop: '0.5rem' }}
              disabled={busy || !prHead.trim()}
              onClick={() =>
                void run(async () => {
                  await openTaskPullRequest(row.id, prHead, prTitle || undefined);
                })
              }
            >
              POST …/pull-request
            </button>
          </div>
        </div>
        <div className="ft-modal-actions" style={{ flexShrink: 0, borderTop: '1px solid var(--mc-border)', padding: '0.75rem 1rem' }}>
          <button type="button" className="ft-btn-primary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
