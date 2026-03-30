import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Activity,
  Bell,
  ChevronLeft,
  LayoutGrid,
  Pause,
  Play,
  RefreshCw,
  Rocket,
  Search,
  Settings,
} from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import { BackendConnectionPill } from './BackendConnectionPill';
import { ThemeCycleButton } from './ThemeCycleButton';
import { MissionControlOverviewModal, type MissionControlWorkspaceStats } from './MissionControlOverviewModal';
import { formatClock } from '../../lib/time';
import { MissionContextCrumb } from './MissionContextCrumb';

export type { MissionControlWorkspaceStats };

export type MissionControlHeaderExtras = {
  boardSearch: string;
  onBoardSearchChange: (v: string) => void;
  agentsPaused: boolean;
  onAgentsPausedToggle: () => void;
  workspaceStats: MissionControlWorkspaceStats;
  liveActivityPanelOpen: boolean;
  onLiveActivityPanelToggle: () => void;
};

type Props = {
  missionControl?: MissionControlHeaderExtras | null;
  onOpenAbout: () => void;
  /** Desktop Mission Control: opens centered command palette (also ⌘K / Ctrl+K). */
  onOpenCommandPalette?: () => void;
  commandPaletteOpen?: boolean;
  /** From URL + catalog; mobile MC header shows crumbs below the desktop breakpoint. */
  missionCrumbParts?: string[];
  workspaceName?: string | null;
};

