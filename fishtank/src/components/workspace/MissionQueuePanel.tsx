import { AlertTriangle, ChevronRight, Plus } from 'lucide-react';
import { useMemo, useState } from 'react';
import { ArmsHttpError } from '../../api/armsClient';
import { useMissionUi } from '../../context/MissionUiContext';
import { KANBAN_COLUMNS, tasksForStatus } from '../../domain/kanban';
import { allowedKanbanTransition } from '../../domain/kanbanTransitions';
import type { Task, TaskStatus } from '../../domain/types';
import { TaskCard } from './TaskCard';
import { NewTaskModal } from './NewTaskModal';
import { TaskDetailModal } from './TaskDetailModal';

export function MissionQueuePanel() {
  const {
    activeWorkspace,
    tasks,
    boardLoading,
    boardLoadFailed,
    patchTaskStatus,
    createTaskForProduct,
    stalledTasks,
    apiError,
  } = useMissionUi();
  const [newOpen, setNewOpen] = useState(false);
  const [selected, setSelected] = useState<Task | null>(null);
  const [dropOver, setDropOver] = useState<TaskStatus | null>(null);
  const [dndError, setDndError] = useState<string | null>(null);

  const scoped = useMemo(() => {
    if (!activeWorkspace) return [];
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  const stalledIds = useMemo(() => new Set(stalledTasks.map((s) => s.taskId)), [stalledTasks]);

  async function handleDrop(columnStatus: TaskStatus, e: React.DragEvent) {
    e.preventDefault();
    setDropOver(null);
    const id = e.dataTransfer.getData('text/task-id');
    if (!id) return;
    const t = scoped.find((x) => x.id === id);
    if (!t) return;
    if (t.status === columnStatus) return;
    if (!allowedKanbanTransition(t.status, columnStatus)) {
      setDndError(`Cannot move from ${t.status} to ${columnStatus} (server rules).`);
      return;
    }
    setDndError(null);
    try {
      await patchTaskStatus(id, columnStatus);
    } catch (e) {
      setDndError(
        e instanceof ArmsHttpError ? `${e.message}${e.code ? ` (${e.code})` : ''}` : 'Could not update status.',
      );
    }
  }

  return (
    <section className="ft-queue-flex ft-queue-relative">
      <div className="ft-border-b" style={{ padding: '0.75rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronRight size={16} className="ft-muted" />
          <span className="ft-upper-label">Mission Queue</span>
        </div>
        <button type="button" className="ft-btn-accent-pink" onClick={() => setNewOpen(true)} disabled={!activeWorkspace}>
          <Plus size={16} />
          New Task
        </button>
      </div>

      {dndError ? (
        <div className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.5rem 0.75rem 0', fontSize: '0.8rem' }}>
          {dndError}
        </div>
      ) : null}

      {stalledTasks.length > 0 ? (
        <div className="ft-banner" role="status" style={{ margin: '0.5rem 0.75rem 0', fontSize: '0.75rem', display: 'flex', alignItems: 'flex-start', gap: '0.5rem' }}>
          <AlertTriangle size={16} style={{ flexShrink: 0, marginTop: 2 }} aria-hidden />
          <span>
            <strong>{stalledTasks.length}</strong> stalled task{stalledTasks.length === 1 ? '' : 's'} (
            <code className="ft-mono">GET …/stalled-tasks</code>). Open a card to send a stall nudge.
          </span>
        </div>
      ) : null}

      <div className="ft-kanban-scroll ft-mission-queue-scroll" style={{ position: 'relative' }}>
        {boardLoading ? (
          <div className="ft-queue-loading" aria-busy="true" aria-label="Loading Kanban">
            <div style={{ display: 'flex', gap: '0.75rem', padding: '0 1rem' }}>
              {[1, 2, 3].map((i) => (
                <div key={i} style={{ width: '200px', flexShrink: 0 }}>
                  <div className="ft-skeleton ft-skeleton--text" style={{ height: '1.5rem', marginBottom: '0.5rem' }} />
                  <div className="ft-skeleton" style={{ height: '4rem', marginBottom: '0.35rem' }} />
                  <div className="ft-skeleton" style={{ height: '4rem' }} />
                </div>
              ))}
            </div>
          </div>
        ) : null}
        {!boardLoading && boardLoadFailed ? (
          <div style={{ padding: '2rem 1rem', textAlign: 'center' }} role="alert">
            <p className="ft-banner ft-banner--error" style={{ display: 'inline-block', textAlign: 'left' }}>
              {apiError ?? 'Failed to load tasks for this product.'}
            </p>
          </div>
        ) : null}
        {!boardLoading && !boardLoadFailed && scoped.length === 0 ? (
          <div className="ft-muted" style={{ padding: '2rem 1rem', textAlign: 'center', fontSize: '0.875rem' }}>
            No tasks in this product yet. Create one with <strong>New Task</strong> (requires an approved idea id).
          </div>
        ) : null}
        {!boardLoadFailed &&
          KANBAN_COLUMNS.map((col) => {
          const colTasks = tasksForStatus(scoped, col.id);
          return (
            <div key={col.id} className={`ft-kanban-col ${col.columnClass}`}>
              <div className="ft-kanban-col-header">
                {col.label} · {colTasks.length}
              </div>
              <div
                className={`ft-kanban-col-body ${dropOver === col.id ? 'ft-kanban-col-body--drop-active' : ''}`}
                onDragOver={(e) => {
                  e.preventDefault();
                  e.dataTransfer.dropEffect = 'move';
                  setDropOver(col.id);
                }}
                onDragLeave={() => setDropOver(null)}
                onDrop={(e) => void handleDrop(col.id, e)}
              >
                {colTasks.map((t) => (
                  <div key={t.id} style={{ position: 'relative' }}>
                    {stalledIds.has(t.id) ? (
                      <span
                        className="ft-chip"
                        style={{
                          position: 'absolute',
                          top: -6,
                          right: 4,
                          zIndex: 1,
                          fontSize: '0.55rem',
                          background: 'color-mix(in srgb, var(--mc-accent-red) 15%, var(--mc-bg))',
                        }}
                        title="Stalled (heartbeat)"
                      >
                        stalled
                      </span>
                    ) : null}
                    <TaskCard task={t} onOpen={() => setSelected(t)} />
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <NewTaskModal open={newOpen} onClose={() => setNewOpen(false)} onCreate={(ideaId, spec) => createTaskForProduct(ideaId, spec)} />
      <TaskDetailModal task={selected} onClose={() => setSelected(null)} />
    </section>
  );
}
