type Props = {
  isOnline: boolean;
};

/** Reflects last successful GET /api/health against the configured arms base URL. */
export function BackendConnectionPill({ isOnline }: Props) {
  return (
    <div
      className={`ft-online-pill ${isOnline ? 'ft-online-pill--on' : 'ft-online-pill--off'}`}
      title={isOnline ? 'Connected to arms' : 'Cannot reach arms (health check failed)'}
    >
      <span
        className={`ft-dot ${isOnline ? 'ft-dot--pulse' : ''}`}
        style={{ background: isOnline ? 'var(--mc-accent-green)' : 'var(--mc-accent-red)' }}
      />
      {isOnline ? 'ONLINE' : 'OFFLINE'}
    </div>
  );
}
