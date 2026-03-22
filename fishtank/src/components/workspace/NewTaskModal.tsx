import { useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import { ArmsHttpError } from '../../api/armsClient';

type Props = {
  open: boolean;
  onClose: () => void;
  onCreate: (ideaId: string, spec: string) => Promise<void>;
};

export function NewTaskModal({ open, onClose, onCreate }: Props) {
  const [ideaId, setIdeaId] = useState('');
  const [spec, setSpec] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    if (!ideaId.trim()) {
      setFormError('Idea id is required (approved idea from arms).');
      return;
    }
    if (!spec.trim()) {
      setFormError('Spec / description is required.');
      return;
    }
    setSubmitting(true);
    try {
      await onCreate(ideaId, spec);
      setIdeaId('');
      setSpec('');
      onClose();
    } catch (err) {
      const msg =
        err instanceof ArmsHttpError
          ? `${err.message}${err.code ? ` (${err.code})` : ''} [${err.status}]`
          : 'Could not create task.';
      setFormError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-new-task-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ width: 'min(100%, 480px)' }}>
        <div className="ft-modal-head">
          <h2 id="ft-new-task-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            New task
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog">
            <X size={18} />
          </button>
        </div>
        <form onSubmit={(e) => void handleSubmit(e)} className="ft-modal-body">
          <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem', lineHeight: 1.5 }}>
            Creates <code className="ft-mono">POST /api/tasks</code> — the idea must already be approved in arms.
          </p>
          {formError ? (
            <p className="ft-banner ft-banner--error" role="alert">
              {formError}
            </p>
          ) : null}
          <label className="ft-field">
            <span className="ft-field-label">Idea id</span>
            <input
              className="ft-input"
              value={ideaId}
              onChange={(e) => setIdeaId(e.target.value)}
              disabled={submitting}
              placeholder="uuid from GET …/ideas"
              autoFocus
            />
          </label>
          <label className="ft-field">
            <span className="ft-field-label">Spec</span>
            <textarea
              className="ft-input"
              rows={6}
              value={spec}
              onChange={(e) => setSpec(e.target.value)}
              disabled={submitting}
              placeholder="First line becomes the card title…"
              style={{ resize: 'vertical', minHeight: '120px' }}
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
