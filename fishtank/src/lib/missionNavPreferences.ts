import {
  buildMissionNavEntries,
  defaultPaletteSelectionIds,
  orderPaletteEntries,
  type NavEntry,
} from './missionNavCatalog';

export const MISSION_NAV_PREFS_STORAGE_KEY = 'ft-mission-nav-prefs-v1';

export const MISSION_NAV_PREFS_CHANGED_EVENT = 'ft-mission-nav-prefs-changed';

export type MissionNavPrefsV1 = {
  v: 1;
  /** Visible sidebar destinations, in order (ids from catalog). */
  sidebarIds?: string[];
  /** Visible command palette destinations; order is normalized with palette priority when resolved. */
  paletteIds?: string[];
};

function parsePrefs(raw: string): MissionNavPrefsV1 | null {
  try {
    const p = JSON.parse(raw) as MissionNavPrefsV1;
    if (p?.v !== 1) return null;
    return { v: 1, sidebarIds: p.sidebarIds, paletteIds: p.paletteIds };
  } catch {
    return null;
  }
}

export function readMissionNavPrefs(): MissionNavPrefsV1 | null {
  try {
    const raw = localStorage.getItem(MISSION_NAV_PREFS_STORAGE_KEY);
    if (!raw) return null;
    return parsePrefs(raw);
  } catch {
    return null;
  }
}

function catalogIdSet(catalog: NavEntry[]): Set<string> {
  return new Set(catalog.map((e) => e.id));
}

export function filterMissionNavIdsToCatalog(catalog: NavEntry[], ids: string[]): string[] {
  const valid = catalogIdSet(catalog);
  return ids.filter((id) => valid.has(id));
}

/** Merge partial updates with existing prefs (and keep v: 1). */
export function patchMissionNavPrefs(
  patch: Partial<Pick<MissionNavPrefsV1, 'sidebarIds' | 'paletteIds'>>,
): void {
  const cur = readMissionNavPrefs();
  writeMissionNavPrefs({ v: 1, ...cur, ...patch });
}

/** Sidebar: full catalog order when unset; otherwise only listed ids, in saved order. */
export function resolveSidebarEntries(catalog: NavEntry[]): NavEntry[] {
  const pref = readMissionNavPrefs();
  const byId = new Map(catalog.map((e) => [e.id, e] as const));
  if (!pref?.sidebarIds?.length) {
    return catalog;
  }
  const ordered = filterMissionNavIdsToCatalog(catalog, pref.sidebarIds)
    .map((id) => byId.get(id))
    .filter((e): e is NavEntry => Boolean(e));
  return ordered.length > 0 ? ordered : catalog;
}

function entriesForPaletteIds(catalog: NavEntry[], ids: string[]): NavEntry[] {
  const byId = new Map(catalog.map((e) => [e.id, e] as const));
  return ids.map((id) => byId.get(id)).filter((e): e is NavEntry => Boolean(e));
}

/** Command palette: PALETTE_PRIORITY ids when unset; otherwise subset with palette priority applied. */
export function resolvePaletteEntries(catalog: NavEntry[]): NavEntry[] {
  const pref = readMissionNavPrefs();
  if (!pref?.paletteIds?.length) {
    const ids = defaultPaletteSelectionIds(catalog);
    const subset = ids.length > 0 ? entriesForPaletteIds(catalog, ids) : catalog;
    return orderPaletteEntries(subset);
  }
  const byId = new Map(catalog.map((e) => [e.id, e] as const));
  const subset = filterMissionNavIdsToCatalog(catalog, pref.paletteIds)
    .map((id) => byId.get(id))
    .filter((e): e is NavEntry => Boolean(e));
  if (subset.length === 0) {
    const ids = defaultPaletteSelectionIds(catalog);
    const fallback = ids.length > 0 ? entriesForPaletteIds(catalog, ids) : catalog;
    return orderPaletteEntries(fallback);
  }
  return orderPaletteEntries(subset);
}

export function writeMissionNavPrefs(prefs: MissionNavPrefsV1): void {
  try {
    localStorage.setItem(MISSION_NAV_PREFS_STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
  notifyMissionNavPrefsChanged();
}

export function clearMissionNavPrefs(): void {
  try {
    localStorage.removeItem(MISSION_NAV_PREFS_STORAGE_KEY);
  } catch {
    /* ignore */
  }
  notifyMissionNavPrefsChanged();
}

export function notifyMissionNavPrefsChanged(): void {
  try {
    window.dispatchEvent(new Event(MISSION_NAV_PREFS_CHANGED_EVENT));
  } catch {
    /* ignore */
  }
}

export function defaultSidebarIdsForCatalog(catalog: NavEntry[]): string[] {
  return catalog.map((e) => e.id);
}

export function defaultPaletteIdsForCatalog(catalog: NavEntry[]): string[] {
  const ids = defaultPaletteSelectionIds(catalog);
  return ids.length > 0 ? ids : catalog.map((e) => e.id);
}

/** Catalog for current hub visibility (same as shell). */
export function missionNavCatalog(showResearchHubNav: boolean): NavEntry[] {
  return buildMissionNavEntries(showResearchHubNav);
}
