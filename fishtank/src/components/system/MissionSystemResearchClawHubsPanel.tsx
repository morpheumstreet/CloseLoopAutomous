import { useCallback, useEffect, useState } from 'react';
import { FlaskConical, Plus, RefreshCw } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import type { ApiResearchHub, ApiResearchSystemSettings } from '../../api/armsTypes';
import { CreateResearchHubModal } from './CreateResearchHubModal';

export function MissionSystemResearchClawHubsPanel({ client }: { client: ArmsClient }) {
  const [rows, setRows] = useState<ApiResearchHub[]>([]);
  const [settings, setSettings] = useState<ApiResearchSystemSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [testNote, setTestNote] = useState<Record<string, string | null>>({});

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<ApiResearchHub | null>(null);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      setRows(await client.listResearchHubs());
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setError(e.message);
      } else {
        setError(e instanceof Error ? e.message : 'Could not load research hubs');
      }
      setRows([]);
    }
    try {
      setSettings(await client.getResearchSystemSettings());
    } catch {
      setSettings(null);
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveSettings(next: Partial<ApiResearchSystemSettings>) {
    if (!settings) return;
    setBusy(true);
    setNote(null);
    setError(null);
    try {
      const body: { auto_research_claw_enabled?: boolean; default_research_hub_id?: string } = {};
      if (next.auto_research_claw_enabled !== undefined) body.auto_research_claw_enabled = next.auto_research_claw_enabled;
      if (next.default_research_hub_id !== undefined) body.default_research_hub_id = next.default_research_hub_id;
      const st = await client.patchResearchSystemSettings(body);
      setSettings(st);
      setNote('Research routing saved.');
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Could not save settings.');
    } finally {
      setBusy(false);
    }
  }

  function openEdit(row: ApiResearchHub) {
    setCreateModalOpen(false);
    setEditTarget(row);
    setNote(null);
  }

  async function onDelete(id: string) {
    if (!window.confirm(`Delete research hub ${id}?`)) return;
    setBusy(true);
    setNote(null);
    try {
      await client.deleteResearchHub(id);
      if (editTarget?.id === id) setEditTarget(null);
      setNote('Deleted.');
      await load();
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Delete failed.');
    } finally {
      setBusy(false);
    }
  }

  async function onTest(id: string) {
    setBusy(true);
    setTestNote((m) => ({ ...m, [id]: null }));
    try {
      const res = await client.postResearchHubTest(id, {});
      if (res.ok) {
        setTestNote((m) => ({ ...m, [id]: 'Reachable (GET /api/health ok).' }));
      } else {
        setTestNote((m) => ({ ...m, [id]: res.error ?? 'Failed.' }));
      }
    } catch (err) {
      setTestNote((m) => ({
        ...m,
        [id]: err instanceof ArmsHttpError ? err.message : 'Test failed.',
      }));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      id="ft-research-claw-hubs"
      style={{
        marginTop: '0.75rem',
        padding: '1rem',
        borderRadius: 'var(--ft-radius-sm)',
        border: '1px solid var(--mc-border)',
        background: 'var(--mc-bg-secondary)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.75rem', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem' }}>
          <FlaskConical size={18} className="ft-muted" aria-hidden />
          <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            ResearchClaw hubs (database)
          </h2>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button
            type="button"
            className="ft-btn-primary"
            style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
            disabled={loading || busy}
            onClick={() => {
              setEditTarget(null);
              setCreateModalOpen(true);
              setNote(null);
            }}
          >
            <Plus size={14} aria-hidden />
            Add hub
          </button>
          <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem' }} disabled={loading || busy} onClick={() => void load()}>
            <RefreshCw size={14} className={loading ? 'ft-spin' : ''} aria-hidden />
            Refresh
          </button>
        </div>
      </div>
      <p className="ft-muted" style={{ margin: '0.5rem 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        Configure one or more ResearchClaw-compatible HTTP roots (see{' '}
        <code className="ft-mono">/api/health</code>, <code className="ft-mono">/api/pipeline/start</code>). When{' '}
        <strong>Auto research via ResearchClaw</strong> is on, scheduled and on-demand research runs use the{' '}
        <strong>default hub</strong> instead of the LLM. API keys are stored server-side; responses never echo secrets.
      </p>

      {settings ? (
        <div
          style={{
            marginBottom: '0.85rem',
            padding: '0.65rem 0.75rem',
            borderRadius: 'var(--ft-radius-sm)',
            border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
            background: 'var(--mc-bg-tertiary)',
          }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem', alignItems: 'center' }}>
            <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.4rem', fontSize: '0.78rem', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={settings.auto_research_claw_enabled}
                disabled={busy}
                onChange={(ev) => void saveSettings({ auto_research_claw_enabled: ev.target.checked })}
              />
              Auto research via ResearchClaw
            </label>
            <label style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem', fontSize: '0.78rem' }}>
              <span className="ft-muted">Default hub</span>
              <select
                className="ft-input"
                style={{ fontSize: '0.75rem', minWidth: '10rem' }}
                value={settings.default_research_hub_id || ''}
                disabled={busy}
                onChange={(ev) => void saveSettings({ default_research_hub_id: ev.target.value })}
              >
                <option value="">(none — use LLM / stub)</option>
                {rows.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.display_name || r.id} ({r.id})
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      ) : null}

      {loading && !rows.length ? <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>Loading…</p> : null}
      {error ? (
        <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.5rem 0' }}>
          {error}
        </p>
      ) : null}
      {note ? (
        <p className="ft-muted" style={{ margin: '0.5rem 0', fontSize: '0.8rem' }}>
          {note}
        </p>
      ) : null}

      <CreateResearchHubModal
        open={createModalOpen || editTarget !== null}
        hub={editTarget}
        onClose={() => {
          setCreateModalOpen(false);
          setEditTarget(null);
        }}
        client={client}
        onSuccess={async (mode) => {
          setNote(mode === 'edit' ? 'Saved.' : 'Research hub created.');
          await load();
        }}
      />

      <div style={{ overflowX: 'auto', marginBottom: '0.5rem' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.75rem' }}>
          <thead>
            <tr style={{ textAlign: 'left', borderBottom: '1px solid var(--mc-border)' }}>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem 0.35rem 0', fontWeight: 600 }}>
                Id
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Name
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Base URL
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                API key
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0', fontWeight: 600 }}>
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id} style={{ borderBottom: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.06))' }}>
                <td style={{ padding: '0.4rem 0.5rem 0.4rem 0', verticalAlign: 'top' }} className="ft-mono">
                  {r.id}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', wordBreak: 'break-word' }}>{r.display_name || '—'}</td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', wordBreak: 'break-all', maxWidth: '14rem' }} className="ft-mono">
                  {r.base_url || '—'}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }}>{r.has_api_key ? '•••• (set)' : '(none)'}</td>
                <td style={{ padding: '0.4rem 0', verticalAlign: 'top', whiteSpace: 'nowrap' }}>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem', marginRight: '0.35rem' }} disabled={busy} onClick={() => openEdit(r)}>
                    Edit
                  </button>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem', marginRight: '0.35rem' }} disabled={busy} onClick={() => void onTest(r.id)}>
                    Test
                  </button>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem' }} disabled={busy} onClick={() => void onDelete(r.id)}>
                    Delete
                  </button>
                  {testNote[r.id] ? (
                    <div className="ft-muted" style={{ fontSize: '0.65rem', marginTop: '0.25rem', maxWidth: '12rem' }}>
                      {testNote[r.id]}
                    </div>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && rows.length === 0 ? <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.8rem' }}>No hubs yet.</p> : null}
      </div>
    </section>
  );
}
