import { useEffect, useRef } from 'react';
import type { ArmsEnv } from '../config/armsEnv';
import { buildLiveEventsUrl } from '../api/armsClient';
import type { FeedEvent } from '../domain/types';
import { ssePayloadToFeedEvent } from '../mappers/missionMappers';

type AppendFn = (event: FeedEvent) => void;

/**
 * Subscribes to arms SSE for one product. Isolated hook (single responsibility).
 * EventSource reconnects on its own; we tear down when productId changes.
 */
export function useArmsLiveFeed(productId: string | null, env: ArmsEnv, append: AppendFn): void {
  const appendRef = useRef(append);
  appendRef.current = append;

  useEffect(() => {
    if (!productId) return;

    let seq = 0;
    const url = buildLiveEventsUrl(env, productId);
    const es = new EventSource(url);

    es.onmessage = (ev: MessageEvent<string>) => {
      let raw: unknown;
      try {
        raw = JSON.parse(ev.data) as unknown;
      } catch {
        return;
      }
      const fe = ssePayloadToFeedEvent(raw, seq++);
      if (fe) appendRef.current(fe);
    };

    return () => es.close();
  }, [productId, env]);
}
