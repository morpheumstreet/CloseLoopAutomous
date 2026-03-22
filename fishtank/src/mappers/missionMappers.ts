import type { ApiAgentHealthItem, ApiProduct, ApiTask, ArmsSsePayload } from '../api/armsTypes';
import type { Agent, FeedEvent, FeedEventType, StalledTaskRow, Task, TaskStatus, WorkspaceStats } from '../domain/types';

const TASK_STATUSES = new Set<string>([
  'planning',
  'inbox',
  'assigned',
  'in_progress',
  'testing',
  'review',
  'done',
  'failed',
  'convoy_active',
]);

const INACTIVE_FOR_ACTIVE_COUNT = new Set<string>(['done', 'failed']);

export function coerceTaskStatus(raw: string): TaskStatus {
  if (TASK_STATUSES.has(raw)) return raw as TaskStatus;
  return 'inbox';
}

export function titleFromSpec(spec: string): string {
  const line = spec.trim().split('\n')[0]?.trim() ?? '';
  if (!line) return '(no spec)';
  return line.length > 88 ? `${line.slice(0, 85)}…` : line;
}

export function apiTaskToTask(t: ApiTask): Task {
  return {
    id: t.id,
    title: titleFromSpec(t.spec),
    status: coerceTaskStatus(t.status),
    workspaceId: t.product_id,
    updatedAt: t.updated_at,
    ideaId: t.idea_id,
    spec: t.spec,
    statusReason: t.status_reason,
    planApproved: t.plan_approved,
    clarificationsJson: t.clarifications_json,
    sandboxPath: t.sandbox_path,
    worktreePath: t.worktree_path,
    pullRequestUrl: t.pull_request_url,
    pullRequestHeadBranch: t.pull_request_head_branch,
    pullRequestNumber: t.pull_request_number,
    currentExecutionAgentId: t.current_execution_agent_id,
    createdAt: t.created_at,
    externalRef: t.external_ref,
  };
}

export function summarizeTaskCounts(tasks: ApiTask[]): { total: number; active: number; done: number } {
  const total = tasks.length;
  const done = tasks.filter((t) => t.status === 'done').length;
  const active = tasks.filter((t) => !INACTIVE_FOR_ACTIVE_COUNT.has(t.status)).length;
  return { total, active, done };
}

export function summarizeAgentCounts(rows: ApiAgentHealthItem[]): { total: number; working: number } {
  const total = rows.length;
  const working = rows.filter((r) => !r.heartbeat_stale && isActiveAgentStatus(r.status)).length;
  return { total, working };
}

function isActiveAgentStatus(status: string): boolean {
  const s = status.toLowerCase();
  if (!s || s === 'unknown' || s === 'idle' || s === 'completed') return false;
  return true;
}

function iconForProduct(p: ApiProduct): string {
  const u = p.icon_url?.trim();
  if (u) return '🖼️';
  const name = p.name.trim();
  if (!name) return '📦';
  const first = [...name][0];
  return first ?? '📦';
}

export function apiProductToWorkspaceStats(
  p: ApiProduct,
  taskCounts: { total: number; active: number; done: number },
  agentCounts: { total: number; working: number },
): WorkspaceStats {
  const slug = p.workspace_id.trim() || 'default';
  const stage = p.stage?.trim();
  const productUpdatedAt = p.updated_at?.trim();
  return {
    id: p.id,
    slug,
    name: p.name,
    icon: iconForProduct(p),
    taskCounts,
    agentCounts,
    ...(stage ? { stage } : {}),
    ...(productUpdatedAt ? { productUpdatedAt } : {}),
  };
}

export function agentHealthToAgent(row: ApiAgentHealthItem): Agent {
  const stale = row.heartbeat_stale;
  const active = !stale && isActiveAgentStatus(row.status);
  let status: Agent['status'] = 'standby';
  if (stale) status = 'offline';
  else if (active) status = 'working';

  const label = row.status?.trim() || 'heartbeat';
  return {
    id: row.task_id,
    name: `${row.task_id.slice(0, 8)} · ${label}`,
    status,
    workspaceId: row.product_id,
  };
}

