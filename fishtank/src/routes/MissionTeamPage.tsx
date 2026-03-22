import { Link } from 'react-router-dom';
import { useCallback, useEffect, useState } from 'react';
import { Bot, Compass, Info, Target, Users } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import { useMissionUi } from '../context/MissionUiContext';
import type { Agent } from '../domain/types';

type TeamOrgLayout = 'outline' | 'org_chart';

type DemoOrgNode = {
  id: string;
  title: string;
  subtitle: string;
  children?: DemoOrgNode[];
};

/** Illustrative hierarchy when product agent-health is empty (no registry / heartbeats). */
const DEMO_ORG: DemoOrgNode = {
  id: 'demo-root',
  title: 'Mission director',
  subtitle: 'Charter, priorities, and cross-stream coordination',
  children: [
    {
      id: 'demo-planning',
      title: 'Planning stream',
      subtitle: 'Specs, scope, and risk before build',
      children: [
        { id: 'demo-spec', title: 'Spec analyst', subtitle: 'Requirements, user stories, acceptance criteria' },
        { id: 'demo-risk', title: 'Risk scout', subtitle: 'Edge cases, failure modes, test matrix' },
      ],
    },
    {
      id: 'demo-delivery',
      title: 'Delivery stream',
      subtitle: 'Implementation through review',
      children: [
        { id: 'demo-build', title: 'Build agent', subtitle: 'Code, integration, local verification' },
        { id: 'demo-verify', title: 'Verification agent', subtitle: 'CI, review loops, quality gates' },
      ],
    },
    {
      id: 'demo-ops',
      title: 'Operations stream',
      subtitle: 'Ship, observe, recover',
      children: [{ id: 'demo-observe', title: 'Observer', subtitle: 'Telemetry, rollbacks, incident hooks' }],
    },
  ],
};

function agentBadge(status: Agent['status']): { label: string; className: string } {
  switch (status) {
    case 'working':
      return { label: 'WORKING', className: 'ft-agent-badge ft-agent-badge--working' };
    case 'offline':
      return { label: 'OFFLINE', className: 'ft-agent-badge ft-agent-badge--offline' };
    default:
      return { label: 'STANDBY', className: 'ft-agent-badge ft-agent-badge--standby' };
  }
}

