import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import { Link, useParams } from 'react-router-dom';
import { CheckCircle2, Circle, MessageSquare, RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiProductFeedback } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import { formatRelativeTime } from '../lib/time';

const LIST_LIMIT = 200;

const SOURCE_PRESETS = ['manual', 'feedback', 'research', 'resurfaced', 'support', 'survey'] as const;

const SENTIMENTS = ['neutral', 'positive', 'negative', 'mixed'] as const;

export function MissionFeedbackPage() {
  const { productId = '' } = useParams<{ productId: string }>();
  const { client } = useMissionUi();

  const [items, setItems] = useState<ApiProductFeedback[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [unprocessedOnly, setUnprocessedOnly] = useState(false);

  const [source, setSource] = useState<string>('manual');
  const [content, setContent] = useState('');
  const [sentiment, setSentiment] = useState<string>('neutral');
  const [category, setCategory] = useState('');
  const [customerId, setCustomerId] = useState('');
  const [ideaId, setIdeaId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [patchingId, setPatchingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!productId) return;
    setLoading(true);
    setError(null);
    try {
      const list = await client.listProductFeedback(productId, { limit: LIST_LIMIT });
      setItems(list);
    } catch (e: unknown) {
      setItems([]);
      setError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not load product feedback.',
      );
    } finally {
      setLoading(false);
    }
  }, [client, productId]);

  useEffect(() => {
    void load();
  }, [load]);

  const visibleItems = useMemo(
    () => (unprocessedOnly ? items.filter((f) => !f.processed) : items),
    [items, unprocessedOnly],
  );

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (!productId) return;
    const src = source.trim();
    const txt = content.trim();
    if (!src || !txt) {
      setFormError('Source and content are required.');
      return;
    }
    setFormError(null);
    setSubmitting(true);
    try {
      const created = await client.appendProductFeedback(productId, {
        source: src,
        content: txt,
        sentiment: sentiment.trim() || undefined,
        category: category.trim() || undefined,
        customer_id: customerId.trim() || undefined,
        idea_id: ideaId.trim() || undefined,
      });
      setItems((prev) => [created, ...prev.filter((x) => x.id !== created.id)]);
      setContent('');
      setCategory('');
      setCustomerId('');
      setIdeaId('');
    } catch (err: unknown) {
      setFormError(
        err instanceof ArmsHttpError ? `${err.message} (${err.status})` : 'Could not save feedback.',
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function toggleProcessed(row: ApiProductFeedback) {
    const next = !row.processed;
    setPatchingId(row.id);
    setError(null);
    try {
      const updated = await client.patchProductFeedback(row.id, { processed: next });
      setItems((prev) => prev.map((x) => (x.id === updated.id ? updated : x)));
    } catch (e: unknown) {
      setError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not update processed state.',
      );
    } finally {
      setPatchingId(null);
    }
  }

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <MessageSquare size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">Feedback</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Capture external product feedback for this workspace. Rows feed the preference model (
              <code className="ft-mono">POST …/preference-model/recompute</code>) and may auto-ingest into knowledge when
              enabled on the server. API: <code className="ft-mono">GET/POST …/feedback</code>,{' '}
              <code className="ft-mono">PATCH /api/product-feedback/&#123;id&#125;</code> (
              <code className="ft-mono">processed</code>).
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <button type="button" className="ft-btn-ghost" disabled={loading || !productId} onClick={() => void load()}>
              <RefreshCw size={16} className={loading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
            <Link to={`/p/${productId}/tasks`} className="ft-btn-primary" style={{ textDecoration: 'none' }}>
              Tasks
            </Link>
          </div>
        </div>

        {error ? (
          <div className="ft-banner ft-banner--error ft-docs-page__banner" role="alert">
            {error}
          </div>
        ) : null}

        {!productId ? (
          <p className="ft-muted">Open a workspace to record feedback.</p>
        ) : (
          <>
            <section
              aria-labelledby="fb-compose-heading"
              style={{
                border: '1px solid var(--mc-border)',
                borderRadius: 'var(--ft-radius-sm)',
                padding: '1rem',
                background: 'var(--mc-bg-secondary)',
                marginBottom: '1.25rem',
              }}
            >
              <h2 id="fb-compose-heading" className="ft-upper-label" style={{ margin: '0 0 0.75rem', fontSize: '0.7rem' }}>
                Add feedback
              </h2>
              <form onSubmit={(e) => void onSubmit(e)} style={{ display: 'flex', flexDirection: 'column', gap: '0.65rem' }}>
                <div style={{ display: 'grid', gap: '0.65rem', gridTemplateColumns: 'repeat(auto-fill, minmax(10rem, 1fr))' }}>
                  <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                    <span className="ft-muted">Source</span>
                    <select
                      className="ft-input ft-input--sm"
                      value={SOURCE_PRESETS.includes(source as (typeof SOURCE_PRESETS)[number]) ? source : '__custom'}
                      onChange={(e) => {
                        const v = e.target.value;
                        if (v === '__custom') setSource('');
                        else setSource(v);
                      }}
                      aria-label="Feedback source"
                    >
                      {SOURCE_PRESETS.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                      <option value="__custom">Custom…</option>
                    </select>
                  </label>
                  {!SOURCE_PRESETS.includes(source as (typeof SOURCE_PRESETS)[number]) ? (
                    <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                      <span className="ft-muted">Custom source</span>
                      <input
                        className="ft-input ft-input--sm"
                        value={source}
                        onChange={(e) => setSource(e.target.value)}
                        placeholder="e.g. zendesk"
                        aria-label="Custom feedback source"
                      />
                    </label>
                  ) : null}
                  <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                    <span className="ft-muted">Sentiment</span>
                    <select
                      className="ft-input ft-input--sm"
                      value={sentiment}
                      onChange={(e) => setSentiment(e.target.value)}
                      aria-label="Sentiment"
                    >
                      {SENTIMENTS.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                    <span className="ft-muted">Category</span>
                    <input
                      className="ft-input ft-input--sm"
                      value={category}
                      onChange={(e) => setCategory(e.target.value)}
                      placeholder="Optional"
                      aria-label="Category"
                    />
                  </label>
                  <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                    <span className="ft-muted">Customer id</span>
                    <input
                      className="ft-input ft-input--sm"
                      value={customerId}
                      onChange={(e) => setCustomerId(e.target.value)}
                      placeholder="Optional"
                      aria-label="Customer id"
                    />
                  </label>
                  <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                    <span className="ft-muted">Idea id</span>
                    <input
                      className="ft-input ft-input--sm ft-mono"
                      value={ideaId}
                      onChange={(e) => setIdeaId(e.target.value)}
                      placeholder="Optional link"
                      aria-label="Linked idea id"
                    />
                  </label>
                </div>
                <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem', fontSize: '0.72rem' }}>
                  <span className="ft-muted">Content</span>
                  <textarea
                    className="ft-input ft-input--sm"
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    placeholder="What did you hear from users or stakeholders?"
                    rows={4}
                    aria-label="Feedback content"
                    style={{ resize: 'vertical', minHeight: '5rem' }}
                  />
                </label>
                {formError ? (
                  <p className="ft-banner ft-banner--error" style={{ margin: 0, padding: '0.5rem 0.65rem', fontSize: '0.8rem' }}>
                    {formError}
                  </p>
                ) : null}
                <div>
                  <button type="submit" className="ft-btn-primary" disabled={submitting || !content.trim()}>
                    {submitting ? 'Saving…' : 'Save feedback'}
                  </button>
                </div>
              </form>
            </section>

            <div className="ft-docs-page__toolbar">
              <label
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '0.4rem',
                  fontSize: '0.8rem',
                  cursor: 'pointer',
                  userSelect: 'none',
                }}
              >
                <input
                  type="checkbox"
                  checked={unprocessedOnly}
                  onChange={(e) => setUnprocessedOnly(e.target.checked)}
                />
                Unprocessed only
              </label>
              <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                {unprocessedOnly ? `${visibleItems.length} shown` : `${items.length} loaded`}
              </span>
            </div>

            {loading && items.length === 0 ? (
              <p className="ft-muted" style={{ padding: '1rem 0', fontSize: '0.875rem', margin: 0 }}>
                Loading…
              </p>
            ) : visibleItems.length === 0 ? (
              <p className="ft-muted" style={{ padding: '1rem 0', fontSize: '0.875rem', margin: 0 }}>
                {items.length === 0
                  ? 'No feedback yet. Add a note above or ingest via the API.'
                  : 'No entries match this filter.'}
              </p>
            ) : (
              <ul
                style={{
                  listStyle: 'none',
                  margin: 0,
                  padding: '0 0 1rem',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '0.5rem',
                }}
              >
                {visibleItems.map((f) => (
                  <li
                    key={f.id}
                    style={{
                      border: '1px solid var(--mc-border)',
                      borderRadius: 'var(--ft-radius-sm)',
                      padding: '0.65rem 0.75rem',
                      background: f.processed ? 'var(--mc-bg-tertiary, var(--mc-bg-secondary))' : 'var(--mc-bg-secondary)',
                      opacity: f.processed ? 0.92 : 1,
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem' }}>
                      <MessageSquare size={16} className="ft-muted" style={{ flexShrink: 0, marginTop: 2 }} />
                      <div style={{ minWidth: 0, flex: 1 }}>
                        <div
                          style={{
                            display: 'flex',
                            flexWrap: 'wrap',
                            alignItems: 'center',
                            gap: '0.35rem 0.5rem',
                            marginBottom: '0.35rem',
                          }}
                        >
                          <code className="ft-mono" style={{ fontSize: '0.7rem' }}>
                            {f.id}
                          </code>
                          <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                            {f.created_at ? formatRelativeTime(f.created_at) : '—'}
                          </span>
                          {f.source ? (
                            <span
                              style={{
                                fontSize: '0.68rem',
                                padding: '0.12rem 0.4rem',
                                borderRadius: 4,
                                background: 'var(--mc-bg-primary)',
                                border: '1px solid var(--mc-border)',
                              }}
                            >
                              {f.source}
                            </span>
                          ) : null}
                          {f.sentiment ? (
                            <span
                              style={{
                                fontSize: '0.68rem',
                                padding: '0.12rem 0.4rem',
                                borderRadius: 4,
                                background: 'var(--mc-bg-primary)',
                                border: '1px solid var(--mc-border)',
                              }}
                            >
                              {f.sentiment}
                            </span>
                          ) : null}
                          {f.category ? (
                            <span
                              style={{
                                fontSize: '0.68rem',
                                padding: '0.12rem 0.4rem',
                                borderRadius: 4,
                                background: 'var(--mc-bg-primary)',
                                border: '1px solid var(--mc-border)',
                              }}
                            >
                              {f.category}
                            </span>
                          ) : null}
                          {f.processed ? (
                            <span style={{ fontSize: '0.68rem', color: 'var(--ft-accent-green, #4ade80)' }}>Processed</span>
                          ) : (
                            <span style={{ fontSize: '0.68rem', color: 'var(--ft-muted)' }}>Open</span>
                          )}
                        </div>
                        <p style={{ margin: 0, fontSize: '0.84rem', lineHeight: 1.5, whiteSpace: 'pre-wrap' }}>
                          {f.content}
                        </p>
                        {(f.customer_id || f.idea_id) && (
                          <div className="ft-muted" style={{ fontSize: '0.72rem', marginTop: '0.35rem' }}>
                            {f.customer_id ? <span>Customer: {f.customer_id}</span> : null}
                            {f.customer_id && f.idea_id ? ' · ' : null}
                            {f.idea_id ? (
                              <span>
                                Idea: <code className="ft-mono">{f.idea_id}</code>
                              </span>
                            ) : null}
                          </div>
                        )}
                      </div>
                      <button
                        type="button"
                        className="ft-btn-ghost"
                        style={{ flexShrink: 0, display: 'inline-flex', alignItems: 'center', gap: 6 }}
                        disabled={patchingId === f.id}
                        onClick={() => void toggleProcessed(f)}
                        title={f.processed ? 'Mark as open' : 'Mark as processed'}
                        aria-pressed={f.processed}
                      >
                        {f.processed ? <CheckCircle2 size={16} aria-hidden /> : <Circle size={16} aria-hidden />}
                        {f.processed ? 'Reopen' : 'Done'}
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </div>
    </div>
  );
}
