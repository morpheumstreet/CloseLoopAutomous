import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { Calendar, ChevronLeft, ChevronRight, Clock, RefreshCw } from 'lucide-react';
import { ArmsHttpError } from '../api/armsClient';
import type { ApiOperationLogEntry, ApiProductSchedule } from '../api/armsTypes';
import { useMissionUi } from '../context/MissionUiContext';

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as const;

function buildMonthCells(
  year: number,
  monthIndex: number,
): { cells: ({ day: number } | null)[]; daysInMonth: number } {
  const first = new Date(year, monthIndex, 1);
  const startPad = first.getDay();
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const cells: ({ day: number } | null)[] = [];
  for (let i = 0; i < startPad; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push({ day: d });
  return { cells, daysInMonth };
}

function matchesLocalDay(iso: string | undefined, year: number, monthIndex: number, day: number): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return false;
  return d.getFullYear() === year && d.getMonth() === monthIndex && d.getDate() === day;
}

function formatDateTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

type ScheduleForm = {
  enabled: boolean;
  cron_expr: string;
  delay_seconds: string;
  spec_json: string;
};

function scheduleToForm(s: ApiProductSchedule): ScheduleForm {
  return {
    enabled: s.enabled,
    cron_expr: (s.cron_expr ?? '').trim(),
    delay_seconds: s.delay_seconds != null && s.delay_seconds > 0 ? String(s.delay_seconds) : '',
    spec_json: s.spec_json?.trim() ? s.spec_json : '{}',
  };
}

