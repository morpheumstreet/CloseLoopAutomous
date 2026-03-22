import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ChevronLeft, ChevronRight, Lightbulb, ListOrdered, RefreshCw, Sparkles } from 'lucide-react';
import { ArmsHttpError, type ArmsClient } from '../api/armsClient';
import type { ApiIdea } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';
import { useIdeationBuckets } from '../hooks/useIdeationBuckets';
import { firstResolvedBucketValue } from '../lib/ideationBucketPreferences';
import { DEFAULT_IDEATION_BUCKET, IDEATION_SOP_NUMBERS, type IdeationBucketValue } from '../lib/ideaCategories';
import { IDEATION_SOPS, IDEATION_SOP_SYSTEM_STEPS, type IdeationSopDefinition } from '../lib/ideationSops';

const SOP_WIZARD_STEP_LABELS = ['Framing & prep', 'Session agenda', 'Rules & evaluation', 'Deliverables'] as const;

type SopWorkshopProps = {
  productId: string;
  mission?: string;
  vision?: string;
  client: ArmsClient;
  createTaskForProduct: (
    ideaId: string | null,
    spec: string,
    newIdeaId?: string | null,
    category?: string | null,
  ) => Promise<void>;
  onQueueSuccess: () => void;
  researchEnabled: boolean;
  researchBusy: boolean;
  pipelineBusy: boolean;
  pipelineLoading: boolean;
  pipelineStage: string;
  onRunResearch: () => void;
};

