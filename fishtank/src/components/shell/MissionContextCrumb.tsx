type Props = {
  workspaceName: string;
  parts: string[];
  className: string;
};

export function MissionContextCrumb({ workspaceName, parts, className }: Props) {
  return (
    <div className={className} aria-label="Current location">
      <span className="ft-mc-context-crumb__ws ft-truncate" title={workspaceName}>
        {workspaceName}
      </span>
      {parts.map((part, i) => (
        <span key={`${i}-${part}`} className="ft-mc-context-crumb__tail">
          <span className="ft-mc-context-crumb__sep" aria-hidden>
            /
          </span>
          <span className="ft-truncate" title={part}>
            {part}
          </span>
        </span>
      ))}
    </div>
  );
}
