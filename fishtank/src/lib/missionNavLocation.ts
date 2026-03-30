import { matchPath } from 'react-router-dom';
import type { NavEntry } from './missionNavCatalog';
import { MISSION_SYSTEM_SUB_NAV } from './missionSystemNav';

/** Full-catalog lookup for the current workspace URL — not affected by sidebar / palette prefs. */
export function findNavEntryForWorkspacePath(
  pathname: string,
  productId: string,
  catalog: NavEntry[],
): NavEntry | null {
  if (
    matchPath({ path: '/p/:productId/system/:section', end: true }, pathname)?.params.productId === productId ||
    matchPath({ path: '/p/:productId/system', end: true }, pathname)?.params.productId === productId
  ) {
    return catalog.find((e) => e.kind === 'workspace' && e.segment === 'system') ?? null;
  }

  const m = matchPath({ path: '/p/:productId/:segment', end: true }, pathname);
  if (m?.params.productId !== productId) return null;
  const seg = m.params.segment;
  if (!seg) return null;
  return catalog.find((e) => e.kind === 'workspace' && e.segment === seg) ?? null;
}

/**
 * If the user hid the current destination in sidebar prefs, merge it back so active state and wayfinding work.
 * Order follows the master catalog.
 */
export function ensureSidebarIncludesCurrent(
  sidebar: NavEntry[],
  catalog: NavEntry[],
  current: NavEntry | null,
): NavEntry[] {
  if (!current) return sidebar;
  if (sidebar.some((e) => e.id === current.id)) return sidebar;
  const ids = new Set(sidebar.map((e) => e.id));
  ids.add(current.id);
  return catalog.filter((e) => ids.has(e.id));
}

/** Labels for Mission Control context trail (sidebar / mobile header); prefs-independent. */
export function missionHeaderCrumbParts(
  pathname: string,
  productId: string,
  catalog: NavEntry[],
): string[] {
  const sys = matchPath({ path: '/p/:productId/system/:section', end: true }, pathname);
  if (sys?.params.productId === productId) {
    const section = sys.params.section;
    const sub = MISSION_SYSTEM_SUB_NAV.find((s) => s.segment === section);
    return sub ? ['System', sub.label] : ['System', section ?? ''];
  }

  if (matchPath({ path: '/p/:productId/system', end: true }, pathname)?.params.productId === productId) {
    return ['System', 'Status'];
  }

  const entry = findNavEntryForWorkspacePath(pathname, productId, catalog);
  if (entry) return [entry.label];

  return [];
}
