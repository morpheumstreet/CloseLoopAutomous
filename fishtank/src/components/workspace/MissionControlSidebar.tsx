import { useEffect, useMemo, useRef } from 'react';
import { ChevronLeft } from 'lucide-react';
import { matchPath, useLocation, useNavigate } from 'react-router-dom';
import { useMissionNavPreferences } from '../../hooks/useMissionNavPreferences';
import { ensureSidebarIncludesCurrent, findNavEntryForWorkspacePath } from '../../lib/missionNavLocation';
import { missionNavCatalog } from '../../lib/missionNavPreferences';
import { MISSION_SYSTEM_SUB_NAV, missionSystemPath } from '../../lib/missionSystemNav';
import { workspacePath } from '../../lib/missionNavCatalog';
import { MissionContextCrumb } from '../shell/MissionContextCrumb';

export type { NavEntry } from '../../lib/missionNavCatalog';
export {
  CORE_NAV_ENTRIES,
  buildMissionNavEntries,
  orderPaletteEntries,
  PALETTE_PRIORITY,
  workspacePath,
} from '../../lib/missionNavCatalog';

type Stats = {
  thisWeek: number;
  inProgress: number;
  total: number;
  completionPct: number;
};

type Props = {
  stats: Stats;
  productId: string;
  onOpenAbout: () => void;
  /** When arms has at least one research hub (or default hub) configured — shows Research hub in the nav. */
  showResearchHubNav?: boolean;
  missionCrumbParts: string[];
  workspaceName: string | null;
};

function isWorkspaceNavActive(pathname: string, segment: string): boolean {
  const exact = matchPath({ path: `/p/:productId/${segment}`, end: true }, pathname);
  if (exact) return true;
  if (segment === 'system') {
    return (
      !!matchPath({ path: '/p/:productId/system', end: true }, pathname) ||
      !!matchPath({ path: '/p/:productId/system/:section', end: true }, pathname)
    );
  }
  return false;
}