function mapArmsTypeToFeedType(armsType: string): FeedEventType {
  switch (armsType) {
    case 'task_dispatched':
      return 'task_dispatched';
    case 'task_completed':
      return 'task_completed';
    case 'task_stall_nudged':
      return 'task_stall_nudged';
    case 'task_execution_reassigned':
      return 'task_execution_reassigned';
    case 'task_chat_message':
    case 'task_chat_queue_ack':
      return 'task_chat_message';
    case 'cost_recorded':
      return 'cost_recorded';
    case 'checkpoint_saved':
      return 'checkpoint_saved';
    case 'pull_request_opened':
      return 'pull_request_opened';
    case 'merge_ship_completed':
      return 'merge_ship_completed';
    case 'convoy_subtask_dispatched':
      return 'convoy_subtask_dispatched';
    case 'convoy_subtask_completed':
      return 'convoy_subtask_completed';
    default:
      return 'system';
  }
}

function formatArmsActivityMessage(p: ArmsSsePayload): string {
  const t = p.type ?? p.event ?? 'event';
  const task = p.task_id ? ` task ${p.task_id.slice(0, 8)}` : '';
  switch (p.type) {
    case 'task_dispatched':
      return `Task dispatched${task}`;
    case 'task_completed':
      return `Task completed${task}`;
    case 'task_stall_nudged':
      return `Stall nudge${task}`;
    case 'task_execution_reassigned':
      return `Execution reassigned${task}`;
    case 'task_chat_message':
      return `Task chat${task}`;
    case 'task_chat_queue_ack':
      return `Chat queue ack${task}`;
    case 'cost_recorded': {
      const amount = p.data?.amount;
      return typeof amount === 'number' ? `Cost recorded: ${amount}` : 'Cost recorded';
    }
    case 'checkpoint_saved':
      return `Checkpoint saved${task}`;
    case 'pull_request_opened': {
      const url = p.data?.html_url;
      return typeof url === 'string' ? `PR opened: ${url}` : 'Pull request opened';
    }
    case 'merge_ship_completed':
      return `Merge ship completed${task}`;
    case 'convoy_subtask_dispatched':
      return `Convoy subtask dispatched${task}`;
    case 'convoy_subtask_completed':
      return `Convoy subtask completed${task}`;
    default:
      return `${t}${task}`;
  }
}

/** Maps one SSE `data:` JSON line to a UI feed row, or null to skip (e.g. hello). */
export function ssePayloadToFeedEvent(raw: unknown, seq: number, includeRaw: boolean): FeedEvent | null {
  if (!raw || typeof raw !== 'object') return null;
  const p = raw as ArmsSsePayload;
  if (p.event === 'hello') return null;
  const armsType = p.type ?? p.event;
  if (!armsType) return null;
  const armsTypeStr = String(armsType);
  const ts = p.ts && typeof p.ts === 'string' ? p.ts : new Date().toISOString();
  const rawObj = includeRaw && raw && typeof raw === 'object' ? { ...(raw as Record<string, unknown>) } : undefined;
  return {
    id: `${ts}-${seq}-${armsTypeStr}`,
    type: mapArmsTypeToFeedType(armsTypeStr),
    message: formatArmsActivityMessage(p),
    createdAt: ts,
    armsType: armsTypeStr,
    raw: rawObj,
  };
}

export function stalledApiToRows(rows: Record<string, unknown>[]): StalledTaskRow[] {
  return rows
    .map((r) => {
      const taskId = typeof r.task_id === 'string' ? r.task_id : '';
      const status = typeof r.status === 'string' ? r.status : '';
      const reason = typeof r.reason === 'string' ? r.reason : '';
      if (!taskId) return null;
      return { taskId, status, reason };
    })
    .filter((x): x is NonNullable<typeof x> => x != null);
}
