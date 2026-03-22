import { useNavigate } from 'react-router-dom';

export function NotFoundPage() {
  const navigate = useNavigate();
  return (
    <div className="ft-screen" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '2rem' }}>
      <div style={{ textAlign: 'center', maxWidth: '24rem' }}>
        <h1 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '0.35rem' }}>404</h1>
        <p className="ft-muted" style={{ marginBottom: '1.25rem' }}>
          This route does not exist in Fishtank.
        </p>
        <button type="button" className="ft-btn-primary" onClick={() => navigate('/')}>
          Go to dashboard
        </button>
      </div>
    </div>
  );
}
