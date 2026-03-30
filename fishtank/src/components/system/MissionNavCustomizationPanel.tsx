import { useCallback, useEffect, useMemo, useState } from 'react';
import { LayoutGrid } from 'lucide-react';
import { useMissionUi } from '../../context/MissionUiContext';
import { buildMissionNavEntries, type NavEntry } from '../../lib/missionNavCatalog';
import {
  clearMissionNavPrefs,
  defaultPaletteIdsForCatalog,
  filterMissionNavIdsToCatalog,
  MISSION_NAV_PREFS_CHANGED_EVENT,
  patchMissionNavPrefs,
  readMissionNavPrefs,
} from '../../lib/missionNavPreferences';

function sameStringSet(a: Set<string>, b: Set<string>): boolean {
  if (a.size !== b.size) return false;
  for (const id of a) {
    if (!b.has(id)) return false;
  }
  return true;
}

function NavToggleTable({
  title,
  description,
  catalog,
  selected,
  onToggle,
}: {
  title: string;
  description: string;
  catalog: NavEntry[];
  selected: Set<string>;
  onToggle: (id: string) => void;
}) {
  return (
    <div style={{ marginTop: '0.85rem' }}>
      <h3 className="ft-field-label" style={{ margin: '0 0 0.35rem', fontSize: '0.68rem', letterSpacing: '0.04em' }}>
        {title}
      </h3>
      <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.78rem', lineHeight: 1.5 }}>
        {description}
      </p>
      <ul style={{ margin: 0, padding: 0, listStyle: 'none' }}>
        {catalog.map((e) => {
          const on = selected.has(e.id);
          return (
            <li
              key={e.id}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: '0.6rem',
                flexWrap: 'wrap',
                padding: '0.35rem 0',
                borderBottom: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.07))',
              }}
            >
              <span style={{ fontSize: '0.78rem', lineHeight: 1.4, flex: '1 1 14rem', minWidth: 0 }}>{e.label}</span>
              <button
                type="button"
                className={on ? 'ft-btn-primary' : 'ft-btn-ghost'}
                style={{ fontSize: '0.72rem', padding: '0.28rem 0.6rem', flexShrink: 0 }}
                onClick={() => onToggle(e.id)}
                aria-pressed={on}
              >
                {on ? 'Shown' : 'Hidden'}
              </button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

export function MissionNavCustomizationPanel() {
  const { client } = useMissionUi();
  const [showResearchHubNav, setShowResearchHubNav] = useState(false);
  const [sidebarOn, setSidebarOn] = useState<Set<string>>(() => new Set());
  const [paletteOn, setPaletteOn] = useState<Set<string>>(() => new Set());
  const [prefsTick, setPrefsTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const [hubs, settings] = await Promise.all([
          client.listResearchHubs(),
          client.getResearchSystemSettings().catch(() => null),
        ]);
        const hasHubUrl = hubs.some((h) => (h.base_url ?? '').trim().length > 0);
        const hasDefault = Boolean(settings?.default_research_hub_id?.trim());
        if (!cancelled) setShowResearchHubNav(hasHubUrl || hasDefault);
      } catch {
        if (!cancelled) setShowResearchHubNav(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [client]);

  const catalog = useMemo(() => buildMissionNavEntries(showResearchHubNav), [showResearchHubNav]);
  const catalogKey = catalog.map((e) => e.id).join(',');

  const syncFromStorage = useCallback(() => {
    const pref = readMissionNavPrefs();
    const nextSidebar = !pref?.sidebarIds?.length
      ? new Set(catalog.map((e) => e.id))
      : new Set(filterMissionNavIdsToCatalog(catalog, pref.sidebarIds));
    const nextPalette = !pref?.paletteIds?.length
      ? new Set(defaultPaletteIdsForCatalog(catalog))
      : new Set(filterMissionNavIdsToCatalog(catalog, pref.paletteIds));
    setSidebarOn((prev) => (sameStringSet(prev, nextSidebar) ? prev : nextSidebar));
    setPaletteOn((prev) => (sameStringSet(prev, nextPalette) ? prev : nextPalette));
  }, [catalog]);

  useEffect(() => {
    syncFromStorage();
  }, [catalogKey, prefsTick, syncFromStorage]);

  useEffect(() => {
    const on = () => setPrefsTick((t) => t + 1);
    window.addEventListener(MISSION_NAV_PREFS_CHANGED_EVENT, on);
    return () => window.removeEventListener(MISSION_NAV_PREFS_CHANGED_EVENT, on);
  }, []);

  const persistSidebar = useCallback(
    (next: Set<string>) => {
      const ordered = catalog.map((e) => e.id).filter((id) => next.has(id));
      if (ordered.length === 0) return;
      patchMissionNavPrefs({ sidebarIds: ordered });
    },
    [catalog],
  );

  const persistPalette = useCallback(
    (next: Set<string>) => {
      const ordered = catalog.map((e) => e.id).filter((id) => next.has(id));
      if (ordered.length === 0) return;
      patchMissionNavPrefs({ paletteIds: ordered });
    },
    [catalog],
  );

  function toggleSidebar(id: string) {
    setSidebarOn((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        if (next.size <= 1) return prev;
        next.delete(id);
      } else {
        next.add(id);
      }
      persistSidebar(next);
      return next;
    });
  }

  function togglePalette(id: string) {
    setPaletteOn((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        if (next.size <= 1) return prev;
        next.delete(id);
      } else {
        next.add(id);
      }
      persistPalette(next);
      return next;
    });
  }

  function resetAll() {
    clearMissionNavPrefs();
    syncFromStorage();
  }

  return (
    <section
      style={{
        marginTop: '0.75rem',
        padding: '1rem',
        borderRadius: 'var(--ft-radius-sm)',
        border: '1px solid var(--mc-border)',
        background: 'var(--mc-bg-secondary)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
        <LayoutGrid size={18} className="ft-muted" aria-hidden />
        <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
          Navigation & command palette
        </h2>
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.5rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        The <strong>left sidebar</strong> and the <strong>Mission palette</strong> (⌘K / Ctrl+K on desktop) use separate lists. Choose
        which destinations appear in each; order follows the list below. By default, the palette only includes Autopilot, Activity log,
        and Projects until you show more. Stored in this browser (<code className="ft-mono">localStorage</code>).
      </p>

      <NavToggleTable
        title="Left sidebar"
        description="Items appear in this order. At least one destination must stay visible."
        catalog={catalog}
        selected={sidebarOn}
        onToggle={toggleSidebar}
      />

      <NavToggleTable
        title="Mission palette (search modal)"
        description="Quick jump list in the palette; can differ from the sidebar. By default only priority destinations are shown; turn others on to include them. At least one destination must stay visible."
        catalog={catalog}
        selected={paletteOn}
        onToggle={togglePalette}
      />

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginTop: '0.75rem' }}>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} onClick={resetAll}>
          Reset to defaults
        </button>
      </div>
    </section>
  );
}
