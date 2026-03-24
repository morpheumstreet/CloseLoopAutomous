import { useEffect, useState, type FormEvent } from 'react';
import { AlertCircle, ArrowLeft, CheckCircle2, Loader2, MinusCircle, ShieldAlert, X, XCircle } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import type {
  ApiGatewayConnectionTestStep,
  ApiGatewayEndpoint,
  GatewayTestConnectionDraft,
  PatchGatewayEndpointBody,
} from '../../api/armsTypes';
import { GATEWAY_DRIVER_OPTIONS } from '../../lib/gatewayDriverOptions';
import { generateOpenclawStyleDeviceId } from '../../lib/openclawDeviceId';

type Props = {
  open: boolean;
  endpoint: ApiGatewayEndpoint | null;
  onClose: () => void;
  client: ArmsClient;
  onSaved: () => void | Promise<void>;
};

/** OpenClaw handshake step when the gateway requests device approval (WebSocket 1008 / pairing). */
function findOpenClawPairingApprovalStep(steps: ApiGatewayConnectionTestStep[] | null | undefined) {
  if (!steps?.length) return undefined;
  return steps.find((s) => s.id === 'ws_openclaw_handshake' && s.status === 'warn');
}

function stepStatusIcon(status: ApiGatewayConnectionTestStep['status']) {
  switch (status) {
    case 'pass':
      return <CheckCircle2 size={18} className="ft-gateway-test-step__icon ft-gateway-test-step__icon--pass" aria-hidden />;
    case 'fail':
      return <XCircle size={18} className="ft-gateway-test-step__icon ft-gateway-test-step__icon--fail" aria-hidden />;
    case 'warn':
      return <AlertCircle size={18} className="ft-gateway-test-step__icon ft-gateway-test-step__icon--warn" aria-hidden />;
    default:
      return <MinusCircle size={18} className="ft-gateway-test-step__icon ft-gateway-test-step__icon--skip" aria-hidden />;
  }
}

