import type { Agent, FeedEvent, Task, WorkspaceStats } from '../domain/types';

const iso = (offsetMin: number) =>
  new Date(Date.now() - offsetMin * 60_000).toISOString();

export const MOCK_WORKSPACES: WorkspaceStats[] = [
  {
    id: 'ws-default',
    slug: 'default',
    name: 'Default',
    icon: '🦞',
    taskCounts: { total: 6, active: 4 },
    agentCounts: { total: 2, working: 1 },
  },
  {
    id: 'ws-product',
    slug: 'product',
    name: 'Product Lab',
    icon: '🚀',
    taskCounts: { total: 3, active: 2 },
    agentCounts: { total: 3, working: 2 },
  },
];

export const MOCK_AGENTS: Agent[] = [
  { id: 'a1', name: 'Builder', status: 'working', workspaceId: 'ws-default' },
  { id: 'a2', name: 'Reviewer', status: 'standby', workspaceId: 'ws-default' },
  { id: 'a3', name: 'Research', status: 'working', workspaceId: 'ws-product' },
  { id: 'a4', name: 'Ship', status: 'offline', workspaceId: 'ws-product' },
];

export const MOCK_TASKS: Task[] = [
  {
    id: 't1',
    title: 'Spec OpenAPI for convoy subtasks',
    status: 'planning',
    workspaceId: 'ws-default',
    updatedAt: iso(120),
  },
  {
    id: 't2',
    title: 'Wire cost caps into dispatch',
    status: 'inbox',
    workspaceId: 'ws-default',
    updatedAt: iso(90),
  },
  {
    id: 't3',
    title: 'Mission queue empty-column compaction',
    status: 'assigned',
    workspaceId: 'ws-default',
    updatedAt: iso(45),
  },
  {
    id: 't4',
    title: 'Autopilot swipe deck polish',
    status: 'in_progress',
    workspaceId: 'ws-default',
    updatedAt: iso(12),
  },
  {
    id: 't5',
    title: 'SSE reconnect on dispatch timeout',
    status: 'testing',
    workspaceId: 'ws-product',
    updatedAt: iso(30),
  },
  {
    id: 't6',
    title: 'SQLite backup before migrations',
    status: 'review',
    workspaceId: 'ws-product',
    updatedAt: iso(8),
  },
  {
    id: 't7',
    title: 'Import README in product wizard',
    status: 'done',
    workspaceId: 'ws-product',
    updatedAt: iso(2000),
  },
];

export const MOCK_EVENTS: FeedEvent[] = [
  {
    id: 'e1',
    type: 'task_status_changed',
    message: 'Task "Autopilot swipe deck polish" → in_progress',
    createdAt: iso(15),
  },
  {
    id: 'e2',
    type: 'agent_status_changed',
    message: 'Agent Builder marked working',
    createdAt: iso(20),
  },
  {
    id: 'e3',
    type: 'task_created',
    message: 'New task: Spec OpenAPI for convoy subtasks',
    createdAt: iso(130),
  },
  {
    id: 'e4',
    type: 'system',
    message: 'Gateway heartbeat OK',
    createdAt: iso(2),
  },
];
