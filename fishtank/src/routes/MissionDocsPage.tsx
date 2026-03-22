import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent } from 'react';
import {
  BookOpen,
  ExternalLink,
  Eye,
  FileText,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  Upload,
} from 'lucide-react';
import { ArmsHttpError, fetchProductTfidfSuggestTags } from '../api/armsClient';
import type { ApiKnowledgeEntry } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import { formatRelativeTime } from '../lib/time';
import { MarkdownReadModal } from '../components/docs/MarkdownReadModal';

const LIST_LIMIT = 150;
/** Debounce before calling arms TF-IDF for category + tags (create/write + edit). */
const CONTENT_METADATA_SUGGEST_MS = 2000;
/** Minimum trimmed length before calling arms (tokenization still needs real words). */
const CONTENT_METADATA_MIN_CHARS = 12;
const TFIDF_EXTRA_CORPUS_MAX_ENTRIES = 80;
const TFIDF_EXTRA_CORPUS_MAX_CHARS = 2000;

function previewLine(content: string, max = 140): string {
  const t = content.trim().replace(/\s+/g, ' ');
  return t.length <= max ? t : `${t.slice(0, max)}…`;
}

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

/** Client-side filter on the loaded page (up to LIST_LIMIT): each token must appear in content or metadata chips. */
function filterKnowledgeByQuery(entries: ApiKnowledgeEntry[], rawQuery: string): ApiKnowledgeEntry[] {
  const q = rawQuery.trim().toLowerCase();
  if (!q) return entries;
  const tokens = q
    .replace(/,/g, ' ')
    .split(/\s+/)
    .map((t) => t.trim())
    .filter(Boolean);
  if (tokens.length === 0) return entries;
  return entries.filter((e) => {
    const content = e.content.toLowerCase();
    const chips = metaChips(e.metadata ?? {}).map((c) => c.toLowerCase());
    const hay = `${content}\n${chips.join('\n')}`;
    return tokens.every((t) => hay.includes(t));
  });
}

function buildMetadata(category: string, tagsRaw: string): Record<string, unknown> | undefined {
  const meta: Record<string, unknown> = {};
  const c = category.trim();
  if (c) meta.category = c;
  const tags = tagsRaw
    .split(',')
    .map((t) => t.trim())
    .filter(Boolean);
  if (tags.length) meta.tags = tags;
  return Object.keys(meta).length ? meta : undefined;
}

function parseMetaForForm(metadata: Record<string, unknown>): { category: string; tags: string } {
  const category = typeof metadata.category === 'string' ? metadata.category : '';
  const tagsVal = metadata.tags;
  let tags = '';
  if (Array.isArray(tagsVal)) {
    tags = tagsVal.filter((t) => typeof t === 'string').join(', ');
  } else if (typeof tagsVal === 'string') {
    tags = tagsVal;
  }
  return { category, tags };
}

/** Maps TF-IDF tokens to a single knowledge `metadata.category` label (freeform, Docs-oriented). */
const KNOWLEDGE_CATEGORY_HINTS: { category: string; needles: string[] }[] = [
  { category: 'newsletter', needles: ['newsletter', 'subscriber', 'email', 'digest', 'campaign', 'unsubscribe'] },
  { category: 'planning', needles: ['plan', 'roadmap', 'milestone', 'strategy', 'backlog', 'priorit', 'objective', 'okr'] },
  { category: 'runbook', needles: ['runbook', 'playbook', 'incident', 'oncall', 'pager', 'outage', 'rollback', 'postmortem'] },
  { category: 'product', needles: ['product', 'feature', 'user', 'persona', 'ux', 'ui'] },
  { category: 'security', needles: ['security', 'auth', 'oauth', 'password', 'csrf', 'xss', 'cve', 'encrypt'] },
  { category: 'integration', needles: ['api', 'webhook', 'integration', 'stripe', 'github', 'slack'] },
  { category: 'operations', needles: ['ops', 'deploy', 'ci', 'cd', 'monitor', 'metric', 'sla', 'dashboard', 'logging'] },
  { category: 'content', needles: ['content', 'copy', 'blog', 'documentation', 'docs', 'guide', 'tutorial'] },
  { category: 'growth', needles: ['growth', 'marketing', 'seo', 'conversion', 'funnel', 'analytics'] },
  { category: 'monetization', needles: ['pricing', 'billing', 'subscription', 'revenue', 'payment', 'checkout'] },
];

function tokenMatchesNeedle(token: string, needle: string): boolean {
  return token === needle || token.includes(needle) || needle.includes(token);
}

