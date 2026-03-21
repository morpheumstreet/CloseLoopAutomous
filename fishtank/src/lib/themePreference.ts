export const THEME_STORAGE_KEY = 'ft-theme';

export type ThemePreference = 'light' | 'dark' | 'system';

export function getThemePreference(): ThemePreference {
  try {
    const v = localStorage.getItem(THEME_STORAGE_KEY);
    if (v === 'light' || v === 'dark') return v;
  } catch {
    /* ignore */
  }
  return 'system';
}

export function setThemePreference(pref: ThemePreference): void {
  const root = document.documentElement;
  try {
    if (pref === 'light' || pref === 'dark') {
      root.setAttribute('data-theme', pref);
      localStorage.setItem(THEME_STORAGE_KEY, pref);
    } else {
      root.removeAttribute('data-theme');
      localStorage.removeItem(THEME_STORAGE_KEY);
    }
  } catch {
    /* ignore */
  }
}

export function cycleThemePreference(): ThemePreference {
  const cur = getThemePreference();
  const next: ThemePreference =
    cur === 'system' ? 'light' : cur === 'light' ? 'dark' : 'system';
  setThemePreference(next);
  return next;
}
