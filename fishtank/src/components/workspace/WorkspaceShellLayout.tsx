import { useEffect, useMemo, useState } from 'react';
import { NavLink, Outlet, useLocation, useParams } from 'react-router-dom';
import { LayoutGrid, Radio, Users } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import type { WorkspaceMainOutletContext } from '../../routes/workspaceMainOutletContext';
import { AboutModal } from '../shell/AboutModal';
import { WorkspaceHeaderBar } from '../shell/WorkspaceHeaderBar';
import { LiveFeedPanel } from './LiveFeedPanel';
import { MissionControlSidebar } from './MissionControlSidebar';

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

const LIVE_ACTIVITY_PANEL_STORAGE_KEY = 'ft-live-activity-panel-open';

function readLiveActivityPanelOpen(): boolean {
  try {
    const v = localStorage.getItem(LIVE_ACTIVITY_PANEL_STORAGE_KEY);
    if (v === '0') return false;
    if (v === '1') return true;
  } catch {
    /* ignore */
  }
  return true;
}

/**
 * Nested layout under `/p/:productId/*`.
 * Desktop (≥1024px): sidebar | main (`Outlet`) | live feed — no three-tab strip.
 * Mobile: stats row + Tasks / Agents / Activity tabs + single `Outlet` (see docs/fishtank-ui-todos.md §1 responsive shell).
 */
export function WorkspaceShellLayout() {
  const { productId } = useParams<{ productId: string }>();
  const location = useLocation();
  const { apiError, dismissError, activeWorkspace, tasks, fetchVersion, armsEnv } = useMissionUi();
  const [boardSearch, setBoardSearch] = useState('');
  const [assigneeAgentId, setAssigneeAgentId] = useState<string | null>(null);
  const [newTaskOpen, setNewTaskOpen] = useState(false);
  const [agentsPaused, setAgentsPaused] = useState(false);
  const [liveActivityPanelOpen, setLiveActivityPanelOpen] = useState(readLiveActivityPanelOpen);
  const [aboutOpen, setAboutOpen] = useState(false);

  useEffect(() => {
    try {
      localStorage.setItem(LIVE_ACTIVITY_PANEL_STORAGE_KEY, liveActivityPanelOpen ? '1' : '0');
    } catch {
      /* ignore */
    }
  }, [liveActivityPanelOpen]);

  const stats = useMemo(() => {
    if (!activeWorkspace) {
      return { thisWeek: 0, inProgress: 0, total: 0, completionPct: 0 };
    }
    const scoped = tasks.filter((t) => t.workspaceId === activeWorkspace.id);
    const now = Date.now();
    const thisWeek = scoped.filter((t) => now - new Date(t.updatedAt).getTime() < WEEK_MS).length;
    const inProgress = scoped.filter((t) =>
      t.status === 'in_progress' || t.status === 'testing' || t.status === 'convoy_active',
    ).length;
    const total = scoped.length;
    const done = scoped.filter((t) => t.status === 'done').length;
    const completionPct = total === 0 ? 0 : Math.round((done / total) * 100);
    return { thisWeek, inProgress, total, completionPct };
  }, [tasks, activeWorkspace]);

  const outletContext = useMemo<WorkspaceMainOutletContext>(
    () => ({
      boardSearch,
      onBoardSearchChange: setBoardSearch,
      assigneeAgentId,
      onAssigneeAgentIdChange: setAssigneeAgentId,
      newTaskOpen,
      onNewTaskOpenChange: setNewTaskOpen,
    }),
    [boardSearch, assigneeAgentId, newTaskOpen],
  );

  const hideDesktopFeedColumn = location.pathname.endsWith('/feed');
  const pid = productId ?? '';

  return (
    <div className="ft-screen-fixed">
      <WorkspaceHeaderBar
        onOpenAbout={() => setAboutOpen(true)}
        missionControl={{
          boardSearch,
          onBoardSearchChange: setBoardSearch,
          agentsPaused,
          onAgentsPausedToggle: () => setAgentsPaused((p) => !p),
          workspaceStats: stats,
          liveActivityPanelOpen,
          onLiveActivityPanelToggle: () => setLiveActivityPanelOpen((o) => !o),
        }}
      />
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

      <div className="ft-mc-workspace-body">
        <div className="ft-desktop-only ft-mc-sidebar-slot">
          <MissionControlSidebar stats={stats} productId={pid} onOpenAbout={() => setAboutOpen(true)} />
        </div>

        <div className="ft-mc-workspace-main">
          <div className="ft-mobile-only">
            <div className="ft-mc-stats-row ft-mc-stats-row--mobile" role="group" aria-label="Workspace stats">
              <div className="ft-mc-stat-pill ft-mc-stat-pill--green">
                <span className="ft-mc-stat-pill-value">{stats.thisWeek}</span>
                <span className="ft-mc-stat-pill-label">Week</span>
              </div>
              <div className="ft-mc-stat-pill ft-mc-stat-pill--blue">
                <span className="ft-mc-stat-pill-value">{stats.inProgress}</span>
                <span className="ft-mc-stat-pill-label">Active</span>
              </div>
              <div className="ft-mc-stat-pill">
                <span className="ft-mc-stat-pill-value">{stats.total}</span>
                <span className="ft-mc-stat-pill-label">Total</span>
              </div>
              <div className="ft-mc-stat-pill ft-mc-stat-pill--progress">
                <span className="ft-mc-stat-pill-value">{stats.completionPct}%</span>
                <span className="ft-mc-stat-pill-label">Done</span>
              </div>
            </div>
            <div className="ft-mobile-tab-bar" role="tablist" aria-label="Workspace modules (mobile only)">
              <NavLink
                to={`/p/${pid}/tasks`}
                role="tab"
                className={({ isActive }) => `ft-mobile-tab ${isActive ? 'ft-mobile-tab--active' : ''}`}
                style={{ textAlign: 'center', textDecoration: 'none', color: 'inherit' }}
              >
                <LayoutGrid size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
                Tasks
              </NavLink>
              <NavLink
                to={`/p/${pid}/agents`}
                role="tab"
                className={({ isActive }) => `ft-mobile-tab ${isActive ? 'ft-mobile-tab--active' : ''}`}
                style={{ textAlign: 'center', textDecoration: 'none', color: 'inherit' }}
              >
                <Users size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
                Agents
              </NavLink>
              <NavLink
                to={`/p/${pid}/feed`}
                role="tab"
                className={({ isActive }) => `ft-mobile-tab ${isActive ? 'ft-mobile-tab--active' : ''}`}
                style={{ textAlign: 'center', textDecoration: 'none', color: 'inherit' }}
              >
                <Radio size={14} style={{ display: 'block', margin: '0 auto 0.2rem' }} aria-hidden />
                Activity
              </NavLink>
            </div>
          </div>

          <div className="ft-mc-outlet-mount">
            <Outlet context={outletContext} />
          </div>
        </div>

        <div className="ft-desktop-only ft-mc-feed-slot">
          {hideDesktopFeedColumn || !liveActivityPanelOpen ? null : <LiveFeedPanel variant="activity" />}
        </div>
      </div>

      <AboutModal
        open={aboutOpen}
        onClose={() => setAboutOpen(false)}
        fetchVersion={fetchVersion}
        armsEnv={armsEnv}
        productIdForSse={activeWorkspace?.id ?? null}
      />
    </div>
  );
}