function inferKnowledgeCategoryFromTags(tags: { token: string; score: number }[]): string {
  let bestCat = '';
  let best = 0;
  for (const { category, needles } of KNOWLEDGE_CATEGORY_HINTS) {
    let s = 0;
    for (const { token, score } of tags) {
      const t = token.toLowerCase();
      for (const n of needles) {
        if (tokenMatchesNeedle(t, n)) {
          s += score;
          break;
        }
      }
    }
    if (s > best) {
      best = s;
      bestCat = category;
    }
  }
  if (bestCat) return bestCat;
  if (tags.length > 0) return tags[0].token;
  return '';
}

type SourceFormat = 'markdown' | 'pdf' | 'word';

function knowledgeKindFromNameAndType(fileName: string, mime: string): SourceFormat | null {
  const name = fileName.toLowerCase();
  const m = (mime || '').toLowerCase();
  if (
    name.endsWith('.md') ||
    name.endsWith('.markdown') ||
    name.endsWith('.txt') ||
    m === 'text/markdown' ||
    m === 'text/x-markdown'
  ) {
    return 'markdown';
  }
  if (m.startsWith('text/') && (name.endsWith('.md') || name.endsWith('.txt') || name.endsWith('.markdown'))) {
    return 'markdown';
  }
  if (name.endsWith('.pdf') || m === 'application/pdf') return 'pdf';
  if (name.endsWith('.docx') || m.includes('wordprocessingml')) {
    return 'word';
  }
  return null;
}

function fileSourceFormat(file: File): SourceFormat | null {
  return knowledgeKindFromNameAndType(file.name, file.type);
}

function isAcceptedKnowledgeFile(file: File): boolean {
  return fileSourceFormat(file) !== null;
}

type DropVisual = 'idle' | 'unknown' | 'ok' | 'bad';

/** Filename from webkit file entry when `getAsFile()` is null during dragover (Safari / some paths). */
function webkitEntryFileName(item: DataTransferItem): string {
  const wk = (item as DataTransferItem & { webkitGetAsEntry?: () => { isFile?: boolean; name?: string } | null })
    .webkitGetAsEntry;
  if (typeof wk !== 'function') return '';
  try {
    const ent = wk.call(item);
    if (ent?.isFile && typeof ent.name === 'string') return ent.name;
  } catch {
    /* ignore */
  }
  return '';
}

function classifyDragDataTransfer(dt: DataTransfer): DropVisual {
  const items = dt.items;
  if (!items?.length) return 'unknown';

  let ok = 0;
  let bad = 0;
  let pending = 0;
  let sawFileSlot = false;

  for (let i = 0; i < items.length; i++) {
    const it = items[i];
    if (it.kind !== 'file') continue;
    sawFileSlot = true;
    const f = it.getAsFile();
    const name = (f?.name ?? webkitEntryFileName(it)).trim();
    const type = f?.type ?? '';

    if (!name && !type && !f) {
      pending++;
      continue;
    }

    if (knowledgeKindFromNameAndType(name, type) !== null) {
      ok++;
      continue;
    }

    const typeL = type.toLowerCase();
    const indeterminateDrag =
      !name &&
      (!typeL || typeL === 'application/octet-stream' || typeL === 'application/x-msdownload');
    if (indeterminateDrag) {
      pending++;
      continue;
    }

    if (name || type) {
      bad++;
      continue;
    }

    pending++;
  }

  if (!sawFileSlot) return 'unknown';
  if (bad > 0) return 'bad';
  if (ok > 0) return 'ok';
  return 'unknown';
}

function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result ?? ''));
    r.onerror = () => reject(r.error ?? new Error('read failed'));
    r.readAsText(file);
  });
}

function mergedCreateMetadata(
  uploadMeta: Record<string, unknown> | null,
  category: string,
  tagsRaw: string,
): Record<string, unknown> | undefined {
  const u: Record<string, unknown> = uploadMeta ? { ...uploadMeta } : {};
  const built = buildMetadata(category, tagsRaw);
  if (built) Object.assign(u, built);
  if (Object.keys(u).length === 0) return undefined;
  return u;
}

