import type { LucideIcon } from 'lucide-react';
import {
  Activity,
  BookOpen,
  Cpu,
  FlaskConical,
  Lightbulb,
  Link2,
  Network,
  Palette,
  Server,
  SlidersHorizontal,
} from 'lucide-react';

export type MissionSystemSubNavItem = {
  segment: string;
  label: string;
  icon: LucideIcon;
};

/** Matches topic sections on the System area; order is sidebar order. */
export const MISSION_SYSTEM_SUB_NAV: MissionSystemSubNavItem[] = [
  { segment: 'status', label: 'Status', icon: Activity },
  { segment: 'host', label: 'Host', icon: Cpu },
  { segment: 'appearance', label: 'Appearance', icon: Palette },
  { segment: 'ideation', label: 'Ideation', icon: Lightbulb },
  { segment: 'navigation', label: 'Navigation', icon: SlidersHorizontal },
  { segment: 'connection', label: 'Connection', icon: Link2 },
  { segment: 'research', label: 'Research', icon: FlaskConical },
  { segment: 'gateway', label: 'Agent Gateways', icon: Network },
  { segment: 'build', label: 'Build', icon: Server },
  { segment: 'api', label: 'API', icon: BookOpen },
];

export function missionSystemPath(productId: string, segment: string): string {
  return `/p/${encodeURIComponent(productId)}/system/${encodeURIComponent(segment)}`;
}

export function missionSystemBasePath(productId: string): string {
  return `/p/${encodeURIComponent(productId)}/system`;
}
