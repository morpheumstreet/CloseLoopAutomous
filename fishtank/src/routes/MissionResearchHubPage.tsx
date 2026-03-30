import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  FlaskConical,
  GitBranch,
  Layers,
  Play,
  RefreshCw,
  Square,
  Terminal,
} from 'lucide-react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { ArmsHttpError, type ArmsClient } from '../api/armsClient';
import type { ApiResearchHub, ApiResearchHubInvokeResult, ApiResearchSystemSettings } from '../api/armsTypes';
import { ResearchClawHubExplorer } from '../components/research/ResearchClawHubExplorer';
import { useMissionUi } from '../context/MissionUiContext';
import { inferPipelineRunState } from '../lib/researchHubPipeline';

type MainTab = 'overview' | 'pipeline' | 'runs' | 'explorer';

function parseInvokeJson(res: ApiResearchHubInvokeResult): unknown | null {
  if (res.json !== undefined) return res.json;
  const t = res.body?.trim();
  if (!t) return null;
  try {
    return JSON.parse(t) as unknown;
  } catch {
    return null;
  }
}

function invokeErrorMessage(res: ApiResearchHubInvokeResult, fallback: string): string {
  const j = parseInvokeJson(res);
  if (j && typeof j === 'object' && j !== null && 'error' in j) {
    const e = (j as { error?: unknown }).error;
    if (typeof e === 'string' && e.trim()) return e;
  }
  if (res.body?.trim()) return `${fallback} (${res.status}): ${res.body.slice(0, 280)}`;
  return `${fallback} (HTTP ${res.status})`;
}

