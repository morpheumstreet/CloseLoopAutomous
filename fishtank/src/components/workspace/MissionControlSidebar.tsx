import { matchPath, useLocation, useNavigate } from 'react-router-dom';
import {
  Activity,
  Bot,
  BookOpen,
  Calendar,
  CheckSquare,
  ClipboardCheck,
  Factory,
  FileText,
  LayoutGrid,
  MessageSquare,
  Network,
  Radar,
  Rocket,
  Settings,
  Users,
  type LucideIcon,
} from 'lucide-react';

type NavEntry =
  | { id: string; label: string; icon: LucideIcon; kind: 'workspace'; segment: string }
  | { id: string; label: string; icon: LucideIcon; kind: 'global'; to: string };

const NAV_ENTRIES: NavEntry[] = [
  { id: 'tasks', label: 'Tasks', icon: CheckSquare, kind: 'workspace', segment: 'tasks' },
  { id: 'agents', label: 'Agents', icon: Bot, kind: 'workspace', segment: 'agents' },
  { id: 'activity_log', label: 'Activity log', icon: Activity, kind: 'global', to: '/activity' },
  { id: 'autopilot', label: 'Autopilot', icon: Rocket, kind: 'global', to: '/autopilot' },
  { id: 'content', label: 'Content', icon: FileText, kind: 'workspace', segment: 'content' },
  { id: 'approvals', label: 'Approvals', icon: ClipboardCheck, kind: 'workspace', segment: 'approvals' },
  { id: 'council', label: 'Council', icon: Users, kind: 'workspace', segment: 'council' },
  { id: 'calendar', label: 'Calendar', icon: Calendar, kind: 'workspace', segment: 'calendar' },
  { id: 'projects', label: 'Projects', icon: LayoutGrid, kind: 'workspace', segment: 'projects' },
  { id: 'memory', label: 'Memory', icon: Network, kind: 'workspace', segment: 'memory' },
  { id: 'docs', label: 'Docs', icon: BookOpen, kind: 'workspace', segment: 'docs' },
  { id: 'people', label: 'People', icon: Users, kind: 'workspace', segment: 'people' },
  { id: 'office', label: 'Office', icon: LayoutGrid, kind: 'workspace', segment: 'office' },
  { id: 'team', label: 'Team', icon: Users, kind: 'workspace', segment: 'team' },
  { id: 'system', label: 'System', icon: Settings, kind: 'workspace', segment: 'system' },
  { id: 'radar', label: 'Radar', icon: Radar, kind: 'workspace', segment: 'radar' },
  { id: 'factory', label: 'Factory', icon: Factory, kind: 'workspace', segment: 'factory' },
  { id: 'pipeline', label: 'Pipeline', icon: Network, kind: 'workspace', segment: 'pipeline' },
  { id: 'feedback', label: 'Feedback', icon: MessageSquare, kind: 'workspace', segment: 'feedback' },
];

type Stats = {
  thisWeek: number;
  inProgress: number;
  total: number;
  completionPct: number;
};

type Props = {
  stats: Stats;
  productId: string;
};

function workspacePath(productId: string, segment: string): string {
  return `/p/${encodeURIComponent(productId)}/${segment}`;
}

function isWorkspaceNavActive(pathname: string, segment: string): boolean {
  return !!matchPath({ path: `/p/:productId/${segment}`, end: true }, pathname);
}

export function MissionControlSidebar({ stats, productId }: Props) {
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;

  return (
    <aside className="ft-mc-sidebar" aria-label="Mission navigation">
      <div className="ft-mc-stats-row" role="group" aria-label="Workspace stats">
        <div className="ft-mc-stat-pill ft-mc-stat-pill--green">
          <span className="ft-mc-stat-pill-value">{stats.thisWeek}</span>
          <span className="ft-mc-stat-pill-label">This week</span>
        </div>
        <div className="ft-mc-stat-pill ft-mc-stat-pill--blue">
          <span className="ft-mc-stat-pill-value">{stats.inProgress}</span>
          <span className="ft-mc-stat-pill-label">In progress</span>
        </div>
        <div className="ft-mc-stat-pill">
          <span className="ft-mc-stat-pill-value">{stats.total}</span>
          <span className="ft-mc-stat-pill-label">Total</span>
        </div>
        <div className="ft-mc-stat-pill ft-mc-stat-pill--progress" title="Share of tasks in Done">
          <span className="ft-mc-stat-pill-value">{stats.completionPct}%</span>
          <span className="ft-mc-stat-pill-label">Done</span>
          <span className="ft-mc-stat-pill-bar" aria-hidden>
            <span className="ft-mc-stat-pill-bar-fill" style={{ width: `${stats.completionPct}%` }} />
          </span>
        </div>
      </div>

      <nav className="ft-mc-nav" aria-label="Primary">
        <ul className="ft-mc-nav-list">
          {NAV_ENTRIES.map((entry) => {
            const Icon = entry.icon;
            const isActive =
              entry.kind === 'workspace'
                ? isWorkspaceNavActive(pathname, entry.segment)
                : pathname === entry.to || pathname.startsWith(`${entry.to}/`);

            function handleClick() {
              if (!productId) return;
              if (entry.kind === 'workspace') navigate(workspacePath(productId, entry.segment));
              else navigate(entry.to);
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
    </aside>
  );
}
