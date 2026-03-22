export type TaskStatus =
  | 'planning'
  | 'inbox'
  | 'assigned'
  | 'in_progress'
  | 'testing'
  | 'review'
  | 'done'
  | 'failed'
  | 'convoy_active';

export interface Workspace {
  id: string;
  slug: string;
  name: string;
  icon: string;
}

export interface WorkspaceStats extends Workspace {
  taskCounts: { total: number; active: number; done: number };
  agentCounts: { total: number; working: number };
  /** Product lifecycle stage from arms (`GET /api/products`). */
  stage?: string;
  /** Product row `updated_at` (ISO) for fallbacks when the operations log has no row. */
  productUpdatedAt?: string;
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
  ideaId: string;
  spec: string;
  statusReason?: string;
  planApproved?: boolean;
  clarificationsJson?: string;
  sandboxPath?: string;
  worktreePath?: string;
  pullRequestUrl?: string;
  pullRequestHeadBranch?: string;
  pullRequestNumber?: number;
  currentExecutionAgentId?: string;
  createdAt?: string;
  externalRef?: string;
}

export type FeedEventType =
  | 'task_created'
  | 'task_status_changed'
  | 'task_completed'
  | 'task_dispatched'
  | 'task_stall_nudged'
  | 'task_execution_reassigned'
  | 'task_chat_message'
  | 'cost_recorded'
  | 'checkpoint_saved'
  | 'pull_request_opened'
  | 'merge_ship_completed'
  | 'convoy_subtask_dispatched'
  | 'convoy_subtask_completed'
  | 'agent_status_changed'
  | 'system';

export interface FeedEvent {
  id: string;
  type: FeedEventType;
  message: string;
  createdAt: string;
  /** Original SSE `type` / `event` string from arms (for debounced board refresh). */
  armsType?: string;
  /** Raw SSE JSON for dev inspector */
  raw?: Record<string, unknown>;
}

export interface KanbanColumn {
  id: TaskStatus;
  label: string;
  columnClass: string;
}

export type StalledTaskRow = {
  taskId: string;
  status: string;
  reason: string;
};
