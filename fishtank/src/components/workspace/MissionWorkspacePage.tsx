import { WorkspaceHeaderBar } from '../shell/WorkspaceHeaderBar';
import { AgentsPanel } from './AgentsPanel';
import { LiveFeedPanel } from './LiveFeedPanel';
import { MissionQueuePanel } from './MissionQueuePanel';

/** Desktop three-pane shell modeled after `src/app/workspace/[slug]/page.tsx` in mission-control. */
export function MissionWorkspacePage() {
  return (
    <div className="ft-screen-fixed">
      <WorkspaceHeaderBar />
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
