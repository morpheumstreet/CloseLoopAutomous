import { useEffect, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, X } from 'lucide-react';
import { type NavEntry, workspacePath } from '../../lib/missionNavCatalog';
import { useMissionNavPreferences } from '@/hooks/useMissionNavPreferences';

type Props = {
  open: boolean;
  onClose: () => void;
  productId: string;
  showResearchHubNav: boolean;
  onOpenAbout: () => void;
  boardSearch: string;
  onBoardSearchChange: (v: string) => void;
};

export function MissionCommandPalette({
  open,
  onClose,
  productId,
  showResearchHubNav,
  onOpenAbout,
  boardSearch,
  onBoardSearchChange,
}: Props) {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);

  const { paletteEntries: entries } = useMissionNavPreferences(showResearchHubNav);

  const q = boardSearch.trim().toLowerCase();
  const filtered = useMemo(() => {
    if (!q) return entries;
    return entries.filter(
      (e) => e.label.toLowerCase().includes(q) || e.id.toLowerCase().includes(q),
    );
  }, [entries, q]);

  useEffect(() => {
    if (!open) return;
    const t = window.setTimeout(() => inputRef.current?.focus(), 0);
    const onEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onEsc);
    return () => {
      window.clearTimeout(t);
      window.removeEventListener('keydown', onEsc);
    };
  }, [open, onClose]);

  if (!open) return null;

  function handleEntry(entry: NavEntry) {
    if (entry.kind === 'about') {
      onOpenAbout();
      onClose();
      return;
    }
    if (!productId) return;
    if (entry.kind === 'workspace') navigate(workspacePath(productId, entry.segment));
    else navigate(entry.to);
    onClose();
  }

  return (
    <div
      className="ft-modal-root ft-mc-command-palette-root"
      role="dialog"
      aria-modal="true"
      aria-labelledby="ft-mc-cmd-title"
    >
      <button type="button" className="ft-modal-backdrop" aria-label="Close" onClick={onClose} />
      <div className="ft-modal-panel ft-mc-command-palette-panel">
        <div className="ft-modal-head ft-mc-command-palette-head">
          <h2 id="ft-mc-cmd-title" className="ft-mc-command-palette-title">
            Mission palette
          </h2>
          <button type="button" className="ft-btn-icon" title="Close" aria-label="Close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="ft-mc-command-palette-search">
          <Search size={16} className="ft-mc-command-palette-search-icon" aria-hidden />
          <input
            ref={inputRef}
            type="search"
            className="ft-mc-command-palette-input"
            placeholder="Search tasks, ideas, specs…"
            aria-label="Filter palette and board"
            value={boardSearch}
            onChange={(e) => onBoardSearchChange(e.target.value)}
          />
        </div>

        <div className="ft-mc-command-palette-actions" role="group" aria-label="Workspace and global destinations">
          {filtered.length === 0 ? (
            <p className="ft-mc-command-palette-empty">No matches.</p>
          ) : (
            filtered.map((entry) => {
              const Icon = entry.icon;
              return (
                <button
                  key={entry.id}
                  type="button"
                  className="ft-mc-command-palette-item"
                  onClick={() => handleEntry(entry)}
                  disabled={entry.kind === 'workspace' && !productId}
                >
                  <Icon size={16} aria-hidden className="ft-mc-command-palette-item-icon" />
                  <span className="ft-mc-command-palette-item-label">{entry.label}</span>
                </button>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
