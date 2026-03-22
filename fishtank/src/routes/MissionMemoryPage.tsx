import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { BookOpen, Brain, FlaskConical, RefreshCw, ScrollText, Search } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiKnowledgeEntry, ApiOperationLogEntry, ApiResearchCycle } from '../api/armsTypes';
import { MarkdownReadModal } from '../components/docs/MarkdownReadModal';
import { useMissionUi } from '../context/MissionUiContext';
import { formatRelativeTime } from '../lib/time';

const OPS_LIMIT = 400;
const KNOWLEDGE_LIMIT = 200;
const RESEARCH_LIMIT = 80;

function metaChips(metadata: Record<string, unknown>): string[] {
  const chips: string[] = [];
  const cat = metadata.category;
  if (typeof cat === 'string' && cat.trim()) chips.push(cat.trim());
  const typ = metadata.type;
  if (typeof typ === 'string' && typ.trim()) chips.push(typ.trim());
  const tags = metadata.tags;
  if (Array.isArray(tags)) {
    for (const t of tags) {
      if (typeof t === 'string' && t.trim()) chips.push(t.trim());
    }
  } else if (typeof tags === 'string' && tags.trim()) {
    tags.split(',').forEach((x) => {
      const s = x.trim();
      if (s) chips.push(s);
    });
  }
  return chips;
}

function previewLine(content: string, max = 200): string {
  const t = content.trim().replace(/\s+/g, ' ');
  return t.length <= max ? t : `${t.slice(0, max)}…`;
}

function searchTokens(raw: string): string[] {
  return raw
    .trim()
    .toLowerCase()
    .replace(/,/g, ' ')
    .split(/\s+/)
    .map((t) => t.trim())
    .filter(Boolean);
}

function matchesTokens(haystack: string, tokens: string[]): boolean {
  if (tokens.length === 0) return true;
  const h = haystack.toLowerCase();
  return tokens.every((t) => h.includes(t));
}

type MemoryItem =
  | { kind: 'op'; ts: string; entry: ApiOperationLogEntry }
  | { kind: 'knowledge'; ts: string; entry: ApiKnowledgeEntry }
  | { kind: 'research'; ts: string; cycle: ApiResearchCycle };

function buildItems(
  ops: ApiOperationLogEntry[],
  knowledge: ApiKnowledgeEntry[],
  cycles: ApiResearchCycle[],
): MemoryItem[] {
  const out: MemoryItem[] = [];
  for (const e of ops) {
    const ts = e.created_at;
    if (ts) out.push({ kind: 'op', ts, entry: e });
  }
  for (const e of knowledge) {
    const ts = e.updated_at || e.created_at;
    if (ts) out.push({ kind: 'knowledge', ts, entry: e });
  }
  for (const c of cycles) {
    if (c.created_at) out.push({ kind: 'research', ts: c.created_at, cycle: c });
  }
  out.sort((a, b) => Date.parse(b.ts) - Date.parse(a.ts));
  return out;
}

function itemSearchBlob(it: MemoryItem): string {
  switch (it.kind) {
    case 'op': {
      const e = it.entry;
      return [e.action, e.resource_type, e.resource_id, e.actor, e.detail_json].filter(Boolean).join('\n');
    }
    case 'knowledge': {
      const e = it.entry;
      return [e.content, ...metaChips(e.metadata ?? {})].join('\n');
    }
    case 'research':
      return `${it.cycle.summary_snapshot}\n${it.cycle.id}`;
    default:
      return '';
  }
}

