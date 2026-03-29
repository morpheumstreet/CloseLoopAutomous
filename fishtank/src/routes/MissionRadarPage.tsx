import { useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Activity, LayoutGrid, Radar, Radio, Users } from 'lucide-react';
import { isDevBuild } from '../config/armsEnv';
import { useMissionUi } from '../context/MissionUiContext';
import type { FeedFilterTab } from '../components/workspace/feedDisplay';
import { FeedEventRow, matchesFeedFilter } from '../components/workspace/feedDisplay';

const RADAR_FILTERS: FeedFilterTab[] = ['all', 'alerts', 'tasks', 'agents', 'shipping', 'convoy'];

export function MissionRadarPage() {
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const {
    events,
    activeWorkspace,
    feedLive,
    bumpFeedReconnect,
    tasks,
    agents,
    stalledTasks,
    isOnline,
  } = useMissionUi();

  const [filter, setFilter] = useState<FeedFilterTab>('alerts');
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const scopedTasks = useMemo(() => {
    if (!activeWorkspace) return tasks;
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  const scopedAgents = useMemo(() => {
    if (!activeWorkspace) return agents;
    return agents.filter((a) => a.workspaceId === activeWorkspace.id);
  }, [agents, activeWorkspace]);

  const inMotion = useMemo(
    () =>
      scopedTasks.filter(
        (t) => t.status === 'in_progress' || t.status === 'testing' || t.status === 'convoy_active',
      ).length,
    [scopedTasks],
  );

  const workingAgents = useMemo(() => scopedAgents.filter((a) => a.status === 'working').length, [scopedAgents]);

  const alertEvents = useMemo(() => events.filter((e) => matchesFeedFilter(e, 'alerts')), [events]);

  const filtered = useMemo(() => events.filter((e) => matchesFeedFilter(e, filter)), [events, filter]);

  const showDevTools = isDevBuild();
  const stalledCount = stalledTasks.length;

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'auto', padding: '1rem 1.25rem' }}>
      <div style={{ maxWidth: '56rem', margin: '0 auto', width: '100%', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
        <header>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
              <span className="ft-muted" aria-hidden>
                <Radar size={22} />
              </span>
              <div>
                <h1 style={{ fontSize: '1.2rem', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>Radar</h1>
                <p className="ft-muted" style={{ margin: '0.25rem 0 0', fontSize: '0.8rem', lineHeight: 1.45, maxWidth: '36rem' }}>
                  Live SSE signals for this workspace, with filters tuned for attention and shipping. Use the operations log for a durable audit trail.
                </p>
              </div>
            </div>
            {activeWorkspace ? (
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                <span className={`ft-feed-status ${feedLive ? 'ft-feed-status--live' : 'ft-feed-status--off'}`}>
                  <Radio size={12} style={{ display: 'inline', verticalAlign: 'middle', marginRight: 4 }} aria-hidden />
                  {feedLive ? 'Stream live' : 'Stream offline'}
                </span>
                {!feedLive ? (
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem' }} onClick={() => bumpFeedReconnect()}>
                    Reconnect
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>
        </header>

        <div
          className="ft-radar-signal-row"
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fill, minmax(9.5rem, 1fr))',
            gap: '0.65rem',
          }}
          role="group"
          aria-label="Workspace signal summary"
        >
          <div
            className={`ft-mc-stat-pill ${isOnline ? 'ft-mc-stat-pill--green' : ''}`}
            style={{ flexDirection: 'column', alignItems: 'stretch', textAlign: 'left', padding: '0.65rem 0.75rem' }}
            title="HTTP reachability of arms from this shell"
          >
            <span className="ft-mc-stat-pill-value" style={{ fontSize: '1rem' }}>
              {isOnline ? 'OK' : '—'}
            </span>
            <span className="ft-mc-stat-pill-label">API</span>
          </div>
          <div
            className="ft-mc-stat-pill"
            style={{
              flexDirection: 'column',
              alignItems: 'stretch',
              textAlign: 'left',
              padding: '0.65rem 0.75rem',
              ...(stalledCount > 0
                ? {
                    borderColor: 'color-mix(in srgb, var(--mc-accent-yellow) 45%, var(--mc-border))',
                    background: 'color-mix(in srgb, var(--mc-accent-yellow) 10%, var(--mc-bg))',
                  }
                : {}),
            }}
            title="Tasks reported stalled (server-side summary)"
          >
            <span
              className="ft-mc-stat-pill-value"
              style={{
                fontSize: '1rem',
                color: stalledCount > 0 ? 'var(--mc-accent-yellow)' : undefined,
              }}
            >
              {stalledCount}
            </span>
            <span className="ft-mc-stat-pill-label">Stalled</span>
          </div>
          <div
            className="ft-mc-stat-pill ft-mc-stat-pill--blue"
            style={{ flexDirection: 'column', alignItems: 'stretch', textAlign: 'left', padding: '0.65rem 0.75rem' }}
          >
            <span className="ft-mc-stat-pill-value" style={{ fontSize: '1rem' }}>
              {inMotion}
            </span>
            <span className="ft-mc-stat-pill-label">In motion</span>
          </div>
          <div
            className="ft-mc-stat-pill"
            style={{ flexDirection: 'column', alignItems: 'stretch', textAlign: 'left', padding: '0.65rem 0.75rem' }}
          >
            <span className="ft-mc-stat-pill-value" style={{ fontSize: '1rem' }}>
              {workingAgents}
            </span>
            <span className="ft-mc-stat-pill-label">Agents working</span>
          </div>
          <div
            className="ft-mc-stat-pill"
            style={{
              flexDirection: 'column',
              alignItems: 'stretch',
              textAlign: 'left',
              padding: '0.65rem 0.75rem',
              ...(alertEvents.length > 0
                ? {
                    borderColor: 'color-mix(in srgb, var(--mc-accent-pink) 40%, var(--mc-border))',
                    background: 'color-mix(in srgb, var(--mc-accent-pink) 8%, var(--mc-bg))',
                  }
                : {}),
            }}
            title="Stalls, reassigns, agent flips, and cost signals in the current buffer"
          >
            <span
              className="ft-mc-stat-pill-value"
              style={{
                fontSize: '1rem',
                color: alertEvents.length > 0 ? 'var(--mc-accent-pink)' : undefined,
              }}
            >
              {alertEvents.length}
            </span>
            <span className="ft-mc-stat-pill-label">Alert events (buffer)</span>
          </div>
        </div>

        <nav className="ft-border-b" style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', paddingBottom: '0.75rem' }} aria-label="Related views">
          <Link
            to="/activity"
            className="ft-btn-ghost"
            style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem', textDecoration: 'none' }}
          >
            <Activity size={14} aria-hidden />
            Operations log
          </Link>
          <Link
            to={`/p/${pid}/feed`}
            className="ft-btn-ghost"
            style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem', textDecoration: 'none' }}
          >
            <Radio size={14} aria-hidden />
            Live activity (full column)
          </Link>
          <Link
            to={`/p/${pid}/tasks`}
            className="ft-btn-ghost"
            style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem', textDecoration: 'none' }}
          >
            <LayoutGrid size={14} aria-hidden />
            Tasks board
          </Link>
          <Link
            to={`/p/${pid}/agents`}
            className="ft-btn-ghost"
            style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem', textDecoration: 'none' }}
          >
            <Users size={14} aria-hidden />
            Agents
          </Link>
        </nav>

        <section aria-labelledby="radar-stream-heading">
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap', marginBottom: '0.65rem' }}>
            <h2 id="radar-stream-heading" className="ft-upper-label" style={{ margin: 0 }}>
              Event stream
            </h2>
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.7rem' }}>
              Newest first · same source as the right-hand feed
            </p>
          </div>
          <div className="ft-tabs" style={{ marginBottom: '0.75rem' }}>
            {RADAR_FILTERS.map((tab) => (
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
          <div
            className="ft-radar-stream"
            style={{
              border: '1px solid var(--mc-border)',
              borderRadius: 'var(--ft-radius-sm)',
              background: 'var(--mc-bg-secondary)',
              padding: '0.5rem',
              display: 'flex',
              flexDirection: 'column',
              gap: '0.35rem',
              minHeight: '12rem',
              maxHeight: 'min(50vh, 28rem)',
              overflowY: 'auto',
            }}
          >
            {!activeWorkspace ? (
              <p className="ft-muted" style={{ textAlign: 'center', padding: '2rem 0.5rem', fontSize: '0.875rem', margin: 0 }}>
                Open a workspace to stream events.
              </p>
            ) : filtered.length === 0 ? (
              <p className="ft-muted" style={{ textAlign: 'center', padding: '2rem 0.5rem', fontSize: '0.875rem', margin: 0 }}>
                No events for this filter yet.
              </p>
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
        </section>
      </div>
    </div>
  );
}