function IdeationSopWorkshop({
  productId,
  mission,
  vision,
  client,
  createTaskForProduct,
  onQueueSuccess,
  researchEnabled,
  researchBusy,
  pipelineBusy,
  pipelineLoading,
  pipelineStage,
  onRunResearch,
}: SopWorkshopProps) {
  const buckets = useIdeationBuckets();
  const [step, setStep] = useState(0);
  const [phaseDone, setPhaseDone] = useState<boolean[][]>(() =>
    IDEATION_SOPS.map(() => SOP_WIZARD_STEP_LABELS.map(() => false)),
  );

  const [captureSpec, setCaptureSpec] = useState('');
  const [suggestedIdeaId, setSuggestedIdeaId] = useState('');
  const [useSuggestedId, setUseSuggestedId] = useState(false);
  const [manualIdeaId, setManualIdeaId] = useState('');
  const [suggestBusy, setSuggestBusy] = useState(false);
  const [submitBusy, setSubmitBusy] = useState(false);
  const [workshopError, setWorkshopError] = useState<string | null>(null);
  const [ideaCategory, setIdeaCategory] = useState<IdeationBucketValue>(DEFAULT_IDEATION_BUCKET);

  useEffect(() => {
    setIdeaCategory((current) => {
      if (buckets.length === 0) return current;
      if (!buckets.some((b) => b.value === current)) return buckets[0]!.value;
      return current;
    });
  }, [buckets]);

  const activeBucket = useMemo(() => buckets.find((b) => b.value === ideaCategory), [buckets, ideaCategory]);

  const sopIdx = useMemo(() => {
    const n = activeBucket?.sop ?? 1;
    const idx = n - 1;
    if (idx < 0) return 0;
    if (idx >= IDEATION_SOPS.length) return IDEATION_SOPS.length - 1;
    return idx;
  }, [activeBucket?.sop]);

  const prevSopRef = useRef<number | undefined>(undefined);
  useEffect(() => {
    const n = activeBucket?.sop ?? 1;
    if (prevSopRef.current === undefined) {
      prevSopRef.current = n;
      return;
    }
    if (prevSopRef.current !== n) {
      prevSopRef.current = n;
      setStep(0);
      setWorkshopError(null);
    }
  }, [activeBucket?.sop]);

  const sop = IDEATION_SOPS[sopIdx]!;

  const togglePhase = useCallback((si: number, pi: number) => {
    setPhaseDone((prev) => {
      const next = prev.map((row) => [...row]);
      const r = [...(next[si] ?? [])];
      r[pi] = !r[pi];
      next[si] = r;
      return next;
    });
  }, []);

  async function runSuggestId() {
    if (!productId.trim()) return;
    const spec = captureSpec.trim();
    if (!spec) {
      setWorkshopError('Write a spec below first so the server can suggest an id from your text.');
      return;
    }
    setWorkshopError(null);
    setSuggestBusy(true);
    try {
      const res = await client.suggestProductIdeaId(productId, { spec });
      const id = (res.idea_id ?? '').trim();
      if (!id) {
        setWorkshopError('Suggest API returned no idea_id.');
        return;
      }
      setSuggestedIdeaId(id);
      setUseSuggestedId(true);
    } catch (e) {
      setWorkshopError(
        e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Could not suggest idea id.',
      );
    } finally {
      setSuggestBusy(false);
    }
  }

  const canSubmitToQueue = useMemo(() => {
    if (!captureSpec.trim()) return false;
    const mid = manualIdeaId.trim();
    const sid = suggestedIdeaId.trim();
    if (mid) return true;
    return useSuggestedId && sid.length > 0;
  }, [captureSpec, manualIdeaId, suggestedIdeaId, useSuggestedId]);

  async function submitToQueue() {
    if (!productId.trim()) return;
    const spec = captureSpec.trim();
    if (!spec) {
      setWorkshopError('Spec is required — paste your aligned concept (first line is often the title).');
      return;
    }
    const mid = manualIdeaId.trim();
    const sid = suggestedIdeaId.trim();
    const newId = mid || (useSuggestedId ? sid : '');
    if (!newId) {
      setWorkshopError('Enter an idea id, or use Suggest id from spec and keep it checked.');
      return;
    }
    setWorkshopError(null);
    setSubmitBusy(true);
    try {
      await createTaskForProduct(null, spec, newId || null, ideaCategory);
      setCaptureSpec('');
      setManualIdeaId('');
      setSuggestedIdeaId('');
      setUseSuggestedId(false);
      setIdeaCategory(firstResolvedBucketValue(buckets));
      onQueueSuccess();
    } catch (e) {
      setWorkshopError(
        e instanceof ArmsHttpError
          ? `${e.message}${e.code ? ` (${e.code})` : ''} [${e.status}]`
          : e instanceof Error
            ? e.message
            : 'Submit failed.',
      );
    } finally {
      setSubmitBusy(false);
    }
  }

  return (
    <section
      className="ft-panel"
      style={{
        borderRadius: 12,
        border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
        padding: '1rem 1.1rem',
        background: 'var(--mc-surface-raised, rgba(255,255,255,0.03))',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.35rem' }}>
        <ListOrdered size={18} className="ft-muted" aria-hidden />
        <h2 style={{ margin: 0, fontSize: '0.95rem', fontWeight: 600 }}>Structured ideation SOPs</h2>
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.85rem', fontSize: '0.8rem', lineHeight: 1.5, maxWidth: '48rem' }}>
        Four one-page workflows. Mission & Vision are the highest-weighted filter — display them in step one and score
        alignment before anything ships. When you are ready, submit the winning concept to the product queue (creates an
        auto-approved manual idea + planning task via{' '}
        <code className="ft-mono">POST /api/tasks</code> with <code className="ft-mono">product_id</code>,{' '}
        <code className="ft-mono">new_idea_id</code>, and <code className="ft-mono">category</code> set to an ideation-bucket
        slug).
      </p>

      <div
        style={{
          padding: '0.65rem 0.75rem',
          borderRadius: 8,
          marginBottom: '1rem',
          background: 'var(--mc-surface, rgba(0,0,0,0.18))',
          border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.06))',
        }}
      >
        <p style={{ margin: 0, fontSize: '0.72rem', fontWeight: 600, letterSpacing: '0.04em', opacity: 0.85 }}>
          MISSION & VISION (reference)
        </p>
        <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.78rem', lineHeight: 1.5 }}>
          <strong style={{ color: 'inherit' }}>Mission</strong>
          {mission?.trim() ? ` — ${mission.trim()}` : ' — not set for this product (edit under product / team settings).'}
        </p>
        <p className="ft-muted" style={{ margin: '0.35rem 0 0', fontSize: '0.78rem', lineHeight: 1.5 }}>
          <strong style={{ color: 'inherit' }}>Vision</strong>
          {vision?.trim() ? ` — ${vision.trim()}` : ' — not set for this product.'}
        </p>
      </div>

      <label className="ft-field" style={{ display: 'block', marginBottom: '1rem' }}>
        <span className="ft-field-label">Ideation bucket</span>
        <p className="ft-muted" style={{ margin: '0 0 0.4rem', fontSize: '0.72rem', lineHeight: 1.45 }}>
          Choose one after validating Mission & Vision — it classifies the idea for the queue and selects which of the four
          SOP workflows is shown below. Customize this list on the{' '}
          <Link to={`/p/${encodeURIComponent(productId)}/system`}>System</Link> page.
        </p>
        <div className="ft-ideation-buckets--mobile">
          <select
            className="ft-input"
            aria-label="Ideation bucket"
            value={ideaCategory}
            disabled={submitBusy || pipelineBusy}
            onChange={(e) => setIdeaCategory(e.target.value as IdeationBucketValue)}
            style={{ fontSize: '0.8rem' }}
          >
            {IDEATION_SOP_NUMBERS.map((sopNum) => {
              const groupBuckets = buckets.filter((c) => c.sop === sopNum);
              if (groupBuckets.length === 0) return null;
              const sopTitle = IDEATION_SOPS[sopNum - 1]?.shortTitle ?? `SOP ${sopNum}`;
              return (
                <optgroup key={sopNum} label={`SOP ${sopNum} — ${sopTitle}`}>
                  {groupBuckets.map((c) => (
                    <option key={c.value} value={c.value}>
                      {c.label}
                    </option>
                  ))}
                </optgroup>
              );
            })}
          </select>
        </div>

        <div className="ft-ideation-buckets--desktop" role="radiogroup" aria-label="Ideation bucket">
          {IDEATION_SOP_NUMBERS.map((sopNum) => {
            const groupBuckets = buckets.filter((c) => c.sop === sopNum);
            if (groupBuckets.length === 0) return null;
            const sopTitle = IDEATION_SOPS[sopNum - 1]?.shortTitle ?? `SOP ${sopNum}`;
            return (
              <div key={sopNum}>
                <div className="ft-ideation-bucket-group__head">
                  SOP {sopNum} — {sopTitle}
                </div>
                <div className="ft-ideation-bucket-tags">
                  {groupBuckets.map((c) => {
                    const on = ideaCategory === c.value;
                    return (
                      <button
                        key={c.value}
                        type="button"
                        role="radio"
                        aria-checked={on}
                        className={on ? 'ft-btn-primary' : 'ft-btn-ghost'}
                        style={{
                          fontSize: '0.72rem',
                          padding: '0.3rem 0.5rem',
                          maxWidth: 'min(100%, 17rem)',
                          textAlign: 'left',
                          justifyContent: 'flex-start',
                        }}
                        disabled={submitBusy || pipelineBusy}
                        title={c.label}
                        onClick={() => setIdeaCategory(c.value)}
                      >
                        {c.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      </label>

      <p style={{ margin: '0 0 0.4rem', fontSize: '0.72rem', fontWeight: 600, letterSpacing: '0.04em', opacity: 0.85 }}>
        HOW TO USE THE SYSTEM
      </p>
      <ol className="ft-muted" style={{ margin: '0 0 1rem', paddingLeft: '1.2rem', fontSize: '0.78rem', lineHeight: 1.55 }}>
        {IDEATION_SOP_SYSTEM_STEPS.map((t) => (
          <li key={t} style={{ marginBottom: '0.2rem' }}>
            {t}
          </li>
        ))}
      </ol>

      <div
        role="group"
        aria-label="Ideation SOP (follows ideation bucket)"
        style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem', marginBottom: '0.35rem' }}
      >
        {IDEATION_SOPS.map((s, i) => (
          <span
            key={s.key}
            className={i === sopIdx ? 'ft-btn-primary' : 'ft-btn-ghost'}
            style={{
              fontSize: '0.72rem',
              padding: '0.35rem 0.55rem',
              display: 'inline-flex',
              alignItems: 'center',
              pointerEvents: 'none',
              userSelect: 'none',
            }}
            aria-current={i === sopIdx ? 'true' : undefined}
          >
            SOP {s.n}: {s.shortTitle}
          </span>
        ))}
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.85rem', fontSize: '0.7rem', lineHeight: 1.45 }}>
        Change the ideation bucket above to switch SOPs; the highlighted row matches your current bucket.
      </p>

      <p className="ft-muted" style={{ margin: '0 0 0.5rem', fontSize: '0.78rem' }}>
        <strong style={{ color: 'inherit' }}>{sop.fullTitle}</strong>
        <br />
        Covers: {sop.covers}
      </p>

      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '0.5rem', marginBottom: '0.65rem' }}>
        {SOP_WIZARD_STEP_LABELS.map((label, i) => (
          <button
            key={label}
            type="button"
            className={i === step ? 'ft-btn-primary' : 'ft-btn-ghost'}
            style={{ fontSize: '0.72rem', padding: '0.3rem 0.5rem' }}
            onClick={() => setStep(i)}
          >
            {i + 1}. {label}
          </button>
        ))}
      </div>

      <div
        style={{
          minHeight: '8rem',
          padding: '0.75rem 0.85rem',
          borderRadius: 8,
          border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
          background: 'var(--mc-surface, rgba(0,0,0,0.12))',
          marginBottom: '0.75rem',
        }}
      >
        <SopStepPanel sop={sop} step={step} />
        <label
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            gap: '0.45rem',
            marginTop: '0.75rem',
            fontSize: '0.76rem',
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={phaseDone[sopIdx]?.[step] ?? false}
            onChange={() => togglePhase(sopIdx, step)}
            style={{ marginTop: '0.15rem' }}
          />
          <span className="ft-muted">Mark this step complete for your session notes.</span>
        </label>
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '0.5rem', flexWrap: 'wrap' }}>
        <button
          type="button"
          className="ft-btn-ghost"
          style={{ fontSize: '0.78rem', display: 'inline-flex', alignItems: 'center', gap: '0.25rem' }}
          disabled={step <= 0}
          onClick={() => setStep((s) => Math.max(0, s - 1))}
        >
          <ChevronLeft size={16} aria-hidden />
          Back
        </button>
        <span className="ft-muted" style={{ fontSize: '0.72rem' }}>
          Step {step + 1} of {SOP_WIZARD_STEP_LABELS.length}
        </span>
        <button
          type="button"
          className="ft-btn-ghost"
          style={{ fontSize: '0.78rem', display: 'inline-flex', alignItems: 'center', gap: '0.25rem' }}
          disabled={step >= SOP_WIZARD_STEP_LABELS.length - 1}
          onClick={() => setStep((s) => Math.min(SOP_WIZARD_STEP_LABELS.length - 1, s + 1))}
        >
          Next
          <ChevronRight size={16} aria-hidden />
        </button>
      </div>

      <hr
        style={{
          margin: '1.15rem 0',
          border: 'none',
          borderTop: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
        }}
      />

      <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.5rem' }}>
        <Sparkles size={16} className="ft-muted" aria-hidden />
        <h3 style={{ margin: 0, fontSize: '0.88rem', fontWeight: 600 }}>Submit aligned idea to queue</h3>
      </div>
      <p className="ft-muted" style={{ margin: '0 0 0.65rem', fontSize: '0.76rem', lineHeight: 1.45 }}>
        Only pass ideas that score high on Mission/Vision alignment. The bucket is set above; add a non-empty spec and an
        idea id (type one or use <strong>Suggest id from spec</strong> and leave it checked). First line of the spec is
        typically the title. Then open <Link to={`/p/${encodeURIComponent(productId)}/tasks`}>Tasks</Link> to plan and
        dispatch.
      </p>

      {workshopError ? (
        <p className="ft-banner ft-banner--error" role="alert" style={{ fontSize: '0.8rem', marginBottom: '0.65rem' }}>
          {workshopError}
        </p>
      ) : null}

      <label className="ft-field" style={{ display: 'block', marginBottom: '0.65rem' }}>
        <span className="ft-field-label">Final spec (aligned concept)</span>
        <textarea
          className="ft-input"
          rows={7}
          value={captureSpec}
          onChange={(e) => setCaptureSpec(e.target.value)}
          disabled={submitBusy || pipelineBusy}
          placeholder="One-line title&#10;Paragraph: problem, alignment to Mission/Vision, success signal…"
          style={{ resize: 'vertical', width: '100%', minHeight: '140px' }}
        />
      </label>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.75rem', alignItems: 'flex-end', marginBottom: '0.65rem' }}>
        <label className="ft-field" style={{ flex: '1 1 12rem', minWidth: 0 }}>
          <span className="ft-field-label">Idea id</span>
          <input
            className="ft-input"
            type="text"
            value={manualIdeaId}
            onChange={(e) => setManualIdeaId(e.target.value)}
            disabled={submitBusy || pipelineBusy}
            placeholder="e.g. my-feature-idea"
            autoComplete="off"
          />
        </label>
        <button
          type="button"
          className="ft-btn-ghost"
          style={{ fontSize: '0.78rem' }}
          disabled={suggestBusy || submitBusy || pipelineBusy}
          onClick={() => void runSuggestId()}
        >
          {suggestBusy ? 'Suggesting…' : 'Suggest id from spec'}
        </button>
      </div>

      {suggestedIdeaId ? (
        <label
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '0.45rem',
            marginBottom: '0.75rem',
            fontSize: '0.76rem',
            cursor: 'pointer',
          }}
        >
          <input
            type="checkbox"
            checked={useSuggestedId}
            onChange={(e) => setUseSuggestedId(e.target.checked)}
            disabled={submitBusy || pipelineBusy}
          />
          <span>
            Use suggested id: <span className="ft-mono">{suggestedIdeaId}</span>
          </span>
        </label>
      ) : null}

      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: '0.6rem' }}>
        <button
          type="button"
          className="ft-btn-primary"
          disabled={!researchEnabled || pipelineBusy}
          onClick={() => void onRunResearch()}
        >
          {researchBusy ? 'Running research…' : 'Run research'}
        </button>
        <button
          type="button"
          className="ft-btn-primary"
          disabled={submitBusy || pipelineBusy || !canSubmitToQueue}
          onClick={() => void submitToQueue()}
        >
          {submitBusy ? 'Submitting…' : 'Submit to queue'}
        </button>
      </div>
      {!researchEnabled && !pipelineBusy && !pipelineLoading && pipelineStage !== 'research' ? (
        <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.72rem', lineHeight: 1.45 }}>
          Run research is only available when pipeline stage is <code className="ft-mono">research</code>.
        </p>
      ) : null}
      {pipelineStage === 'research' && !researchEnabled && !researchBusy && !pipelineLoading ? (
        <p className="ft-muted" style={{ margin: '0.5rem 0 0', fontSize: '0.72rem' }}>
          Wait for the current pipeline action to finish.
        </p>
      ) : null}
    </section>
  );
}

function SopStepPanel({ sop, step }: { sop: IdeationSopDefinition; step: number }) {
  if (step === 0) {
    return (
      <div style={{ fontSize: '0.78rem', lineHeight: 1.55 }}>
        <p style={{ margin: '0 0 0.5rem' }}>
          <strong>Purpose</strong> — {sop.purpose}
        </p>
        <p style={{ margin: '0 0 0.5rem' }}>
          <strong>Scope</strong> — {sop.scope}
        </p>
        <p style={{ margin: '0 0 0.5rem' }}>
          <strong>Participants</strong> — {sop.participants}
        </p>
        <p style={{ margin: 0 }}>
          <strong>Preparation</strong> — {sop.preparation}
        </p>
      </div>
    );
  }
  if (step === 1) {
    return (
      <div style={{ overflowX: 'auto' }}>
        <table className="ft-table" style={{ fontSize: '0.74rem', minWidth: '100%' }}>
          <thead>
            <tr>
              <th>Time</th>
              <th>Phase</th>
              <th>Techniques</th>
              <th>Output</th>
            </tr>
          </thead>
          <tbody>
            {sop.agenda.map((row) => (
              <tr key={`${row.phase}-${row.time}`}>
                <td>{row.time}</td>
                <td>{row.phase}</td>
                <td style={{ maxWidth: '14rem', wordBreak: 'break-word' }}>{row.techniques}</td>
                <td>{row.output}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }
  if (step === 2) {
    return (
      <div>
        <p style={{ margin: '0 0 0.45rem', fontSize: '0.78rem', fontWeight: 600 }}>Ground rules</p>
        <ul className="ft-muted" style={{ margin: '0 0 0.85rem', paddingLeft: '1.1rem', fontSize: '0.76rem', lineHeight: 1.5 }}>
          {sop.groundRules.map((r) => (
            <li key={r} style={{ marginBottom: '0.15rem' }}>
              {r}
            </li>
          ))}
        </ul>
        <p style={{ margin: '0 0 0.35rem', fontSize: '0.78rem', fontWeight: 600 }}>Evaluation criteria (score 1–5)</p>
        <div style={{ overflowX: 'auto' }}>
          <table className="ft-table" style={{ fontSize: '0.74rem', minWidth: '100%' }}>
            <thead>
              <tr>
                <th>Criterion</th>
                <th>Weight</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {sop.evaluation.map((row) => (
                <tr key={row.criterion}>
                  <td style={{ wordBreak: 'break-word' }}>{row.criterion}</td>
                  <td>{row.weight}</td>
                  <td style={{ wordBreak: 'break-word' }}>{row.description}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    );
  }
  return (
    <div style={{ fontSize: '0.78rem', lineHeight: 1.55 }}>
      <p style={{ margin: '0 0 0.5rem' }}>
        <strong>Deliverables & post-session</strong> — {sop.deliverables}
      </p>
      <p style={{ margin: 0 }}>
        <strong>Cadence & tools</strong> — {sop.cadenceTools}
      </p>
    </div>
  );
}

function ideaNeedsSwipe(i: ApiIdea): boolean {
  if (i.decided === true) return false;
  if ((i.task_id ?? '').trim() !== '') return false;
  return true;
}

function stageDescription(stage: string): string {
  switch (stage) {
    case 'research':
      return 'Run research first (button next to Submit to queue below). When it finishes, the product moves to ideation and you can generate ideas.';
    case 'ideation':
      return 'Research is saved on this product. Run ideation to create draft ideas, then review them on Approvals.';
    case 'swipe':
      return 'New drafts are on the swipe queue. Open Approvals to pass, maybe, yes, or now.';
    case 'planning':
    case 'execution':
    case 'review':
    case 'shipped':
      return 'Autopilot has moved past the ideation step for this cycle. Run research again only when the product returns to the research stage.';
    default:
      return 'Check the current stage below before running pipeline actions.';
  }
}

export function MissionIdeationPage() {
  const { productId } = useParams<{ productId: string }>();
  const pid = productId ?? '';
  const { client, productDetail, refreshActiveBoard, boardLoading, createTaskForProduct } = useMissionUi();

  const [ideas, setIdeas] = useState<ApiIdea[]>([]);
  const [ideasLoading, setIdeasLoading] = useState(true);
  const [ideasError, setIdeasError] = useState<string | null>(null);
  const [actionBusy, setActionBusy] = useState<'research' | 'ideation' | null>(null);
  const [pageError, setPageError] = useState<string | null>(null);

  const stage = (productDetail?.stage ?? '').trim().toLowerCase();

  const loadIdeas = useCallback(async () => {
    if (!pid.trim()) return;
    setIdeasError(null);
    setIdeasLoading(true);
    try {
      const list = await client.listProductIdeas(pid);
      list.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
      setIdeas(list);
    } catch (e) {
      setIdeas([]);
      setIdeasError(e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Could not load ideas.');
    } finally {
      setIdeasLoading(false);
    }
  }, [client, pid]);

  useEffect(() => {
    void refreshActiveBoard({ silent: true });
  }, [refreshActiveBoard]);

  useEffect(() => {
    void loadIdeas();
  }, [loadIdeas]);

  const refreshAll = useCallback(async () => {
    setPageError(null);
    await Promise.all([loadIdeas(), refreshActiveBoard({ silent: true })]);
  }, [loadIdeas, refreshActiveBoard]);

  const swipeQueue = useMemo(() => ideas.filter(ideaNeedsSwipe), [ideas]);

  async function runResearch() {
    if (!pid.trim()) return;
    setPageError(null);
    setActionBusy('research');
    try {
      await client.runProductResearch(pid);
      await refreshAll();
    } catch (e) {
      setPageError(
        e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Research request failed.',
      );
    } finally {
      setActionBusy(null);
    }
  }

  async function runIdeation() {
    if (!pid.trim()) return;
    setPageError(null);
    setActionBusy('ideation');
    try {
      await client.runProductIdeation(pid);
      await refreshAll();
    } catch (e) {
      setPageError(
        e instanceof ArmsHttpError ? e.message : e instanceof Error ? e.message : 'Ideation request failed.',
      );
    } finally {
      setActionBusy(null);
    }
  }

  const loading = ideasLoading || boardLoading;
  const researchEnabled = stage === 'research' && !actionBusy && !loading;
  const ideationEnabled = stage === 'ideation' && !actionBusy && !loading;

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, overflow: 'auto', padding: '1rem 1.25rem' }}>
      <div style={{ maxWidth: '56rem', margin: '0 auto', width: '100%', display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
        <header>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: '1rem', flexWrap: 'wrap' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.6rem' }}>
              <span className="ft-muted" aria-hidden>
                <Lightbulb size={22} />
              </span>
              <div>
                <h1 style={{ fontSize: '1.2rem', fontWeight: 700, margin: 0, letterSpacing: '-0.02em' }}>Ideation</h1>
                <p className="ft-muted" style={{ margin: '0.25rem 0 0', fontSize: '0.8rem', lineHeight: 1.45, maxWidth: '40rem' }}>
                  Drive autopilot ideation from the pipeline card; run on-demand research next to{' '}
                  <strong>Submit to queue</strong> below. Calls <code className="ft-mono">POST /api/products/…/research</code> and{' '}
                  <code className="ft-mono">POST /api/products/…/ideation</code>. After ideation, open{' '}
                  <Link to={`/p/${encodeURIComponent(pid)}/approvals`}>Approvals</Link>{' '}
                  to swipe new drafts.
                </p>
              </div>
            </div>
            <button
              type="button"
              className="ft-btn-ghost"
              style={{ fontSize: '0.8rem', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
              disabled={loading || actionBusy != null}
              onClick={() => void refreshAll()}
            >
              <RefreshCw size={14} className={loading || actionBusy != null ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
          </div>
        </header>

        {pageError ? (
          <p style={{ margin: 0, fontSize: '0.85rem', color: 'var(--mc-danger, #dc2626)' }} role="alert">
            {pageError}
          </p>
        ) : null}

        <section
          className="ft-panel"
          style={{
            borderRadius: 12,
            border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
            padding: '1rem 1.1rem',
            background: 'var(--mc-surface-raised, rgba(255,255,255,0.03))',
          }}
        >
          <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'baseline', justifyContent: 'space-between', gap: '0.75rem' }}>
            <h2 style={{ margin: 0, fontSize: '0.95rem', fontWeight: 600 }}>Pipeline stage</h2>
            <span
              className="ft-mono"
              style={{
                fontSize: '0.78rem',
                padding: '0.2rem 0.5rem',
                borderRadius: 6,
                background: 'var(--mc-surface, rgba(0,0,0,0.2))',
                border: '1px solid var(--mc-border-subtle, rgba(255,255,255,0.08))',
              }}
            >
              {stage || '…'}
            </span>
          </div>
          <p className="ft-muted" style={{ margin: '0.6rem 0 0', fontSize: '0.82rem', lineHeight: 1.5 }}>
            {stageDescription(stage)}
          </p>

          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.6rem', marginTop: '1rem' }}>
            <button
              type="button"
              className="ft-btn-primary"
              disabled={!ideationEnabled}
              onClick={() => void runIdeation()}
            >
              {actionBusy === 'ideation' ? 'Running ideation…' : 'Run ideation'}
            </button>
          </div>
          {!ideationEnabled && stage !== 'ideation' && stage !== 'research' ? (
            <p className="ft-muted" style={{ margin: '0.75rem 0 0', fontSize: '0.78rem' }}>
              Run ideation only in the <code className="ft-mono">ideation</code> stage. Run research from the SOP section
              next to <strong>Submit to queue</strong> when stage is <code className="ft-mono">research</code>.
            </p>
          ) : null}
          {stage === 'ideation' && !ideationEnabled && !loading ? (
            <p className="ft-muted" style={{ margin: '0.75rem 0 0', fontSize: '0.78rem' }}>
              Finish loading or wait for the current action to complete.
            </p>
          ) : null}
        </section>

        <IdeationSopWorkshop
          productId={pid}
          mission={productDetail?.mission_statement}
          vision={productDetail?.vision_statement}
          client={client}
          createTaskForProduct={createTaskForProduct}
          onQueueSuccess={() => void refreshAll()}
          researchEnabled={researchEnabled}
          researchBusy={actionBusy === 'research'}
          pipelineBusy={actionBusy != null}
          pipelineLoading={loading}
          pipelineStage={stage}
          onRunResearch={() => void runResearch()}
        />

        <section>
          <h2 style={{ margin: '0 0 0.5rem', fontSize: '0.95rem', fontWeight: 600 }}>Ideas on this product</h2>
          {ideasError ? (
            <p className="ft-banner ft-banner--error" role="alert" style={{ fontSize: '0.82rem' }}>
              {ideasError}
            </p>
          ) : ideasLoading ? (
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.82rem' }}>
              Loading ideas…
            </p>
          ) : (
            <p className="ft-muted" style={{ margin: 0, fontSize: '0.82rem', lineHeight: 1.5 }}>
              <strong style={{ color: 'inherit' }}>{ideas.length}</strong> total ·{' '}
              <strong style={{ color: 'inherit' }}>{swipeQueue.length}</strong> waiting for swipe
              {swipeQueue.length > 0 ? (
                <>
                  {' '}
                  —{' '}
                  <Link to={`/p/${encodeURIComponent(pid)}/approvals`}>Review on Approvals</Link>
                </>
              ) : null}
            </p>
          )}
        </section>
      </div>
    </div>
  );
}
