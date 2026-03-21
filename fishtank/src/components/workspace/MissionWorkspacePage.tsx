import { useMissionUi } from '../../context/MissionUiContext';
import { WorkspaceHeaderBar } from '../shell/WorkspaceHeaderBar';
import { AgentsPanel } from './AgentsPanel';
import { LiveFeedPanel } from './LiveFeedPanel';
import { MissionQueuePanel } from './MissionQueuePanel';

/** Desktop three-pane shell modeled after `src/app/workspace/[slug]/page.tsx` in mission-control. */
export function MissionWorkspacePage() {
  const { apiError, dismissError } = useMissionUi();
  return (
    <div className="ft-screen-fixed">
      <WorkspaceHeaderBar />
      {apiError ? (
        <div className="ft-container" style={{ padding: '0.35rem 1rem 0' }}>
          <div className="ft-banner ft-banner--error" role="alert" style={{ display: 'flex', justifyContent: 'space-between', gap: '0.5rem', alignItems: 'center' }}>
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
      <div className="ft-mobile-only">
        <MissionQueuePanel />
      </div>
    </div>
  );
}
