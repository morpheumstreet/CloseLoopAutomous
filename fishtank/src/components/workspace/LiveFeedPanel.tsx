import { ChevronDown, ChevronRight, ChevronUp, Clock, Radio } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { FeedEvent, FeedEventType } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

type FeedFilter = 'all' | 'tasks' | 'agents' | 'shipping' | 'convoy';

const FILTERS: FeedFilter[] = ['all', 'tasks', 'agents', 'shipping', 'convoy'];

export function LiveFeedPanel() {
  const { events, activeWorkspace, feedLive, bumpFeedReconnect } = useMissionUi();
  const [filter, setFilter] = useState<FeedFilter>('all');
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const filtered = useMemo(() => events.filter((e) => matchesFilter(e, filter)), [events, filter]);

  const showDevTools = import.meta.env.DEV;

  return (
    <aside className="ft-feed">
      <div className="ft-sr-only" aria-live="polite">
        {activeWorkspace && !feedLive ? 'Live feed disconnected.' : ''}
      </div>
      <div className="ft-border-b" style={{ padding: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
            <ChevronRight size={16} className="ft-muted" />
            <span className="ft-upper-label">Live Feed</span>
          </div>
          {activeWorkspace ? (
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
              <span className={`ft-feed-status ${feedLive ? 'ft-feed-status--live' : 'ft-feed-status--off'}`}>
                <Radio size={12} style={{ display: 'inline', verticalAlign: 'middle', marginRight: 4 }} aria-hidden />
                {feedLive ? 'Live' : 'Disconnected'}
              </span>
              {!feedLive ? (
                <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem' }} onClick={() => bumpFeedReconnect()}>
                  Reconnect
                </button>
              ) : null}
            </div>
          ) : null}
        </div>
        <div className="ft-tabs" style={{ marginTop: '0.65rem' }}>
          {FILTERS.map((tab) => (
            <button
              key={tab}
              type="button"
              className={`ft-tab ${filter === tab ? 'ft-tab--active' : ''}`}
              onClick={() => setFilter(tab)}
            >
              {tab}
            </button>
          ))}
        </div>
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
        {filtered.length === 0 ? (
          <div className="ft-muted" style={{ textAlign: 'center', padding: '2rem 0.5rem', fontSize: '0.875rem' }}>
            {activeWorkspace ? 'No events yet for this filter' : 'Open a workspace to stream events'}
          </div>
        ) : (
          filtered.map((e) => (
            <FeedItem
              key={e.id}
              event={e}
              showRaw={showDevTools}
              expanded={!!expanded[e.id]}
              onToggleRaw={() => setExpanded((prev) => ({ ...prev, [e.id]: !prev[e.id] }))}
            />
          ))
        )}
      </div>
    </aside>
  );
}

function matchesFilter(event: FeedEvent, filter: FeedFilter): boolean {
  if (filter === 'all') return true;
  const taskTypes: FeedEventType[] = [
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
  const agentTypes: FeedEventType[] = ['agent_status_changed', 'cost_recorded'];
  const shippingTypes: FeedEventType[] = ['pull_request_opened', 'merge_ship_completed'];
  const convoyTypes: FeedEventType[] = ['convoy_subtask_dispatched', 'convoy_subtask_completed'];
  if (filter === 'tasks') return taskTypes.includes(event.type);
  if (filter === 'agents') return agentTypes.includes(event.type);
  if (filter === 'shipping') return shippingTypes.includes(event.type);
  if (filter === 'convoy') return convoyTypes.includes(event.type);
  return true;
}

function FeedItem({
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
    <div className="ft-feed-item">
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.35rem' }}>
        <span aria-hidden style={{ flexShrink: 0 }}>
          {eventIcon(event.type)}
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ lineHeight: 1.35 }}>{event.message}</div>
          <div className="ft-muted" style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', marginTop: '0.35rem', fontSize: '0.65rem' }}>
            <Clock size={12} aria-hidden />
            {formatRelativeTime(event.createdAt)}
          </div>
          {showRaw && event.raw ? (
            <div style={{ marginTop: '0.35rem' }}>
              <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.65rem', display: 'inline-flex', alignItems: 'center', gap: '0.2rem' }} onClick={onToggleRaw}>
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

function eventIcon(type: FeedEventType): string {
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
