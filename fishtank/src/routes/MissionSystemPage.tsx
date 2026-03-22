import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  Activity,
  Copy,
  Cpu,
  ExternalLink,
  HardDrive,
  MemoryStick,
  Radio,
  RefreshCw,
  Server,
  Settings,
} from 'lucide-react';
import { ArmsHttpError, buildLiveEventsUrl, buildLiveEventsUrlTemplate } from '../api/armsClient';
import type { ApiHostMetrics, ApiVersion } from '../api/armsTypes';
import { ThemeCycleButton } from '../components/shell/ThemeCycleButton';
import { useMissionUi } from '../context/MissionUiContext';

function displayVersion(v: ApiVersion): string {
  const n = v.number?.trim();
  if (n) return n;
  const t = v.tag?.trim();
  if (t) return t;
  return v.version?.trim() || '—';
}

function maskSecret(s: string): string {
  const t = s.trim();
  if (!t) return '(unset)';
  if (t.length <= 6) return '••••••';
  return `${t.slice(0, 3)}…${t.slice(-2)}`;
}

function formatBytes(n: number): string {
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

function formatPercent(n: number, digits = 1): string {
  if (!Number.isFinite(n)) return '—';
  return `${n.toFixed(digits)}%`;
}

function UsageBar({ pct }: { pct: number }) {
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

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    /* ignore */
  }
}

