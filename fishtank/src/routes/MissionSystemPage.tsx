import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  Activity,
  Copy,
  Cpu,
  ExternalLink,
  HardDrive,
  Lightbulb,
  MemoryStick,
  Network,
  Radio,
  RefreshCw,
  Server,
  Settings,
} from 'lucide-react';
import { ArmsClient, ArmsHttpError, buildLiveEventsUrl, buildLiveEventsUrlTemplate } from '../api/armsClient';
import type { ApiGatewayEndpoint, ApiHostMetrics, ApiVersion, PatchGatewayEndpointBody } from '../api/armsTypes';
import { ThemeCycleButton } from '../components/shell/ThemeCycleButton';
import { useMissionUi } from '../context/MissionUiContext';
import { IDEATION_BUCKETS, IDEATION_SOP_NUMBERS, type IdeationBucketValue } from '../lib/ideaCategories';
import {
  clearIdeationBucketPrefs,
  initialSelectedSetFromStorage,
  writeIdeationBucketPrefs,
} from '../lib/ideationBucketPreferences';
import { IDEATION_SOPS } from '../lib/ideationSops';

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

function IdeationBucketsSettingsPanel() {
  const [selected, setSelected] = useState<Set<IdeationBucketValue>>(() => initialSelectedSetFromStorage());

  function persist(next: Set<IdeationBucketValue>) {
    const slugs = IDEATION_BUCKETS.filter((b) => next.has(b.value)).map((b) => b.value);
    if (slugs.length === 0) return;
    setSelected(next);
    writeIdeationBucketPrefs({ v: 1, selectedSlugs: slugs });
  }

  function toggle(v: IdeationBucketValue) {
    const next = new Set(selected);
    if (next.has(v)) {
      if (next.size <= 1) return;
      next.delete(v);
    } else {
      next.add(v);
    }
    persist(next);
  }

  function selectAll() {
    persist(new Set(IDEATION_BUCKETS.map((b) => b.value)));
  }

  function resetPrefs() {
    clearIdeationBucketPrefs();
    setSelected(new Set(IDEATION_BUCKETS.map((b) => b.value)));
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
        <Lightbulb size={18} className="ft-muted" aria-hidden />
        <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
          Ideation buckets
        </h2>
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        Each item maps to one of the four SOPs. Mark buckets <strong>Active</strong> (shown on Ideation) or{' '}
        <strong>Inactive</strong> (hidden). On narrow screens you get a list with Active/Inactive buttons; on desktop,
        buckets are pill tags — click a tag to toggle (highlighted = active). At least one must stay active. Stored in
        this browser (<code className="ft-mono">localStorage</code>); arms still receives the slug as{' '}
        <code className="ft-mono">category</code>.
      </p>

      {IDEATION_SOP_NUMBERS.map((sopNum) => {
        const sopMeta = IDEATION_SOPS[sopNum - 1];
        const rows = IDEATION_BUCKETS.filter((b) => b.sop === sopNum);
        return (
          <div key={sopNum} style={{ marginBottom: '0.9rem' }}>
            <div className="ft-ideation-bucket-group__head" style={{ marginBottom: '0.4rem' }}>
              SOP {sopNum} — {sopMeta?.shortTitle ?? '—'}
            </div>

            <div className="ft-sys-ideation-bucket-settings--mobile">
              <ul style={{ margin: 0, padding: 0, listStyle: 'none' }}>
                {rows.map((b) => (
                  <li
                    key={b.value}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: '0.6rem',
                      flexWrap: 'wrap',
                      padding: '0.4rem 0',
                      borderBottom: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.07))',
                    }}
                  >
                    <span style={{ fontSize: '0.78rem', lineHeight: 1.4, flex: '1 1 14rem', minWidth: 0 }}>{b.label}</span>
                    <button
                      type="button"
                      className={selected.has(b.value) ? 'ft-btn-primary' : 'ft-btn-ghost'}
                      style={{ fontSize: '0.72rem', padding: '0.28rem 0.6rem', flexShrink: 0 }}
                      onClick={() => toggle(b.value)}
                    >
                      {selected.has(b.value) ? 'Active' : 'Inactive'}
                    </button>
                  </li>
                ))}
              </ul>
            </div>

            <div className="ft-sys-ideation-bucket-settings--desktop">
              <div className="ft-ideation-bucket-tags" role="group" aria-label={`SOP ${sopNum} buckets`}>
                {rows.map((b) => {
                  const on = selected.has(b.value);
                  return (
                    <button
                      key={b.value}
                      type="button"
                      className={`ft-ideation-bucket-tag${on ? ' ft-ideation-bucket-tag--on' : ''}`}
                      aria-pressed={on}
                      title={on ? 'Active — click to hide on Ideation' : 'Inactive — click to show on Ideation'}
                      onClick={() => toggle(b.value)}
                    >
                      {b.label}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        );
      })}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginTop: '0.5rem' }}>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} onClick={selectAll}>
          Activate all
        </button>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} onClick={resetPrefs}>
          Reset to defaults
        </button>
      </div>
    </section>
  );
}

