import type { Task } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

export function TaskCard({ task }: { task: Task }) {
  return (
    <article className="ft-task-card ft-animate-slide-in">
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.35rem', alignItems: 'baseline' }}>
        <div className="ft-task-card-title" style={{ flex: 1, minWidth: 0 }}>
          {task.title}
        </div>
        <span className="ft-task-status-pill" title="Kanban status">
          {task.status}
        </span>
      </div>
      <div className="ft-task-meta">Updated {formatRelativeTime(task.updatedAt)}</div>
    </article>
  );
}
