import { useCallback, useEffect, useState } from 'react';
import { Network, Plus, RefreshCw } from 'lucide-react';
import { ArmsClient, ArmsHttpError } from '../../api/armsClient';
import type { ApiGatewayEndpoint } from '../../api/armsTypes';
import { CreateGatewayEndpointModal } from './CreateGatewayEndpointModal';
import { EditGatewayEndpointModal } from './EditGatewayEndpointModal';

export function MissionSystemGatewayEndpointsPanel({
  client,
  defaultProductId,
}: {
  client: ArmsClient;
  defaultProductId?: string;
}) {
  const [rows, setRows] = useState<ApiGatewayEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);

  const [editTarget, setEditTarget] = useState<ApiGatewayEndpoint | null>(null);

  const load = useCallback(async () => {
    setError(null);
    setLoading(true);
    try {
      const list = await client.listGatewayEndpoints();
      setRows(list);
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setError(e.message);
      } else {
        setError(e instanceof Error ? e.message : 'Could not load gateway endpoints');
      }
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [client]);

  useEffect(() => {
    void load();
  }, [load]);

  function openEdit(row: ApiGatewayEndpoint) {
    setEditTarget(row);
    setNote(null);
  }

  async function onDelete(id: string) {
    if (!window.confirm(`Delete gateway endpoint ${id}? This cannot be undone.`)) return;
    setBusy(true);
    setNote(null);
    setError(null);
    try {
      await client.deleteGatewayEndpoint(id);
      setNote('Deleted.');
      if (editTarget?.id === id) setEditTarget(null);
      await load();
    } catch (err) {
      setNote(err instanceof ArmsHttpError ? err.message : 'Delete failed.');
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
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
          <Network size={18} className="ft-muted" aria-hidden />
          <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
            Agent Gateways
          </h2>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button
            type="button"
            className="ft-btn-primary"
            style={{ fontSize: '0.75rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
            disabled={loading || busy}
            onClick={() => setCreateModalOpen(true)}
          >
            <Plus size={14} aria-hidden />
            Add endpoint
          </button>
          <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem' }} disabled={loading || busy} onClick={() => void load()}>
            <RefreshCw size={14} className={loading ? 'ft-spin' : ''} aria-hidden />
            Refresh
          </button>
        </div>
      </div>
      <p className="ft-muted" style={{ margin: '0.5rem 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        Manual dispatch profiles stored in arms (<code className="ft-mono">gateway_endpoints</code>). Register execution agents with{' '}
        <code className="ft-mono">gateway_endpoint_id</code> + <code className="ft-mono">session_key</code>. API responses never include stored tokens; use{' '}
        <code className="ft-mono">has_gateway_token</code> or set a new token here.
      </p>
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

      <div style={{ overflowX: 'auto', marginBottom: '1rem' }}>
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
                Driver
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                URL
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Connect
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Token
              </th>
              <th className="ft-muted" style={{ padding: '0.35rem 0.5rem', fontWeight: 600 }}>
                Product
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
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }} className="ft-mono">
                  {r.driver}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', wordBreak: 'break-all', maxWidth: '14rem' }}>
                  {r.gateway_url || '—'}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top', fontSize: '0.7rem', maxWidth: '12rem' }}>
                  {r.connection_status === 'pairing_required' ? (
                    <span className="ft-banner ft-banner--warn" style={{ display: 'block', padding: '0.35rem 0.45rem', textAlign: 'left' }}>
                      Pairing required
                      {r.pairing_request_id ? (
                        <>
                          <br />
                          <code className="ft-mono">openclaw devices approve {r.pairing_request_id}</code>
                        </>
                      ) : null}
                    </span>
                  ) : r.connection_status ? (
                    <span className="ft-muted">{r.connection_status}</span>
                  ) : (
                    '—'
                  )}
                </td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }}>{r.has_gateway_token ? '•••• (set)' : '(none)'}</td>
                <td style={{ padding: '0.4rem 0.5rem', verticalAlign: 'top' }} className="ft-mono">
                  {r.product_id || '—'}
                </td>
                <td style={{ padding: '0.4rem 0', verticalAlign: 'top', whiteSpace: 'nowrap' }}>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem', marginRight: '0.35rem' }} disabled={busy} onClick={() => openEdit(r)}>
                    Edit
                  </button>
                  <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.7rem' }} disabled={busy} onClick={() => void onDelete(r.id)}>
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && rows.length === 0 ? <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.8rem' }}>No rows yet.</p> : null}
      </div>

      <CreateGatewayEndpointModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        client={client}
        defaultProductId={defaultProductId}
        onCreated={async () => {
          setNote('Gateway endpoint created.');
          setError(null);
          await load();
        }}
      />

      <EditGatewayEndpointModal
        open={editTarget !== null}
        endpoint={editTarget}
        onClose={() => setEditTarget(null)}
        client={client}
        onSaved={async () => {
          setNote('Saved.');
          setError(null);
          await load();
        }}
      />
    </section>
  );
}