export function MissionControlSidebar({
  stats,
  productId,
  onOpenAbout,
  showResearchHubNav = false,
  missionCrumbParts,
  workspaceName,
}: Props) {
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const prevNonSystemPathRef = useRef<string | null>(null);

  const { sidebarEntries: prefsSidebarEntries } = useMissionNavPreferences(showResearchHubNav);
  const catalog = useMemo(() => missionNavCatalog(showResearchHubNav), [showResearchHubNav]);
  const currentNavEntry = useMemo(
    () => (productId ? findNavEntryForWorkspacePath(pathname, productId, catalog) : null),
    [pathname, productId, catalog],
  );
  const navEntries = useMemo(
    () => ensureSidebarIncludesCurrent(prefsSidebarEntries, catalog, currentNavEntry),
    [prefsSidebarEntries, catalog, currentNavEntry],
  );

  useEffect(() => {
    if (!pathname.includes('/system')) {
      prevNonSystemPathRef.current = pathname;
    }
  }, [pathname]);

  const systemSubNavOpen =
    !!productId &&
    (!!matchPath({ path: '/p/:productId/system', end: true }, pathname) ||
      !!matchPath({ path: '/p/:productId/system/:section', end: true }, pathname));

  const systemSectionMatch = matchPath({ path: '/p/:productId/system/:section', end: true }, pathname);
  const activeSystemSection =
    systemSectionMatch?.params.section ??
    (matchPath({ path: '/p/:productId/system', end: true }, pathname) ? 'status' : null);

  function exitSystemNav() {
    if (!productId) return;
    const fallback = workspacePath(productId, 'tasks');
    const prev = prevNonSystemPathRef.current;
    if (prev && !prev.includes('/system')) {
      navigate(prev);
    } else {
      navigate(fallback);
    }
  }

  return (
    <aside className="ft-mc-sidebar" aria-label="Mission navigation">
      {missionCrumbParts.length > 0 && workspaceName ? (
        <MissionContextCrumb
          workspaceName={workspaceName}
          parts={missionCrumbParts}
          className="ft-mc-context-crumb ft-mc-context-crumb--sidebar"
        />
      ) : null}
      <div className="ft-mc-stats-row" role="group" aria-label="Workspace stats">
        <div className="ft-mc-stat-pill ft-mc-stat-pill--green">
          <span className="ft-mc-stat-pill-value">{stats.thisWeek}</span>
          <span className="ft-mc-stat-pill-label">This week</span>
        </div>
        <div className="ft-mc-stat-pill ft-mc-stat-pill--blue">
          <span className="ft-mc-stat-pill-value">{stats.inProgress}</span>
          <span className="ft-mc-stat-pill-label">In progress</span>
        </div>
        <div className="ft-mc-stat-pill ft-mc-stat-pill--progress" title="Share of tasks in Done">
          <span className="ft-mc-stat-pill-value">{stats.completionPct}%</span>
          <span className="ft-mc-stat-pill-label">Done</span>
          <span className="ft-mc-stat-pill-bar" aria-hidden>
            <span className="ft-mc-stat-pill-bar-fill" style={{ width: `${stats.completionPct}%` }} />
          </span>
        </div>
      </div>

      <div className="ft-mc-nav-viewport">
        <div className={`ft-mc-nav-track ${systemSubNavOpen ? 'ft-mc-nav-track--system' : ''}`}>
          <nav className="ft-mc-nav ft-mc-nav--primary" aria-label="Primary" aria-hidden={systemSubNavOpen}>
            <ul className="ft-mc-nav-list">
            {navEntries.map((entry) => {
              const Icon = entry.icon;
              const isActive =
                entry.kind === 'about'
                  ? false
                  : entry.kind === 'workspace'
                    ? isWorkspaceNavActive(pathname, entry.segment)
                    : pathname === entry.to || pathname.startsWith(`${entry.to}/`);

              function handleClick() {
                if (entry.kind === 'about') {
                  onOpenAbout();
                  return;
                }
                if (!productId) return;
                if (entry.kind === 'workspace') {
                  if (entry.segment === 'system') {
                    navigate(missionSystemPath(productId, 'status'));
                    return;
                  }
                  navigate(workspacePath(productId, entry.segment));
                } else navigate(entry.to);
              }

              const disabled = entry.kind === 'workspace' && !productId;

              return (
                <li key={entry.id}>
                  <button
                    type="button"
                    className={`ft-mc-nav-item ${isActive ? 'ft-mc-nav-item--active' : ''}`}
                    onClick={handleClick}
                    disabled={disabled}
                  >
                    <Icon size={16} aria-hidden className="ft-mc-nav-icon" />
                    <span>{entry.label}</span>
                  </button>
                </li>
              );
            })}
            </ul>
          </nav>

        <nav
          className="ft-mc-nav ft-mc-nav--system-sub"
          aria-label="System sections"
          aria-hidden={!systemSubNavOpen}
        >
          <div className="ft-mc-system-sub-head">
            <button type="button" className="ft-mc-system-sub-back" onClick={exitSystemNav}>
              <ChevronLeft size={16} aria-hidden />
              <span>Main menu</span>
            </button>
          </div>
          <ul className="ft-mc-nav-list ft-mc-nav-list--system-sub">
            {MISSION_SYSTEM_SUB_NAV.map((item) => {
              const SubIcon = item.icon;
              const isSubActive = activeSystemSection === item.segment;
              return (
                <li key={item.segment}>
                  <button
                    type="button"
                    className={`ft-mc-nav-item ft-mc-nav-item--system-sub ${isSubActive ? 'ft-mc-nav-item--active' : ''}`}
                    onClick={() => {
                      if (!productId) return;
                      navigate(missionSystemPath(productId, item.segment));
                    }}
                    disabled={!productId}
                    title={item.label}
                  >
                    <SubIcon size={14} aria-hidden className="ft-mc-nav-icon" />
                    <span className="ft-mc-nav-item__system-label">{item.label}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        </nav>
        </div>
      </div>
    </aside>
  );
}
