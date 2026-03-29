import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { Send } from 'lucide-react';
import { ArmsHttpError, type ArmsClient } from '../../api/armsClient';
import type { ApiResearchHub, ApiResearchHubInvokeResult, ApiResearchSystemSettings } from '../../api/armsTypes';
import {
  RESEARCH_CLAW_OPENAPI_SPEC_TITLE,
  RESEARCH_CLAW_OPENAPI_SPEC_VERSION,
  RESEARCH_CLAW_OPERATIONS,
  type ResearchClawCatalogOp,
  type ResearchClawOpTag,
  buildResearchClawPath,
  groupResearchClawOpsByTag,
  researchClawTagLabel,
} from '../../lib/researchClawOpenapiCatalog';

export type ResearchHubExplorerHubContext = {
  hubs: ApiResearchHub[];
  settings: ApiResearchSystemSettings | null;
  selectedHubId: string | null;
  setSelectedHubId: (id: string) => void;
  loading: boolean;
  loadError: string | null;
  reload: () => Promise<void>;
};

type Props = {
  client: ArmsClient;
  /** When set, hub list and selection are owned by the parent (no duplicate fetch). */
  hubContext?: ResearchHubExplorerHubContext;
  /** `embedded` hides the OpenAPI blurb (parent page provides context). */
  variant?: 'default' | 'embedded';
  /** When false, hub dropdown is omitted (parent renders hub selection). */
  showHubSelector?: boolean;
};

