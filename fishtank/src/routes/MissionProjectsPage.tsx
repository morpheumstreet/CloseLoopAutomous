import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiOperationLogEntry } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import type { WorkspaceStats } from '../domain/types';
import { formatRelativeTime } from '../lib/time';

const LOG_LIMIT = 2000;

function formatStageLabel(stage?: string): string {
  const s = stage?.trim();
  if (!s) return 'No stage';
  return s
    .replace(/_/g, ' ')
    .split(/\s+/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(' ');
}

function initiatorLabel(actor: string | undefined): string {
  const a = actor?.trim();
  if (!a) return 'Unknown';
  if (a.toLowerCase() === 'http') return 'Mission Control';
  return a;
}

function buildJournalHints(entries: ApiOperationLogEntry[]) {
  const lastJournalAtByProduct = new Map<string, string>();
  for (const e of entries) {
    const pid = e.product_id?.trim();
    const at = e.created_at?.trim();
    if (!pid || !at) continue;
    const prev = lastJournalAtByProduct.get(pid);
    if (!prev || new Date(at).getTime() > new Date(prev).getTime()) lastJournalAtByProduct.set(pid, at);
  }

  const createMetaByProduct = new Map<string, { at: number; actor: string }>();
  for (const e of entries) {
    if (e.action !== 'product.create') continue;
    const id = e.resource_id?.trim();
    const at = e.created_at?.trim();
    if (!id || !at) continue;
    const t = new Date(at).getTime();
    const cur = createMetaByProduct.get(id);
    if (!cur || t < cur.at) createMetaByProduct.set(id, { at: t, actor: e.actor?.trim() ?? '' });
  }

  return { lastJournalAtByProduct, createMetaByProduct };
}

function completionPct(w: WorkspaceStats): number {
  const { total, done } = w.taskCounts;
  if (total === 0) return 0;
  return Math.round((done / total) * 100);
}

function updatedMeta(
  w: WorkspaceStats,
  lastJournalAtByProduct: Map<string, string>,
): { label: string; title: string } {
  const journalAt = lastJournalAtByProduct.get(w.id);
  const productAt = w.productUpdatedAt?.trim();
  if (journalAt && productAt) {
    const jt = new Date(journalAt).getTime();
    const pt = new Date(productAt).getTime();
    if (jt >= pt) {
      return {
        label: formatRelativeTime(journalAt),
        title: 'Latest operations-log entry scoped to this product',
      };
    }
    return {
      label: formatRelativeTime(productAt),
      title: 'No recent journal row; using product updated_at from arms',
    };
  }
  if (journalAt) {
    return {
      label: formatRelativeTime(journalAt),
      title: 'Latest operations-log entry scoped to this product',
    };
  }
  if (productAt) {
    return {
      label: formatRelativeTime(productAt),
      title: 'Using product updated_at from arms',
    };
  }
  return { label: '—', title: 'No timestamp available' };
}

export function MissionProjectsPage() {
  const navigate = useNavigate();
  const { workspaces, listLoading, refreshWorkspaces, fetchOperationsLog } = useMissionUi();
  const [entries, setEntries] = useState<ApiOperationLogEntry[]>([]);
  const [logLoading, setLogLoading] = useState(true);
  const [logError, setLogError] = useState<string | null>(null);

  const loadLog = useCallback(async () => {
    setLogLoading(true);
    setLogError(null);
    try {
      const list = await fetchOperationsLog({ limit: LOG_LIMIT });
      setEntries(list);
    } catch (e) {
      setEntries([]);
      setLogError(e instanceof ArmsHttpError ? `${e.message} (${e.status})` : 'Could not load operations log.');
    } finally {
      setLogLoading(false);
    }
  }, [fetchOperationsLog]);

  useEffect(() => {
    void loadLog();
  }, [loadLog]);

  const { lastJournalAtByProduct, createMetaByProduct } = useMemo(() => buildJournalHints(entries), [entries]);

  const sortedWorkspaces = useMemo(() => {
    return [...workspaces].sort((a, b) => {
      const ja = lastJournalAtByProduct.get(a.id) ?? a.productUpdatedAt ?? '';
      const jb = lastJournalAtByProduct.get(b.id) ?? b.productUpdatedAt ?? '';
      const ta = ja ? new Date(ja).getTime() : 0;
      const tb = jb ? new Date(jb).getTime() : 0;
      return tb - ta;
    });
  }, [workspaces, lastJournalAtByProduct]);

  const busy = listLoading || (logLoading && entries.length === 0);

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-projects-page">
      <div className="ft-projects-page__head">
        <div>
          <h1 className="ft-projects-page__title">Projects</h1>
          <p className="ft-muted ft-projects-page__blurb">
            Every arms product in one place — open a card to jump into that workspace. Progress is done tasks over total;
            “updated” prefers the operations log, then the product record.
          </p>
        </div>
        <button
          type="button"
          className="ft-btn-ghost"
          disabled={busy}
          onClick={() => {
            void refreshWorkspaces();
            void loadLog();
          }}
          title="Reload products and journal hints"
        >
          <RefreshCw size={16} className={busy ? 'ft-spin' : ''} />
          Refresh
        </button>
      </div>

      {logError ? (
        <div className="ft-banner ft-banner--error ft-projects-page__banner" role="alert">
          {logError} Initiator and some “updated” times may be incomplete.
        </div>
      ) : null}

      {busy && workspaces.length === 0 ? (
        <div className="ft-projects-page__skeleton-grid" aria-busy="true" aria-label="Loading projects">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="ft-skeleton ft-project-card-skeleton" />
          ))}
        </div>
      ) : sortedWorkspaces.length === 0 ? (
        <p className="ft-muted">No products yet. Create one from the Mission Control home screen.</p>
      ) : (
        <div className="ft-grid-ws ft-projects-page__grid">
          {sortedWorkspaces.map((w) => (
            <ProjectFleetCard
              key={w.id}
              workspace={w}
              initiator={initiatorLabel(createMetaByProduct.get(w.id)?.actor)}
              updated={updatedMeta(w, lastJournalAtByProduct)}
              onOpen={() => navigate(`/p/${encodeURIComponent(w.id)}/tasks`)}
            />
          ))}
        </div>
      )}
      </div>
    </div>
  );
}

