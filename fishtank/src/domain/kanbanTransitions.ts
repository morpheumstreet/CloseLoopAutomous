import type { TaskStatus } from './types';

/** Mirrors `domain.AllowedKanbanTransition` in arms for drag-and-drop hints. */
export function allowedKanbanTransition(from: TaskStatus, to: TaskStatus): boolean {
  if (from === to) return true;
  switch (from) {
    case 'planning':
      return to === 'inbox';
    case 'inbox':
      return to === 'assigned' || to === 'planning' || to === 'convoy_active';
    case 'assigned':
      return to === 'in_progress' || to === 'inbox' || to === 'failed' || to === 'convoy_active';
    case 'in_progress':
      return (
        to === 'testing' ||
        to === 'review' ||
        to === 'failed' ||
        to === 'assigned' ||
        to === 'convoy_active'
      );
    case 'testing':
      return to === 'review' || to === 'in_progress' || to === 'failed';
    case 'review':
      return to === 'done' || to === 'testing' || to === 'in_progress' || to === 'failed';
    case 'done':
    case 'failed':
      return to === 'planning' || to === 'inbox';
    case 'convoy_active':
      return to === 'review' || to === 'failed' || to === 'in_progress' || to === 'inbox';
    default:
      return false;
  }
}
