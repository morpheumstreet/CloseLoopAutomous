import { DEFAULT_IDEATION_BUCKET, IDEATION_BUCKETS, type IdeationBucketValue } from './ideaCategories';

const STORAGE_KEY = 'ft-ideation-bucket-prefs-v1';

export const IDEATION_BUCKETS_CHANGED_EVENT = 'ft-ideation-buckets-changed';

export type IdeationBucketPrefsV1 = {
  v: 1;
  /** Shown buckets, in catalog order. Must be non-empty when saved. */
  selectedSlugs: IdeationBucketValue[];
};

export type ResolvedIdeationBucket = {
  value: IdeationBucketValue;
  label: string;
  sop: 1 | 2 | 3 | 4;
};

function parsePrefs(raw: string): IdeationBucketPrefsV1 | null {
  try {
    const p = JSON.parse(raw) as IdeationBucketPrefsV1 & { labelOverrides?: unknown };
    if (p?.v !== 1 || !Array.isArray(p.selectedSlugs)) return null;
    return { v: 1, selectedSlugs: p.selectedSlugs };
  } catch {
    return null;
  }
}

const catalogValues = new Set(IDEATION_BUCKETS.map((b) => b.value));

function isIdeationBucketValue(s: string): s is IdeationBucketValue {
  return catalogValues.has(s as IdeationBucketValue);
}

export function readIdeationBucketPrefs(): IdeationBucketPrefsV1 | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return parsePrefs(raw);
  } catch {
    return null;
  }
}

/** Ordered list for Ideation UI. Falls back to full built-in catalog when unset or invalid. */
export function resolveIdeationBuckets(): ResolvedIdeationBucket[] {
  const pref = readIdeationBucketPrefs();
  if (!pref?.selectedSlugs?.length) {
    return IDEATION_BUCKETS.map((b) => ({ value: b.value, label: b.label, sop: b.sop }));
  }
  const out: ResolvedIdeationBucket[] = [];
  for (const b of IDEATION_BUCKETS) {
    if (!pref.selectedSlugs.includes(b.value)) continue;
    out.push({ value: b.value, label: b.label, sop: b.sop });
  }
  if (out.length === 0) {
    return IDEATION_BUCKETS.map((b) => ({ value: b.value, label: b.label, sop: b.sop }));
  }
  return out;
}

export function writeIdeationBucketPrefs(prefs: IdeationBucketPrefsV1): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore quota / private mode */
  }
  notifyIdeationBucketsChanged();
}

export function clearIdeationBucketPrefs(): void {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* ignore */
  }
  notifyIdeationBucketsChanged();
}

export function notifyIdeationBucketsChanged(): void {
  try {
    window.dispatchEvent(new Event(IDEATION_BUCKETS_CHANGED_EVENT));
  } catch {
    /* ignore */
  }
}

export function defaultSelectedSlugs(): IdeationBucketValue[] {
  return IDEATION_BUCKETS.map((b) => b.value);
}

export function initialSelectedSetFromStorage(): Set<IdeationBucketValue> {
  const pref = readIdeationBucketPrefs();
  if (!pref?.selectedSlugs?.length) {
    return new Set(defaultSelectedSlugs());
  }
  const next = new Set<IdeationBucketValue>();
  for (const s of pref.selectedSlugs) {
    if (isIdeationBucketValue(s)) next.add(s);
  }
  return next.size > 0 ? next : new Set(defaultSelectedSlugs());
}

export function firstResolvedBucketValue(buckets: ResolvedIdeationBucket[]): IdeationBucketValue {
  return buckets[0]?.value ?? DEFAULT_IDEATION_BUCKET;
}