function ProjectFleetCard({
  workspace,
  initiator,
  updated,
  onOpen,
}: {
  workspace: WorkspaceStats;
  initiator: string;
  updated: { label: string; title: string };
  onOpen: () => void;
}) {
  const pct = completionPct(workspace);
  const subtitle = `/${workspace.slug} · ${workspace.taskCounts.total} tasks · ${workspace.agentCounts.working}/${workspace.agentCounts.total} agents active`;

  return (
    <button type="button" className="ft-ws-card ft-project-card" onClick={onOpen}>
      <div className="ft-project-card__top">
        <div className="ft-project-card__title-row">
          <span className="ft-project-card__icon" aria-hidden>
            {workspace.icon}
          </span>
          <div className="ft-project-card__titles">
            <h2 className="ft-truncate ft-project-card__name">{workspace.name}</h2>
            <p className="ft-muted ft-project-card__subtitle ft-truncate" title={subtitle}>
              {subtitle}
            </p>
          </div>
        </div>
        <span className="ft-stage-pill" title="Product stage from arms">
          {formatStageLabel(workspace.stage)}
        </span>
      </div>

      <div className="ft-project-card__progress-block">
        <div className="ft-project-card__progress-labels">
          <span>Progress</span>
          <span className="ft-muted">{pct}%</span>
        </div>
        <div className="ft-project-card__progress-track" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
          <div className="ft-project-card__progress-fill" style={{ width: `${pct}%` }} />
        </div>
      </div>

      <div className="ft-project-card__meta">
        <div className="ft-project-card__meta-row">
          <span className="ft-muted">Initiated by</span>
          <span className="ft-project-card__meta-value">{initiator}</span>
        </div>
        <div className="ft-project-card__meta-row">
          <span className="ft-muted">Updated</span>
          <span className="ft-project-card__meta-value" title={updated.title}>
            {updated.label}
          </span>
        </div>
      </div>
    </button>
  );
}
