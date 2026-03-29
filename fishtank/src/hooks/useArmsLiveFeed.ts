import { useEffect, useRef } from 'react';
import { isDevBuild, type ArmsEnv } from '../config/armsEnv';
import { buildLiveEventsUrl } from '../api/armsClient';
import type { FeedEvent } from '../domain/types';
import { ssePayloadToFeedEvent } from '../mappers/missionMappers';

type AppendFn = (event: FeedEvent) => void;

export type LiveFeedOptions = {
  /** Increment to close and reopen EventSource (manual reconnect). */
  reconnectEpoch?: number;
  onConnectionLive?: (live: boolean) => void;
};

/**
 * Subscribes to arms SSE for one product. Tears down when productId or epoch changes.
 */
export function useArmsLiveFeed(
  productId: string | null,
  env: ArmsEnv,
  append: AppendFn,
  options?: LiveFeedOptions,
): void {
  const appendRef = useRef(append);
  appendRef.current = append;

  const onLiveRef = useRef(options?.onConnectionLive);
  onLiveRef.current = options?.onConnectionLive;

  const epoch = options?.reconnectEpoch ?? 0;
  const includeRaw = isDevBuild();

  useEffect(() => {
    if (!productId) {
      onLiveRef.current?.(false);
      return;
    }

    onLiveRef.current?.(false);
    let seq = 0;
    const url = buildLiveEventsUrl(env, productId);
    const es = new EventSource(url);

    es.onopen = () => {
      onLiveRef.current?.(true);
    };

    es.onmessage = (ev: MessageEvent<string>) => {
      let raw: unknown;
      try {
        raw = JSON.parse(ev.data) as unknown;
      } catch {
        return;
      }
      const fe = ssePayloadToFeedEvent(raw, seq++, includeRaw);
      if (fe) appendRef.current(fe);
    };

    es.onerror = () => {
      onLiveRef.current?.(false);
    };

    return () => {
      es.close();
      onLiveRef.current?.(false);
    };
  }, [productId, env, epoch, includeRaw]);
}