function groupByDayDesc(items: MemoryItem[]): { dayKey: string; dayTitle: string; items: MemoryItem[] }[] {
  const groups: { dayKey: string; dayTitle: string; items: MemoryItem[] }[] = [];
  for (const it of items) {
    const d = new Date(it.ts);
    const dayKey = d.toISOString().slice(0, 10);
    const dayTitle = d.toLocaleDateString(undefined, {
      weekday: 'long',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
    const last = groups[groups.length - 1];
    if (last && last.dayKey === dayKey) last.items.push(it);
    else groups.push({ dayKey, dayTitle, items: [it] });
  }
  return groups;
}

function opEntryKey(e: ApiOperationLogEntry, i: number): string {
  if (e.id !== undefined && e.id !== null) return `op-${e.id}`;
  return `op-${e.created_at ?? i}-${i}`;
}

export function MissionMemoryPage() {
  const { productId = '' } = useParams<{ productId: string }>();
  const { client } = useMissionUi();

  const [ops, setOps] = useState<ApiOperationLogEntry[]>([]);
  const [knowledge, setKnowledge] = useState<ApiKnowledgeEntry[]>([]);
  const [research, setResearch] = useState<ApiResearchCycle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [knowledgeOff, setKnowledgeOff] = useState(false);
  const [searchInput, setSearchInput] = useState('');
  const [readModal, setReadModal] = useState<{ title: string; content: string } | null>(null);

  const load = useCallback(async () => {
    if (!productId) return;
    setLoading(true);
    setError(null);
    setKnowledgeOff(false);

    const opsP = client
      .listOperationsLog({ product_id: productId, limit: OPS_LIMIT })
      .then((v) => ({ ok: true as const, v }))
      .catch((e: unknown) => ({ ok: false as const, e }));

    const knowP = client
      .listProductKnowledge(productId, { limit: KNOWLEDGE_LIMIT })
      .then((v) => ({ ok: true as const, v }))
      .catch((e: unknown) => ({ ok: false as const, e }));

    const resP = client
      .listProductResearchCycles(productId, { limit: RESEARCH_LIMIT })
      .catch(() => [] as ApiResearchCycle[]);

    const [opsR, knowR, cycles] = await Promise.all([opsP, knowP, resP]);

    if (opsR.ok) setOps(opsR.v);
    else {
      setOps([]);
      const e = opsR.e;
      setError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not load operations log for this product.',
      );
    }

    if (knowR.ok) {
      setKnowledge(knowR.v);
    } else {
      setKnowledge([]);
      const e = knowR.e;
      if (e instanceof ArmsHttpError && e.status === 503) {
        setKnowledgeOff(true);
      } else if (opsR.ok) {
        setError(
          e instanceof ArmsHttpError
            ? `${e.message} (${e.status})`
            : 'Could not load knowledge for this product.',
        );
      }
    }

    setResearch(cycles);
    setLoading(false);
  }, [client, productId]);

  useEffect(() => {
    void load();
  }, [load]);

  const tokens = useMemo(() => searchTokens(searchInput), [searchInput]);

  const allItems = useMemo(
    () => buildItems(ops, knowledge, research),
    [ops, knowledge, research],
  );

  const filteredItems = useMemo(() => {
    if (tokens.length === 0) return allItems;
    return allItems.filter((it) => matchesTokens(itemSearchBlob(it), tokens));
  }, [allItems, tokens]);

  const dayGroups = useMemo(() => groupByDayDesc(filteredItems), [filteredItems]);

  const taskLinkForOp = (e: ApiOperationLogEntry): string | null => {
    const rid = e.resource_id?.trim();
    if (!rid || !productId) return null;
    const rt = (e.resource_type ?? '').toLowerCase();
    const act = (e.action ?? '').toLowerCase();
    if (rt.includes('task') || act.includes('task') || rt === 'tasks') {
      return `/p/${productId}/tasks`;
    }
    return null;
  };

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <Brain size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">Memory</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Chronological journal for this workspace:{' '}
              <code className="ft-mono">GET /api/operations-log?product_id=…</code>, knowledge from{' '}
              <code className="ft-mono">GET …/knowledge</code>, and research snapshots from{' '}
              <code className="ft-mono">GET …/research-cycles</code>. Newest first, grouped by day. Use the search box to
              filter across all sources. For a global audit table (all products), open{' '}
              <Link to="/activity" className="ft-docs-operator__link">
                Activity
              </Link>
              .
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <button type="button" className="ft-btn-ghost" disabled={loading || !productId} onClick={() => void load()}>
              <RefreshCw size={16} className={loading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
            <Link to={`/p/${productId}/docs`} className="ft-btn-primary" style={{ textDecoration: 'none' }}>
              <BookOpen size={16} aria-hidden />
              Docs
            </Link>
          </div>
        </div>

        {knowledgeOff ? (
          <div className="ft-banner ft-docs-page__banner" role="status">
            Knowledge is not configured on this arms instance (503). Operations log and research history still appear when
            available.
          </div>
        ) : null}

        {error ? (
          <div className="ft-banner ft-banner--error ft-docs-page__banner" role="alert">
            {error}
          </div>
        ) : null}

        {!productId ? (
          <p className="ft-muted">Open a workspace to view memory.</p>
        ) : (
          <>
            <div className="ft-docs-page__toolbar">
              <div style={{ position: 'relative', flex: '1 1 12rem', minWidth: 0, maxWidth: '22rem' }}>
                <Search
                  size={16}
                  className="ft-muted"
                  style={{
                    position: 'absolute',
                    left: 10,
                    top: '50%',
                    transform: 'translateY(-50%)',
                    pointerEvents: 'none',
                  }}
                  aria-hidden
                />
                <input
                  className="ft-input ft-input--sm ft-input--leading-icon"
                  value={searchInput}
                  onChange={(e) => setSearchInput(e.target.value)}
                  placeholder="Search journal (ops, docs, research)…"
                  aria-label="Search memory journal"
                  style={{ width: '100%' }}
                />
              </div>
              <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                {searchInput.trim()
                  ? `${filteredItems.length} of ${allItems.length} entries`
                  : `${allItems.length} entries loaded`}
              </span>
            </div>

            {loading && allItems.length === 0 ? (
              <p className="ft-muted" style={{ padding: '1rem 0', fontSize: '0.875rem', margin: 0 }}>
                Loading…
              </p>
            ) : allItems.length === 0 ? (
              <p className="ft-muted" style={{ padding: '1rem 0', fontSize: '0.875rem', margin: 0 }}>
                No journal data yet for this product. Product actions will appear in the operations log; add long-form
                notes under{' '}
                <Link to={`/p/${productId}/docs`} className="ft-docs-operator__link">
                  Docs
                </Link>
                .
              </p>
            ) : filteredItems.length === 0 ? (
              <p className="ft-muted" style={{ padding: '1rem 0', fontSize: '0.875rem', margin: 0 }}>
                No entries match this search.
              </p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem', paddingBottom: '1rem' }}>
                {dayGroups.map((g) => (
                  <section key={g.dayKey} aria-labelledby={`mem-day-${g.dayKey}`}>
                    <h2
                      id={`mem-day-${g.dayKey}`}
                      className="ft-upper-label"
                      style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.06em' }}
                    >
                      {g.dayTitle}
                    </h2>
                    <ul
                      style={{
                        listStyle: 'none',
                        margin: 0,
                        padding: 0,
                        display: 'flex',
                        flexDirection: 'column',
                        gap: '0.5rem',
                      }}
                    >
                      {g.items.map((it, idx) => {
                        if (it.kind === 'op') {
                          const e = it.entry;
                          const taskTo = taskLinkForOp(e);
                          return (
                            <li
                              key={`${g.dayKey}-${opEntryKey(e, idx)}`}
                              style={{
                                border: '1px solid var(--mc-border)',
                                borderRadius: 'var(--ft-radius-sm)',
                                padding: '0.65rem 0.75rem',
                                background: 'var(--mc-bg-secondary)',
                              }}
                            >
                              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem' }}>
                                <ScrollText size={16} className="ft-muted" style={{ flexShrink: 0, marginTop: 2 }} />
                                <div style={{ minWidth: 0, flex: 1 }}>
                                  <div
                                    style={{
                                      display: 'flex',
                                      flexWrap: 'wrap',
                                      alignItems: 'baseline',
                                      gap: '0.35rem 0.65rem',
                                      marginBottom: '0.25rem',
                                    }}
                                  >
                                    <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Operation</span>
                                    <code className="ft-mono" style={{ fontSize: '0.72rem' }}>
                                      {e.action ?? '—'}
                                    </code>
                                    <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                                      {e.created_at ? formatRelativeTime(e.created_at) : '—'}
                                    </span>
                                  </div>
                                  <div className="ft-muted" style={{ fontSize: '0.75rem', lineHeight: 1.45 }}>
                                    {(e.resource_type || e.resource_id) && (
                                      <span>
                                        {e.resource_type ?? '—'}
                                        {e.resource_id ? (
                                          <>
                                            {' '}
                                            ·{' '}
                                            {taskTo ? (
                                              <Link to={taskTo} className="ft-docs-operator__link">
                                                <code className="ft-mono" style={{ fontSize: '0.7rem' }}>
                                                  {e.resource_id}
                                                </code>
                                              </Link>
                                            ) : (
                                              <code className="ft-mono" style={{ fontSize: '0.7rem' }}>
                                                {e.resource_id}
                                              </code>
                                            )}
                                          </>
                                        ) : null}
                                      </span>
                                    )}
                                    {e.actor ? (
                                      <span style={{ display: 'block', marginTop: '0.15rem' }}>Actor: {e.actor}</span>
                                    ) : null}
                                    {e.detail_json ? (
                                      <pre
                                        className="ft-mono ft-muted"
                                        style={{
                                          margin: '0.35rem 0 0',
                                          fontSize: '0.68rem',
                                          whiteSpace: 'pre-wrap',
                                          wordBreak: 'break-word',
                                          maxHeight: '6rem',
                                          overflow: 'auto',
                                        }}
                                      >
                                        {e.detail_json}
                                      </pre>
                                    ) : null}
                                  </div>
                                </div>
                              </div>
                            </li>
                          );
                        }
                        if (it.kind === 'knowledge') {
                          const e = it.entry;
                          const chips = metaChips(e.metadata ?? {});
                          return (
                            <li
                              key={`know-${e.id}`}
                              style={{
                                border: '1px solid var(--mc-border)',
                                borderRadius: 'var(--ft-radius-sm)',
                                padding: '0.65rem 0.75rem',
                                background: 'var(--mc-bg-secondary)',
                              }}
                            >
                              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem' }}>
                                <BookOpen size={16} className="ft-muted" style={{ flexShrink: 0, marginTop: 2 }} />
                                <div style={{ minWidth: 0, flex: 1 }}>
                                  <div
                                    style={{
                                      display: 'flex',
                                      flexWrap: 'wrap',
                                      alignItems: 'baseline',
                                      gap: '0.35rem 0.65rem',
                                      marginBottom: '0.25rem',
                                    }}
                                  >
                                    <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Knowledge</span>
                                    <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                                      #{e.id} · {e.updated_at ? formatRelativeTime(e.updated_at) : '—'}
                                    </span>
                                  </div>
                                  {chips.length > 0 ? (
                                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.25rem', marginBottom: '0.35rem' }}>
                                      {chips.slice(0, 8).map((c) => (
                                        <span
                                          key={c}
                                          style={{
                                            fontSize: '0.65rem',
                                            padding: '0.1rem 0.35rem',
                                            borderRadius: 4,
                                            background: 'var(--mc-bg-tertiary, rgba(255,255,255,0.06))',
                                          }}
                                        >
                                          {c}
                                        </span>
                                      ))}
                                    </div>
                                  ) : null}
                                  <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem', lineHeight: 1.45 }}>
                                    {previewLine(e.content)}
                                  </p>
                                  <div style={{ marginTop: '0.45rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                                    <button
                                      type="button"
                                      className="ft-btn-ghost"
                                      style={{ fontSize: '0.75rem', padding: '0.2rem 0.5rem' }}
                                      onClick={() => setReadModal({ title: `Knowledge #${e.id}`, content: e.content })}
                                    >
                                      Read
                                    </button>
                                    <Link
                                      to={`/p/${productId}/docs`}
                                      className="ft-btn-ghost"
                                      style={{ fontSize: '0.75rem', padding: '0.2rem 0.5rem', textDecoration: 'none' }}
                                    >
                                      Edit in Docs
                                    </Link>
                                  </div>
                                </div>
                              </div>
                            </li>
                          );
                        }
                        const c = it.cycle;
                        return (
                          <li
                            key={`res-${c.id}`}
                            style={{
                              border: '1px solid var(--mc-border)',
                              borderRadius: 'var(--ft-radius-sm)',
                              padding: '0.65rem 0.75rem',
                              background: 'var(--mc-bg-secondary)',
                            }}
                          >
                            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem' }}>
                              <FlaskConical size={16} className="ft-muted" style={{ flexShrink: 0, marginTop: 2 }} />
                              <div style={{ minWidth: 0, flex: 1 }}>
                                <div
                                  style={{
                                    display: 'flex',
                                    flexWrap: 'wrap',
                                    alignItems: 'baseline',
                                    gap: '0.35rem 0.65rem',
                                    marginBottom: '0.25rem',
                                  }}
                                >
                                  <span style={{ fontWeight: 600, fontSize: '0.82rem' }}>Research snapshot</span>
                                  <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                                    {c.created_at ? formatRelativeTime(c.created_at) : '—'}
                                  </span>
                                </div>
                                <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem', lineHeight: 1.45 }}>
                                  {previewLine(c.summary_snapshot, 320)}
                                </p>
                              </div>
                            </div>
                          </li>
                        );
                      })}
                    </ul>
                  </section>
                ))}
              </div>
            )}
          </>
        )}
      </div>

      <MarkdownReadModal
        open={readModal !== null}
        onClose={() => setReadModal(null)}
        title={readModal?.title ?? ''}
        content={readModal?.content ?? ''}
      />
    </div>
  );
}
