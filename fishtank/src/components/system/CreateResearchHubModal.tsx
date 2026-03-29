import { useEffect, useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import type { ApiResearchHub } from '../../api/armsTypes';

type Props = {
  open: boolean;
  onClose: () => void;
  client: ArmsClient;
  /** When set, edit this hub; otherwise create a new one. */
  hub?: ApiResearchHub | null;
  onSuccess: (mode: 'create' | 'edit') => void | Promise<void>;
};

export function CreateResearchHubModal({ open, onClose, client, hub, onSuccess }: Props) {
  const [displayName, setDisplayName] = useState('');
  const [baseUrl, setBaseUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const isEdit = Boolean(hub);

  useEffect(() => {
    if (!open) return;
    if (hub) {
      setDisplayName(hub.display_name ?? '');
      setBaseUrl(hub.base_url ?? '');
      setApiKey('');
    } else {
      setDisplayName('');
      setBaseUrl('');
      setApiKey('');
    }
    setFormError(null);
  }, [open, hub]);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    setSubmitting(true);
    try {
      if (hub) {
        const body: { display_name?: string; base_url?: string; api_key?: string } = {
          display_name: displayName.trim(),
          base_url: baseUrl.trim(),
        };
        if (apiKey.trim() !== '') body.api_key = apiKey.trim();
        await client.patchResearchHub(hub.id, body);
        await onSuccess('edit');
      } else {
        await client.createResearchHub({
          display_name: displayName.trim() || undefined,
          base_url: baseUrl.trim(),
          api_key: apiKey.trim() || undefined,
        });
        await onSuccess('create');
      }
      onClose();
    } catch (err) {
      setFormError(err instanceof ArmsHttpError ? err.message : isEdit ? 'Save failed.' : 'Create failed.');
    } finally {
      setSubmitting(false);
    }
  }

  const titleId = isEdit ? 'ft-edit-research-hub-title' : 'ft-create-research-hub-title';

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby={titleId}>
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ maxWidth: '26rem' }}>
        <div className="ft-modal-head">
          <h2 id={titleId} style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            {isEdit ? 'Edit ResearchClaw hub' : 'Add ResearchClaw hub'}
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
          <p className="ft-muted" style={{ margin: '0 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.45 }}>
            HTTP root for a ResearchClaw-compatible API (<code className="ft-mono">/api/health</code>,{' '}
            <code className="ft-mono">/api/pipeline/start</code>). Secrets stay on the server.
          </p>
          {isEdit ? (
            <p className="ft-muted ft-mono" style={{ margin: '0 0 0.75rem', fontSize: '0.78rem' }}>
              {hub?.id}
            </p>
          ) : null}
          <label className="ft-field">
            <span className="ft-field-label">Display name</span>
            <input
              className="ft-input ft-input--sm"
              value={displayName}
              onChange={(ev) => setDisplayName(ev.target.value)}
              disabled={submitting}
              placeholder="e.g. Lab ResearchClaw"
              autoFocus={!isEdit}
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Base URL</span>
            <input
              className="ft-input ft-input--sm"
              value={baseUrl}
              onChange={(ev) => setBaseUrl(ev.target.value)}
              disabled={submitting}
              placeholder="https://host:port"
              required
              autoComplete="off"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">{isEdit ? 'New API key (optional)' : 'API key (optional)'}</span>
            <input
              className="ft-input ft-input--sm"
              type="password"
              value={apiKey}
              onChange={(ev) => setApiKey(ev.target.value)}
              disabled={submitting}
              placeholder={isEdit && hub?.has_api_key ? 'Leave blank to keep stored key' : 'Bearer token if required'}
              autoComplete="off"
            />
          </label>
          <div className="ft-modal-actions">
            <button type="button" className="ft-btn-ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button type="submit" className="ft-btn-primary" disabled={submitting}>
              {submitting ? (isEdit ? 'Saving…' : 'Adding…') : isEdit ? 'Save' : 'Add hub'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