export function MissionResearchHubPage() {
  const { client } = useMissionUi();
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const [searchParams, setSearchParams] = useSearchParams();

  const [hubs, setHubs] = useState<ApiResearchHub[]>([]);
  const [settings, setSettings] = useState<ApiResearchSystemSettings | null>(null);
  const [loadingHubs, setLoadingHubs] = useState(true);
  const [hubLoadError, setHubLoadError] = useState<string | null>(null);
  const [selectedHubId, setSelectedHubIdState] = useState<string | null>(null);

  const [mainTab, setMainTab] = useState<MainTab>('overview');

  const [overviewBusy, setOverviewBusy] = useState(false);
  const [healthJson, setHealthJson] = useState<unknown | null>(null);
  const [versionJson, setVersionJson] = useState<unknown | null>(null);
  const [overviewErr, setOverviewErr] = useState<string | null>(null);

  const [statusJson, setStatusJson] = useState<unknown | null>(null);
  const [statusErr, setStatusErr] = useState<string | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [stagesJson, setStagesJson] = useState<unknown | null>(null);
  const [pipelineAction, setPipelineAction] = useState<'start' | 'stop' | null>(null);

  const [runsJson, setRunsJson] = useState<unknown | null>(null);
  const [runsErr, setRunsErr] = useState<string | null>(null);
  const [runsLoading, setRunsLoading] = useState(false);
  const [runIdInput, setRunIdInput] = useState('');
  const [runDetailJson, setRunDetailJson] = useState<unknown | null>(null);
  const [runDetailErr, setRunDetailErr] = useState<string | null>(null);
  const [runDetailLoading, setRunDetailLoading] = useState(false);

  const reloadHubs = useCallback(async () => {
    setHubLoadError(null);
    setLoadingHubs(true);
    try {
      const [list, st] = await Promise.all([
        client.listResearchHubs(),
        client.getResearchSystemSettings().catch(() => null),
      ]);
      setHubs(list);
      setSettings(st);
    } catch (e) {
      setHubLoadError(e instanceof Error ? e.message : 'Could not load hubs');
      setHubs([]);
      setSettings(null);
    } finally {
      setLoadingHubs(false);
    }
  }, [client]);

  useEffect(() => {
    void reloadHubs();
  }, [reloadHubs]);

  const setSelectedHubId = useCallback(
    (id: string) => {
      setSelectedHubIdState(id);
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          next.set('hub', id);
          return next;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  useEffect(() => {
    if (hubs.length === 0 || selectedHubId) return;
    const q = searchParams.get('hub');
    if (q && hubs.some((h) => h.id === q)) {
      setSelectedHubIdState(q);
      return;
    }
    const def = settings?.default_research_hub_id?.trim();
    const pick = def && hubs.some((h) => h.id === def) ? def : hubs[0].id;
    setSelectedHubId(pick);
  }, [hubs, settings, searchParams, selectedHubId, setSelectedHubId]);

  const activeHub = useMemo(() => hubs.find((h) => h.id === selectedHubId) ?? null, [hubs, selectedHubId]);

  const hubContext = useMemo(
    () => ({
      hubs,
      settings,
      selectedHubId,
      setSelectedHubId,
      loading: loadingHubs,
      loadError: hubLoadError,
      reload: reloadHubs,
    }),
    [hubs, settings, selectedHubId, setSelectedHubId, loadingHubs, hubLoadError, reloadHubs],
  );

  const runInvoke = useCallback(
    async (c: ArmsClient, hubId: string, method: string, path: string, json_body?: unknown) => {
      return c.postResearchHubInvoke(hubId, { method, path, json_body });
    },
    [],
  );

  const fetchPipelineStatus = useCallback(async () => {
    if (!selectedHubId) return;
    setStatusErr(null);
    setStatusLoading(true);
    try {
      const res = await runInvoke(client, selectedHubId, 'GET', '/api/pipeline/status');
      if (res.status >= 200 && res.status < 300) {
        setStatusJson(parseInvokeJson(res));
      } else {
        setStatusJson(null);
        setStatusErr(invokeErrorMessage(res, 'Pipeline status'));
      }
    } catch (e) {
      setStatusJson(null);
      setStatusErr(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Status request failed');
    } finally {
      setStatusLoading(false);
    }
  }, [client, runInvoke, selectedHubId]);

  const fetchStages = useCallback(async () => {
    if (!selectedHubId) return;
    try {
      const res = await runInvoke(client, selectedHubId, 'GET', '/api/pipeline/stages');
      if (res.status >= 200 && res.status < 300) {
        setStagesJson(parseInvokeJson(res));
      } else {
        setStagesJson(null);
      }
    } catch {
      setStagesJson(null);
    }
  }, [client, runInvoke, selectedHubId]);

  useEffect(() => {
    if (mainTab !== 'pipeline' || !selectedHubId) return;
    void fetchPipelineStatus();
    void fetchStages();
  }, [mainTab, selectedHubId, fetchPipelineStatus, fetchStages]);

  const pipelineState = useMemo(() => inferPipelineRunState(statusJson), [statusJson]);

  useEffect(() => {
    if (mainTab !== 'pipeline' || !selectedHubId) return;
    if (pipelineState !== 'running') return;
    const t = window.setInterval(() => {
      void fetchPipelineStatus();
    }, 4000);
    return () => window.clearInterval(t);
  }, [mainTab, selectedHubId, pipelineState, fetchPipelineStatus]);

  const loadOverview = useCallback(async () => {
    if (!selectedHubId) return;
    setOverviewErr(null);
    setOverviewBusy(true);
    try {
      const [h, v] = await Promise.all([
        runInvoke(client, selectedHubId, 'GET', '/api/health'),
        runInvoke(client, selectedHubId, 'GET', '/api/version'),
      ]);
      setHealthJson(h.status >= 200 && h.status < 300 ? parseInvokeJson(h) : null);
      setVersionJson(v.status >= 200 && v.status < 300 ? parseInvokeJson(v) : null);
      if (h.status < 200 || h.status >= 300 || v.status < 200 || v.status >= 300) {
        const parts: string[] = [];
        if (h.status < 200 || h.status >= 300) parts.push(`health ${h.status}`);
        if (v.status < 200 || v.status >= 300) parts.push(`version ${v.status}`);
        setOverviewErr(parts.join('; '));
      }
    } catch (e) {
      setOverviewErr(e instanceof Error ? e.message : 'Overview request failed');
    } finally {
      setOverviewBusy(false);
    }
  }, [client, runInvoke, selectedHubId]);

  useEffect(() => {
    if (mainTab !== 'overview' || !selectedHubId) return;
    void loadOverview();
  }, [mainTab, selectedHubId, loadOverview]);

  const fetchRuns = useCallback(async () => {
    if (!selectedHubId) return;
    setRunsErr(null);
    setRunsLoading(true);
    try {
      const res = await runInvoke(client, selectedHubId, 'GET', '/api/runs');
      if (res.status >= 200 && res.status < 300) {
        setRunsJson(parseInvokeJson(res));
      } else {
        setRunsJson(null);
        setRunsErr(invokeErrorMessage(res, 'List runs'));
      }
    } catch (e) {
      setRunsJson(null);
      setRunsErr(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Runs request failed');
    } finally {
      setRunsLoading(false);
    }
  }, [client, runInvoke, selectedHubId]);

  useEffect(() => {
    if (mainTab !== 'runs' || !selectedHubId) return;
    void fetchRuns();
  }, [mainTab, selectedHubId, fetchRuns]);

  async function fetchRunDetail() {
    const rid = runIdInput.trim();
    if (!selectedHubId || !rid) {
      setRunDetailErr('Enter a run id.');
      return;
    }
    setRunDetailErr(null);
    setRunDetailLoading(true);
    try {
      const path = `/api/runs/${encodeURIComponent(rid)}`;
      const res = await runInvoke(client, selectedHubId, 'GET', path);
      if (res.status >= 200 && res.status < 300) {
        setRunDetailJson(parseInvokeJson(res));
      } else {
        setRunDetailJson(null);
        setRunDetailErr(invokeErrorMessage(res, 'Run detail'));
      }
    } catch (e) {
      setRunDetailJson(null);
      setRunDetailErr(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Run detail failed');
    } finally {
      setRunDetailLoading(false);
    }
  }

  async function startPipeline() {
    if (!selectedHubId) return;
    setPipelineAction('start');
    setStatusErr(null);
    try {
      const res = await runInvoke(client, selectedHubId, 'POST', '/api/pipeline/start', {
        topic: null,
        config_overrides: null,
        auto_approve: true,
      });
      if (res.status < 200 || res.status >= 300) {
        setStatusErr(invokeErrorMessage(res, 'Start pipeline'));
      }
      await fetchPipelineStatus();
    } catch (e) {
      setStatusErr(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Start failed');
    } finally {
      setPipelineAction(null);
    }
  }

  async function stopPipeline() {
    if (!selectedHubId) return;
    setPipelineAction('stop');
    setStatusErr(null);
    try {
      const res = await runInvoke(client, selectedHubId, 'POST', '/api/pipeline/stop', {});
      if (res.status < 200 || res.status >= 300) {
        setStatusErr(invokeErrorMessage(res, 'Stop pipeline'));
      }
      await fetchPipelineStatus();
    } catch (e) {
      setStatusErr(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Stop failed');
    } finally {
      setPipelineAction(null);
    }
  }

  const startDisabled = pipelineAction != null || statusLoading || pipelineState === 'running';
  const stopDisabled = pipelineAction != null || statusLoading || pipelineState === 'idle';

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem 1rem', overflow: 'auto' }}>
      <div style={{ maxWidth: '56rem', margin: '0 auto', width: '100%', display: 'flex', flexDirection: 'column', gap: '1rem' }}>
        <header>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.55rem' }}>
              <FlaskConical size={22} className="ft-muted" aria-hidden />
              <div>
                <h1 style={{ fontSize: '1.2rem', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>Research hub</h1>
                <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.8rem', lineHeight: 1.45, maxWidth: '40rem' }}>
                  ResearchClaw via arms — allowlisted REST only. Configure hubs on{' '}
                  {pid ? (
                    <Link to={`/p/${encodeURIComponent(pid)}/system`}>System</Link>
                  ) : (
                    'System'
                  )}
                  . WebSockets are not proxied here.
                </p>
              </div>
            </div>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.75rem' }}
              disabled={loadingHubs}
              onClick={() => void reloadHubs()}
            >
              <RefreshCw size={14} className={loadingHubs ? 'ft-spin' : ''} aria-hidden />
              Hubs
            </button>
          </div>

          {!loadingHubs && hubs.length > 0 ? (
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.65rem', alignItems: 'center', marginTop: '0.85rem' }}>
              <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.4rem', fontSize: '0.78rem' }}>
                <span className="ft-muted">Hub</span>
                <select
                  className="ft-input"
                  style={{ fontSize: '0.75rem', minWidth: 'min(100%, 18rem)' }}
                  value={selectedHubId ?? ''}
                  onChange={(ev) => setSelectedHubId(ev.target.value)}
                >
                  {hubs.map((h) => (
                    <option key={h.id} value={h.id}>
                      {h.display_name || h.id} — {h.base_url}
                    </option>
                  ))}
                </select>
              </label>
              {activeHub ? (
                <span className="ft-muted" style={{ fontSize: '0.72rem', wordBreak: 'break-all' }}>
                  <code className="ft-mono">{activeHub.base_url}</code>
                  {activeHub.has_api_key ? ' · key stored' : ' · no API key'}
                </span>
              ) : null}
            </div>
          ) : null}
        </header>

        {hubLoadError ? (
          <p className="ft-banner ft-banner--error" role="alert" style={{ margin: 0 }}>
            {hubLoadError}
          </p>
        ) : null}

        {!loadingHubs && hubs.length === 0 ? (
          <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
            No hub configured.{' '}
            {pid ? (
              <Link to={`/p/${encodeURIComponent(pid)}/system/research`}>Add one on System</Link>
            ) : (
              'Add a ResearchClaw hub under System.'
            )}
          </p>
        ) : null}

        {selectedHubId && hubs.length > 0 ? (
          <>
            <nav className="ft-border-b" style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem', paddingBottom: '0.5rem' }} aria-label="Research hub sections">
              <div className="ft-tabs" style={{ flexWrap: 'wrap' }}>
                {(
                  [
                    { id: 'overview' as const, label: 'Overview', icon: Activity },
                    { id: 'pipeline' as const, label: 'Pipeline', icon: GitBranch },
                    { id: 'runs' as const, label: 'Runs', icon: Layers },
                    { id: 'explorer' as const, label: 'API explorer', icon: Terminal },
                  ] as const
                ).map(({ id, label, icon: Icon }) => (
                  <button
                    key={id}
                    type="button"
                    className={`ft-tab ${mainTab === id ? 'ft-tab--active' : ''}`}
                    onClick={() => setMainTab(id)}
                  >
                    <Icon size={14} style={{ marginRight: 6, verticalAlign: 'middle', opacity: 0.85 }} aria-hidden />
                    {label}
                  </button>
                ))}
              </div>
            </nav>

            {mainTab === 'overview' ? (
              <section style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                  <button
                    type="button"
                    className="ft-btn-ghost"
                    style={{ fontSize: '0.75rem' }}
                    disabled={overviewBusy}
                    onClick={() => void loadOverview()}
                  >
                    <RefreshCw size={14} className={overviewBusy ? 'ft-spin' : ''} aria-hidden />
                    Refresh
                  </button>
                  {overviewErr ? (
                    <span className="ft-muted" style={{ fontSize: '0.75rem' }}>
                      {overviewErr}
                    </span>
                  ) : null}
                </div>
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(14rem, 1fr))',
                    gap: '0.65rem',
                  }}
                >
                  <div
                    className="ft-mc-stat-pill"
                    style={{ flexDirection: 'column', alignItems: 'stretch', textAlign: 'left', padding: '0.75rem' }}
                  >
                    <span className="ft-mc-stat-pill-label" style={{ marginBottom: '0.35rem' }}>
                      GET /api/health
                    </span>
                    <pre
                      style={{
                        margin: 0,
                        fontSize: '0.68rem',
                        overflow: 'auto',
                        maxHeight: '12rem',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontFamily: 'ui-monospace, monospace',
                      }}
                    >
                      {healthJson !== null ? JSON.stringify(healthJson, null, 2) : overviewBusy ? '…' : '—'}
                    </pre>
                  </div>
                  <div
                    className="ft-mc-stat-pill ft-mc-stat-pill--blue"
                    style={{ flexDirection: 'column', alignItems: 'stretch', textAlign: 'left', padding: '0.75rem' }}
                  >
                    <span className="ft-mc-stat-pill-label" style={{ marginBottom: '0.35rem' }}>
                      GET /api/version
                    </span>
                    <pre
                      style={{
                        margin: 0,
                        fontSize: '0.68rem',
                        overflow: 'auto',
                        maxHeight: '12rem',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-word',
                        fontFamily: 'ui-monospace, monospace',
                      }}
                    >
                      {versionJson !== null ? JSON.stringify(versionJson, null, 2) : overviewBusy ? '…' : '—'}
                    </pre>
                  </div>
                </div>
              </section>
            ) : null}

            {mainTab === 'pipeline' ? (
              <section style={{ display: 'flex', flexDirection: 'column', gap: '0.85rem' }}>
                <div
                  style={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    alignItems: 'center',
                    gap: '0.5rem',
                    justifyContent: 'space-between',
                  }}
                >
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                    <span
                      className={`ft-feed-status ${pipelineState === 'running' ? 'ft-feed-status--live' : 'ft-feed-status--off'}`}
                      title="Inferred from GET /api/pipeline/status"
                    >
                      {statusLoading ? 'Checking…' : pipelineState === 'running' ? 'Pipeline running' : pipelineState === 'idle' ? 'Pipeline idle' : 'State unknown'}
                    </span>
                    <button
                      type="button"
                      className="ft-btn-ghost"
                      style={{ fontSize: '0.75rem' }}
                      disabled={statusLoading}
                      onClick={() => void fetchPipelineStatus()}
                    >
                      <RefreshCw size={14} className={statusLoading ? 'ft-spin' : ''} aria-hidden />
                      Status
                    </button>
                  </div>
                  <div style={{ display: 'flex', gap: '0.45rem', flexWrap: 'wrap' }}>
                    <button
                      type="button"
                      className="ft-btn-primary"
                      style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
                      disabled={startDisabled}
                      onClick={() => void startPipeline()}
                      title={pipelineState === 'running' ? 'Pipeline already running' : 'POST /api/pipeline/start'}
                    >
                      <Play size={14} aria-hidden />
                      {pipelineAction === 'start' ? 'Starting…' : 'Start'}
                    </button>
                    <button
                      type="button"
                      className="ft-btn-ghost"
                      style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
                      disabled={stopDisabled}
                      onClick={() => void stopPipeline()}
                      title={pipelineState === 'idle' ? 'Nothing to stop' : 'POST /api/pipeline/stop'}
                    >
                      <Square size={14} aria-hidden />
                      {pipelineAction === 'stop' ? 'Stopping…' : 'Stop'}
                    </button>
                  </div>
                </div>
                {pipelineState === 'unknown' ? (
                  <p className="ft-muted" style={{ margin: 0, fontSize: '0.75rem', lineHeight: 1.45 }}>
                    Status shape may differ by ResearchClaw build — Start/Stop stay available; use raw JSON below to confirm{' '}
                    <code className="ft-mono">pipeline_running</code>, <code className="ft-mono">running</code>, or <code className="ft-mono">status</code>.
                  </p>
                ) : null}
                {statusErr ? (
                  <p className="ft-banner ft-banner--error" role="alert" style={{ margin: 0, fontSize: '0.8rem' }}>
                    {statusErr}
                  </p>
                ) : null}
                <div
                  style={{
                    borderRadius: 'var(--ft-radius-sm)',
                    border: '1px solid var(--mc-border)',
                    background: 'var(--mc-bg-secondary)',
                    padding: '0.75rem',
                  }}
                >
                  <div className="ft-muted" style={{ fontSize: '0.7rem', marginBottom: '0.35rem', fontWeight: 600 }}>
                    GET /api/pipeline/status
                  </div>
                  <pre
                    style={{
                      margin: 0,
                      fontSize: '0.68rem',
                      overflow: 'auto',
                      maxHeight: '14rem',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontFamily: 'ui-monospace, monospace',
                    }}
                  >
                    {statusJson !== null ? JSON.stringify(statusJson, null, 2) : statusLoading ? '…' : '—'}
                  </pre>
                </div>
                <details style={{ fontSize: '0.78rem' }}>
                  <summary style={{ cursor: 'pointer', fontWeight: 600 }}>Pipeline stages (GET /api/pipeline/stages)</summary>
                  <pre
                    style={{
                      margin: '0.5rem 0 0',
                      padding: '0.5rem',
                      fontSize: '0.65rem',
                      overflow: 'auto',
                      maxHeight: '16rem',
                      borderRadius: 'var(--ft-radius-sm)',
                      border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
                      background: 'var(--mc-bg-tertiary)',
                      fontFamily: 'ui-monospace, monospace',
                    }}
                  >
                    {stagesJson !== null ? JSON.stringify(stagesJson, null, 2) : '—'}
                  </pre>
                </details>
              </section>
            ) : null}

            {mainTab === 'runs' ? (
              <section style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
                  <button
                    type="button"
                    className="ft-btn-ghost"
                    style={{ fontSize: '0.75rem' }}
                    disabled={runsLoading}
                    onClick={() => void fetchRuns()}
                  >
                    <RefreshCw size={14} className={runsLoading ? 'ft-spin' : ''} aria-hidden />
                    Refresh list
                  </button>
                  {runsErr ? (
                    <span className="ft-banner ft-banner--error" style={{ margin: 0, fontSize: '0.78rem' }}>
                      {runsErr}
                    </span>
                  ) : null}
                </div>
                <pre
                  style={{
                    margin: 0,
                    padding: '0.65rem',
                    fontSize: '0.68rem',
                    overflow: 'auto',
                    maxHeight: '18rem',
                    borderRadius: 'var(--ft-radius-sm)',
                    border: '1px solid var(--mc-border)',
                    background: 'var(--mc-bg-secondary)',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    fontFamily: 'ui-monospace, monospace',
                  }}
                >
                  {runsJson !== null ? JSON.stringify(runsJson, null, 2) : runsLoading ? '…' : '—'}
                </pre>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
                  <input
                    className="ft-input"
                    style={{ fontSize: '0.75rem', flex: '1 1 12rem', minWidth: 0 }}
                    placeholder="run id"
                    value={runIdInput}
                    onChange={(e) => setRunIdInput(e.target.value)}
                  />
                  <button
                    type="button"
                    className="ft-btn-primary"
                    style={{ fontSize: '0.75rem' }}
                    disabled={runDetailLoading}
                    onClick={() => void fetchRunDetail()}
                  >
                    {runDetailLoading ? 'Loading…' : 'Load run'}
                  </button>
                </div>
                {runDetailErr ? (
                  <p className="ft-banner ft-banner--error" role="alert" style={{ margin: 0, fontSize: '0.8rem' }}>
                    {runDetailErr}
                  </p>
                ) : null}
                {runDetailJson != null ? (
                  <pre
                    style={{
                      margin: 0,
                      padding: '0.65rem',
                      fontSize: '0.68rem',
                      overflow: 'auto',
                      maxHeight: '16rem',
                      borderRadius: 'var(--ft-radius-sm)',
                      border: '1px solid var(--mc-border)',
                      background: 'var(--mc-bg-tertiary)',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      fontFamily: 'ui-monospace, monospace',
                    }}
                  >
                    {JSON.stringify(runDetailJson, null, 2)}
                  </pre>
                ) : null}
              </section>
            ) : null}

            {mainTab === 'explorer' ? (
              <ResearchClawHubExplorer
                client={client}
                hubContext={hubContext}
                variant="embedded"
                showHubSelector={false}
              />
            ) : null}
          </>
        ) : null}
      </div>
    </div>
  );
}