export function EditGatewayEndpointModal({ open, endpoint, onClose, client, onSaved }: Props) {
  const [name, setName] = useState('');
  const [driver, setDriver] = useState('');
  const [url, setUrl] = useState('');
  const [device, setDevice] = useState('');
  const [timeoutField, setTimeoutField] = useState('');
  const [product, setProduct] = useState('');
  const [newToken, setNewToken] = useState('');
  const [clearToken, setClearToken] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [formInfo, setFormInfo] = useState<string | null>(null);
  const [deviceIdGenerating, setDeviceIdGenerating] = useState(false);

  const [testPane, setTestPane] = useState(false);
  const [testLoading, setTestLoading] = useState(false);
  const [testSteps, setTestSteps] = useState<ApiGatewayConnectionTestStep[] | null>(null);
  const [testErr, setTestErr] = useState<string | null>(null);

  useEffect(() => {
    if (!open || !endpoint) return;
    setName(endpoint.display_name);
    setDriver(endpoint.driver);
    setUrl(endpoint.gateway_url);
    setDevice(endpoint.device_id);
    setTimeoutField(String(endpoint.timeout_sec ?? 0));
    setProduct(endpoint.product_id ?? '');
    setNewToken('');
    setClearToken(false);
    setFormError(null);
    setFormInfo(null);
    setDeviceIdGenerating(false);
    setTestPane(false);
    setTestLoading(false);
    setTestSteps(null);
    setTestErr(null);
  }, [open, endpoint]);

  if (!open || !endpoint) return null;

  const pairingApprovalStep = testLoading ? undefined : findOpenClawPairingApprovalStep(testSteps);

  function buildTestDraft(): GatewayTestConnectionDraft {
    const ts = parseInt(timeoutField.trim(), 10);
    const timeoutSec = Number.isFinite(ts) ? ts : endpoint.timeout_sec;
    const draft: GatewayTestConnectionDraft = {
      gateway_url: url.trim(),
      driver,
      device_id: device.trim(),
      timeout_sec: timeoutSec,
    };
    if (clearToken) draft.gateway_token = '';
    else if (newToken.trim() !== '') draft.gateway_token = newToken.trim();
    return draft;
  }

  async function runTest() {
    setFormError(null);
    setTestErr(null);
    setTestPane(true);
    setTestLoading(true);
    setTestSteps(null);
    try {
      const res = await client.postGatewayTestConnection(endpoint.id, { draft: buildTestDraft() });
      setTestSteps(res.steps ?? []);
    } catch (err) {
      setTestErr(err instanceof ArmsHttpError ? err.message : 'Connection test failed.');
      setTestSteps([]);
    } finally {
      setTestLoading(false);
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    setFormInfo(null);
    const ts = parseInt(timeoutField.trim(), 10);
    const timeoutSec = Number.isFinite(ts) ? ts : endpoint.timeout_sec;
    const body: PatchGatewayEndpointBody = {};
    if (name.trim() !== endpoint.display_name) body.display_name = name.trim();
    if (driver !== endpoint.driver) body.driver = driver;
    if (url.trim() !== endpoint.gateway_url) body.gateway_url = url.trim();
    if (device.trim() !== endpoint.device_id) body.device_id = device.trim();
    if (timeoutSec !== endpoint.timeout_sec) body.timeout_sec = timeoutSec;
    const prevPid = endpoint.product_id ?? '';
    if (product.trim() !== prevPid) body.product_id = product.trim();
    if (clearToken) body.gateway_token = '';
    else if (newToken.trim() !== '') body.gateway_token = newToken.trim();
    if (Object.keys(body).length === 0) {
      setFormInfo('No changes to save.');
      return;
    }
    setSubmitting(true);
    try {
      await client.patchGatewayEndpoint(endpoint.id, body);
      await onSaved();
      onClose();
    } catch (err) {
      setFormError(err instanceof ArmsHttpError ? err.message : 'Update failed.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-edit-gateway-endpoint-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel ft-modal-panel--gateway-edit">
        <div className="ft-modal-head">
          <h2 id="ft-edit-gateway-endpoint-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            Edit gateway endpoint
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog" disabled={submitting}>
            <X size={18} />
          </button>
        </div>
        <div className="ft-gateway-modal-slide-shell">
          <div
            className="ft-gateway-modal-slide-track"
            style={{ transform: testPane ? 'translateX(-50%)' : 'translateX(0)' }}
          >
            <div className="ft-gateway-modal-slide-pane">
              <form onSubmit={(e) => void handleSubmit(e)} className="ft-modal-body">
                <p className="ft-muted" style={{ margin: '0 0 0.25rem', fontSize: '0.8rem' }}>
                  Id <span className="ft-mono">{endpoint.id}</span>
                </p>
                {formError ? (
                  <p className="ft-banner ft-banner--error" role="alert">
                    {formError}
                  </p>
                ) : null}
                {formInfo ? (
                  <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
                    {formInfo}
                  </p>
                ) : null}
                <label className="ft-field">
                  <span className="ft-field-label">Display name</span>
                  <input className="ft-input ft-input--sm" value={name} onChange={(ev) => setName(ev.target.value)} disabled={submitting} autoFocus />
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">Driver</span>
                  <select className="ft-input ft-input--sm" value={driver} onChange={(ev) => setDriver(ev.target.value)} disabled={submitting}>
                    {!GATEWAY_DRIVER_OPTIONS.some(([v]) => v === driver) ? (
                      <option value={driver}>
                        {driver}
                      </option>
                    ) : null}
                    {GATEWAY_DRIVER_OPTIONS.map(([v, label]) => (
                      <option key={v} value={v}>
                        {label}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">Gateway URL</span>
                  <input className="ft-input ft-input--sm" value={url} onChange={(ev) => setUrl(ev.target.value)} disabled={submitting} />
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">New gateway token</span>
                  <input
                    className="ft-input ft-input--sm"
                    value={newToken}
                    onChange={(ev) => setNewToken(ev.target.value)}
                    disabled={submitting || clearToken}
                    placeholder="Leave blank to keep current"
                    autoComplete="off"
                  />
                </label>
                <label className="ft-field" style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                  <input type="checkbox" checked={clearToken} onChange={(ev) => setClearToken(ev.target.checked)} disabled={submitting} />
                  <span style={{ fontSize: '0.85rem' }}>Clear stored token</span>
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">Device id</span>
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'stretch' }}>
                    <input
                      className="ft-input ft-input--sm"
                      style={{ flex: 1, minWidth: 0 }}
                      value={device}
                      onChange={(ev) => setDevice(ev.target.value)}
                      disabled={submitting}
                      spellCheck={false}
                      autoComplete="off"
                    />
                    <button
                      type="button"
                      className="ft-btn-ghost"
                      style={{ flexShrink: 0, whiteSpace: 'nowrap' }}
                      disabled={submitting || deviceIdGenerating}
                      onClick={() => {
                        void (async () => {
                          setFormError(null);
                          setDeviceIdGenerating(true);
                          try {
                            setDevice(await generateOpenclawStyleDeviceId());
                          } catch {
                            setFormError('Could not generate device id (Ed25519 / SHA-256). Try a secure context (HTTPS) or a newer browser.');
                          } finally {
                            setDeviceIdGenerating(false);
                          }
                        })();
                      }}
                    >
                      {deviceIdGenerating ? '…' : 'Generate'}
                    </button>
                  </div>
                  <span className="ft-muted" style={{ fontSize: '0.75rem', marginTop: '0.2rem', display: 'block' }}>
                    OpenClaw-style: SHA-256 hex of a fresh Ed25519 public key.
                  </span>
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">Timeout (sec)</span>
                  <input
                    className="ft-input ft-input--sm"
                    value={timeoutField}
                    onChange={(ev) => setTimeoutField(ev.target.value)}
                    disabled={submitting}
                    inputMode="numeric"
                  />
                </label>
                <label className="ft-field">
                  <span className="ft-field-label">Product id</span>
                  <input
                    className="ft-input ft-input--sm"
                    value={product}
                    onChange={(ev) => setProduct(ev.target.value)}
                    disabled={submitting}
                    placeholder="Empty = global"
                  />
                </label>
                <div className="ft-modal-actions ft-modal-actions--gateway-edit">
                  <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                    <button type="button" className="ft-btn-ghost" onClick={onClose} disabled={submitting}>
                      Cancel
                    </button>
                    <button
                      type="button"
                      className="ft-btn-ghost"
                      disabled={submitting || testLoading}
                      onClick={() => void runTest()}
                    >
                      Test
                    </button>
                  </div>
                  <button type="submit" className="ft-btn-primary" disabled={submitting}>
                    {submitting ? 'Saving…' : 'Save changes'}
                  </button>
                </div>
              </form>
            </div>
            <div className="ft-gateway-modal-slide-pane ft-modal-body">
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.75rem' }}>
                <button
                  type="button"
                  className="ft-btn-ghost"
                  style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
                  onClick={() => setTestPane(false)}
                  disabled={testLoading}
                >
                  <ArrowLeft size={16} aria-hidden />
                  Back
                </button>
              </div>
              <h3 style={{ margin: '0 0 0.35rem', fontSize: '1rem', fontWeight: 600 }}>Connection test</h3>
              <p className="ft-muted" style={{ margin: '0 0 1rem', fontSize: '0.8rem', lineHeight: 1.45 }}>
                Checks configuration, URL, transport (HTTP, HTTPS, or WebSocket), and bearer auth where supported — using the values in the form (including unsaved token changes). For OpenClaw-class WebSocket drivers, after a successful handshake the server runs{' '}
                <code className="ft-mono">agents.list</code> next (before the &quot;Dispatch path&quot; row) to confirm operator scopes (e.g. <code className="ft-mono">operator.read</code>) and pairing — the same RPC as fleet discovery.
              </p>
              {testErr ? (
                <p className="ft-banner ft-banner--error" role="alert" style={{ marginBottom: '0.75rem' }}>
                  {testErr}
                </p>
              ) : null}
              {pairingApprovalStep ? (
                <div
                  className="ft-banner ft-banner--warn"
                  role="status"
                  aria-live="polite"
                  style={{
                    marginBottom: '0.85rem',
                    padding: '0.65rem 0.75rem',
                    textAlign: 'left',
                    borderWidth: '1px',
                  }}
                >
                  <div style={{ display: 'flex', gap: '0.55rem', alignItems: 'flex-start' }}>
                    <ShieldAlert size={22} className="ft-gateway-test-step__icon ft-gateway-test-step__icon--warn" aria-hidden />
                    <div style={{ minWidth: 0 }}>
                      <div style={{ fontWeight: 700, fontSize: '0.88rem' }}>Admin approval required (WebSocket 1008)</div>
                      <p style={{ margin: '0.35rem 0 0', fontSize: '0.78rem', lineHeight: 1.5, opacity: 0.95 }}>
                        The gateway rejected the OpenClaw handshake with policy close code <strong>1008</strong> (pairing). A
                        privileged operator on the <strong>gateway host</strong> must approve this client before tasks can
                        dispatch.
                      </p>
                      {pairingApprovalStep.detail ? (
                        <pre
                          className="ft-mono ft-muted"
                          style={{
                            margin: '0.5rem 0 0',
                            padding: '0.45rem 0.5rem',
                            fontSize: '0.72rem',
                            lineHeight: 1.45,
                            whiteSpace: 'pre-wrap',
                            wordBreak: 'break-word',
                            borderRadius: 'var(--ft-radius-sm, 6px)',
                            background: 'var(--mc-bg-tertiary, rgba(0,0,0,0.2))',
                            border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
                          }}
                        >
                          {pairingApprovalStep.detail}
                        </pre>
                      ) : null}
                    </div>
                  </div>
                </div>
              ) : null}
              {testLoading ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', color: 'var(--mc-text-secondary)' }}>
                  <Loader2 size={20} className="ft-spin" aria-hidden />
                  <span>Running checks…</span>
                </div>
              ) : null}
              {!testLoading && testSteps && testSteps.length > 0 ? (
                <ul className="ft-gateway-test-step-list" role="list">
                  {testSteps.map((s) => (
                    <li key={s.id} className="ft-gateway-test-step">
                      {stepStatusIcon(s.status)}
                      <div className="ft-gateway-test-step__body">
                        <div className="ft-gateway-test-step__title">{s.title}</div>
                        {s.detail ? <div className="ft-gateway-test-step__detail ft-muted">{s.detail}</div> : null}
                        {typeof s.elapsed_ms === 'number' ? (
                          <div className="ft-gateway-test-step__meta ft-muted">{s.elapsed_ms} ms</div>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ul>
              ) : null}
              {!testLoading && testSteps && testSteps.length === 0 && !testErr ? (
                <p className="ft-muted" style={{ fontSize: '0.85rem' }}>
                  No steps returned.
                </p>
              ) : null}
              <div className="ft-modal-actions" style={{ marginTop: '1rem' }}>
                <button type="button" className="ft-btn-primary" disabled={testLoading} onClick={() => void runTest()}>
                  Run again
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