export function ResearchClawHubExplorer({ client, hubContext, variant = 'default', showHubSelector = true }: Props) {
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const [searchParams, setSearchParams] = useSearchParams();
  const [internalHubs, setInternalHubs] = useState<ApiResearchHub[]>([]);
  const [internalSettings, setInternalSettings] = useState<ApiResearchSystemSettings | null>(null);
  const [internalLoading, setInternalLoading] = useState(!hubContext);
  const [internalLoadError, setInternalLoadError] = useState<string | null>(null);
  const [internalSelectedHubId, setInternalSelectedHubId] = useState<string | null>(null);

  const hubs = hubContext?.hubs ?? internalHubs;
  const settings = hubContext?.settings ?? internalSettings;
  const loading = hubContext?.loading ?? internalLoading;
  const loadError = hubContext?.loadError ?? internalLoadError;
  const selectedHubId = hubContext?.selectedHubId ?? internalSelectedHubId;
  const setSelectedHubId = hubContext?.setSelectedHubId ?? setInternalSelectedHubId;
  const [tag, setTag] = useState<ResearchClawOpTag>('core');
  const [selectedOpId, setSelectedOpId] = useState<string>(RESEARCH_CLAW_OPERATIONS[0].id);
  const [pathValues, setPathValues] = useState<Record<string, string>>({});
  const [bodyText, setBodyText] = useState('');
  const [invoking, setInvoking] = useState(false);
  const [invokeResult, setInvokeResult] = useState<ApiResearchHubInvokeResult | null>(null);
  const [invokeError, setInvokeError] = useState<string | null>(null);

  const byTag = useMemo(() => groupResearchClawOpsByTag(), []);
  const tagOrder = useMemo(() => (['core', 'pipeline', 'projects', 'voice'] as const), []);

  const load = useCallback(async () => {
    if (hubContext) {
      await hubContext.reload();
      return;
    }
    setInternalLoadError(null);
    setInternalLoading(true);
    try {
      const [list, st] = await Promise.all([
        client.listResearchHubs(),
        client.getResearchSystemSettings().catch(() => null),
      ]);
      setInternalHubs(list);
      setInternalSettings(st);
    } catch (e) {
      setInternalLoadError(e instanceof Error ? e.message : 'Could not load hubs');
      setInternalHubs([]);
      setInternalSettings(null);
    } finally {
      setInternalLoading(false);
    }
  }, [client, hubContext]);

  useEffect(() => {
    if (hubContext) return;
    void load();
  }, [hubContext, load]);

  useEffect(() => {
    if (hubContext) return;
    if (hubs.length === 0 || selectedHubId) return;
    const q = searchParams.get('hub');
    if (q && hubs.some((h) => h.id === q)) {
      setSelectedHubId(q);
      return;
    }
    const def = settings?.default_research_hub_id?.trim();
    const pick = def && hubs.some((h) => h.id === def) ? def : hubs[0].id;
    setSelectedHubId(pick);
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set('hub', pick);
        return next;
      },
      { replace: true },
    );
  }, [hubContext, hubs, settings, searchParams, selectedHubId, setSearchParams]);

  const selectedOp = useMemo((): ResearchClawCatalogOp | null => {
    return RESEARCH_CLAW_OPERATIONS.find((o) => o.id === selectedOpId) ?? RESEARCH_CLAW_OPERATIONS[0];
  }, [selectedOpId]);

  useEffect(() => {
    if (!selectedOp) return;
    if (!selectedOp.pathParams?.length) {
      setPathValues({});
      return;
    }
    setPathValues((prev) => {
      const next: Record<string, string> = {};
      for (const p of selectedOp.pathParams!) {
        next[p.name] = prev[p.name] ?? '';
      }
      return next;
    });
  }, [selectedOp?.id]);

  useEffect(() => {
    if (!selectedOp) return;
    if (selectedOp.method === 'POST' && selectedOp.invokeSupported && selectedOp.bodyJsonExample !== undefined) {
      setBodyText(JSON.stringify(selectedOp.bodyJsonExample, null, 2));
    } else {
      setBodyText('');
    }
  }, [selectedOp?.id]);

  useEffect(() => {
    const grp = byTag[tag];
    if (!grp?.length) return;
    if (!grp.some((o) => o.id === selectedOpId)) {
      setSelectedOpId(grp[0].id);
    }
  }, [tag, byTag, selectedOpId]);

  const activeHub = useMemo(() => hubs.find((h) => h.id === selectedHubId) ?? null, [hubs, selectedHubId]);

  async function runInvoke() {
    if (!selectedOp || !selectedHubId || !selectedOp.invokeSupported) return;
    setInvokeError(null);
    setInvokeResult(null);
    let builtPath: string;
    try {
      builtPath = buildResearchClawPath(selectedOp, pathValues);
    } catch {
      setInvokeError('Fill all path parameters.');
      return;
    }
    if (selectedOp.pathParams?.some((p) => !(pathValues[p.name] ?? '').trim())) {
      setInvokeError('Fill all path parameters.');
      return;
    }
    const payload: { method: string; path: string; json_body?: unknown } = {
      method: selectedOp.method,
      path: builtPath,
    };
    if (selectedOp.method === 'POST') {
      try {
        const trimmed = bodyText.trim();
        payload.json_body = trimmed ? JSON.parse(trimmed) : {};
      } catch {
        setInvokeError('Request body must be valid JSON.');
        return;
      }
    }
    setInvoking(true);
    try {
      const res = await client.postResearchHubInvoke(selectedHubId, payload);
      setInvokeResult(res);
    } catch (e) {
      setInvokeError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Invoke failed.');
    } finally {
      setInvoking(false);
    }
  }

  const opsInTag = byTag[tag] ?? [];

  const shellStyle =
    variant === 'embedded'
      ? { marginTop: 0, padding: 0, border: 'none', background: 'transparent' as const }
      : {
          marginTop: '0.65rem',
          padding: '1rem',
          borderRadius: 'var(--ft-radius-sm)',
          border: '1px solid var(--mc-border)',
          background: 'var(--mc-bg-secondary)',
        };

  return (
    <section style={shellStyle}>
      {variant === 'default' ? (
        <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.78rem', lineHeight: 1.5 }}>
          {RESEARCH_CLAW_OPENAPI_SPEC_TITLE} <code className="ft-mono">v{RESEARCH_CLAW_OPENAPI_SPEC_VERSION}</code> — allowlisted REST only. WebSockets{' '}
          <code className="ft-mono">/ws/events</code>, <code className="ft-mono">/ws/chat</code> are not proxied here.
        </p>
      ) : null}

      {loading ? <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>Loading hubs…</p> : null}
      {loadError ? (
        <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.5rem 0' }}>
          {loadError}
        </p>
      ) : null}

      {!loading && hubs.length === 0 ? (
        <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
          No hub configured.{' '}
          {pid ? (
            <Link to={`/p/${encodeURIComponent(pid)}/system#ft-research-claw-hubs`}>Add one on System</Link>
          ) : (
            'Add a ResearchClaw hub under System.'
          )}
        </p>
      ) : null}

      {showHubSelector && hubs.length > 0 ? (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem', alignItems: 'center', marginBottom: '0.85rem' }}>
          <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.4rem', fontSize: '0.78rem' }}>
            <span className="ft-muted">Hub</span>
            <select
              className="ft-input"
              style={{ fontSize: '0.75rem', minWidth: '14rem' }}
              value={selectedHubId ?? ''}
              onChange={(ev) => {
                const id = ev.target.value;
                setSelectedHubId(id);
                setSearchParams(
                  (prev) => {
                    const next = new URLSearchParams(prev);
                    next.set('hub', id);
                    return next;
                  },
                  { replace: true },
                );
              }}
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

      {selectedHubId && hubs.length > 0 ? (
        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            gap: '0.75rem',
            alignItems: 'flex-start',
          }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem', flex: '1 1 11rem', minWidth: 0, maxWidth: '16rem' }}>
            <div className="ft-tabs" style={{ flexWrap: 'wrap' }} role="tablist" aria-label="ResearchClaw API groups">
              {tagOrder.map((t) => (
                <button
                  key={t}
                  type="button"
                  role="tab"
                  aria-selected={t === tag}
                  className={`ft-tab ${t === tag ? 'ft-tab--active' : ''}`}
                  onClick={() => setTag(t)}
                >
                  {researchClawTagLabel(t)}
                </button>
              ))}
            </div>
            <div
              style={{
                borderTop: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
                paddingTop: '0.45rem',
              }}
            >
              <span className="ft-muted" style={{ fontSize: '0.65rem', letterSpacing: '0.04em', fontWeight: 600 }}>
                Operations
              </span>
              <ul style={{ listStyle: 'none', margin: '0.35rem 0 0', padding: 0, display: 'flex', flexDirection: 'column', gap: '0.2rem' }}>
                {opsInTag.map((op) => (
                  <li key={op.id}>
                    <button
                      type="button"
                      className={op.id === selectedOpId ? 'ft-btn-primary' : 'ft-btn-ghost'}
                      style={{
                        fontSize: '0.68rem',
                        width: '100%',
                        justifyContent: 'flex-start',
                        fontWeight: op.id === selectedOpId ? 600 : 400,
                        lineHeight: 1.35,
                      }}
                      onClick={() => setSelectedOpId(op.id)}
                    >
                      <span className="ft-mono" style={{ marginRight: '0.35rem' }}>
                        {op.method}
                      </span>
                      {op.summary}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          </div>

          {selectedOp ? (
            <div
              style={{
                padding: '0.75rem',
                borderRadius: 'var(--ft-radius-sm)',
                border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
                background: 'var(--mc-bg-tertiary)',
                minWidth: 0,
                flex: '3 1 18rem',
              }}
            >
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'baseline', marginBottom: '0.35rem' }}>
                <span className="ft-mono" style={{ fontSize: '0.78rem', fontWeight: 700 }}>
                  {selectedOp.method}
                </span>
                <code className="ft-mono" style={{ fontSize: '0.72rem', wordBreak: 'break-all' }}>
                  {selectedOp.path}
                </code>
              </div>
              <p style={{ margin: '0 0 0.35rem', fontSize: '0.82rem', fontWeight: 600 }}>{selectedOp.summary}</p>
              {selectedOp.description ? (
                <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.78rem', lineHeight: 1.5 }}>
                  {selectedOp.description}
                </p>
              ) : null}

              {!selectedOp.invokeSupported ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.78rem' }}>
                  {selectedOp.invokeNote ?? 'Not available via this explorer.'}
                </p>
              ) : (
                <>
                  {selectedOp.pathParams?.map((p) => (
                    <label key={p.name} style={{ display: 'block', marginBottom: '0.5rem', fontSize: '0.75rem' }}>
                      <span className="ft-muted">{p.name}</span>
                      {p.description ? (
                        <span className="ft-muted" style={{ marginLeft: '0.35rem' }}>
                          — {p.description}
                        </span>
                      ) : null}
                      <input
                        className="ft-input"
                        style={{ display: 'block', width: '100%', marginTop: '0.25rem', fontSize: '0.75rem' }}
                        value={pathValues[p.name] ?? ''}
                        onChange={(ev) => setPathValues((m) => ({ ...m, [p.name]: ev.target.value }))}
                        placeholder={p.name}
                      />
                    </label>
                  ))}

                  {selectedOp.method === 'POST' ? (
                    <label style={{ display: 'block', marginBottom: '0.5rem', fontSize: '0.75rem' }}>
                      <span className="ft-muted">JSON body</span>
                      <textarea
                        className="ft-input"
                        style={{
                          display: 'block',
                          width: '100%',
                          marginTop: '0.25rem',
                          fontSize: '0.72rem',
                          fontFamily: 'ui-monospace, monospace',
                          minHeight: '7rem',
                          resize: 'vertical',
                        }}
                        value={bodyText}
                        onChange={(ev) => setBodyText(ev.target.value)}
                        spellCheck={false}
                      />
                    </label>
                  ) : null}

                  <button
                    type="button"
                    className="ft-btn-primary"
                    style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
                    disabled={invoking}
                    onClick={() => void runInvoke()}
                  >
                    <Send size={14} aria-hidden />
                    {invoking ? 'Sending…' : 'Send via arms'}
                  </button>

                  {invokeError ? (
                    <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.65rem 0 0', fontSize: '0.78rem' }}>
                      {invokeError}
                    </p>
                  ) : null}

                  {invokeResult ? (
                    <div style={{ marginTop: '0.65rem' }}>
                      <div className="ft-muted" style={{ fontSize: '0.7rem', marginBottom: '0.25rem' }}>
                        Hub HTTP status: <strong style={{ color: 'var(--mc-text-primary)' }}>{invokeResult.status}</strong>
                      </div>
                      <pre
                        style={{
                          margin: 0,
                          padding: '0.5rem',
                          fontSize: '0.68rem',
                          overflow: 'auto',
                          maxHeight: '18rem',
                          borderRadius: 'var(--ft-radius-sm)',
                          border: '1px solid var(--mc-border)',
                          background: 'var(--mc-bg-secondary)',
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                        }}
                      >
                        {invokeResult.json !== undefined
                          ? JSON.stringify(invokeResult.json, null, 2)
                          : invokeResult.body || '—'}
                      </pre>
                    </div>
                  ) : null}
                </>
              )}
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
