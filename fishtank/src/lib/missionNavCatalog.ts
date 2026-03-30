import {
  Activity,
  Bot,
  BookOpen,
  Calendar,
  CheckSquare,
  ClipboardCheck,
  Factory,
  FileText,
  FlaskConical,
  LayoutGrid,
  Lightbulb,
  MessageSquare,
  Network,
  Radar,
  Info,
  Rocket,
  Settings,
  Users,
  type LucideIcon,
} from 'lucide-react';

export type NavEntry =
  | { id: string; label: string; icon: LucideIcon; kind: 'workspace'; segment: string }
  | { id: string; label: string; icon: LucideIcon; kind: 'global'; to: string }
  | { id: string; label: string; icon: LucideIcon; kind: 'about' };

export const CORE_NAV_ENTRIES: NavEntry[] = [
  { id: 'tasks', label: 'Tasks', icon: CheckSquare, kind: 'workspace', segment: 'tasks' },
  { id: 'agents', label: 'Agents', icon: Bot, kind: 'workspace', segment: 'agents' },
  { id: 'activity_log', label: 'Activity log', icon: Activity, kind: 'global', to: '/activity' },
  { id: 'autopilot', label: 'Autopilot hub', icon: Rocket, kind: 'global', to: '/autopilot' },
  { id: 'content', label: 'Content', icon: FileText, kind: 'workspace', segment: 'content' },
  { id: 'ideation', label: 'Ideation', icon: Lightbulb, kind: 'workspace', segment: 'ideation' },
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
  { id: 'about', label: 'About', icon: Info, kind: 'about' },
];

export function workspacePath(productId: string, segment: string): string {
  return `/p/${encodeURIComponent(productId)}/${segment}`;
}

export function buildMissionNavEntries(showResearchHubNav: boolean): NavEntry[] {
  if (!showResearchHubNav) return CORE_NAV_ENTRIES;
  const idx = CORE_NAV_ENTRIES.findIndex((e) => e.id === 'system');
  const row: NavEntry = {
    id: 'research_hub',
    label: 'Research hub',
    icon: FlaskConical,
    kind: 'workspace',
    segment: 'research-hub',
  };
  if (idx < 0) return [...CORE_NAV_ENTRIES, row];
  return [...CORE_NAV_ENTRIES.slice(0, idx + 1), row, ...CORE_NAV_ENTRIES.slice(idx + 1)];
}

/** Palette (⌘K) puts these destinations first when present; rest follow catalog order. */
export const PALETTE_PRIORITY = ['autopilot', 'activity_log', 'projects'] as const;

/** Ids shown in the Mission palette when the user has not customized palette visibility (others default to hidden). */
export function defaultPaletteSelectionIds(catalog: NavEntry[]): string[] {
  return (PALETTE_PRIORITY as readonly string[]).filter((id) => catalog.some((e) => e.id === id));
}

export function orderPaletteEntries(entries: NavEntry[]): NavEntry[] {
  const prioritySet = new Set<string>(PALETTE_PRIORITY);
  const top = PALETTE_PRIORITY.map((id) => entries.find((e) => e.id === id)).filter(
    (e): e is NavEntry => Boolean(e),
  );
  const rest = entries.filter((e) => !prioritySet.has(e.id));
  return [...top, ...rest];
}
