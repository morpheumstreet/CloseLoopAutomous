import { ArmsHttpError } from '../api/armsClient';

/** `PATCH /api/products/{id}` optimistic lock failed (`updated_at` changed). */
export function isStaleEntityError(e: unknown): e is ArmsHttpError {
  return e instanceof ArmsHttpError && e.status === 409 && e.code === 'stale_entity';
}

export const STALE_PRODUCT_HELP =
  'This workspace was updated while you were editing—another save, a second tab, or a background process changed it first. Your draft was not saved. Reload from the server, then edit and save again if you still want those changes.';
