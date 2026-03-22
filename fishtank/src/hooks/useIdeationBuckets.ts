import { useEffect, useState } from 'react';
import {
  IDEATION_BUCKETS_CHANGED_EVENT,
  resolveIdeationBuckets,
  type ResolvedIdeationBucket,
} from '../lib/ideationBucketPreferences';

/**
 * Live list of ideation buckets (built-in catalog ± System page customizations in localStorage).
 */
export function useIdeationBuckets(): ResolvedIdeationBucket[] {
  const [buckets, setBuckets] = useState<ResolvedIdeationBucket[]>(() => resolveIdeationBuckets());

  useEffect(() => {
    const sync = () => setBuckets(resolveIdeationBuckets());
    window.addEventListener('storage', sync);
    window.addEventListener(IDEATION_BUCKETS_CHANGED_EVENT, sync);
    return () => {
      window.removeEventListener('storage', sync);
      window.removeEventListener(IDEATION_BUCKETS_CHANGED_EVENT, sync);
    };
  }, []);

  return buckets;
}
