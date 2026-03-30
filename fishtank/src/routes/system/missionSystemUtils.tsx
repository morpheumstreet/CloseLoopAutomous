import type { ApiVersion } from '../../api/armsTypes';

export function displayVersion(v: ApiVersion): string {
  const n = v.number?.trim();
  if (n) return n;
  const t = v.tag?.trim();
  if (t) return t;
  return v.version?.trim() || '—';
}

export function maskSecret(s: string): string {
  const t = s.trim();
  if (!t) return '(unset)';
  if (t.length <= 6) return '••••••';
  return `${t.slice(0, 3)}…${t.slice(-2)}`;
}

export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n < 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i += 1;
  }
  const rounded = i === 0 ? Math.round(v) : v < 10 ? Number(v.toFixed(1)) : Math.round(v);
  return `${rounded} ${units[i]}`;
}

export function formatPercent(n: number, digits = 1): string {
  if (!Number.isFinite(n)) return '—';
  return `${n.toFixed(digits)}%`;
}

export function UsageBar({ pct }: { pct: number }) {
  const w = Math.min(100, Math.max(0, pct));
  return (
    <div
      style={{
        height: 6,
        borderRadius: 3,
        background: 'var(--mc-bg-tertiary)',
        overflow: 'hidden',
        marginTop: '0.35rem',
      }}
    >
      <div
        style={{
          width: `${w}%`,
          height: '100%',
          background: 'color-mix(in srgb, var(--mc-accent) 80%, var(--mc-border))',
          borderRadius: 3,
        }}
      />
    </div>
  );
}

export async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    /* ignore */
  }
}
