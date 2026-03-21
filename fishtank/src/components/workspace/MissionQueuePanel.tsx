import { ChevronRight, Plus } from 'lucide-react';
import { useMemo } from 'react';
import { useMissionUi } from '../../context/MissionUiContext';
import { KANBAN_COLUMNS, tasksForStatus } from '../../domain/kanban';
import { TaskCard } from './TaskCard';

export function MissionQueuePanel() {
  const { activeWorkspace, tasks } = useMissionUi();

  const scoped = useMemo(() => {
    if (!activeWorkspace) return [];
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  return (
    <section className="ft-queue-flex">
      <div className="ft-border-b" style={{ padding: '0.75rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronRight size={16} className="ft-muted" />
          <span className="ft-upper-label">Mission Queue</span>
        </div>
        <button type="button" className="ft-btn-accent-pink">
          <Plus size={16} />
          New Task
        </button>
      </div>
      <div className="ft-kanban-scroll ft-mission-queue-scroll">
        {KANBAN_COLUMNS.map((col) => {
          const colTasks = tasksForStatus(scoped, col.id);
          return (
            <div key={col.id} className={`ft-kanban-col ${col.columnClass}`}>
              <div className="ft-kanban-col-header">
                {col.label} · {colTasks.length}
              </div>
              <div className="ft-kanban-col-body">
                {colTasks.map((t) => (
                  <TaskCard key={t.id} task={t} />
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
