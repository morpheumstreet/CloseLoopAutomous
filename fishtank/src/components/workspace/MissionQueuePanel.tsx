import { AlertTriangle, Plus } from 'lucide-react';
import { useMemo, useState } from 'react';
import { ArmsHttpError } from '../../api/armsClient';
import { useMissionUi } from '../../context/MissionUiContext';
import { KANBAN_COLUMNS, tasksForStatus } from '../../domain/kanban';
import { allowedKanbanTransition } from '../../domain/kanbanTransitions';
import type { Task, TaskStatus } from '../../domain/types';
import { TaskCard } from './TaskCard';
import { NewTaskModal } from './NewTaskModal';
import { TaskDetailModal } from './TaskDetailModal';

function kanbanColumnMcClass(id: TaskStatus): string {
  if (id === 'planning' || id === 'inbox' || id === 'assigned') return 'ft-kanban-col--mc-backlog';
  if (id === 'in_progress' || id === 'testing' || id === 'convoy_active') return 'ft-kanban-col--mc-active';
  if (id === 'review' || id === 'failed') return 'ft-kanban-col--mc-review';
  if (id === 'done') return 'ft-kanban-col--mc-done';
  return '';
}

type MissionQueuePanelProps = {
  boardSearch?: string;
  assigneeAgentId?: string | null;
  onAssigneeAgentIdChange?: (id: string | null) => void;
  newTaskOpen: boolean;
  onNewTaskOpenChange: (open: boolean) => void;
};

export function MissionQueuePanel({
  boardSearch = '',
  assigneeAgentId = null,
  onAssigneeAgentIdChange,
  newTaskOpen,
  onNewTaskOpenChange,
}: MissionQueuePanelProps) {
  const {
    activeWorkspace,
    tasks,
    agents,
    boardLoading,
    boardLoadFailed,
    patchTaskStatus,
    createTaskForProduct,
    stalledTasks,
    apiError,
  } = useMissionUi();
  const [selected, setSelected] = useState<Task | null>(null);
  const [dropOver, setDropOver] = useState<TaskStatus | null>(null);
  const [dndError, setDndError] = useState<string | null>(null);

  const scoped = useMemo(() => {
    if (!activeWorkspace) return [];
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  const scopedAgents = useMemo(() => {
    if (!activeWorkspace) return [];
    return agents.filter((a) => a.workspaceId === activeWorkspace.id);
  }, [agents, activeWorkspace]);

  const filtered = useMemo(() => {
    let list = scoped;
    const q = boardSearch.trim().toLowerCase();
    if (q) {
      list = list.filter(
        (t) => t.title.toLowerCase().includes(q) || t.spec.toLowerCase().includes(q) || t.ideaId.toLowerCase().includes(q),
      );
    }
    if (assigneeAgentId) {
      list = list.filter((t) => t.currentExecutionAgentId === assigneeAgentId);
    }
    return list;
  }, [scoped, boardSearch, assigneeAgentId]);

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
    <section className="ft-queue-flex ft-queue-relative ft-mc-main">
      <div className="ft-mc-filter-bar ft-border-b">
        <button
          type="button"
          className="ft-btn-primary ft-mc-new-task-main"
          onClick={() => onNewTaskOpenChange(true)}
          disabled={!activeWorkspace}
        >
          <Plus size={18} aria-hidden />
          New Task
        </button>
        <div className="ft-mc-filter-chips" role="group" aria-label="Filter by assignee">
          <button
            type="button"
            className={`ft-mc-assignee-chip ${assigneeAgentId == null ? 'ft-mc-assignee-chip--on' : ''}`}
            onClick={() => onAssigneeAgentIdChange?.(null)}
          >
            All
          </button>
          {scopedAgents.slice(0, 6).map((a) => (
            <button
              key={a.id}
              type="button"
              className={`ft-mc-assignee-chip ${assigneeAgentId === a.id ? 'ft-mc-assignee-chip--on' : ''}`}
              onClick={() => onAssigneeAgentIdChange?.(a.id)}
              title={a.name}
            >
              <span className="ft-mc-assignee-avatar" aria-hidden>
                {initialsFromName(a.name)}
              </span>
              <span className="ft-mc-assignee-name">{firstName(a.name)}</span>
            </button>
          ))}
        </div>
        <label className="ft-mc-project-select-wrap">
          <span className="ft-sr-only">Project filter</span>
          <select className="ft-mc-project-select" disabled aria-disabled="true" defaultValue="all" title="Single workspace — all projects">
            <option value="all">All projects</option>
          </select>
        </label>
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
            No tasks in this product yet. Use <strong>New Task</strong> after you have an approved idea (yes / now) without a linked task.
          </div>
        ) : null}
        {!boardLoading && !boardLoadFailed && scoped.length > 0 && filtered.length === 0 ? (
          <div className="ft-muted ft-mc-filter-empty" role="status">
            No tasks match the current search or assignee filter.
          </div>
        ) : null}
        {!boardLoadFailed &&
          KANBAN_COLUMNS.map((col) => {
          const colTasks = tasksForStatus(filtered, col.id);
          return (
            <div key={col.id} className={`ft-kanban-col ${col.columnClass} ${kanbanColumnMcClass(col.id)}`}>
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
                onDragLeave={(e) => {
                  const rel = e.relatedTarget as Node | null;
                  if (rel && e.currentTarget.contains(rel)) return;
                  setDropOver(null);
                }}
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
                    <TaskCard task={t} agents={scopedAgents} onOpen={() => setSelected(t)} />
                  </div>
                ))}
              </div>
            </div>
          );
        })}
      </div>

      <NewTaskModal
        open={newTaskOpen}
        productId={activeWorkspace?.id ?? ''}
        onClose={() => onNewTaskOpenChange(false)}
        onCreate={(ideaId, spec, newIdeaId) => createTaskForProduct(ideaId, spec, newIdeaId)}
      />
      <TaskDetailModal task={selected} onClose={() => setSelected(null)} />
    </section>
  );
}

function firstName(name: string): string {
  const part = name.trim().split(/\s+/)[0];
  return part || name;
}

function initialsFromName(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}
