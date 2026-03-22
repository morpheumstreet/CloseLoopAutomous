import { GripVertical } from 'lucide-react';
import type { Task } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

type Props = {
  task: Task;
  onOpen: () => void;
};

export function TaskCard({ task, onOpen }: Props) {
  return (
    <article className="ft-task-card ft-animate-slide-in" style={{ display: 'flex', gap: '0.35rem', alignItems: 'flex-start' }}>
      <button
        type="button"
        className="ft-btn-icon"
        draggable
        onDragStart={(e) => {
          e.dataTransfer.setData('text/task-id', task.id);
          e.dataTransfer.effectAllowed = 'move';
          e.currentTarget.closest('.ft-task-card')?.classList.add('ft-task-card--dragging');
        }}
        onDragEnd={(e) => {
          e.currentTarget.closest('.ft-task-card')?.classList.remove('ft-task-card--dragging');
        }}
        title="Drag to another column"
        aria-label="Drag task to another column"
        style={{ flexShrink: 0, padding: '0.15rem', margin: '-0.15rem 0 0 -0.15rem' }}
      >
        <GripVertical size={14} className="ft-muted" />
      </button>
      <button
        type="button"
        onClick={() => onOpen()}
        className="ft-task-card-body-btn"
        style={{
          flex: 1,
          minWidth: 0,
          textAlign: 'left',
          background: 'none',
          border: 'none',
          padding: 0,
          color: 'inherit',
          cursor: 'pointer',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.35rem', alignItems: 'baseline' }}>
          <div className="ft-task-card-title" style={{ flex: 1, minWidth: 0 }}>
            {task.title}
          </div>
          <span className="ft-task-status-pill" title="Kanban status">
            {task.status}
          </span>
        </div>
        <div className="ft-task-meta">Updated {formatRelativeTime(task.updatedAt)}</div>
      </button>
    </article>
  );
}