export function MissionCalendarPage() {
  const { productId } = useParams<{ productId: string }>();
  const { client, fetchOperationsLog } = useMissionUi();

  const [view, setView] = useState(() => {
    const n = new Date();
    return { year: n.getFullYear(), month: n.getMonth() };
  });
  const [schedule, setSchedule] = useState<ApiProductSchedule | null>(null);
  const [form, setForm] = useState<ScheduleForm>({
    enabled: true,
    cron_expr: '',
    delay_seconds: '',
    spec_json: '{}',
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saveNote, setSaveNote] = useState<string | null>(null);
  const [logEntries, setLogEntries] = useState<ApiOperationLogEntry[]>([]);

  const load = useCallback(async () => {
    if (!productId) return;
    setError(null);
    setSaveNote(null);
    setLoading(true);
    try {
      const [sch, log] = await Promise.all([
        client.getProductSchedule(productId),
        fetchOperationsLog({ product_id: productId, limit: 40 }).catch(() => [] as ApiOperationLogEntry[]),
      ]);
      setSchedule(sch);
      setForm(scheduleToForm(sch));
      setLogEntries(log);
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setError(e.message);
      } else {
        setError(e instanceof Error ? e.message : 'Could not load schedule');
      }
      setSchedule(null);
    } finally {
      setLoading(false);
    }
  }, [client, fetchOperationsLog, productId]);

  useEffect(() => {
    void load();
  }, [load]);

  const scheduleAudit = useMemo(
    () => logEntries.filter((e) => (e.action ?? '').includes('product_schedule')).slice(0, 10),
    [logEntries],
  );

  const { cells } = useMemo(() => buildMonthCells(view.year, view.month), [view.year, view.month]);

  const today = useMemo(() => {
    const n = new Date();
    return { y: n.getFullYear(), m: n.getMonth(), d: n.getDate() };
  }, []);

  const monthTitle = useMemo(
    () =>
      new Date(view.year, view.month, 1).toLocaleString(undefined, { month: 'long', year: 'numeric' }),
    [view.year, view.month],
  );

  const prevMonth = () => {
    setView((v) => {
      if (v.month === 0) return { year: v.year - 1, month: 11 };
      return { year: v.year, month: v.month - 1 };
    });
  };

  const nextMonth = () => {
    setView((v) => {
      if (v.month === 11) return { year: v.year + 1, month: 0 };
      return { year: v.year, month: v.month + 1 };
    });
  };

  const goThisMonth = () => {
    const n = new Date();
    setView({ year: n.getFullYear(), month: n.getMonth() });
  };

  const save = async () => {
    if (!productId) return;
    let specJson: string;
    try {
      const parsed = JSON.parse(form.spec_json.trim() || '{}');
      specJson = JSON.stringify(parsed);
    } catch {
      setSaveNote('Spec JSON must be valid JSON.');
      return;
    }
    const delayRaw = form.delay_seconds.trim();
    const delayNum = delayRaw === '' ? 0 : Number.parseInt(delayRaw, 10);
    if (delayRaw !== '' && (Number.isNaN(delayNum) || delayNum < 0)) {
      setSaveNote('Delay must be a non-negative integer (seconds), or empty for none.');
      return;
    }

    setSaving(true);
    setSaveNote(null);
    setError(null);
    try {
      const updated = await client.patchProductSchedule(productId, {
        enabled: form.enabled,
        spec_json: specJson,
        cron_expr: form.cron_expr.trim(),
        delay_seconds: delayNum,
      });
      setSchedule(updated);
      setForm(scheduleToForm(updated));
      setSaveNote('Schedule saved.');
    } catch (e) {
      if (e instanceof ArmsHttpError) {
        setSaveNote(e.message);
      } else {
        setSaveNote(e instanceof Error ? e.message : 'Save failed');
      }
    } finally {
      setSaving(false);
    }
  };

  if (!productId) {
    return (
      <div className="ft-queue-flex" style={{ flex: 1, padding: '1rem' }}>
        <p className="ft-muted">No workspace selected.</p>
      </div>
    );
  }

  return (
    <div className="ft-queue-flex" style={{ flex: 1, minWidth: 0, minHeight: 0, padding: '0.75rem', overflow: 'auto' }}>
      <div className="ft-docs-page" style={{ maxWidth: '72rem', margin: '0 auto', width: '100%' }}>
        <div className="ft-docs-page__head">
          <div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.45rem', marginBottom: '0.35rem' }}>
              <Calendar size={20} className="ft-muted" aria-hidden />
              <h1 className="ft-docs-page__title">Calendar</h1>
            </div>
            <p className="ft-muted ft-docs-page__blurb">
              Product autopilot cadence from <code className="ft-mono">GET/PATCH …/product-schedule</code>. The grid
              highlights the server-reported next tick and last enqueue when they fall in the visible month. Recurring
              projection beyond the next run needs Redis, <code className="ft-mono">arms-worker</code>, and a valid cron
              expression.
            </p>
          </div>
          <div className="ft-docs-page__actions">
            <button type="button" className="ft-btn-ghost" disabled={loading} onClick={() => void load()} title="Reload">
              <RefreshCw size={16} className={loading ? 'ft-spin' : ''} aria-hidden />
              Refresh
            </button>
            {productId ? (
              <Link
                to={`/p/${productId}/tasks`}
                className="ft-btn-ghost"
                style={{ textDecoration: 'none', display: 'inline-flex', alignItems: 'center', gap: '0.35rem' }}
              >
                Tasks
              </Link>
            ) : null}
          </div>
        </div>

        {error ? (
          <div className="ft-docs-page__banner ft-banner ft-banner--error" role="alert" style={{ marginTop: '0.5rem' }}>
            {error}
          </div>
        ) : null}

        <div
          style={{
            marginTop: '0.75rem',
            display: 'flex',
            flexWrap: 'wrap',
            gap: '0.75rem',
            alignItems: 'stretch',
          }}
        >
          <section
            style={{
              flex: '1 1 19rem',
              minWidth: 0,
              padding: '1rem',
              borderRadius: 'var(--ft-radius-sm)',
              border: '1px solid var(--mc-border)',
              background: 'var(--mc-bg-secondary)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem', marginBottom: '0.75rem', flexWrap: 'wrap' }}>
              <h2 className="ft-field-label" style={{ margin: 0, fontSize: '0.7rem', letterSpacing: '0.04em' }}>
                Month
              </h2>
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.25rem' }}>
                <button type="button" className="ft-btn-ghost" aria-label="Previous month" onClick={prevMonth}>
                  <ChevronLeft size={18} aria-hidden />
                </button>
                <span style={{ fontSize: '0.9rem', fontWeight: 600, minWidth: '10rem', textAlign: 'center' }}>{monthTitle}</span>
                <button type="button" className="ft-btn-ghost" aria-label="Next month" onClick={nextMonth}>
                  <ChevronRight size={18} aria-hidden />
                </button>
                <button type="button" className="ft-btn-ghost" style={{ fontSize: '0.75rem', marginLeft: '0.25rem' }} onClick={goThisMonth}>
                  Today
                </button>
              </div>
            </div>

            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(7, 1fr)',
                gap: '2px',
                fontSize: '0.65rem',
                fontWeight: 700,
                letterSpacing: '0.04em',
                textTransform: 'uppercase',
                color: 'var(--mc-text-secondary)',
                marginBottom: '0.35rem',
              }}
            >
              {WEEKDAYS.map((w) => (
                <div key={w} style={{ textAlign: 'center', padding: '0.2rem 0' }}>
                  {w}
                </div>
              ))}
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: '2px' }}>
              {cells.map((cell, idx) => {
                if (!cell) {
                  return <div key={`e-${idx}`} style={{ minHeight: '3.25rem', background: 'transparent' }} />;
                }
                const { day } = cell;
                const isToday = today.y === view.year && today.m === view.month && today.d === day;
                const isNext =
                  schedule?.next_scheduled_at &&
                  matchesLocalDay(schedule.next_scheduled_at, view.year, view.month, day);
                const isLast =
                  schedule?.last_enqueued_at &&
                  matchesLocalDay(schedule.last_enqueued_at, view.year, view.month, day) &&
                  !isNext;

                const bg = isNext
                  ? 'color-mix(in srgb, var(--mc-accent) 22%, var(--mc-bg-secondary))'
                  : isLast
                    ? 'color-mix(in srgb, var(--mc-text-secondary) 12%, var(--mc-bg-secondary))'
                    : 'var(--mc-bg-tertiary)';
                const ring = isToday ? '2px solid color-mix(in srgb, var(--mc-accent) 65%, transparent)' : '1px solid var(--mc-border)';

                return (
                  <div
                    key={day}
                    style={{
                      minHeight: '3.25rem',
                      borderRadius: 'var(--ft-radius-sm)',
                      border: ring,
                      background: bg,
                      padding: '0.35rem 0.4rem',
                      fontSize: '0.8rem',
                      fontWeight: isToday ? 700 : 500,
                    }}
                  >
                    <span style={{ opacity: isToday ? 1 : 0.92 }}>{day}</span>
                    {isNext ? (
                      <div style={{ fontSize: '0.6rem', marginTop: '0.2rem', lineHeight: 1.25, color: 'var(--mc-text-secondary)' }}>
                        Next tick
                      </div>
                    ) : null}
                    {isLast ? (
                      <div style={{ fontSize: '0.6rem', marginTop: '0.15rem', lineHeight: 1.25, color: 'var(--mc-text-secondary)' }}>
                        Last enqueue
                      </div>
                    ) : null}
                  </div>
                );
              })}
            </div>

            <div style={{ marginTop: '0.85rem', display: 'flex', flexWrap: 'wrap', gap: '0.5rem', fontSize: '0.72rem', color: 'var(--mc-text-secondary)' }}>
              <span className="ft-chip" style={{ borderColor: 'color-mix(in srgb, var(--mc-accent) 40%, var(--mc-border))' }}>
                Accent fill: next scheduled autopilot tick
              </span>
              <span className="ft-chip">Muted: last enqueue</span>
              <span className="ft-chip">Ring: today</span>
            </div>
          </section>

          <div style={{ flex: '1 1 17rem', minWidth: 0, display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <section
              style={{
                padding: '1rem',
                borderRadius: 'var(--ft-radius-sm)',
                border: '1px solid var(--mc-border)',
                background: 'var(--mc-bg-secondary)',
              }}
            >
              <h2 className="ft-field-label" style={{ margin: '0 0 0.65rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
                Product schedule
              </h2>
              {loading ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.85rem' }}>
                  Loading…
                </p>
              ) : (
                <>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '0.65rem' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.85rem', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={form.enabled}
                        onChange={(e) => setForm((f) => ({ ...f, enabled: e.target.checked }))}
                      />
                      Autopilot schedule enabled
                    </label>
                    <div>
                      <label className="ft-field-label" style={{ display: 'block', marginBottom: '0.35rem' }}>
                        Cron (5-field, UTC-based on worker)
                      </label>
                      <input
                        type="text"
                        className="ft-input"
                        style={{ width: '100%', fontFamily: 'var(--ft-mono, ui-monospace, monospace)', fontSize: '0.8rem' }}
                        placeholder="e.g. 0 */6 * * *"
                        value={form.cron_expr}
                        onChange={(e) => setForm((f) => ({ ...f, cron_expr: e.target.value }))}
                        autoComplete="off"
                        spellCheck={false}
                      />
                    </div>
                    <div>
                      <label className="ft-field-label" style={{ display: 'block', marginBottom: '0.35rem' }}>
                        Delay (seconds, one-shot)
                      </label>
                      <input
                        type="text"
                        inputMode="numeric"
                        className="ft-input"
                        style={{ width: '100%', maxWidth: '12rem', fontSize: '0.85rem' }}
                        placeholder="0"
                        value={form.delay_seconds}
                        onChange={(e) => setForm((f) => ({ ...f, delay_seconds: e.target.value.replace(/\D/g, '') }))}
                      />
                    </div>
                    <div>
                      <label className="ft-field-label" style={{ display: 'block', marginBottom: '0.35rem' }}>
                        Spec JSON
                      </label>
                      <textarea
                        className="ft-input"
                        rows={4}
                        style={{ width: '100%', fontFamily: 'var(--ft-mono, ui-monospace, monospace)', fontSize: '0.78rem', resize: 'vertical' }}
                        value={form.spec_json}
                        onChange={(e) => setForm((f) => ({ ...f, spec_json: e.target.value }))}
                        spellCheck={false}
                      />
                    </div>
                  </div>

                  <div style={{ marginTop: '0.75rem', display: 'flex', flexWrap: 'wrap', gap: '0.5rem', alignItems: 'center' }}>
                    <button type="button" className="ft-btn-primary" disabled={saving} onClick={() => void save()}>
                      {saving ? 'Saving…' : 'Save schedule'}
                    </button>
                    {saveNote ? (
                      <span className="ft-muted" style={{ fontSize: '0.8rem' }}>
                        {saveNote}
                      </span>
                    ) : null}
                  </div>

                  <div
                    style={{
                      marginTop: '1rem',
                      paddingTop: '0.85rem',
                      borderTop: '1px solid var(--mc-border)',
                      fontSize: '0.78rem',
                      lineHeight: 1.5,
                      color: 'var(--mc-text-secondary)',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: '0.35rem', marginBottom: '0.35rem' }}>
                      <Clock size={14} style={{ marginTop: '0.12rem', flexShrink: 0 }} aria-hidden />
                      <div>
                        <div>
                          <strong style={{ color: 'var(--mc-text)' }}>Next scheduled:</strong> {formatDateTime(schedule?.next_scheduled_at)}
                        </div>
                        <div style={{ marginTop: '0.25rem' }}>
                          <strong style={{ color: 'var(--mc-text)' }}>Last enqueued:</strong> {formatDateTime(schedule?.last_enqueued_at)}
                        </div>
                        {schedule?.asynq_task_id ? (
                          <div style={{ marginTop: '0.25rem' }} className="ft-mono">
                            Task id: {schedule.asynq_task_id}
                          </div>
                        ) : null}
                        {schedule?.updated_at ? (
                          <div style={{ marginTop: '0.25rem' }}>Row updated {formatDateTime(schedule.updated_at)}</div>
                        ) : null}
                      </div>
                    </div>
                  </div>
                </>
              )}
            </section>

            <section
              style={{
                padding: '1rem',
                borderRadius: 'var(--ft-radius-sm)',
                border: '1px solid var(--mc-border)',
                background: 'var(--mc-bg-secondary)',
              }}
            >
              <h2 className="ft-field-label" style={{ margin: '0 0 0.5rem', fontSize: '0.7rem', letterSpacing: '0.04em' }}>
                Schedule changes (ops log)
              </h2>
              {scheduleAudit.length === 0 ? (
                <p className="ft-muted" style={{ margin: 0, fontSize: '0.8rem' }}>
                  No recent <code className="ft-mono">product_schedule</code> entries for this workspace.
                </p>
              ) : (
                <ul style={{ margin: 0, paddingLeft: '1.1rem', fontSize: '0.78rem', lineHeight: 1.45, color: 'var(--mc-text-secondary)' }}>
                  {scheduleAudit.map((e) => (
                    <li key={`${e.id ?? e.created_at}-${e.action}`} style={{ marginBottom: '0.35rem' }}>
                      <span className="ft-mono">{e.action}</span>
                      {e.created_at ? <span> — {formatDateTime(e.created_at)}</span> : null}
                      {e.actor ? <span> ({e.actor})</span> : null}
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>
        </div>
      </div>
    </div>
  );
}
