import { useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import { ArmsHttpError } from '../../api/armsClient';

type Props = {
  open: boolean;
  onClose: () => void;
  onCreate: (name: string, workspaceId: string) => Promise<void>;
};

export function CreateProductModal({ open, onClose, onCreate }: Props) {
  const [name, setName] = useState('');
  const [workspaceId, setWorkspaceId] = useState('default');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    if (!name.trim()) {
      setFormError('Name is required.');
      return;
    }
    if (!workspaceId.trim()) {
      setFormError('Workspace id is required.');
      return;
    }
    setSubmitting(true);
    try {
      await onCreate(name, workspaceId);
      setName('');
      setWorkspaceId('default');
      onClose();
    } catch (err) {
      const msg =
        err instanceof ArmsHttpError ? `${err.message} (${err.status})` : 'Could not create product.';
      setFormError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-create-product-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel">
        <div className="ft-modal-head">
          <h2 id="ft-create-product-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            New product
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog">
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
            <span className="ft-field-label">Name</span>
            <input
              className="ft-input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
              disabled={submitting}
              placeholder="My product"
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Workspace id</span>
            <input
              className="ft-input"
              value={workspaceId}
              onChange={(e) => setWorkspaceId(e.target.value)}
              disabled={submitting}
              placeholder="default"
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
