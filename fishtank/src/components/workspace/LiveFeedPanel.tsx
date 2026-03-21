import { ChevronRight, Clock } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { FeedEvent, FeedEventType } from '../../domain/types';
import { formatRelativeTime } from '../../lib/time';

type FeedFilter = 'all' | 'tasks' | 'agents';

const FILTERS: FeedFilter[] = ['all', 'tasks', 'agents'];

export function LiveFeedPanel() {
  const { events } = useMissionUi();
  const [filter, setFilter] = useState<FeedFilter>('all');

  const filtered = useMemo(() => events.filter((e) => matchesFilter(e, filter)), [events, filter]);

  return (
    <aside className="ft-feed">
      <div className="ft-border-b" style={{ padding: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronRight size={16} className="ft-muted" />
          <span className="ft-upper-label">Live Feed</span>
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
            No events yet
          </div>
        ) : (
          filtered.map((e) => <FeedItem key={e.id} event={e} />)
        )}
      </div>
    </aside>
  );
}

function matchesFilter(event: FeedEvent, filter: FeedFilter): boolean {
  if (filter === 'all') return true;
  const taskTypes: FeedEventType[] = ['task_created', 'task_status_changed', 'task_completed'];
  const agentTypes: FeedEventType[] = ['agent_status_changed'];
  if (filter === 'tasks') return taskTypes.includes(event.type);
  if (filter === 'agents') return agentTypes.includes(event.type);
  return true;
}

function FeedItem({ event }: { event: FeedEvent }) {
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
    case 'agent_status_changed':
      return '🔔';
    default:
      return '⚙️';
  }
}