function DemoOrgChartTopDown({ node }: { node: DemoOrgNode }) {
  const children = node.children;
  return (
    <div className="ft-team-org-chart-node">
      <div className="ft-team-org-chart-card">
        <div style={{ fontWeight: 600, fontSize: '0.82rem', lineHeight: 1.25 }}>{node.title}</div>
        <div className="ft-muted" style={{ fontSize: '0.65rem', marginTop: '0.25rem', lineHeight: 1.35 }}>
          {node.subtitle}
        </div>
      </div>
      {children?.length ? (
        <div className="ft-team-org-chart-subtree">
          <div className="ft-team-org-chart-stem" aria-hidden />
          <div className="ft-team-org-chart-hline" aria-hidden />
          <div className="ft-team-org-chart-branches">
            {children.map((ch) => (
              <div key={ch.id} className="ft-team-org-chart-branch">
                <div className="ft-team-org-chart-stem" aria-hidden />
                <DemoOrgChartTopDown node={ch} />
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function DemoOrgBranch({ node, depth }: { node: DemoOrgNode; depth: number }) {
  return (
    <div
      style={{
        marginTop: depth === 0 ? 0 : '0.65rem',
        marginLeft: depth === 0 ? 0 : '0.5rem',
        paddingLeft: depth === 0 ? 0 : '0.85rem',
        borderLeft: depth === 0 ? 'none' : '2px solid color-mix(in srgb, var(--mc-border) 70%, transparent)',
      }}
    >
      <div
        style={{
          borderRadius: 'var(--ft-radius-sm)',
          border: '1px solid var(--mc-border)',
          background: 'var(--mc-bg-secondary)',
          padding: '0.55rem 0.75rem',
        }}
      >
        <div style={{ fontWeight: 600, fontSize: '0.85rem' }}>{node.title}</div>
        <div className="ft-muted" style={{ fontSize: '0.7rem', marginTop: '0.2rem', lineHeight: 1.35 }}>
          {node.subtitle}
        </div>
      </div>
      {node.children?.length ? (
        <div style={{ marginTop: '0.35rem' }}>
          {node.children.map((ch) => (
            <DemoOrgBranch key={ch.id} node={ch} depth={depth + 1} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

export function MissionTeamPage() {
  const { activeWorkspace, agents, productDetail, client, refreshActiveBoard } = useMissionUi();
  const [orgLayout, setOrgLayout] = useState<TeamOrgLayout>('outline');
  const productId = activeWorkspace?.id ?? '';
  const liveAgents = productId ? agents.filter((a) => a.workspaceId === productId) : [];

  const [missionDraft, setMissionDraft] = useState('');
  const [visionDraft, setVisionDraft] = useState('');
  const [statementsBusy, setStatementsBusy] = useState(false);
  const [statementsError, setStatementsError] = useState<string | null>(null);
  const [statementsSaved, setStatementsSaved] = useState(false);

  useEffect(() => {
    setMissionDraft(productDetail?.mission_statement?.trim() ?? '');
    setVisionDraft(productDetail?.vision_statement?.trim() ?? '');
    setStatementsSaved(false);
  }, [productDetail?.id, productDetail?.mission_statement, productDetail?.vision_statement]);

  const saveStatements = useCallback(async () => {
    if (!productId) return;
    setStatementsError(null);
    setStatementsSaved(false);
    setStatementsBusy(true);
    try {
      const updated = await client.patchProduct(productId, {
        mission_statement: missionDraft.trim(),
        vision_statement: visionDraft.trim(),
      });
      setMissionDraft(updated.mission_statement?.trim() ?? '');
      setVisionDraft(updated.vision_statement?.trim() ?? '');
      setStatementsSaved(true);
      await refreshActiveBoard({ silent: true });
    } catch (e) {
      setStatementsError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Save failed');
    } finally {
      setStatementsBusy(false);
    }
  }, [client, missionDraft, productId, refreshActiveBoard, visionDraft]);

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div style={{ maxWidth: orgLayout === 'org_chart' ? '52rem' : '40rem', width: '100%' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
          <Users size={18} className="ft-muted" aria-hidden />
          <h1 style={{ fontSize: '1.05rem', fontWeight: 700, margin: 0 }}>Team</h1>
        </div>
        <p className="ft-muted" style={{ fontSize: '0.78rem', margin: '0 0 1rem', lineHeight: 1.45 }}>
          Mission org for{' '}
          <strong style={{ color: 'var(--mc-fg, inherit)' }}>{activeWorkspace?.name ?? 'this workspace'}</strong>.
          Live roster comes from agent health; this page always shows a reference layout when the registry is empty.
        </p>

        {productId ? (
          <section
            style={{
              marginBottom: '1.25rem',
              padding: '1rem',
              borderRadius: 'var(--ft-radius-sm)',
              border: '1px solid var(--mc-border)',
              background: 'var(--mc-bg-secondary)',
            }}
          >
            <div className="ft-upper-label" style={{ marginBottom: '0.65rem' }}>
              Mission &amp; vision
            </div>
            <p className="ft-muted" style={{ fontSize: '0.72rem', margin: '0 0 0.85rem', lineHeight: 1.45 }}>
              Optional statements stored on the product via <code className="ft-mono">{'PATCH /api/products/{id}'}</code>. Save with empty fields to
              clear. Shown here for the team; use them in reviews and onboarding as you like.
            </p>
            <div style={{ display: 'grid', gap: '0.85rem' }}>
              <label style={{ display: 'block' }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', fontSize: '0.72rem', fontWeight: 600, marginBottom: '0.35rem' }}>
                  <Target size={14} className="ft-muted" aria-hidden />
                  Mission statement
                </span>
                <textarea
                  className="ft-input"
                  rows={3}
                  value={missionDraft}
                  onChange={(e) => {
                    setMissionDraft(e.target.value);
                    setStatementsSaved(false);
                  }}
                  placeholder="Why this mission exists — outcomes, scope, or north star (optional)"
                  style={{ width: '100%', resize: 'vertical', minHeight: '4.5rem', fontSize: '0.8rem' }}
                />
              </label>
              <label style={{ display: 'block' }}>
                <span style={{ display: 'flex', alignItems: 'center', gap: '0.35rem', fontSize: '0.72rem', fontWeight: 600, marginBottom: '0.35rem' }}>
                  <Compass size={14} className="ft-muted" aria-hidden />
                  Vision statement
                </span>
                <textarea
                  className="ft-input"
                  rows={3}
                  value={visionDraft}
                  onChange={(e) => {
                    setVisionDraft(e.target.value);
                    setStatementsSaved(false);
                  }}
                  placeholder="Where we are headed longer term — optional"
                  style={{ width: '100%', resize: 'vertical', minHeight: '4.5rem', fontSize: '0.8rem' }}
                />
              </label>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '0.5rem', marginTop: '0.75rem' }}>
              <button type="button" className="ft-btn-primary" disabled={statementsBusy} onClick={() => void saveStatements()}>
                {statementsBusy ? 'Saving…' : 'Save statements'}
              </button>
              {statementsSaved ? (
                <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
                  Saved.
                </span>
              ) : null}
            </div>
            {statementsError ? (
              <p className="ft-banner ft-banner--error" role="alert" style={{ margin: '0.65rem 0 0', fontSize: '0.75rem' }}>
                {statementsError}
              </p>
            ) : null}
          </section>
        ) : (
          <p className="ft-muted" style={{ fontSize: '0.72rem', margin: '0 0 1rem' }}>
            Open a workspace to edit mission and vision statements.
          </p>
        )}

        {liveAgents.length > 0 ? (
          <section style={{ marginBottom: '1.25rem' }}>
            <div className="ft-upper-label" style={{ marginBottom: '0.5rem' }}>
              Registered agents
            </div>
            <div
              style={{
                borderRadius: 'var(--ft-radius-sm)',
                border: '1px solid var(--mc-border)',
                background: 'var(--mc-bg-secondary)',
                padding: '0.5rem',
              }}
            >
              {liveAgents.map((agent) => {
                const badge = agentBadge(agent.status);
                return (
                  <div
                    key={agent.id}
                    className="ft-agent-row"
                    style={{ borderRadius: 'var(--ft-radius-sm)', marginBottom: '0.35rem' }}
                  >
                    <Bot size={16} color="var(--mc-accent)" aria-hidden />
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div className="ft-truncate" style={{ fontWeight: 600, fontSize: '0.8rem' }}>
                        {agent.name}
                      </div>
                      <div style={{ fontSize: '0.65rem' }} className="ft-muted">
                        {agent.id}
                      </div>
                    </div>
                    <span className={badge.className}>{badge.label}</span>
                  </div>
                );
              })}
            </div>
          </section>
        ) : null}

        {liveAgents.length === 0 ? (
          <div
            style={{
              display: 'flex',
              alignItems: 'flex-start',
              gap: '0.5rem',
              padding: '0.6rem 0.75rem',
              marginBottom: '0.85rem',
              borderRadius: 'var(--ft-radius-sm)',
              border: '1px solid color-mix(in srgb, var(--mc-accent) 35%, var(--mc-border))',
              background: 'color-mix(in srgb, var(--mc-accent) 6%, var(--mc-bg-secondary))',
              fontSize: '0.75rem',
              lineHeight: 1.45,
            }}
          >
            <Info size={16} className="ft-muted" style={{ flexShrink: 0, marginTop: '0.1rem' }} aria-hidden />
            <div>
              No agent heartbeats for this product yet (or agent health is disabled on the server). The tree below is{' '}
              <strong>illustrative</strong> so you can preview how a mission org might look once agents register.
            </div>
          </div>
        ) : (
          <p className="ft-muted" style={{ fontSize: '0.72rem', margin: '0 0 0.75rem' }}>
            Example structure for onboarding and staffing discussions:
          </p>
        )}

        <div
          style={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '0.5rem',
            marginBottom: '0.5rem',
          }}
        >
          <div className="ft-upper-label" style={{ margin: 0 }}>
            {liveAgents.length === 0 ? 'Demo organization' : 'Reference org (demo)'}
          </div>
          <div className="ft-tabs" style={{ margin: 0 }} role="tablist" aria-label="Organization layout">
            <button
              type="button"
              role="tab"
              aria-selected={orgLayout === 'outline'}
              className={`ft-tab ${orgLayout === 'outline' ? 'ft-tab--active' : ''}`}
              onClick={() => setOrgLayout('outline')}
            >
              Outline
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={orgLayout === 'org_chart'}
              className={`ft-tab ${orgLayout === 'org_chart' ? 'ft-tab--active' : ''}`}
              onClick={() => setOrgLayout('org_chart')}
            >
              Org chart
            </button>
          </div>
        </div>
        {orgLayout === 'outline' ? (
          <DemoOrgBranch node={DEMO_ORG} depth={0} />
        ) : (
          <div
            style={{
              borderRadius: 'var(--ft-radius-sm)',
              border: '1px solid var(--mc-border)',
              background: 'color-mix(in srgb, var(--mc-bg-secondary) 92%, transparent)',
              padding: '1rem 0.75rem 1.15rem',
              overflowX: 'auto',
            }}
          >
            <DemoOrgChartTopDown node={DEMO_ORG} />
          </div>
        )}

        {productId ? (
          <p style={{ fontSize: '0.72rem', marginTop: '1.1rem' }} className="ft-muted">
            Heartbeats and status:{' '}
            <Link
              to={`/p/${productId}/agents`}
              style={{ color: 'var(--mc-accent)', textDecoration: 'underline', textUnderlineOffset: '2px' }}
            >
              Agents
            </Link>
          </p>
        ) : null}
      </div>
    </div>
  );
}
