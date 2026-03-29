import { useState, type FormEvent } from 'react';
import { Link2, Smartphone } from 'lucide-react';
import { trimBase } from '../../config/armsEnv';
import { saveArmsConnection } from '../../config/armsLocalStorage';

type Tab = 'endpoint' | 'qr';

type Props = {
  onSaved: () => void;
};

function parseHttpUrl(raw: string): string {
  const t = raw.trim();
  if (!t) throw new Error('Enter the arms API base URL.');
  let u: URL;
  try {
    u = new URL(t);
  } catch {
    throw new Error('Use a full URL including http:// or https://');
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    throw new Error('Only http and https URLs are supported.');
  }
  return trimBase(u.toString());
}

export function ConnectionSetupModal({ onSaved }: Props) {
  const [tab, setTab] = useState<Tab>('endpoint');
  const [endpoint, setEndpoint] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const base = parseHttpUrl(endpoint);
      saveArmsConnection(base, apiKey);
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save connection.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-connection-setup-title" style={{ position: 'relative' }}>
      <div className="ft-modal-backdrop" aria-hidden style={{ pointerEvents: 'none' }} />
      <div className="ft-modal-panel" style={{ maxWidth: '28rem', width: 'min(100% - 2rem, 28rem)' }}>
        <div className="ft-modal-head">
          <h2 id="ft-connection-setup-title" style={{ margin: 0, fontSize: '1.15rem', fontWeight: 600 }}>
            Connect to arms
          </h2>
        </div>

        <div className="ft-tabs" style={{ margin: '0 1rem', paddingTop: '0.35rem' }} role="tablist" aria-label="Connection method">
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'endpoint'}
            className={`ft-tab ${tab === 'endpoint' ? 'ft-tab--active' : ''}`}
            onClick={() => setTab('endpoint')}
          >
            <Link2 size={14} style={{ marginRight: '0.35rem', verticalAlign: 'middle' }} aria-hidden />
            Endpoint &amp; API key
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'qr'}
            className={`ft-tab ${tab === 'qr' ? 'ft-tab--active' : ''}`}
            onClick={() => setTab('qr')}
          >
            <Smartphone size={14} style={{ marginRight: '0.35rem', verticalAlign: 'middle' }} aria-hidden />
            Mobile (QR)
          </button>
        </div>

        {tab === 'endpoint' ? (
          <form onSubmit={(e) => void handleSubmit(e)} className="ft-modal-body">
            <p className="ft-muted" style={{ margin: '0 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.45 }}>
              The HTTP root of your arms server (must expose <code className="ft-mono">GET /api/health</code>). Values are stored in this browser only (
              <code className="ft-mono">localStorage</code>).
            </p>
            {error ? (
              <p className="ft-banner ft-banner--error" role="alert" style={{ marginBottom: '0.75rem' }}>
                {error}
              </p>
            ) : null}
            <label className="ft-field">
              <span className="ft-field-label">API endpoint</span>
              <input
                className="ft-input ft-input--sm"
                value={endpoint}
                onChange={(ev) => setEndpoint(ev.target.value)}
                disabled={submitting}
                placeholder="https://arms.example.com"
                autoComplete="url"
                autoFocus
              />
            </label>
            <label className="ft-field">
              <span className="ft-field-label">API key (Bearer)</span>
              <input
                className="ft-input ft-input--sm"
                value={apiKey}
                onChange={(ev) => setApiKey(ev.target.value)}
                disabled={submitting}
                placeholder="Optional if the server has no MC_API_TOKEN"
                autoComplete="off"
                type="password"
              />
            </label>
            <div className="ft-modal-actions" style={{ marginTop: '1rem' }}>
              <button type="submit" className="ft-btn-primary" disabled={submitting}>
                {submitting ? 'Saving…' : 'Continue'}
              </button>
            </div>
          </form>
        ) : (
          <div className="ft-modal-body">
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.875rem', lineHeight: 1.5 }}>
              Pairing this browser with a phone or tablet via QR code will be added here. For now, use the &quot;Endpoint &amp; API key&quot; tab to connect
              from this device.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
