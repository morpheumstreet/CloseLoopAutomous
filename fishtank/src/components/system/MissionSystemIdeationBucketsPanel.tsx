import { useState } from 'react';
import { Lightbulb } from 'lucide-react';
import { IDEATION_BUCKETS, IDEATION_SOP_NUMBERS, type IdeationBucketValue } from '../../lib/ideaCategories';
import {
  clearIdeationBucketPrefs,
  initialSelectedSetFromStorage,
  writeIdeationBucketPrefs,
} from '../../lib/ideationBucketPreferences';
import { IDEATION_SOPS } from '../../lib/ideationSops';

export function MissionSystemIdeationBucketsPanel() {
  const [selected, setSelected] = useState<Set<IdeationBucketValue>>(() => initialSelectedSetFromStorage());

  function persist(next: Set<IdeationBucketValue>) {
    const slugs = IDEATION_BUCKETS.filter((b) => next.has(b.value)).map((b) => b.value);
    if (slugs.length === 0) return;
    setSelected(next);
    writeIdeationBucketPrefs({ v: 1, selectedSlugs: slugs });
  }

  function toggle(v: IdeationBucketValue) {
    const next = new Set(selected);
    if (next.has(v)) {
      if (next.size <= 1) return;
      next.delete(v);
    } else {
      next.add(v);
    }
    persist(next);
  }

  function selectAll() {
    persist(new Set(IDEATION_BUCKETS.map((b) => b.value)));
  }

  function resetPrefs() {
    clearIdeationBucketPrefs();
    setSelected(new Set(IDEATION_BUCKETS.map((b) => b.value)));
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
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
        <Lightbulb size={18} className="ft-muted" aria-hidden />
        <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
          Ideation buckets
        </h2>
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.75rem', fontSize: '0.8rem', lineHeight: 1.5 }}>
        Each item maps to one of the four SOPs. Mark buckets <strong>Active</strong> (shown on Ideation) or{' '}
        <strong>Inactive</strong> (hidden). On narrow screens you get a list with Active/Inactive buttons; on desktop,
        buckets are pill tags — click a tag to toggle (highlighted = active). At least one must stay active. Stored in
        this browser (<code className="ft-mono">localStorage</code>); arms still receives the slug as{' '}
        <code className="ft-mono">category</code>.
      </p>

      {IDEATION_SOP_NUMBERS.map((sopNum) => {
        const sopMeta = IDEATION_SOPS[sopNum - 1];
        const rows = IDEATION_BUCKETS.filter((b) => b.sop === sopNum);
        return (
          <div key={sopNum} style={{ marginBottom: '0.9rem' }}>
            <div className="ft-ideation-bucket-group__head" style={{ marginBottom: '0.4rem' }}>
              SOP {sopNum} — {sopMeta?.shortTitle ?? '—'}
            </div>

            <div className="ft-sys-ideation-bucket-settings--mobile">
              <ul style={{ margin: 0, padding: 0, listStyle: 'none' }}>
                {rows.map((b) => (
                  <li
                    key={b.value}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      gap: '0.6rem',
                      flexWrap: 'wrap',
                      padding: '0.4rem 0',
                      borderBottom: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.07))',
                    }}
                  >
                    <span style={{ fontSize: '0.78rem', lineHeight: 1.4, flex: '1 1 14rem', minWidth: 0 }}>{b.label}</span>
                    <button
                      type="button"
                      className={selected.has(b.value) ? 'ft-btn-primary' : 'ft-btn-ghost'}
                      style={{ fontSize: '0.72rem', padding: '0.28rem 0.6rem', flexShrink: 0 }}
                      onClick={() => toggle(b.value)}
                    >
                      {selected.has(b.value) ? 'Active' : 'Inactive'}
                    </button>
                  </li>
                ))}
              </ul>
            </div>

            <div className="ft-sys-ideation-bucket-settings--desktop">
              <div className="ft-ideation-bucket-tags" role="group" aria-label={`SOP ${sopNum} buckets`}>
                {rows.map((b) => {
                  const on = selected.has(b.value);
                  return (
                    <button
                      key={b.value}
                      type="button"
                      className={`ft-ideation-bucket-tag${on ? ' ft-ideation-bucket-tag--on' : ''}`}
                      aria-pressed={on}
                      title={on ? 'Active — click to hide on Ideation' : 'Inactive — click to show on Ideation'}
                      onClick={() => toggle(b.value)}
                    >
                      {b.label}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        );
      })}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginTop: '0.5rem' }}>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} onClick={selectAll}>
          Activate all
        </button>
        <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.78rem' }} onClick={resetPrefs}>
          Reset to defaults
        </button>
      </div>
    </section>
  );
}