export function MissionDocsPage() {
  const { activeWorkspace, client, armsEnv } = useMissionUi();
  const productId = activeWorkspace?.id ?? '';

  const [entries, setEntries] = useState<ApiKnowledgeEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [knowledgeOff, setKnowledgeOff] = useState(false);

  const [searchInput, setSearchInput] = useState('');

  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [creating, setCreating] = useState(false);
  const [editing, setEditing] = useState(false);

  const [formContent, setFormContent] = useState('');
  const [formCategory, setFormCategory] = useState('');
  const [formTags, setFormTags] = useState('');
  const [formBusy, setFormBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [metadataSuggestBusy, setMetadataSuggestBusy] = useState(false);
  const [metadataSuggestError, setMetadataSuggestError] = useState<string | null>(null);
  const [readOpen, setReadOpen] = useState(false);

  const [createMode, setCreateMode] = useState<'write' | 'upload'>('upload');
  const [dropVisual, setDropVisual] = useState<DropVisual>('idle');
  const [uploadMeta, setUploadMeta] = useState<Record<string, unknown> | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  /** When editing, skip TF-IDF until content differs from this snapshot (set in openEdit). */
  const contentSuggestBaselineRef = useRef<string | null>(null);
  /** Ignore stale list responses when productId changes or multiple loads overlap. */
  const knowledgeFetchGen = useRef(0);

  const load = useCallback(async () => {
    if (!productId) return;
    const gen = ++knowledgeFetchGen.current;
    setLoading(true);
    setError(null);
    setKnowledgeOff(false);
    try {
      const list = await client.listProductKnowledge(productId, {
        limit: LIST_LIMIT,
      });
      if (gen !== knowledgeFetchGen.current) return;
      setEntries(list);
    } catch (e) {
      if (gen !== knowledgeFetchGen.current) return;
      setEntries([]);
      if (e instanceof ArmsHttpError && e.status === 503) {
        setKnowledgeOff(true);
      } else {
        setError(e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not load knowledge.');
      }
    } finally {
      if (gen === knowledgeFetchGen.current) setLoading(false);
    }
  }, [client, productId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!productId) {
      knowledgeFetchGen.current += 1;
      setEntries([]);
      setLoading(false);
      setError(null);
      setKnowledgeOff(false);
    }
  }, [productId]);

  useEffect(() => {
    setReadOpen(false);
  }, [selectedId]);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    let cancelled = false;

    if (!productId || formBusy) {
      return undefined;
    }
    // Run for any create/edit with enough text — not only Write tab (import sets Write; this avoids missing a beat if mode lags).
    if (!creating && !editing) {
      return undefined;
    }

    const text = formContent.trim();
    if (text.length < CONTENT_METADATA_MIN_CHARS) {
      setMetadataSuggestError(null);
      return undefined;
    }
    if (
      editing &&
      contentSuggestBaselineRef.current != null &&
      text === contentSuggestBaselineRef.current.trim()
    ) {
      return undefined;
    }

    const categoryTrimmed = formCategory.trim();
    const hasTags = formTags.split(',').some((t) => t.trim().length > 0);
    if (categoryTrimmed !== '' && hasTags) {
      setMetadataSuggestError(null);
      return undefined;
    }

    timer = window.setTimeout(() => {
      void (async () => {
        if (cancelled) return;
        setMetadataSuggestBusy(true);
        setMetadataSuggestError(null);
        try {
          const extraCorpus = entries
            .filter((e) => !(editing && selectedId != null && e.id === selectedId))
            .slice(0, TFIDF_EXTRA_CORPUS_MAX_ENTRIES)
            .map((e) => e.content.slice(0, TFIDF_EXTRA_CORPUS_MAX_CHARS))
            .filter((s) => s.trim().length > 0);
          const res = await fetchProductTfidfSuggestTags(armsEnv, productId, {
            text,
            top_k: 16,
            min_token_len: 2,
            ...(extraCorpus.length ? { extra_corpus: extraCorpus } : {}),
          });
          if (cancelled) return;
          const tags = res.tags ?? [];
          if (tags.length === 0) {
            setMetadataSuggestError(
              'No keywords returned for this text (often stopwords only). Add more specific words or set category/tags manually.',
            );
            return;
          }
          setMetadataSuggestError(null);
          setFormTags(tags.map((t) => t.token).slice(0, 12).join(', '));
          const cat = inferKnowledgeCategoryFromTags(tags);
          if (cat) setFormCategory(cat);
        } catch (e) {
          if (cancelled) return;
          if (e instanceof ArmsHttpError) {
            if (e.status === 403) {
              setMetadataSuggestError(
                'Tag suggestions need arms credentials that allow POST (HTTP Basic with role "read" can list docs but not this endpoint — use a Bearer token or Basic admin).',
              );
            } else if (e.status === 0 || e.code === 'network') {
              setMetadataSuggestError(
                `Browser could not complete the request to ${armsEnv.baseUrl} (${e.message}). ` +
                  'Cross-origin POST needs CORS: set arms env ARMS_CORS_ALLOW_ORIGIN to this page’s exact origin ' +
                  '(e.g. http://localhost:5173 for Bun dev — see docs/setup-guide.md). Then restart arms.',
              );
            } else {
              setMetadataSuggestError(`${e.message} (${e.status})`);
            }
          } else {
            const raw = e instanceof Error ? e.message : String(e);
            if (raw.includes('is not a function')) {
              setMetadataSuggestError(
                'Tag suggestions failed: the UI bundle was out of date (restart `bun run dev` and hard-refresh). If this persists, pull latest Fishtank sources.',
              );
            } else {
              setMetadataSuggestError(
                `Tag suggestions failed: ${raw}. Check VITE_ARMS_URL (using ${armsEnv.baseUrl}).`,
              );
            }
          }
        } finally {
          if (!cancelled) setMetadataSuggestBusy(false);
        }
      })();
    }, CONTENT_METADATA_SUGGEST_MS);

    return () => {
      cancelled = true;
      if (timer !== undefined) window.clearTimeout(timer);
      setMetadataSuggestBusy(false);
    };
  }, [
    productId,
    formContent,
    formCategory,
    formTags,
    creating,
    editing,
    entries,
    selectedId,
    formBusy,
    armsEnv,
  ]);

  const listEntries = useMemo(
    () => filterKnowledgeByQuery(entries, searchInput),
    [entries, searchInput],
  );

  const selected = useMemo(
    () => entries.find((e) => e.id === selectedId) ?? null,
    [entries, selectedId],
  );

  useEffect(() => {
    if (selectedId != null && !entries.some((e) => e.id === selectedId)) {
      setSelectedId(null);
      setEditing(false);
    }
  }, [entries, selectedId]);

  const resetForm = () => {
    setFormContent('');
    setFormCategory('');
    setFormTags('');
    setFormError(null);
    setMetadataSuggestError(null);
    setUploadMeta(null);
    setCreateMode('upload');
    setDropVisual('idle');
    contentSuggestBaselineRef.current = null;
  };

  const openCreate = () => {
    setCreating(true);
    setSelectedId(null);
    setEditing(false);
    resetForm();
  };

  const cancelCreate = () => {
    setCreating(false);
    resetForm();
  };

  useEffect(() => {
    if (createMode !== 'upload') setDropVisual('idle');
  }, [createMode]);

  const importKnowledgeFile = useCallback(async (file: File) => {
    if (!isAcceptedKnowledgeFile(file)) {
      setFormError('Use .md, .markdown, .txt, .pdf, or .docx only.');
      return;
    }
    const fmt = fileSourceFormat(file)!;
    setFormError(null);
    setUploadMeta({
      source_filename: file.name,
      source_format: fmt,
    });
    try {
      if (fmt === 'markdown') {
        const text = await readFileAsText(file);
        setFormContent(text);
      } else {
        const kind = fmt === 'pdf' ? 'PDF' : 'Word';
        setFormContent(
          `# ${file.name}\n\n_${kind} file — knowledge stores plain text. Replace this note with pasted or extracted content._\n`,
        );
      }
      setCreateMode('write');
    } catch {
      setFormError('Could not read that file.');
      setUploadMeta(null);
    }
  }, []);

  const onDropZoneDragEnter = (e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDropVisual(classifyDragDataTransfer(e.dataTransfer));
  };

  const onDropZoneDragLeave = (e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const rel = e.relatedTarget as Node | null;
    if (rel && e.currentTarget.contains(rel)) return;
    setDropVisual('idle');
  };

  const onDropZoneDragOver = (e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const next = classifyDragDataTransfer(e.dataTransfer);
    setDropVisual(next);
    e.dataTransfer.dropEffect = next === 'bad' ? 'none' : 'copy';
  };

  const onDropZoneDrop = (e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDropVisual('idle');
    const files = Array.from(e.dataTransfer.files ?? []).filter(isAcceptedKnowledgeFile);
    if (files.length === 0) {
      setFormError('Drop a .md, .txt, .pdf, or .docx file.');
      return;
    }
    if (files.length > 1) {
      setFormError('Drop one file at a time.');
      return;
    }
    void importKnowledgeFile(files[0]);
  };

  const openEdit = () => {
    if (!selected) return;
    const meta = selected.metadata ?? {};
    const { category, tags } = parseMetaForForm(meta);
    setFormContent(selected.content);
    setFormCategory(category);
    setFormTags(tags);
    setFormError(null);
    setMetadataSuggestError(null);
    setEditing(true);
    contentSuggestBaselineRef.current = selected.content;
  };

  const cancelEdit = () => {
    setEditing(false);
    setFormError(null);
    setMetadataSuggestError(null);
  };

  const submitCreate = async () => {
    if (!productId) return;
    const content = formContent.trim();
    if (!content) {
      setFormError('Content is required.');
      return;
    }
    setFormBusy(true);
    setFormError(null);
    try {
      const meta = mergedCreateMetadata(uploadMeta, formCategory, formTags);
      const row = await client.createProductKnowledge(productId, { content, metadata: meta });
      setCreating(false);
      resetForm();
      await load();
      setSelectedId(row.id);
    } catch (e) {
      setFormError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not create entry.',
      );
    } finally {
      setFormBusy(false);
    }
  };

  const submitEdit = async () => {
    if (!productId || !selected) return;
    const content = formContent.trim();
    if (!content) {
      setFormError('Content is required.');
      return;
    }
    setFormBusy(true);
    setFormError(null);
    try {
      const prev: Record<string, unknown> = { ...(selected.metadata ?? {}) };
      const cat = formCategory.trim();
      if (cat) prev.category = cat;
      else delete prev.category;
      const tagList = formTags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean);
      if (tagList.length) prev.tags = tagList;
      else delete prev.tags;
      await client.patchProductKnowledgeEntry(productId, selected.id, {
        content,
        metadata: prev,
      });
      setEditing(false);
      resetForm();
      await load();
    } catch (e) {
      setFormError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not save changes.',
      );
    } finally {
      setFormBusy(false);
    }
  };

  const confirmDelete = async () => {
    if (!productId || !selected) return;
    if (!window.confirm(`Delete knowledge entry #${selected.id}? This cannot be undone.`)) return;
    setFormBusy(true);
    setFormError(null);
    try {
      await client.deleteProductKnowledgeEntry(productId, selected.id);
      setSelectedId(null);
      setEditing(false);
      await load();
    } catch (e) {
      setFormError(
        e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not delete entry.',
      );
    } finally {
      setFormBusy(false);
    }
  };

  const routesUrl = `${armsEnv.baseUrl.replace(/\/+$/, '')}/api/docs/routes`;

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page">
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <BookOpen size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">Docs</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Product-scoped knowledge from <code className="ft-mono">GET /api/products/…/knowledge</code> — the list loads
              newest first (limit {LIST_LIMIT}); the search box filters those rows by content and by{' '}
              <code className="ft-mono">metadata</code> (category, type, tags). On the <strong>Write</strong> tab (or after file
              import), pause typing for 2 seconds to
              call <code className="ft-mono">POST …/nlp/tfidf-suggest-tags</code> — arms must allow <strong>POST</strong>{' '}
              (Bearer or Basic <strong>admin</strong>; Basic <code className="ft-mono">read</code> returns 403). Category is
              inferred in the browser from tag scores. Operator API references are linked below.
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <button
              type="button"
              className="ft-btn-ghost"
              disabled={loading || knowledgeOff || !productId}
              onClick={() => void load()}
              title="Reload list"
            >
              <RefreshCw size={16} className={loading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
            <button
              type="button"
              className="ft-btn-primary"
              disabled={knowledgeOff || !productId}
              onClick={openCreate}
            >
              <Plus size={16} aria-hidden />
              New doc
            </button>
          </div>
        </div>

        {knowledgeOff ? (
          <div className="ft-banner ft-docs-page__banner" role="status">
            Knowledge is not configured on this arms instance (503). Enable persistence and wire the knowledge service
            to use this module.
          </div>
        ) : null}

        {error ? (
          <div className="ft-banner ft-banner--error ft-docs-page__banner" role="alert">
            {error}
          </div>
        ) : null}

        {!productId ? (
          <p className="ft-muted">Open a workspace to manage docs.</p>
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
                  placeholder="Search knowledge…"
                  aria-label="Search knowledge"
                  style={{ width: '100%' }}
                  disabled={knowledgeOff}
                />
              </div>
              <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                {searchInput.trim()
                  ? `Showing ${listEntries.length} of ${entries.length} — “${searchInput.trim()}”`
                  : 'Newest first'}
              </span>
            </div>

            <div className="ft-docs-page__layout">
              <div>
                <div className="ft-upper-label" style={{ marginBottom: '0.4rem' }}>
                  Entries
                </div>
                <div className="ft-docs-list" aria-busy={loading}>
                  {loading && entries.length === 0 ? (
                    <p className="ft-muted" style={{ padding: '1rem', fontSize: '0.8rem', margin: 0 }}>
                      Loading…
                    </p>
                  ) : entries.length === 0 ? (
                    <p className="ft-muted" style={{ padding: '1rem', fontSize: '0.8rem', margin: 0 }}>
                      No knowledge entries yet. Add one with New doc.
                    </p>
                  ) : listEntries.length === 0 ? (
                    <p className="ft-muted" style={{ padding: '1rem', fontSize: '0.8rem', margin: 0 }}>
                      No entries match this search.
                    </p>
                  ) : (
                    listEntries.map((e) => {
                      const chips = metaChips(e.metadata ?? {});
                      const active = e.id === selectedId && !creating;
                      return (
                        <button
                          key={e.id}
                          type="button"
                          className={`ft-docs-row${active ? ' ft-docs-row--active' : ''}`}
                          onClick={() => {
                            setCreating(false);
                            setEditing(false);
                            setSelectedId(e.id);
                            resetForm();
                          }}
                        >
                          <div className="ft-docs-row__id">#{e.id}</div>
                          <div className="ft-docs-row__preview">{previewLine(e.content)}</div>
                          {chips.length ? (
                            <div className="ft-docs-row__chips">
                              {chips.slice(0, 4).map((c, i) => (
                                <span key={`${i}-${c}`} className="ft-docs-meta-chip">
                                  {c}
                                </span>
                              ))}
                              {chips.length > 4 ? (
                                <span className="ft-docs-meta-chip ft-docs-meta-chip--more">
                                  +{chips.length - 4}
                                </span>
                              ) : null}
                            </div>
                          ) : null}
                          <div className="ft-docs-row__time">{formatRelativeTime(e.updated_at)}</div>
                        </button>
                      );
                    })
                  )}
                </div>
              </div>

              <div>
                <div className="ft-upper-label" style={{ marginBottom: '0.4rem' }}>
                  Preview
                </div>
                <div className="ft-docs-preview">
                  {creating ? (
                    <div>
                      <p className="ft-muted" style={{ fontSize: '0.78rem', marginTop: 0 }}>
                        New entry — drop or browse a file below, or switch to Write to type or paste content.
                      </p>
                      {formError ? (
                        <p className="ft-banner ft-banner--error" style={{ marginBottom: '0.65rem', fontSize: '0.75rem' }}>
                          {formError}
                        </p>
                      ) : null}
                      <div className="ft-docs-seg" role="tablist" aria-label="New document source">
                        <button
                          type="button"
                          role="tab"
                          aria-selected={createMode === 'write'}
                          className={`ft-docs-seg__btn${createMode === 'write' ? ' ft-docs-seg__btn--active' : ''}`}
                          onClick={() => setCreateMode('write')}
                        >
                          Write
                        </button>
                        <button
                          type="button"
                          role="tab"
                          aria-selected={createMode === 'upload'}
                          className={`ft-docs-seg__btn${createMode === 'upload' ? ' ft-docs-seg__btn--active' : ''}`}
                          onClick={() => setCreateMode('upload')}
                        >
                          Upload
                        </button>
                      </div>
                      {createMode === 'upload' ? (
                        <div>
                          <input
                            ref={fileInputRef}
                            type="file"
                            className="ft-docs-file-input-hidden"
                            accept=".md,.markdown,.txt,.pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/markdown,text/plain"
                            aria-hidden
                            tabIndex={-1}
                            onChange={(ev) => {
                              const f = ev.target.files?.[0];
                              if (f) void importKnowledgeFile(f);
                              ev.target.value = '';
                            }}
                          />
                          <div
                            className={`ft-docs-dropzone ft-docs-dropzone--${dropVisual}`}
                            onDragEnter={onDropZoneDragEnter}
                            onDragLeave={onDropZoneDragLeave}
                            onDragOver={onDropZoneDragOver}
                            onDrop={onDropZoneDrop}
                            role="region"
                            aria-label="Drop files to import"
                          >
                            <Upload size={28} className="ft-docs-dropzone__icon" aria-hidden />
                            <p className="ft-docs-dropzone__title">Drop files here</p>
                            <p className="ft-docs-dropzone__hint ft-muted">
                              Accepted: <strong>.md</strong>, <strong>.txt</strong>, <strong>.pdf</strong>, <strong>.docx</strong>.
                              Highlight turns{' '}
                              <span className="ft-docs-dropzone__legend ft-docs-dropzone__legend--ok">green</span> when all
                              dragged files match,{' '}
                              <span className="ft-docs-dropzone__legend ft-docs-dropzone__legend--bad">red</span> when any
                              file is unsupported or the selection is mixed.
                            </p>
                          </div>
                          <button
                            type="button"
                            className="ft-btn-ghost"
                            style={{ marginTop: '0.5rem', fontSize: '0.75rem' }}
                            disabled={formBusy}
                            onClick={() => fileInputRef.current?.click()}
                          >
                            Browse files
                          </button>
                          <p className="ft-muted" style={{ fontSize: '0.72rem', marginTop: '0.65rem', lineHeight: 1.45 }}>
                            Markdown and text are read into the editor. PDF and Word add a starter note (replace with pasted
                            text). After import we open the Write tab to edit before Create.
                          </p>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginTop: '0.75rem' }}>
                            <button type="button" className="ft-btn-ghost" disabled={formBusy} onClick={cancelCreate}>
                              Cancel
                            </button>
                          </div>
                        </div>
                      ) : (
                        <div>
                          <label className="ft-docs-field-label">Content</label>
                          <textarea
                            className="ft-input ft-docs-textarea"
                            value={formContent}
                            onChange={(e) => setFormContent(e.target.value)}
                            rows={10}
                            placeholder="Plan summary, newsletter draft, runbook…"
                            disabled={formBusy}
                          />
                          {metadataSuggestBusy ? (
                            <p className="ft-muted" style={{ fontSize: '0.72rem', margin: '0.35rem 0 0.5rem' }}>
                              Suggesting category & tags (TF-IDF)…
                            </p>
                          ) : null}
                          {metadataSuggestError ? (
                            <p
                              className="ft-banner ft-banner--error"
                              style={{ marginBottom: '0.5rem', fontSize: '0.72rem', lineHeight: 1.4 }}
                              role="status"
                            >
                              {metadataSuggestError}
                            </p>
                          ) : null}
                          <label className="ft-docs-field-label">Category (metadata)</label>
                          <input
                            className="ft-input ft-input--sm"
                            value={formCategory}
                            onChange={(e) => setFormCategory(e.target.value)}
                            placeholder="Auto from content after pause, or set manually"
                            disabled={formBusy}
                            style={{ width: '100%', marginBottom: '0.5rem' }}
                          />
                          <label className="ft-docs-field-label">Tags (comma-separated)</label>
                          <input
                            className="ft-input ft-input--sm"
                            value={formTags}
                            onChange={(e) => setFormTags(e.target.value)}
                            placeholder="Auto after pause in content, or comma-separated"
                            disabled={formBusy}
                            style={{ width: '100%', marginBottom: '0.75rem' }}
                          />
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
                            <button
                              type="button"
                              className="ft-btn-primary"
                              disabled={formBusy}
                              onClick={() => void submitCreate()}
                            >
                              Create
                            </button>
                            <button type="button" className="ft-btn-ghost" disabled={formBusy} onClick={cancelCreate}>
                              Cancel
                            </button>
                          </div>
                        </div>
                      )}
                    </div>
                  ) : selected ? (
                    <div>
                      <div className="ft-docs-preview__head">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
                          <FileText size={16} className="ft-muted" aria-hidden />
                          <span style={{ fontWeight: 600, fontSize: '0.88rem' }}>Entry #{selected.id}</span>
                        </div>
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.35rem' }}>
                          {!editing ? (
                            <>
                              <button
                                type="button"
                                className="ft-btn-ghost"
                                style={{ fontSize: '0.72rem' }}
                                disabled={formBusy || knowledgeOff}
                                onClick={() => setReadOpen(true)}
                              >
                                <Eye size={14} aria-hidden />
                                Read
                              </button>
                              <button
                                type="button"
                                className="ft-btn-ghost"
                                style={{ fontSize: '0.72rem' }}
                                disabled={formBusy || knowledgeOff}
                                onClick={openEdit}
                              >
                                <Pencil size={14} aria-hidden />
                                Edit
                              </button>
                              <button
                                type="button"
                                className="ft-btn-ghost"
                                style={{ fontSize: '0.72rem' }}
                                disabled={formBusy || knowledgeOff}
                                onClick={() => void confirmDelete()}
                              >
                                <Trash2 size={14} aria-hidden />
                                Delete
                              </button>
                            </>
                          ) : null}
                        </div>
                      </div>
                      {metaChips(selected.metadata ?? {}).length ? (
                        <div className="ft-docs-row__chips" style={{ marginBottom: '0.65rem' }}>
                          {metaChips(selected.metadata ?? {}).map((c, i) => (
                            <span key={`${i}-${c}`} className="ft-docs-meta-chip">
                              {c}
                            </span>
                          ))}
                        </div>
                      ) : null}
                      <p className="ft-muted" style={{ fontSize: '0.68rem', margin: '0 0 0.75rem' }}>
                        Updated {formatRelativeTime(selected.updated_at)}
                        {selected.task_id ? (
                          <>
                            {' '}
                            · task <code className="ft-mono">{selected.task_id}</code>
                          </>
                        ) : null}
                      </p>
                      {formError ? (
                        <p className="ft-banner ft-banner--error" style={{ marginBottom: '0.65rem', fontSize: '0.75rem' }}>
                          {formError}
                        </p>
                      ) : null}
                      {editing ? (
                        <>
                          <label className="ft-docs-field-label">Content</label>
                          <textarea
                            className="ft-input ft-docs-textarea"
                            value={formContent}
                            onChange={(e) => setFormContent(e.target.value)}
                            rows={10}
                            disabled={formBusy}
                          />
                          {metadataSuggestBusy ? (
                            <p className="ft-muted" style={{ fontSize: '0.72rem', margin: '0.35rem 0 0.5rem' }}>
                              Suggesting category & tags (TF-IDF)…
                            </p>
                          ) : null}
                          {metadataSuggestError ? (
                            <p
                              className="ft-banner ft-banner--error"
                              style={{ marginBottom: '0.5rem', fontSize: '0.72rem', lineHeight: 1.4 }}
                              role="status"
                            >
                              {metadataSuggestError}
                            </p>
                          ) : null}
                          <label className="ft-docs-field-label">Category</label>
                          <input
                            className="ft-input ft-input--sm"
                            value={formCategory}
                            onChange={(e) => setFormCategory(e.target.value)}
                            placeholder="Auto from content after pause, or set manually"
                            disabled={formBusy}
                            style={{ width: '100%', marginBottom: '0.5rem' }}
                          />
                          <label className="ft-docs-field-label">Tags</label>
                          <input
                            className="ft-input ft-input--sm"
                            value={formTags}
                            onChange={(e) => setFormTags(e.target.value)}
                            placeholder="Auto after pause in content, or comma-separated"
                            disabled={formBusy}
                            style={{ width: '100%', marginBottom: '0.75rem' }}
                          />
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
                            <button
                              type="button"
                              className="ft-btn-primary"
                              disabled={formBusy}
                              onClick={() => void submitEdit()}
                            >
                              Save
                            </button>
                            <button type="button" className="ft-btn-ghost" disabled={formBusy} onClick={cancelEdit}>
                              Cancel
                            </button>
                          </div>
                        </>
                      ) : (
                        <pre className="ft-docs-body">{selected.content}</pre>
                      )}
                    </div>
                  ) : (
                    <p className="ft-muted" style={{ fontSize: '0.82rem', margin: 0 }}>
                      Select an entry to preview, or use <strong>New doc</strong> to add knowledge for{' '}
                      {activeWorkspace?.name ?? 'this product'}.
                    </p>
                  )}
                </div>

                <div className="ft-docs-operator">
                  <div className="ft-upper-label" style={{ marginBottom: '0.4rem' }}>
                    Operator references
                  </div>
                  <ul className="ft-docs-operator__list">
                    <li>
                      <a href={routesUrl} target="_blank" rel="noreferrer" className="ft-docs-operator__link">
                        <ExternalLink size={14} aria-hidden />
                        Route catalog (JSON)
                      </a>
                      <span className="ft-muted"> — same inventory as running arms</span>
                    </li>
                    <li className="ft-muted" style={{ fontSize: '0.78rem', lineHeight: 1.45 }}>
                      OpenAPI 3.1 (import into Swagger / Redoc):{' '}
                      <code className="ft-mono">docs/openapi/arms-openapi.yaml</code> in the CloseLoop repo.
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
      {selected ? (
        <MarkdownReadModal
          open={readOpen}
          onClose={() => setReadOpen(false)}
          title={`Entry #${selected.id}`}
          content={selected.content}
        />
      ) : null}
    </div>
  );
}
