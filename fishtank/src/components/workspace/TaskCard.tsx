import type { Task } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

export function TaskCard({ task }: { task: Task }) {
  return (
    <article className="ft-task-card ft-animate-slide-in">
      <div className="ft-task-card-title">{task.title}</div>
      <div className="ft-task-meta">Updated {formatRelativeTime(task.updatedAt)}</div>
    </article>
  );
}
