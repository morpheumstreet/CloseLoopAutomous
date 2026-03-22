import { useEffect, useState } from 'react';
import { Copy, X } from 'lucide-react';
import { ArmsHttpError, buildLiveEventsUrl, buildLiveEventsUrlTemplate } from '../../api/armsClient';
import type { ApiVersion } from '../../api/armsTypes';
import type { ArmsEnv } from '../../config/armsEnv';

type Props = {
  open: boolean;
  onClose: () => void;
  fetchVersion: () => Promise<ApiVersion>;
  armsEnv: ArmsEnv;
  /** When set, SSE URL uses this product id; otherwise shows a template with &lt;product_id&gt;. */
  productIdForSse?: string | null;
};

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

async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    /* ignore */
  }
}

export function AboutModal({ open, onClose, fetchVersion, armsEnv, productIdForSse }: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<ApiVersion | null>(null);

  useEffect(() => {
    if (!open) return;
    setError(null);
    setInfo(null);
    setLoading(true);
    let cancelled = false;
    void (async () => {
      try {
        const data = await fetchVersion();
        if (!cancelled) setInfo(data);
      } catch (e) {
        if (cancelled) return;
        if (e instanceof ArmsHttpError) {
          setError(e.message);
        } else {
          setError(e instanceof Error ? e.message : 'Could not load version');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, fetchVersion]);

  if (!open) return null;

  const sseUrl = productIdForSse ? buildLiveEventsUrl(armsEnv, productIdForSse) : buildLiveEventsUrlTemplate(armsEnv);

  return (
    <div className="ft-modal-root" role="dialog" aria-modal="true" aria-labelledby="ft-about-title">
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel" style={{ width: 'min(100%, 480px)' }}>
        <div className="ft-modal-head">
          <h2 id="ft-about-title" style={{ margin: 0, fontSize: '1.1rem', fontWeight: 600 }}>
            About Fishtank
          </h2>
          <button type="button" className="ft-btn-icon" onClick={onClose} aria-label="Close dialog">
            <X size={18} />
          </button>
        </div>
        <div className="ft-modal-body">
          <p className="ft-muted" style={{ margin: 0, fontSize: '0.875rem', lineHeight: 1.5 }}>
            Mission Control UI for arms. Backend build metadata from{' '}
            <code className="ft-mono">GET /api/version</code>.
          </p>

          <section style={{ marginTop: '1rem', paddingTop: '1rem', borderTop: '1px solid var(--mc-border)' }}>
            <h3 className="ft-field-label" style={{ marginBottom: '0.5rem' }}>
              Connection (VITE_ARMS_*)
            </h3>
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
                  Basic user
                </dt>
                <dd style={{ margin: 0 }} className="ft-mono">
                  {armsEnv.basicUser || '(unset — VITE_ARMS_BASIC_USER)'}
                </dd>
              </div>
            </dl>
            <div style={{ marginTop: '0.65rem' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.35rem', marginBottom: '0.25rem' }}>
                <span className="ft-muted" style={{ fontSize: '0.7rem' }}>
                  SSE URL (<code className="ft-mono">?token=</code> for EventSource)
                </span>
                <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem', display: 'inline-flex', alignItems: 'center', gap: '0.25rem' }} onClick={() => void copyText(sseUrl)}>
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
                }}
              >
                {sseUrl}
              </pre>
              {!productIdForSse ? (
                <p className="ft-muted" style={{ fontSize: '0.7rem', marginTop: '0.35rem', marginBottom: 0 }}>
                  Replace <code className="ft-mono">&lt;product_id&gt;</code> with a real id, or open a workspace and revisit About for a ready-made URL.
                </p>
              ) : null}
            </div>
          </section>

          <section style={{ marginTop: '1rem', paddingTop: '1rem', borderTop: '1px solid var(--mc-border)' }}>
            <h3 className="ft-field-label" style={{ marginBottom: '0.35rem' }}>
              API docs
            </h3>
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem', lineHeight: 1.5 }}>
              Machine-readable spec in the repo: <code className="ft-mono">docs/openapi/arms-openapi.yaml</code> — import into Swagger UI or Redocly. Route inventory:{' '}
              <code className="ft-mono">GET /api/docs/routes</code>.
            </p>
          </section>

          {loading ? <p className="ft-muted" style={{ margin: '1rem 0 0' }}>Loading version…</p> : null}
          {error ? (
            <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '1rem 0 0' }}>
              {error}
            </p>
          ) : null}

          {info && !loading ? (
            <dl
              style={{
                margin: '1rem 0 0',
                display: 'grid',
                gap: '0.65rem',
                fontSize: '0.875rem',
              }}
            >
              <div>
                <dt className="ft-field-label" style={{ marginBottom: '0.2rem' }}>
                  Arms version
                </dt>
                <dd style={{ margin: 0, fontWeight: 700, fontSize: '1.35rem', letterSpacing: '-0.02em' }}>
                  {displayVersion(info)}
                </dd>
              </div>
              <div>
                <dt className="ft-field-label" style={{ marginBottom: '0.2rem' }}>
                  Describe
                </dt>
                <dd style={{ margin: 0, wordBreak: 'break-all' }} className="ft-mono">
                  {info.version || '—'}
                </dd>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem' }}>
                <div>
                  <dt className="ft-field-label" style={{ marginBottom: '0.2rem' }}>
                    Tag
                  </dt>
                  <dd style={{ margin: 0 }} className="ft-mono">
                    {info.tag || '—'}
                  </dd>
                </div>
                <div>
                  <dt className="ft-field-label" style={{ marginBottom: '0.2rem' }}>
                    Commit
                  </dt>
                  <dd style={{ margin: 0 }} className="ft-mono">
                    {info.commit || '—'}
                  </dd>
                </div>
              </div>
              {(info.commits_after_tag > 0 || info.dirty) && (
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
                  {info.commits_after_tag > 0 ? (
                    <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
                      +{info.commits_after_tag} commit{info.commits_after_tag === 1 ? '' : 's'} after tag
                    </span>
                  ) : null}
                  {info.dirty ? (
                    <span className="ft-chip" style={{ fontSize: '0.75rem' }}>
                      dirty working tree
                    </span>
                  ) : null}
                </div>
              )}
            </dl>
          ) : null}

          <div className="ft-modal-actions">
            <button type="button" className="ft-btn-primary" onClick={onClose}>
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
