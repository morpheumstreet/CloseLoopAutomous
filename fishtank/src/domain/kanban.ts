import type { KanbanColumn, TaskStatus } from './types';

/**
 * Column definitions — labels and accent colors mirror mission-control MissionQueue.
 * Mission Control (YouTube reference) shorthand: Planning + Inbox + Assigned ≈ **Backlog**;
 * In progress + Testing + Convoy active ≈ **In progress**; Review + Failed ≈ **Review**; Done ≈ **Done**.
 */
export const KANBAN_COLUMNS: KanbanColumn[] = [
  { id: 'planning', label: '📋 Planning', columnClass: 'ft-col-planning' },
  { id: 'inbox', label: 'Inbox', columnClass: 'ft-col-inbox' },
  { id: 'assigned', label: 'Assigned', columnClass: 'ft-col-assigned' },
  { id: 'in_progress', label: 'In Progress', columnClass: 'ft-col-in_progress' },
  { id: 'testing', label: 'Testing', columnClass: 'ft-col-testing' },
  { id: 'review', label: 'Review', columnClass: 'ft-col-review' },
  { id: 'convoy_active', label: 'Convoy', columnClass: 'ft-col-assigned' },
  { id: 'done', label: 'Done', columnClass: 'ft-col-done' },
  { id: 'failed', label: 'Failed', columnClass: 'ft-col-review' },
];

export function tasksForStatus<T extends { status: TaskStatus }>(
  tasks: T[],
  status: TaskStatus,
): T[] {
  return tasks.filter((t) => t.status === status);
}
