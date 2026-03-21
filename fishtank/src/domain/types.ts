export type TaskStatus =
  | 'planning'
  | 'inbox'
  | 'assigned'
  | 'in_progress'
  | 'testing'
  | 'review'
  | 'done';

export interface Workspace {
  id: string;
  slug: string;
  name: string;
  icon: string;
}

export interface WorkspaceStats extends Workspace {
  taskCounts: { total: number; active: number };
  agentCounts: { total: number; working: number };
}

export interface Agent {
  id: string;
  name: string;
  status: 'standby' | 'working' | 'offline';
  workspaceId: string;
}

export interface Task {
  id: string;
  title: string;
  status: TaskStatus;
  workspaceId: string;
  updatedAt: string;
}

export type FeedEventType =
  | 'task_created'
  | 'task_status_changed'
  | 'task_completed'
  | 'agent_status_changed'
  | 'system';

export interface FeedEvent {
  id: string;
  type: FeedEventType;
  message: string;
  createdAt: string;
}

export interface KanbanColumn {
  id: TaskStatus;
  label: string;
  columnClass: string;
}
