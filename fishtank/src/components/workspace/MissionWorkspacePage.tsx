import { useState } from 'react';
import { LayoutGrid, Radio, Users } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import { WorkspaceHeaderBar } from '../shell/WorkspaceHeaderBar';
import { AgentsPanel } from './AgentsPanel';
import { LiveFeedPanel } from './LiveFeedPanel';
import { MissionQueuePanel } from './MissionQueuePanel';

type MobileTab = 'queue' | 'agents' | 'feed';

/** Desktop three-pane shell; mobile uses tabs for Agents + Live Feed parity. */
export function MissionWorkspacePage() {
  const { apiError, dismissError } = useMissionUi();
  const [tab, setTab] = useState<MobileTab>('queue');

  return (
    <div className="ft-screen-fixed">
      <WorkspaceHeaderBar />
      {apiError ? (
        <div className="ft-container" style={{ padding: '0.35rem 1rem 0' }}>
          <div
            className="ft-banner ft-banner--error"
            role="alert"
            style={{ display: 'flex', justifyContent: 'space-between', gap: '0.5rem', alignItems: 'center' }}
          >
            <span style={{ fontSize: '0.8rem' }}>{apiError}</span>
            <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem' }} onClick={dismissError}>
              Dismiss
            </button>
          </div>
        </div>
      ) : null}

      <div className="ft-desktop-only" style={{ flex: 1, minHeight: 0 }}>
        <AgentsPanel />
        <MissionQueuePanel />
        <LiveFeedPanel />
      </div>

      <div className="ft-mobile-only" style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        <div className="ft-mobile-tab-bar" role="tablist" aria-label="Workspace panels">
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'queue'}
            className={`ft-mobile-tab ${tab === 'queue' ? 'ft-mobile-tab--active' : ''}`}
            onClick={() => setTab('queue')}
          >
            <LayoutGrid size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
            Queue
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'agents'}
            className={`ft-mobile-tab ${tab === 'agents' ? 'ft-mobile-tab--active' : ''}`}
            onClick={() => setTab('agents')}
          >
            <Users size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
            Agents
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'feed'}
            className={`ft-mobile-tab ${tab === 'feed' ? 'ft-mobile-tab--active' : ''}`}
            onClick={() => setTab('feed')}
          >
            <Radio size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
            Feed
          </button>
        </div>
        <div style={{ flex: 1, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }} role="tabpanel">
          {tab === 'queue' ? <MissionQueuePanel /> : null}
          {tab === 'agents' ? <AgentsPanel /> : null}
          {tab === 'feed' ? <LiveFeedPanel /> : null}
        </div>
      </div>
    </div>
  );
}
