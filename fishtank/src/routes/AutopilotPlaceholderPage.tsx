import { useNavigate } from 'react-router-dom';
import { ChevronLeft } from 'lucide-react';

/** Ideas, swipe deck, schedules, and preference model will plug in here (see fishtank-ui-todos). */
export function AutopilotPlaceholderPage() {
  const navigate = useNavigate();
  return (
    <div className="ft-screen">
      <header className="ft-border-b" style={{ padding: '1rem', background: 'var(--mc-bg-secondary)' }}>
        <button type="button" className="ft-btn-ghost" onClick={() => navigate(-1)} style={{ display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}>
          <ChevronLeft size={18} />
          Back
        </button>
      </header>
      <main className="ft-container" style={{ paddingBlock: '2rem' }}>
        <h1 style={{ fontSize: '1.35rem', fontWeight: 700, marginBottom: '0.5rem' }}>Autopilot</h1>
        <p className="ft-muted" style={{ maxWidth: '36rem', lineHeight: 1.6 }}>
          Entry point for ideas, swipe deck, maybe pool, research cycles, and product schedules. Wire these views to the arms autopilot routes when you are ready to ship the full flow.
        </p>
      </main>
    </div>
  );
}
