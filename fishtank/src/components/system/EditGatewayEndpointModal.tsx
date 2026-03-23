import { useEffect, useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import type { ApiGatewayEndpoint, PatchGatewayEndpointBody } from '../../api/armsTypes';
import { GATEWAY_DRIVER_OPTIONS } from '../../lib/gatewayDriverOptions';
import { generateOpenclawStyleDeviceId } from '../../lib/openclawDeviceId';

type Props = {
  open: boolean;
  endpoint: ApiGatewayEndpoint | null;
  onClose: () => void;
  client: ArmsClient;
  onSaved: () => void | Promise<void>;
};

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
  }, [open, endpoint]);

  if (!open || !endpoint) return null;

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
      <div className="ft-modal-panel" style={{ maxWidth: '26rem' }}>
        <div className="ft-modal-head">
          <h2 id="ft-edit-gateway-endpoint-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            Edit gateway endpoint
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog" disabled={submitting}>
            <X size={18} />
          </button>
        </div>
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
          <div className="ft-modal-actions">
            <button type="button" className="ft-btn-ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className="ft-btn-primary" disabled={submitting}>
              {submitting ? 'Saving…' : 'Save changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
