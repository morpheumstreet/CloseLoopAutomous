import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react';
import { X } from 'lucide-react';
import type { ApiIdea } from '../../api/armsTypes';
import { ArmsHttpError } from '../../api/armsClient';
import { useMissionUi } from '../../context/MissionUiContext';

type Props = {
  open: boolean;
  productId: string;
  onClose: () => void;
  onCreate: (ideaId: string | null, spec: string, newIdeaId?: string | null) => Promise<void>;
};

/** Approved (yes/now) and not already linked to a task. */
function ideaEligibleForNewTask(i: ApiIdea): boolean {
  if (!i.decided) return false;
  const d = (i.decision ?? '').toLowerCase();
  if (d !== 'yes' && d !== 'now') return false;
  if ((i.task_id ?? '').trim() !== '') return false;
  return true;
}

function ideaLabel(i: ApiIdea): string {
  const t = (i.title ?? '').trim() || '(no title)';
  return `${t} · ${i.id}`;
}

function ideaMatchesQuery(i: ApiIdea, q: string): boolean {
  const s = q.trim().toLowerCase();
  if (!s) return true;
  const id = (i.id ?? '').toLowerCase();
  const title = (i.title ?? '').toLowerCase();
  return id.includes(s) || title.includes(s);
}

export function NewTaskModal({ open, productId, onClose, onCreate }: Props) {
  const { client } = useMissionUi();
  const [ideas, setIdeas] = useState<ApiIdea[]>([]);
  const [ideasLoading, setIdeasLoading] = useState(false);
  const [ideasError, setIdeasError] = useState<string | null>(null);
  const [selectedIdeaId, setSelectedIdeaId] = useState('');
  const [comboOpen, setComboOpen] = useState(false);
  const [comboFilter, setComboFilter] = useState('');
  const [spec, setSpec] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const comboRef = useRef<HTMLDivElement>(null);

  const eligible = useMemo(() => ideas.filter(ideaEligibleForNewTask), [ideas]);

  const filteredEligible = useMemo(
    () => eligible.filter((i) => ideaMatchesQuery(i, comboFilter)),
    [eligible, comboFilter],
  );

  const selectedIdea = useMemo(
    () => eligible.find((i) => i.id === selectedIdeaId) ?? null,
    [eligible, selectedIdeaId],
  );

  const hasIdeaChoice = eligible.length > 0 && selectedIdeaId !== '' && selectedIdea != null;
  /** Open dropdown + active filter with zero hits — treat as “no idea found” for this query. */
  const filterHasNoIdea =
    comboOpen && comboFilter.trim() !== '' && filteredEligible.length === 0;
  const specAndCreateDisabled =
    ideasLoading || !!ideasError || !hasIdeaChoice || submitting || filterHasNoIdea;

  useEffect(() => {
    if (!open || !productId.trim()) {
      return;
    }
    let cancelled = false;
    setIdeasLoading(true);
    setIdeasError(null);
    setIdeas([]);
    setSelectedIdeaId('');
    setComboFilter('');
    setComboOpen(false);
    setSpec('');
    void client.listProductIdeas(productId.trim()).then(
      (list) => {
        if (cancelled) return;
        setIdeas(list);
        const elig = list.filter(ideaEligibleForNewTask);
        if (elig.length > 0) {
          setSelectedIdeaId(elig[0].id);
        }
        setIdeasLoading(false);
      },
      (err) => {
        if (cancelled) return;
        setIdeasLoading(false);
        setIdeasError(err instanceof ArmsHttpError ? err.message : 'Could not load ideas.');
      },
    );
    return () => {
      cancelled = true;
    };
  }, [open, productId, client]);

  useEffect(() => {
    if (!comboOpen) return;
    function onPointerDown(e: PointerEvent) {
      if (comboRef.current?.contains(e.target as Node)) return;
      setComboOpen(false);
      setComboFilter('');
    }
    document.addEventListener('pointerdown', onPointerDown);
    return () => document.removeEventListener('pointerdown', onPointerDown);
  }, [comboOpen]);

  const openCombo = useCallback(() => {
    setComboOpen(true);
    setComboFilter('');
  }, []);

  const pickIdea = useCallback((i: ApiIdea) => {
    setSelectedIdeaId(i.id);
    setComboOpen(false);
    setComboFilter('');
  }, []);

  if (!open) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setFormError(null);
    if (!productId.trim()) {
      setFormError('Open a workspace (product) first.');
      return;
    }
    if (!hasIdeaChoice || !selectedIdea) {
      setFormError('Choose an approved idea without a task.');
      return;
    }
    if (!spec.trim()) {
      setFormError('Spec / description is required.');
      return;
    }

    setSubmitting(true);
    try {
      await onCreate(selectedIdeaId, spec, null);
      setSelectedIdeaId('');
      setSpec('');
      setComboFilter('');
      setComboOpen(false);
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

  const inputDisplay = comboOpen ? comboFilter : selectedIdea ? ideaLabel(selectedIdea) : '';

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-new-task-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ width: 'min(100%, 480px)' }}>
        <div className="ft-modal-head">
          <h2 id="ft-new-task-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            New Task
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
          {!productId.trim() ? (
            <p className="ft-banner ft-banner--error" role="alert">
              No active workspace — open a product first.
            </p>
          ) : null}
          <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem', lineHeight: 1.5 }}>
            <strong>Note:</strong> every task must be tied to an <strong>approved</strong> idea (yes / now) that does not
            already have a task. Pick one below; there is no task without that backing idea.
          </p>

          {ideasLoading ? (
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.82rem' }}>
              Loading ideas…
            </p>
          ) : null}
          {ideasError ? (
            <p className="ft-banner ft-banner--error" role="alert">
              {ideasError}
            </p>
          ) : null}

          {!ideasLoading && !ideasError && productId.trim() ? (
            <label className="ft-field">
              <span className="ft-field-label">Idea</span>
              {eligible.length === 0 ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.82rem', lineHeight: 1.45 }}>
                  No approved ideas without a linked task. Approve an idea on the deck (yes / now) and ensure it has no
                  task yet, then try again.
                </p>
              ) : (
                <div ref={comboRef} style={{ position: 'relative' }}>
                  <input
                    type="text"
                    className="ft-input"
                    role="combobox"
                    aria-expanded={comboOpen}
                    aria-controls="ft-new-task-idea-listbox"
                    aria-autocomplete="list"
                    value={inputDisplay}
                    onChange={(e) => {
                      setComboFilter(e.target.value);
                      if (!comboOpen) setComboOpen(true);
                    }}
                    onFocus={() => openCombo()}
                    disabled={submitting}
                    placeholder="Search by title or id…"
                    autoComplete="off"
                    aria-label="Search and select idea"
                    autoFocus={!ideasLoading && eligible.length > 0}
                  />
                  {comboOpen ? (
                    <ul
                      id="ft-new-task-idea-listbox"
                      role="listbox"
                      className="ft-input"
                      style={{
                        position: 'absolute',
                        left: 0,
                        right: 0,
                        top: '100%',
                        marginTop: 2,
                        zIndex: 50,
                        maxHeight: 220,
                        overflowY: 'auto',
                        padding: '0.25rem 0',
                        listStyle: 'none',
                        marginBottom: 0,
                      }}
                    >
                      {filteredEligible.length === 0 ? (
                        <li className="ft-muted" style={{ padding: '0.5rem 0.75rem', fontSize: '0.82rem' }}>
                          No matching ideas.
                        </li>
                      ) : (
                        filteredEligible.map((i) => (
                          <li key={i.id} role="presentation">
                            <button
                              type="button"
                              role="option"
                              aria-selected={i.id === selectedIdeaId}
                              className="ft-btn-ghost"
                              style={{
                                width: '100%',
                                textAlign: 'left',
                                justifyContent: 'flex-start',
                                borderRadius: 0,
                                fontWeight: i.id === selectedIdeaId ? 600 : 400,
                                fontSize: '0.82rem',
                                padding: '0.45rem 0.75rem',
                              }}
                              onMouseDown={(e) => e.preventDefault()}
                              onClick={() => pickIdea(i)}
                            >
                              {ideaLabel(i)}
                            </button>
                          </li>
                        ))
                      )}
                    </ul>
                  ) : null}
                </div>
              )}
            </label>
          ) : null}

          <label className="ft-field">
            <span className="ft-field-label">Spec</span>
            <textarea
              className="ft-input"
              rows={6}
              value={spec}
              onChange={(e) => setSpec(e.target.value)}
              disabled={specAndCreateDisabled}
              placeholder={
                specAndCreateDisabled && !ideasLoading && !ideasError
                  ? 'Select an idea above to enable the spec…'
                  : 'Task spec — first line is often used as the card title…'
              }
              style={{ resize: 'vertical', minHeight: '120px' }}
            />
          </label>

          <div className="ft-modal-actions">
            <button type="button" className="ft-btn-ghost" onClick={onClose} disabled={submitting}>
              Cancel
            </button>
            <button
              type="submit"
              className="ft-btn-primary"
              disabled={specAndCreateDisabled || !spec.trim() || ideasLoading}
            >
              {submitting ? 'Creating…' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
