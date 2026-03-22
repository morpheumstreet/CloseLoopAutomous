import { LiveFeedPanel } from '../components/workspace/LiveFeedPanel';

/** Mobile-first full-column activity; desktop hides duplicate right rail when this route is active. */
export function MissionFeedPage() {
  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <LiveFeedPanel variant="activity" />
    </div>
  );
}