export function WorkspaceHeaderBar({
  missionControl = null,
  onOpenAbout,
  onOpenCommandPalette,
  commandPaletteOpen = false,
  missionCrumbParts = [],
  workspaceName = null,
}: Props) {
  const navigate = useNavigate();
  const {
    activeWorkspace,
    productDetail,
    isOnline,
    goHome,
    tasks,
    agents,
    fetchVersion,
    refreshActiveBoard,
  } = useMissionUi();
  const [now, setNow] = useState(() => new Date());
  const [mcOverviewOpen, setMcOverviewOpen] = useState(false);

  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const scopedTasks = useMemo(() => {
    if (!activeWorkspace) return tasks;
    return tasks.filter((t) => t.workspaceId === activeWorkspace.id);
  }, [tasks, activeWorkspace]);

  const scopedAgents = useMemo(() => {
    if (!activeWorkspace) return agents;
    return agents.filter((a) => a.workspaceId === activeWorkspace.id);
  }, [agents, activeWorkspace]);

  const workingAgents = scopedAgents.filter((a) => a.status === 'working').length;
  const tasksInQueue = scopedTasks.filter((t) => t.status !== 'done' && t.status !== 'review').length;

  const mqPending = productDetail?.merge_queue_pending;
  const mergeMethod = productDetail?.merge_policy?.merge_method;

  function handleGoDashboard() {
    goHome();
    navigate('/');
  }

  const useMcShell = Boolean(missionControl && activeWorkspace);

  if (useMcShell && missionControl) {
    return (
      <header className="ft-header-bar ft-header-bar--mc">
        <div className="ft-header-mc-left">
          <button type="button" className="ft-btn-icon" onClick={handleGoDashboard} title="All workspaces">
            <ChevronLeft size={16} aria-hidden />
            <LayoutGrid size={16} aria-hidden />
          </button>
          <button
            type="button"
            className="ft-mc-brand"
            title="Mission Control — workspace overview"
            aria-label="Mission Control — workspace overview"
            aria-haspopup="dialog"
            aria-expanded={mcOverviewOpen}
            onClick={() => setMcOverviewOpen(true)}
          >
            <span className="ft-mc-brand-icon" aria-hidden>
              <Rocket size={18} />
            </span>
            <span className="ft-mc-brand-text ft-hide-below-lg">Mission Control</span>
          </button>
          {missionCrumbParts.length > 0 && workspaceName ? (
            <MissionContextCrumb
              workspaceName={workspaceName}
              parts={missionCrumbParts}
              className="ft-mc-context-crumb ft-mc-context-crumb--mobile-header"
            />
          ) : null}
        </div>

        <div className="ft-header-mc-center">
          <div className="ft-mc-global-search ft-desktop-only">
            <button
              type="button"
              className="ft-mc-global-search-trigger"
              onClick={() => onOpenCommandPalette?.()}
              aria-haspopup="dialog"
              aria-expanded={commandPaletteOpen}
              aria-label="Open command palette"
            >
              <Search size={16} className="ft-mc-global-search-icon" aria-hidden />
              <span
                className={
                  missionControl.boardSearch.trim()
                    ? 'ft-mc-global-search-trigger-label ft-mc-global-search-trigger-label--value'
                    : 'ft-mc-global-search-trigger-label'
                }
              >
                {missionControl.boardSearch.trim()
                  ? missionControl.boardSearch
                  : 'Search tasks, ideas, specs…'}
              </span>
              <span className="ft-mc-global-search-kbd" aria-hidden>
                {typeof navigator !== 'undefined' && /Mac|iPhone|iPad|iPod/.test(navigator.platform ?? '')
                  ? '⌘K'
                  : 'Ctrl+K'}
              </span>
            </button>
          </div>
          <div className="ft-mc-global-search ft-mobile-only">
            <Search size={16} className="ft-mc-global-search-icon" aria-hidden />
            <input
              type="search"
              className="ft-mc-global-search-input"
              placeholder="Search tasks, ideas, specs…"
              aria-label="Global task search"
              value={missionControl.boardSearch}
              onChange={(e) => missionControl.onBoardSearchChange(e.target.value)}
            />
          </div>
        </div>

        <div className="ft-header-mc-right">
          <BackendConnectionPill isOnline={isOnline} className="ft-hide-below-lg" />
          <button
            type="button"
            className="ft-btn-icon"
            title={
              missionControl.liveActivityPanelOpen
                ? 'Hide Live Activity panel'
                : 'Show Live Activity panel'
            }
            aria-label={
              missionControl.liveActivityPanelOpen
                ? 'Hide Live Activity panel'
                : 'Show Live Activity panel'
            }
            aria-pressed={missionControl.liveActivityPanelOpen}
            onClick={() => missionControl.onLiveActivityPanelToggle()}
          >
            <Bell size={18} />
          </button>
          <button
            type="button"
            className="ft-btn-icon"
            title={missionControl.agentsPaused ? 'Resume agents (UI only)' : 'Pause agents (UI only)'}
            aria-pressed={missionControl.agentsPaused}
            onClick={() => missionControl.onAgentsPausedToggle()}
          >
            {missionControl.agentsPaused ? <Play size={18} /> : <Pause size={18} />}
          </button>
 
          <button
            type="button"
            className="ft-btn-icon ft-hide-below-lg"
            title="Refresh board"
            aria-label="Refresh board"
            onClick={() => void refreshActiveBoard()}
          >
            <RefreshCw size={18} />
          </button>
          <ThemeCycleButton />
          <button type="button" className="ft-btn-icon" title="About / settings" onClick={onOpenAbout}>
            <Settings size={18} />
          </button>
         
        </div>

        {activeWorkspace ? (
          <MissionControlOverviewModal
            open={mcOverviewOpen}
            onClose={() => setMcOverviewOpen(false)}
            workspaceName={activeWorkspace.name}
            workspaceIcon={activeWorkspace.icon}
            isOnline={isOnline}
            fetchVersion={fetchVersion}
            productDetail={productDetail}
            productTasks={scopedTasks}
            workspaceStats={missionControl.workspaceStats}
          />
        ) : null}
      </header>
    );
  }

  return (
    <header className="ft-header-bar">
      <div className="ft-header-left">
        {activeWorkspace ? (
          <>
            <button type="button" className="ft-btn-icon" onClick={handleGoDashboard} title="All workspaces">
              <ChevronLeft size={16} />
              <LayoutGrid size={16} />
            </button>
            <div className="ft-chip ft-truncate" style={{ maxWidth: 'min(40vw, 280px)' }}>
              <span aria-hidden>{activeWorkspace.icon}</span>
              <span className="ft-truncate" style={{ fontWeight: 600 }}>
                {activeWorkspace.name}
              </span>
            </div>
            {mqPending != null && mqPending > 0 ? (
              <span className="ft-chip ft-show-lg" style={{ fontSize: '0.65rem' }} title="Merge queue pending (read-only)">
                MQ pending: {mqPending}
                {mergeMethod ? ` · ${mergeMethod}` : ''}
              </span>
            ) : mergeMethod ? (
              <span className="ft-chip ft-show-lg" style={{ fontSize: '0.65rem' }} title="Merge policy (read-only)">
                Merge: {mergeMethod}
              </span>
            ) : null}
          </>
        ) : (
          <button type="button" className="ft-chip" onClick={() => navigate('/')} title="Home">
            <LayoutGrid size={16} />
            <span style={{ fontSize: '0.875rem' }}>All Workspaces</span>
          </button>
        )}
      </div>

      {activeWorkspace ? (
        <div className="ft-show-lg">
          <div className="ft-stat-lg">
            <div className="ft-stat-lg-value" style={{ color: 'var(--mc-accent-blue)' }}>
              {workingAgents}
            </div>
            <div className="ft-stat-lg-label">Agents active</div>
          </div>
          <div className="ft-stat-lg">
            <div className="ft-stat-lg-value" style={{ color: 'var(--mc-accent-purple)' }}>
              {tasksInQueue}
            </div>
            <div className="ft-stat-lg-label">Tasks in queue</div>
          </div>
        </div>
      ) : null}

      <div className="ft-header-right">
        <span className="ft-time" style={{ fontVariantNumeric: 'tabular-nums' }}>
          {formatClock(now)}
        </span>
        <BackendConnectionPill isOnline={isOnline} />
        <ThemeCycleButton />
        <button type="button" className="ft-btn-icon" title="Autopilot hub (placeholder)" onClick={() => navigate('/autopilot')}>
          <Rocket size={20} />
        </button>
        <button type="button" className="ft-btn-icon" title="Activity / operations log" onClick={() => navigate('/activity')}>
          <Activity size={20} />
        </button>
        <button type="button" className="ft-btn-icon" title="About Fishtank / arms version" onClick={onOpenAbout}>
          <Settings size={20} />
        </button>
      </div>
    </header>
  );
}
