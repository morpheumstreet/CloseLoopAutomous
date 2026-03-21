import { Moon, Monitor, Sun } from 'lucide-react';
import { useCallback, useState } from 'react';
import {
  cycleThemePreference,
  getThemePreference,
  type ThemePreference,
} from '../../lib/themePreference';

export function ThemeCycleButton() {
  const [pref, setPref] = useState<ThemePreference>(() => getThemePreference());

  const onClick = useCallback(() => {
    setPref(cycleThemePreference());
  }, []);

  const title =
    pref === 'system'
      ? 'Theme: Auto (follow system). Click for light mode.'
      : pref === 'light'
        ? 'Theme: Light. Click for dark mode.'
        : 'Theme: Dark. Click for auto (system).';

  const label = pref === 'system' ? 'Auto' : pref === 'light' ? 'Light' : 'Dark';

  const Icon = pref === 'system' ? Monitor : pref === 'light' ? Sun : Moon;

  return (
    <button
      type="button"
      className="ft-chip ft-theme-switch"
      onClick={onClick}
      title={title}
      aria-label={title}
    >
      <Icon size={16} aria-hidden />
      <span className="ft-theme-switch__label">{label}</span>
    </button>
  );
}