export function MissionSystemPage() {
  const { productId } = useParams<{ productId: string }>();
  const {
    armsEnv,
    client,
    isOnline,
    feedLive,
    bumpFeedReconnect,
    fetchVersion,
    refreshWorkspaces,
  } = useMissionUi();

  const [versionLoading, setVersionLoading] = useState(true);
  const [versionError, setVersionError] = useState<string | null>(null);
  const [versionInfo, setVersionInfo] = useState<ApiVersion | null>(null);

  const [hostLoading, setHostLoading] = useState(true);
  const [hostError, setHostError] = useState<string | null>(null);
  const [hostMetrics, setHostMetrics] = useState<ApiHostMetrics | null>(null);

  const [pingBusy, setPingBusy] = useState(false);
  const [pingNote, setPingNote] = useState<string | null>(null);

  const loadVersion = useCallback(async () => {
    setVersionError(null);
    setVersionLoading(true);
    try {
      const data = await fetchVersion();
      setVersionInfo(data);
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setVersionError(e.message);
      } else {
        setVersionError(e instanceof Error ? e.message : 'Could not load version');
      }
      setVersionInfo(null);
    } finally {
      setVersionLoading(false);
    }
  }, [fetchVersion]);

  useEffect(() => {
    void loadVersion();
  }, [loadVersion]);

  const loadHostMetrics = useCallback(async () => {
    setHostError(null);
    setHostLoading(true);
    try {
      const data = await client.hostMetrics();
      setHostMetrics(data);
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setHostError(e.message);
      } else {
        setHostError(e instanceof Error ? e.message : 'Could not load host metrics');
      }
      setHostMetrics(null);
    } finally {
      setHostLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void loadHostMetrics();
  }, [loadHostMetrics]);

  const sseUrl = productId ? buildLiveEventsUrl(armsEnv, productId) : buildLiveEventsUrlTemplate(armsEnv);
  const routesUrl = `${armsEnv.baseUrl.replace(/\/+$/, '')}/api/docs/routes`;

  const pingArms = async () => {
    setPingBusy(true);
    setPingNote(null);
    try {
      await client.health();
      setPingNote('Health check succeeded.');
      void refreshWorkspaces();
    } catch (e) {
      setPingNote(e instanceof Error ? e.message : 'Health check failed.');
    } finally {
      setPingBusy(false);
    }
  };

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <Settings size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">System</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Workspace connection, appearance, and operator references. Values come from Vite env (<code className="ft-mono">VITE_ARMS_*</code>),
              <code className="ft-mono"> GET /api/version</code>, and <code className="ft-mono">GET /api/ops/host-metrics</code> (CPU / RAM / disk for the
              machine running arms). SSE URLs include <code className="ft-mono">?token=</code> when a bearer token is configured.
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <button type="button" className="ft-btn-ghost" disabled={versionLoading} onClick={() => void loadVersion()} title="Reload version">
              <RefreshCw size={16} className={versionLoading ? 'ft-spin' : ''} aria-hidden />
              Refresh version
            </button>
            <button type="button" className="ft-btn-ghost" disabled={hostLoading} onClick={() => void loadHostMetrics()} title="Reload host metrics">
              <RefreshCw size={16} className={hostLoading ? 'ft-spin' : ''} aria-hidden />
              Refresh host
            </button>
          </div>
        </div>

        <section
          style={{
            marginTop: '0.5rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Status
          </h2>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
            <span
              className="ft-chip"
              style={{
                fontSize: '0.75rem',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '0.35rem',
                borderColor: isOnline ? 'color-mix(in srgb, var(--mc-accent) 35%, var(--mc-border))' : undefined,
              }}
            >
              <Server size={14} aria-hidden />
              Backend: {isOnline ? 'reachable' : 'unreachable'}
            </span>
            <span className="ft-chip" style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
              <Radio size={14} className={feedLive ? '' : 'ft-muted'} aria-hidden />
              Live feed: {feedLive ? 'streaming' : 'idle / reconnecting'}
            </span>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.75rem' }}
              disabled={pingBusy}
              onClick={() => void pingArms()}
            >
              <RefreshCw size={14} className={pingBusy ? 'ft-spin' : ''} aria-hidden />
              Ping arms
            </button>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.75rem' }}
              onClick={() => bumpFeedReconnect()}
              title="Bumps EventSource subscription"
            >
              Reconnect SSE
            </button>
            <Link
              to="/activity"
              className="ft-btn-ghost"
              style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '0.35rem', fontSize: '0.75rem' }}
            >
              <Activity size={14} aria-hidden />
              Activity log
            </Link>
          </div>
          {pingNote ? (
            <p className="ft-muted" style={{ margin: '0.65rem 0 0', fontSize: '0.8rem' }}>
              {pingNote}
            </p>
          ) : null}
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Host resources (arms server)
          </h2>
          <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
            Snapshot from <code className="ft-mono">GET /api/ops/host-metrics</code>. CPU usage uses a short server-side sample; values describe the host
            where the API runs (not your browser machine).
          </p>
          {hostLoading ? <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>Loading…</p> : null}
          {hostError ? (
            <p className="ft-banner ft-banner--error" role="alert" style={{ margin: hostLoading ? '0.5rem 0 0' : 0 }}>
              {hostError}
            </p>
          ) : null}
          {hostMetrics && !hostLoading ? (
            <div style={{ display: 'grid', gap: '1rem' }}>
              <div
                style={{
                  padding: '0.65rem',
                  borderRadius: 'var(--ft-radius-sm)',
                  border: '1px solid var(--mc-border)',
                  background: 'var(--mc-bg-tertiary)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', marginBottom: '0.35rem' }}>
                  <Cpu size={16} className="ft-muted" aria-hidden />
                  <span style={{ fontSize: '0.8rem', fontWeight: 600 }}>CPU</span>
                </div>
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.75rem' }}>
                  {hostMetrics.cpu.logical_cores} logical · {hostMetrics.cpu.physical_cores} physical cores ·{' '}
                  <strong style={{ color: 'var(--mc-text-primary)' }}>{formatPercent(hostMetrics.cpu.percent_total)}</strong> busy (
                  {hostMetrics.cpu.sample_interval} sample)
                </p>
                {hostMetrics.cpu.load_avg ? (
                  <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.75rem' }}>
                    Load avg: {hostMetrics.cpu.load_avg.load1.toFixed(2)} / {hostMetrics.cpu.load_avg.load5.toFixed(2)} /{' '}
                    {hostMetrics.cpu.load_avg.load15.toFixed(2)}
                  </p>
                ) : (
                  <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.75rem' }}>
                    Load average not available on this OS.
                  </p>
                )}
              </div>

              <div
                style={{
                  padding: '0.65rem',
                  borderRadius: 'var(--ft-radius-sm)',
                  border: '1px solid var(--mc-border)',
                  background: 'var(--mc-bg-tertiary)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', marginBottom: '0.35rem' }}>
                  <MemoryStick size={16} className="ft-muted" aria-hidden />
                  <span style={{ fontSize: '0.8rem', fontWeight: 600 }}>Memory</span>
                </div>
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.75rem' }}>
                  {formatBytes(hostMetrics.memory.used_bytes)} / {formatBytes(hostMetrics.memory.total_bytes)} used ·{' '}
                  <strong style={{ color: 'var(--mc-text-primary)' }}>{formatPercent(hostMetrics.memory.used_percent)}</strong>
                </p>
                <UsageBar pct={hostMetrics.memory.used_percent} />
                <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.7rem' }}>
                  {formatBytes(hostMetrics.memory.available_bytes)} available
                </p>
              </div>

              <div
                style={{
                  padding: '0.65rem',
                  borderRadius: 'var(--ft-radius-sm)',
                  border: '1px solid var(--mc-border)',
                  background: 'var(--mc-bg-tertiary)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', marginBottom: '0.35rem' }}>
                  <HardDrive size={16} className="ft-muted" aria-hidden />
                  <span style={{ fontSize: '0.8rem', fontWeight: 600 }}>Disk</span>
                </div>
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.75rem', wordBreak: 'break-all' }}>
                  <code className="ft-mono">{hostMetrics.disk.path}</code> — {formatBytes(hostMetrics.disk.used_bytes)} /{' '}
                  {formatBytes(hostMetrics.disk.total_bytes)} ·{' '}
                  <strong style={{ color: 'var(--mc-text-primary)' }}>{formatPercent(hostMetrics.disk.used_percent)}</strong>
                </p>
                <UsageBar pct={hostMetrics.disk.used_percent} />
                <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.7rem' }}>
                  {formatBytes(hostMetrics.disk.free_bytes)} free
                </p>
                {hostMetrics.disk.inodes_total > 0 ? (
                  <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.7rem' }}>
                    Inodes: {hostMetrics.disk.inodes_used.toLocaleString()} / {hostMetrics.disk.inodes_total.toLocaleString()} (
                    {formatPercent(hostMetrics.disk.inodes_percent)})
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Appearance
          </h2>
          <p className="ft-muted" style={{ margin: '0 0 0.5rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
            Theme matches the header control; stored in <code className="ft-mono">localStorage</code> for light/dark, or follow system when set to Auto.
          </p>
          <ThemeCycleButton />
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Connection (build-time env)
          </h2>
          <dl style={{ margin: 0, display: 'grid', gap: '0.5rem', fontSize: '0.8rem' }}>
            <div>
              <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                Base URL
              </dt>
              <dd style={{ margin: 0, wordBreak: 'break-all' }} className="ft-mono">
                {armsEnv.baseUrl}
              </dd>
            </div>
            <div>
              <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                Bearer token
              </dt>
              <dd style={{ margin: 0 }} className="ft-mono">
                {armsEnv.token ? maskSecret(armsEnv.token) : '(unset — VITE_ARMS_TOKEN)'}
              </dd>
            </div>
            <div>
              <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                Basic auth user
              </dt>
              <dd style={{ margin: 0 }} className="ft-mono">
                {armsEnv.basicUser || '(unset — VITE_ARMS_BASIC_USER)'}
              </dd>
            </div>
            <div>
              <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                Basic auth password
              </dt>
              <dd style={{ margin: 0 }} className="ft-mono">
                {armsEnv.basicPassword ? maskSecret(armsEnv.basicPassword) : '(unset — VITE_ARMS_BASIC_PASSWORD)'}
              </dd>
            </div>
          </dl>

          <div style={{ marginTop: '0.85rem' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.35rem', marginBottom: '0.25rem' }}>
              <span className="ft-muted" style={{ fontSize: '0.7rem' }}>
                SSE URL (EventSource)
              </span>
              <button
                type="button"
                className="ft-btn-ghost"
                style={{ fontSize: '0.7rem', display: 'inline-flex', alignItems: 'center', gap: '0.25rem' }}
                onClick={() => void copyText(sseUrl)}
              >
                <Copy size={12} />
                Copy
              </button>
            </div>
            <pre
              style={{
                margin: 0,
                padding: '0.45rem',
                fontSize: '0.65rem',
                wordBreak: 'break-all',
                whiteSpace: 'pre-wrap',
                background: 'var(--mc-bg-tertiary)',
                border: '1px solid var(--mc-border)',
                borderRadius: 'var(--ft-radius-sm)',
              }}
            >
              {sseUrl}
            </pre>
            {!productId ? (
              <p className="ft-muted" style={{ fontSize: '0.7rem', marginTop: '0.35rem', marginBottom: 0 }}>
                Replace <code className="ft-mono">&lt;product_id&gt;</code> with this workspace id, or stay on this page while a product is open.
              </p>
            ) : null}
          </div>
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Arms build
          </h2>
          {versionLoading ? <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>Loading…</p> : null}
          {versionError ? (
            <p className="ft-banner ft-banner--error" role="alert" style={{ margin: versionLoading ? '0.5rem 0 0' : 0 }}>
              {versionError}
            </p>
          ) : null}
          {versionInfo && !versionLoading ? (
            <dl style={{ margin: 0, display: 'grid', gap: '0.5rem', fontSize: '0.85rem' }}>
              <div>
                <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                  Version
                </dt>
                <dd style={{ margin: 0, fontWeight: 700, fontSize: '1.15rem' }}>{displayVersion(versionInfo)}</dd>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem' }}>
                <div>
                  <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                    Tag
                  </dt>
                  <dd style={{ margin: 0 }} className="ft-mono">
                    {versionInfo.tag || '—'}
                  </dd>
                </div>
                <div>
                  <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                    Commit
                  </dt>
                  <dd style={{ margin: 0 }} className="ft-mono">
                    {versionInfo.commit || '—'}
                  </dd>
                </div>
              </div>
              <div>
                <dt className="ft-muted" style={{ fontSize: '0.7rem' }}>
                  Describe string
                </dt>
                <dd style={{ margin: 0, wordBreak: 'break-all' }} className="ft-mono">
                  {versionInfo.version || '—'}
                </dd>
              </div>
            </dl>
          ) : null}
        </section>

        <section
          style={{
            marginTop: '0.75rem',
            padding: '1rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-secondary)',
          }}
        >
          <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            API references
          </h2>
          <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
            OpenAPI spec in the repo: <code className="ft-mono">docs/openapi/arms-openapi.yaml</code>
          </p>
          <a href={routesUrl} target="_blank" rel="noreferrer" className="ft-btn-ghost" style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem', textDecoration: 'none' }}>
            <ExternalLink size={14} aria-hidden />
            GET /api/docs/routes
          </a>
        </section>
      </div>
    </div>
  );
}
