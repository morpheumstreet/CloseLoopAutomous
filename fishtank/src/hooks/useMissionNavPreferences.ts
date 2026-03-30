import { useEffect, useMemo, useState } from 'react';
import {
  MISSION_NAV_PREFS_CHANGED_EVENT,
  MISSION_NAV_PREFS_STORAGE_KEY,
  missionNavCatalog,
  resolvePaletteEntries,
  resolveSidebarEntries,
} from '../lib/missionNavPreferences';
import type { NavEntry } from '../lib/missionNavCatalog';

export function useMissionNavPreferences(showResearchHubNav: boolean): {
  sidebarEntries: NavEntry[];
  paletteEntries: NavEntry[];
} {
  const [tick, setTick] = useState(0);

  useEffect(() => {
    const bump = () => setTick((t) => t + 1);
    window.addEventListener(MISSION_NAV_PREFS_CHANGED_EVENT, bump);
    const onStorage = (e: StorageEvent) => {
      if (e.key === MISSION_NAV_PREFS_STORAGE_KEY) bump();
    };
    window.addEventListener('storage', onStorage);
    return () => {
      window.removeEventListener(MISSION_NAV_PREFS_CHANGED_EVENT, bump);
      window.removeEventListener('storage', onStorage);
    };
  }, []);

  const catalog = useMemo(() => missionNavCatalog(showResearchHubNav), [showResearchHubNav]);

  const sidebarEntries = useMemo(() => resolveSidebarEntries(catalog), [catalog, tick]);
  const paletteEntries = useMemo(() => resolvePaletteEntries(catalog), [catalog, tick]);

  return { sidebarEntries, paletteEntries };
}
