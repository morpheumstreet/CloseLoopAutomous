import { Link, useParams } from 'react-router-dom';
import { BookOpen, Construction } from 'lucide-react';

const COPY: Record<string, { title: string; blurb: string; designRef: string }> = {
  content: {
    title: 'Content',
    blurb: 'Content pipeline and drafts — wire to catalog / autopilot APIs when exposed.',
    designRef: '§2 Other / extensible',
  },
  approvals: {
    title: 'Approvals',
    blurb: 'Approval gates and sign-off — connect plan approve/reject and merge policy UX.',
    designRef: '§2 Other / extensible',
  },
  council: {
    title: 'Council',
    blurb: 'Multi-agent review surface — placeholder until convoy / operator queue patterns land.',
    designRef: '§2 Other / extensible',
  },
  calendar: {
    title: 'Calendar',
    blurb: 'Cron and recurring jobs — next: product schedule API + month/grid view ([D] in fishtank-ui-todos).',
    designRef: '§2 Calendar',
  },
  projects: {
    title: 'Projects',
    blurb: 'High-level initiatives — use workspace dashboard + product detail; extend with progress cards.',
    designRef: '§2 Projects',
  },
  memory: {
    title: 'Memory',
    blurb: 'Long-term journal — aggregate chat, knowledge, and ops log when backends unify.',
    designRef: '§2 Memories',
  },
  docs: {
    title: 'Docs',
    blurb: 'Searchable artifacts — wire knowledge CRUD and previews ([G]).',
    designRef: '§2 Docs',
  },
  people: {
    title: 'People',
    blurb: 'Human operators and roles — optional; most agent roster lives under Agents / Team.',
    designRef: '§2 Other',
  },
  office: {
    title: 'Office',
    blurb: '2D / pixel activity layer — fun optional view ([I]).',
    designRef: '§2 Office',
  },
  team: {
    title: 'Team',
    blurb: 'Org chart and mission line — extend Agents with hierarchy and charter copy ([H]).',
    designRef: '§2 Team',
  },
  system: {
    title: 'System',
    blurb: 'Settings, connection, and env — About + arms connection pill ([K]).',
    designRef: '§4 NFR',
  },
  radar: {
    title: 'Radar',
    blurb: 'Signals and alerts — feed filters and ops log deep links.',
    designRef: '§2 Other',
  },
  factory: {
    title: 'Factory',
    blurb: 'Batch / factory runs — map to convoy and dispatch waves ([J]).',
    designRef: '§2 Other',
  },
  pipeline: {
    title: 'Pipeline',
    blurb: 'End-to-end delivery — merge queue, PR flow, CI hooks ([J]).',
    designRef: '§2 Other',
  },
  feedback: {
    title: 'Feedback',
    blurb: 'Product feedback capture — autopilot feedback endpoints ([J]).',
    designRef: '§2 Other',
  },
};

const FALLBACK = { title: 'Module', blurb: 'Placeholder — see docs/fishtank-ui-todos.md.', designRef: '§2' };

export function WorkspaceModulePlaceholder({ segment }: { segment: string }) {
  const { productId } = useParams<{ productId: string }>();
  const meta = COPY[segment] ?? FALLBACK;

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '1.25rem', overflow: 'auto' }}>
      <div
        style={{
          maxWidth: '32rem',
          border: '1px dashed var(--mc-border)',
          borderRadius: 'var(--ft-radius-sm)',
          padding: '1.25rem',
          background: 'var(--mc-bg-secondary)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.75rem' }}>
          <Construction size={20} className="ft-muted" aria-hidden />
          <h1 style={{ fontSize: '1.1rem', fontWeight: 700, margin: 0 }}>{meta.title}</h1>
        </div>
        <p style={{ fontSize: '0.875rem', lineHeight: 1.5, margin: '0 0 0.75rem' }}>{meta.blurb}</p>
        <p className="ft-muted" style={{ fontSize: '0.75rem', margin: '0 0 1rem', display: 'flex', alignItems: 'center', gap: '0.35rem' }}>
          <BookOpen size={14} aria-hidden />
          Design: <code className="ft-mono">{meta.designRef}</code> — see <code className="ft-mono">docs/fishtank-ui-todos.md</code>
        </p>
        {productId ? (
          <Link to={`/p/${productId}/tasks`} className="ft-btn-primary" style={{ display: 'inline-block', textDecoration: 'none' }}>
            Back to Tasks
          </Link>
        ) : null}
      </div>
    </div>
  );
}
