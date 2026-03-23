import { AgentsPanel } from '../components/workspace/AgentsPanel';

/** Full-width agents view — execution agent registry (`GET /api/agents`) + per-product task heartbeats. */
export function MissionAgentsPage() {
  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <AgentsPanel />
    </div>
  );
}
