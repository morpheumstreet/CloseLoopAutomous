import { ChevronRight, Radio } from 'lucide-react';
import { useMemo, useState } from 'react';
import { isDevBuild } from '../../config/armsEnv';
import { useMissionUi } from '../../context/MissionUiContext';
import type { FeedFilterTab } from './feedDisplay';
import { FeedEventRow, matchesFeedFilter } from './feedDisplay';

const FILTERS: FeedFilterTab[] = ['all', 'tasks', 'agents', 'shipping', 'convoy'];

type LiveFeedPanelProps = { variant?: 'default' | 'activity' };

export function LiveFeedPanel({ variant = 'default' }: LiveFeedPanelProps) {
  const { events, activeWorkspace, feedLive, bumpFeedReconnect } = useMissionUi();
  const [filter, setFilter] = useState<FeedFilterTab>('all');
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const filtered = useMemo(() => events.filter((e) => matchesFeedFilter(e, filter)), [events, filter]);

  const showDevTools = isDevBuild();

  const title = variant === 'activity' ? 'Live Activity' : 'Live Feed';
  const asideClass = variant === 'activity' ? 'ft-feed ft-feed--mc' : 'ft-feed';

  return (
    <aside className={asideClass}>
      <div className="ft-sr-only" aria-live="polite">
        {activeWorkspace && !feedLive ? 'Live feed disconnected.' : ''}
      </div>
      <div className="ft-border-b" style={{ padding: '0.75rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
            <ChevronRight size={16} className="ft-muted" />
            <span className="ft-upper-label">{title}</span>
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
        {variant === 'default' ? (
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
        ) : (
          <p className="ft-muted" style={{ marginTop: '0.5rem', fontSize: '0.7rem', lineHeight: 1.4 }}>
            Newest events first — agents, tasks, and shipping updates for this workspace.
          </p>
        )}
      </div>
      <div style={{ flex: 1, overflowY: 'auto', padding: '0.5rem', display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
        {filtered.length === 0 ? (
          <div className="ft-muted" style={{ textAlign: 'center', padding: '2rem 0.5rem', fontSize: '0.875rem' }}>
            {activeWorkspace ? 'No events yet for this filter' : 'Open a workspace to stream events'}
          </div>
        ) : (
          filtered.map((e) => (
            <FeedEventRow
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