const GATEWAY_DRIVER_OPTIONS = [
  ['stub', 'stub'],
  ['openclaw_ws', 'openclaw_ws'],
  ['nemoclaw_ws', 'nemoclaw_ws'],
  ['nullclaw_ws', 'nullclaw_ws'],
  ['nullclaw_a2a', 'nullclaw_a2a'],
  ['picoclaw_ws', 'picoclaw_ws'],
  ['zeroclaw_ws', 'zeroclaw_ws'],
  ['clawlet_ws', 'clawlet_ws'],
  ['ironclaw_ws', 'ironclaw_ws'],
  ['mimiclaw_ws', 'mimiclaw_ws'],
  ['zclaw_relay_http', 'zclaw_relay_http'],
  ['nanobot_cli', 'nanobot_cli'],
  ['inkos_cli', 'inkos_cli'],
  ['mistermorph_http', 'mistermorph_http'],
  ['copaw_http', 'copaw_http'],
  ['metaclaw_http', 'metaclaw_http'],
] as const;

function GatewayEndpointsPanel({ client, defaultProductId }: { client: ArmsClient; defaultProductId?: string }) {
  const [rows, setRows] = useState<ApiGatewayEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);

  const [cName, setCName] = useState('');
  const [cDriver, setCDriver] = useState<string>('openclaw_ws');
  const [cUrl, setCUrl] = useState('');
  const [cToken, setCToken] = useState('');
  const [cDevice, setCDevice] = useState('');
  const [cTimeout, setCTimeout] = useState('');
  const [cProduct, setCProduct] = useState(defaultProductId ?? '');

  const [editTarget, setEditTarget] = useState<ApiGatewayEndpoint | null>(null);
  const [eName, setEName] = useState('');
  const [eDriver, setEDriver] = useState('');
  const [eUrl, setEUrl] = useState('');
  const [eDevice, setEDevice] = useState('');
  const [eTimeout, setETimeout] = useState('');
  const [eProduct, setEProduct] = useState('');
  const [eNewToken, setENewToken] = useState('');
  const [eClearToken, setEClearToken] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      const list = await client.listGatewayEndpoints();
      setRows(list);
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setError(e.message);
      } else {
        setError(e instanceof Error ? e.message : 'Could not load gateway endpoints');
      }
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setCProduct(defaultProductId ?? '');
  }, [defaultProductId]);

  function openEdit(row: ApiGatewayEndpoint) {
    setEditTarget(row);
    setEName(row.display_name);
    setEDriver(row.driver);
    setEUrl(row.gateway_url);
    setEDevice(row.device_id);
    setETimeout(String(row.timeout_sec ?? 0));
    setEProduct(row.product_id ?? '');
    setENewToken('');
    setEClearToken(false);
    setNote(null);
  }

  function closeEdit() {
    setEditTarget(null);
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setNote(null);
    setError(null);
    const timeoutSec = parseInt(cTimeout.trim(), 10);
    try {
      await client.createGatewayEndpoint({
        display_name: cName.trim() || undefined,
        driver: cDriver,
        gateway_url: cUrl.trim() || undefined,
        gateway_token: cToken.trim() || undefined,
        device_id: cDevice.trim() || undefined,
        timeout_sec: Number.isFinite(timeoutSec) ? timeoutSec : undefined,
        product_id: cProduct.trim() || undefined,
      });
      setNote('Gateway endpoint created.');
      setCName('');
      setCUrl('');
      setCToken('');
      setCDevice('');
      setCTimeout('');
      await load();
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Create failed.');
    } finally {
      setBusy(false);
    }
  }

  async function onSaveEdit(e: FormEvent) {
    e.preventDefault();
    if (!editTarget) return;
    setBusy(true);
    setNote(null);
    setError(null);
    const ts = parseInt(eTimeout.trim(), 10);
    const timeoutSec = Number.isFinite(ts) ? ts : editTarget.timeout_sec;
    const body: PatchGatewayEndpointBody = {};
    if (eName.trim() !== editTarget.display_name) body.display_name = eName.trim();
    if (eDriver !== editTarget.driver) body.driver = eDriver;
    if (eUrl.trim() !== editTarget.gateway_url) body.gateway_url = eUrl.trim();
    if (eDevice.trim() !== editTarget.device_id) body.device_id = eDevice.trim();
    if (timeoutSec !== editTarget.timeout_sec) body.timeout_sec = timeoutSec;
    const prevPid = editTarget.product_id ?? '';
    if (eProduct.trim() !== prevPid) body.product_id = eProduct.trim();
    if (eClearToken) body.gateway_token = '';
    else if (eNewToken.trim() !== '') body.gateway_token = eNewToken.trim();
    try {
      if (Object.keys(body).length === 0) {
        setNote('No changes to save.');
      } else {
        await client.patchGatewayEndpoint(editTarget.id, body);
        setNote('Saved.');
        closeEdit();
        await load();
      }
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Update failed.');
    } finally {
      setBusy(false);
    }
  }

  async function onDelete(id: string) {
    if (!window.confirm(`Delete gateway endpoint ${id}? This cannot be undone.`)) return;
    setBusy(true);
    setNote(null);
    setError(null);
    try {
      await client.deleteGatewayEndpoint(id);
      setNote('Deleted.');
      if (editTarget?.id === id) closeEdit();
      await load();
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Delete failed.');
    } finally {
      setBusy(false);
    }
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
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
          <Network size={18} className="ft-muted" aria-hidden />
          <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Gateway endpoints (database)
          </h2>
        </div>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem' }} disabled={loading || busy} onClick={() => void load()}>
          <RefreshCw size={14} className={loading ? 'ft-spin' : ''} aria-hidden />
          Refresh
        </button>
      </div>
      <p className="ft-muted" style={{ margin: '0.5rem 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        Manual dispatch profiles stored in arms (<code className="ft-mono">gateway_endpoints</code>). Register execution agents with{' '}
        <code className="ft-mono">gateway_endpoint_id</code> + <code className="ft-mono">session_key</code>. API responses never include stored tokens; use{' '}
        <code className="ft-mono">has_gateway_token</code> or set a new token here.
      </p>
      {loading && !rows.length ? <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>Loading…</p> : null}
      {error ? (
        <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.5rem 0' }}>
          {error}
        </p>
      ) : null}
      {note ? (
        <p className="ft-muted" style={{ margin: '0.5rem 0', fontSize: '0.8rem' }}>
          {note}
        </p>
      ) : null}

      <div style={{ overflowX: 'auto', marginBottom: '1rem' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.75rem' }}>
          <thead>
            <tr style={{ textAlign: 'left', borderBottom: '1px solid var(--mc-border)' }}>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem 0.35rem 0', fontWeight: 600 }}>
                Id
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Name
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Driver
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                URL
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Token
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Product
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0', fontWeight: 600 }}>
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id} style={{ borderBottom: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.06))' }}>
                <td style={{ padding: '0.4rem 0.5rem 0.4rem 0', verticalAlign: 'top' }} className="ft-mono">
                  {r.id}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', wordBreak: 'break-word' }}>{r.display_name || '—'}</td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }} className="ft-mono">
                  {r.driver}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', wordBreak: 'break-all', maxWidth: '14rem' }}>
                  {r.gateway_url || '—'}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }}>{r.has_gateway_token ? '•••• (set)' : '(none)'}</td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }} className="ft-mono">
                  {r.product_id || '—'}
                </td>
                <td style={{ padding: '0.4rem 0', verticalAlign: 'top', whiteSpace: 'nowrap' }}>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem', marginRight: '0.35rem' }} disabled={busy} onClick={() => openEdit(r)}>
                    Edit
                  </button>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem' }} disabled={busy} onClick={() => void onDelete(r.id)}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && rows.length === 0 ? <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.8rem' }}>No rows yet.</p> : null}
      </div>

      <h3 className="ft-field-label" style={{ margin: '0 0 0.5rem', fontSize: '0.65rem', letterSpacing: '0.04em' }}>
        Add endpoint
      </h3>
      <form onSubmit={(ev) => void onCreate(ev)} style={{ display: 'grid', gap: '0.5rem', maxWidth: '36rem' }}>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Display name
          </span>
          <input className="ft-input ft-input--sm" value={cName} onChange={(ev) => setCName(ev.target.value)} disabled={busy} placeholder="Optional" />
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Driver
          </span>
          <select className="ft-input ft-input--sm" value={cDriver} onChange={(ev) => setCDriver(ev.target.value)} disabled={busy}>
            {GATEWAY_DRIVER_OPTIONS.map(([v, label]) => (
              <option key={v} value={v}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Gateway URL
          </span>
          <input className="ft-input ft-input--sm" value={cUrl} onChange={(ev) => setCUrl(ev.target.value)} disabled={busy} placeholder="wss://… (not required for stub)" />
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Gateway token
          </span>
          <input className="ft-input ft-input--sm" value={cToken} onChange={(ev) => setCToken(ev.target.value)} disabled={busy} placeholder="Optional" autoComplete="off" />
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Device id
          </span>
          <input className="ft-input ft-input--sm" value={cDevice} onChange={(ev) => setCDevice(ev.target.value)} disabled={busy} placeholder="Optional" />
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Timeout (sec, 0 = server default)
          </span>
          <input className="ft-input ft-input--sm" value={cTimeout} onChange={(ev) => setCTimeout(ev.target.value)} disabled={busy} placeholder="0" inputMode="numeric" />
        </label>
        <label className="ft-field" style={{ margin: 0 }}>
          <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
            Product id (optional scope)
          </span>
          <input className="ft-input ft-input--sm" value={cProduct} onChange={(ev) => setCProduct(ev.target.value)} disabled={busy} placeholder="Workspace / product UUID" />
        </label>
        <div>
          <button type="submit" className="ft-btn-primary" style={{ fontSize: '0.78rem' }} disabled={busy}>
            Create gateway endpoint
          </button>
        </div>
      </form>

      {editTarget ? (
        <div
          style={{
            marginTop: '1.25rem',
            padding: '0.85rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border)',
            background: 'var(--mc-bg-tertiary)',
          }}
        >
          <h3 className="ft-field-label" style={{ margin: '0 0 0.5rem', fontSize: '0.65rem', letterSpacing: '0.04em' }}>
            Edit <span className="ft-mono">{editTarget.id}</span>
          </h3>
          <form onSubmit={(ev) => void onSaveEdit(ev)} style={{ display: 'grid', gap: '0.5rem', maxWidth: '36rem' }}>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Display name
              </span>
              <input className="ft-input ft-input--sm" value={eName} onChange={(ev) => setEName(ev.target.value)} disabled={busy} />
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Driver
              </span>
              <select className="ft-input ft-input--sm" value={eDriver} onChange={(ev) => setEDriver(ev.target.value)} disabled={busy}>
                {GATEWAY_DRIVER_OPTIONS.map(([v, label]) => (
                  <option key={v} value={v}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Gateway URL
              </span>
              <input className="ft-input ft-input--sm" value={eUrl} onChange={(ev) => setEUrl(ev.target.value)} disabled={busy} />
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                New gateway token
              </span>
              <input
                className="ft-input ft-input--sm"
                value={eNewToken}
                onChange={(ev) => setENewToken(ev.target.value)}
                disabled={busy || eClearToken}
                placeholder="Leave blank to keep current"
                autoComplete="off"
              />
            </label>
            <label className="ft-field" style={{ margin: 0, display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <input type="checkbox" checked={eClearToken} onChange={(ev) => setEClearToken(ev.target.checked)} disabled={busy} />
              <span style={{ fontSize: '0.78rem' }}>Clear stored token</span>
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Device id
              </span>
              <input className="ft-input ft-input--sm" value={eDevice} onChange={(ev) => setEDevice(ev.target.value)} disabled={busy} />
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Timeout (sec)
              </span>
              <input className="ft-input ft-input--sm" value={eTimeout} onChange={(ev) => setETimeout(ev.target.value)} disabled={busy} inputMode="numeric" />
            </label>
            <label className="ft-field" style={{ margin: 0 }}>
              <span className="ft-field-label" style={{ fontSize: '0.65rem' }}>
                Product id
              </span>
              <input className="ft-input ft-input--sm" value={eProduct} onChange={(ev) => setEProduct(ev.target.value)} disabled={busy} placeholder="Empty = global" />
            </label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
              <button type="submit" className="ft-btn-primary" style={{ fontSize: '0.78rem' }} disabled={busy}>
                Save changes
              </button>
              <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} disabled={busy} onClick={closeEdit}>
                Cancel
              </button>
            </div>
          </form>
        </div>
      ) : null}
    </section>
  );
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

        <IdeationBucketsSettingsPanel />

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

        <GatewayEndpointsPanel client={client} defaultProductId={productId} />

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
