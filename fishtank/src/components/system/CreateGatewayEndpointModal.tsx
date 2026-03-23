import { useEffect, useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import { GATEWAY_DRIVER_OPTIONS } from '../../lib/gatewayDriverOptions';

type Props = {
  open: boolean;
  onClose: () => void;
  client: ArmsClient;
  defaultProductId?: string;
  onCreated: () => void | Promise<void>;
};

export function CreateGatewayEndpointModal({ open, onClose, client, defaultProductId, onCreated }: Props) {
  const [name, setName] = useState('');
  const [driver, setDriver] = useState<string>('openclaw_ws');
  const [url, setUrl] = useState('');
  const [token, setToken] = useState('');
  const [device, setDevice] = useState('');
  const [timeoutField, setTimeoutField] = useState('');
  const [product, setProduct] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName('');
    setDriver('openclaw_ws');
    setUrl('');
    setToken('');
    setDevice('');
    setTimeoutField('');
    setProduct(defaultProductId ?? '');
    setFormError(null);
  }, [open, defaultProductId]);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    setSubmitting(true);
    const timeoutSec = parseInt(timeoutField.trim(), 10);
    try {
      await client.createGatewayEndpoint({
        display_name: name.trim() || undefined,
        driver,
        gateway_url: url.trim() || undefined,
        gateway_token: token.trim() || undefined,
        device_id: device.trim() || undefined,
        timeout_sec: Number.isFinite(timeoutSec) ? timeoutSec : undefined,
        product_id: product.trim() || undefined,
      });
      await onCreated();
      onClose();
    } catch (err) {
      setFormError(err instanceof ArmsHttpError ? err.message : 'Create failed.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-create-gateway-endpoint-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ maxWidth: '26rem' }}>
        <div className="ft-modal-head">
          <h2 id="ft-create-gateway-endpoint-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            Create gateway endpoint
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog" disabled={submitting}>
            <X size={18} />
          </button>
        </div>
        <form onSubmit={(e) => void handleSubmit(e)} className="ft-modal-body">
          {formError ? (
            <p className="ft-banner ft-banner--error" role="alert">
              {formError}
            </p>
          ) : null}
          <label className="ft-field">
            <span className="ft-field-label">Display name</span>
            <input
              className="ft-input ft-input--sm"
              value={name}
              onChange={(ev) => setName(ev.target.value)}
              disabled={submitting}
              placeholder="Optional"
              autoFocus
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Driver</span>
            <select className="ft-input ft-input--sm" value={driver} onChange={(ev) => setDriver(ev.target.value)} disabled={submitting}>
              {GATEWAY_DRIVER_OPTIONS.map(([v, label]) => (
                <option key={v} value={v}>
                  {label}
                </option>
              ))}
            </select>
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Gateway URL</span>
            <input
              className="ft-input ft-input--sm"
              value={url}
              onChange={(ev) => setUrl(ev.target.value)}
              disabled={submitting}
              placeholder="wss://… (not required for stub)"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Gateway token</span>
            <input
              className="ft-input ft-input--sm"
              value={token}
              onChange={(ev) => setToken(ev.target.value)}
              disabled={submitting}
              placeholder="Optional"
              autoComplete="off"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Device id</span>
            <input
              className="ft-input ft-input--sm"
              value={device}
              onChange={(ev) => setDevice(ev.target.value)}
              disabled={submitting}
              placeholder="Optional"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Timeout (sec, 0 = server default)</span>
            <input
              className="ft-input ft-input--sm"
              value={timeoutField}
              onChange={(ev) => setTimeoutField(ev.target.value)}
              disabled={submitting}
              placeholder="0"
              inputMode="numeric"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Product id (optional scope)</span>
            <input
              className="ft-input ft-input--sm"
              value={product}
              onChange={(ev) => setProduct(ev.target.value)}
              disabled={submitting}
              placeholder="Workspace / product UUID"
            />
          </label>
          <div className="ft-modal-actions">
            <button type="button" className="ft-btn-ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className="ft-btn-primary" disabled={submitting}>
              {submitting ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
