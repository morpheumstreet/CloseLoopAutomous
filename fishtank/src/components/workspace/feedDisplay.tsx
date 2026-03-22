import { ChevronDown, ChevronUp, Clock } from 'lucide-react';
import type { FeedEvent, FeedEventType } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

export type FeedFilterTab = 'all' | 'tasks' | 'agents' | 'shipping' | 'convoy' | 'alerts';

const TASK_TYPES: FeedEventType[] = [
  'task_created',
  'task_status_changed',
  'task_completed',
  'task_dispatched',
  'task_stall_nudged',
  'task_execution_reassigned',
  'task_chat_message',
  'checkpoint_saved',
  'pull_request_opened',
];

const AGENT_TYPES: FeedEventType[] = ['agent_status_changed', 'cost_recorded'];
const SHIPPING_TYPES: FeedEventType[] = ['pull_request_opened', 'merge_ship_completed'];
const CONVOY_TYPES: FeedEventType[] = ['convoy_subtask_dispatched', 'convoy_subtask_completed'];
const ALERT_TYPES: FeedEventType[] = [
  'task_stall_nudged',
  'task_execution_reassigned',
  'agent_status_changed',
  'cost_recorded',
];

export function matchesFeedFilter(event: FeedEvent, filter: FeedFilterTab): boolean {
  if (filter === 'all') return true;
  if (filter === 'alerts') return ALERT_TYPES.includes(event.type);
  if (filter === 'tasks') return TASK_TYPES.includes(event.type);
  if (filter === 'agents') return AGENT_TYPES.includes(event.type);
  if (filter === 'shipping') return SHIPPING_TYPES.includes(event.type);
  if (filter === 'convoy') return CONVOY_TYPES.includes(event.type);
  return true;
}

export function feedItemModifierClass(type: FeedEventType): string {
  switch (type) {
    case 'task_completed':
    case 'convoy_subtask_completed':
    case 'merge_ship_completed':
      return 'ft-feed-item--positive';
    case 'task_stall_nudged':
    case 'task_execution_reassigned':
    case 'cost_recorded':
      return 'ft-feed-item--warn';
    default:
      return '';
  }
}

export function feedEventIcon(type: FeedEventType): string {
  switch (type) {
    case 'task_created':
      return '📋';
    case 'task_status_changed':
      return '🔄';
    case 'task_completed':
      return '✅';
    case 'task_dispatched':
      return '🚀';
    case 'task_stall_nudged':
      return '⏱️';
    case 'task_execution_reassigned':
      return '🔁';
    case 'task_chat_message':
      return '💬';
    case 'checkpoint_saved':
      return '💾';
    case 'cost_recorded':
      return '💰';
    case 'pull_request_opened':
      return '🔗';
    case 'merge_ship_completed':
      return '🚢';
    case 'convoy_subtask_dispatched':
      return '🧩';
    case 'convoy_subtask_completed':
      return '✔️';
    case 'agent_status_changed':
      return '🔔';
    default:
      return '⚙️';
  }
}

export function FeedEventRow({
  event,
  showRaw,
  expanded,
  onToggleRaw,
}: {
  event: FeedEvent;
  showRaw: boolean;
  expanded: boolean;
  onToggleRaw: () => void;
}) {
  return (
    <div className={`ft-feed-item ${feedItemModifierClass(event.type)}`}>
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.35rem' }}>
        <span aria-hidden style={{ flexShrink: 0 }}>
          {feedEventIcon(event.type)}
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ lineHeight: 1.35 }}>{event.message}</div>
          <div className="ft-muted" style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', marginTop: '0.35rem', fontSize: '0.65rem' }}>
            <Clock size={12} aria-hidden />
            {formatRelativeTime(event.createdAt)}
          </div>
          {showRaw && event.raw ? (
            <div style={{ marginTop: '0.35rem' }}>
              <button
                type="button"
                className="ft-btn-ghost"
                style={{ fontSize: '0.65rem', display: 'inline-flex', alignItems: 'center', gap: '0.2rem' }}
                onClick={onToggleRaw}
              >
                {expanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
                JSON
              </button>
              {expanded ? (
                <pre
                  style={{
                    marginTop: '0.25rem',
                    padding: '0.35rem',
                    fontSize: '0.6rem',
                    overflow: 'auto',
                    maxHeight: '8rem',
                    background: 'var(--mc-bg-tertiary)',
                    border: '1px solid var(--mc-border)',
                  }}
                >
                  {JSON.stringify(event.raw, null, 2)}
                </pre>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
