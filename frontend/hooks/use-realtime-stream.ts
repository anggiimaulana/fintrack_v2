import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { dispatchDomainEvent, type DomainEvent } from "@/lib/realtime-dispatcher";
import { publishDomainEvent } from "@/lib/realtime-bus";
import { refreshSession } from "@/lib/api";

/** Reconnect delays in ms: 1s, 2s, 4s, 8s, then capped at 16s. */
export const BACKOFF_STEPS_MS = [1000, 2000, 4000, 8000, 16000] as const;

/**
 * backoffDelay returns the delay for a given attempt (0-based), capped at the
 * final step. `jitter` (0..1) spreads reconnects so many clients do not
 * stampede the server after a restart; it is injectable for deterministic tests.
 */
export function backoffDelay(attempt: number, jitter = Math.random): number {
  const idx = Math.min(Math.max(attempt, 0), BACKOFF_STEPS_MS.length - 1);
  const base = BACKOFF_STEPS_MS[idx];
  // Full jitter over the lower half keeps the cap intact: [base/2, base].
  const factor = 0.5 + 0.5 * jitter();
  return Math.round(base * factor);
}

export type StreamStatus = "connecting" | "open" | "reconnecting" | "closed";

/**
 * Handle for a scheduled reconnect. Typed as `number` because the stream only
 * runs in the browser (inside an effect), where `window.setTimeout` returns a
 * number — unlike Node's `Timeout`, which the ambient typings also expose.
 */
export type ReconnectTimer = number;

export interface UseRealtimeStreamOptions {
  /** Disable the stream entirely (e.g. not authenticated). */
  enabled?: boolean;
  /** Override for tests. */
  eventSourceFactory?: (url: string) => EventSource;
}

/**
 * Subscribes to the backend SSE stream and dispatches domain events into the
 * TanStack Query cache.
 *
 * Reconnection is managed manually rather than relying on EventSource's built-in
 * retry, so the backoff follows the P07 spec (1s→16s with jitter). Because a
 * manually re-created EventSource does not resend `Last-Event-ID`, the last seen
 * id is passed as the `last_event_id` query param so the server can replay
 * anything missed (the header wins when both are present).
 */
export function useRealtimeStream(options: UseRealtimeStreamOptions = {}) {
  const { enabled = true, eventSourceFactory } = options;
  const queryClient = useQueryClient();
  // Internal status for the connected case; the effective value is derived below
  // so the effect body never calls setState synchronously (which would cause a
  // cascading render).
  const [streamStatus, setStreamStatus] = useState<StreamStatus>("connecting");
  const lastEventIdRef = useRef<string>("");
  const attemptRef = useRef(0);
  const esRef = useRef<EventSource | null>(null);
  // `undefined` (not null) so clearTimeout accepts it without a guard.
  const timerRef = useRef<ReconnectTimer | undefined>(undefined);
  const stoppedRef = useRef(false);
  // Tracks whether the stream has ever opened in this mount. Used to
  // distinguish a genuine reconnect (which needs a resync) from the first open.
  const hadOpenRef = useRef(false);

  const handleEvent = useCallback(
    (raw: MessageEvent) => {
      let parsed: DomainEvent | null = null;
      try {
        parsed = JSON.parse(raw.data) as DomainEvent;
      } catch {
        return; // ignore malformed frames rather than breaking the stream
      }
      if (parsed?.id) lastEventIdRef.current = parsed.id;
      if (parsed?.event) {
        // Cache invalidation (data freshness) and the notification bus (UI
        // side-effects) both consume the same event — one stream, two sinks.
        dispatchDomainEvent(queryClient, parsed);
        publishDomainEvent(parsed);
      }
    },
    [queryClient]
  );

  const status: StreamStatus = enabled ? streamStatus : "closed";

  useEffect(() => {
    if (!enabled) {
      return;
    }

    stoppedRef.current = false;
    const make = eventSourceFactory ?? ((url: string) => new EventSource(url, { withCredentials: true }));

    const connect = () => {
      if (stoppedRef.current) return;

      const params = new URLSearchParams();
      if (lastEventIdRef.current) params.set("last_event_id", lastEventIdRef.current);
      const qs = params.toString();
      const url = `/api/v1/realtime/stream${qs ? `?${qs}` : ""}`;

      setStreamStatus(attemptRef.current === 0 ? "connecting" : "reconnecting");
      const es = make(url);
      esRef.current = es;

      es.onopen = () => {
        const isReconnect = attemptRef.current > 0 || hadOpenRef.current;
        attemptRef.current = 0;
        setStreamStatus("open");

        // After a drop, buffered events are replayed on resume — but a gap
        // longer than the server buffer TTL (or a Redis-less deployment) can
        // still lose events. Synthesise one `stream_reconnected` so consumers
        // resync exactly once, replacing any need for polling. The very first
        // open needs no resync: the initial queries fetch fresh data anyway.
        if (isReconnect) {
          const ev: DomainEvent = {
            event: "stream_reconnected",
            timestamp: Math.floor(Date.now() / 1000),
          };
          dispatchDomainEvent(queryClient, ev);
          publishDomainEvent(ev);
        }
        hadOpenRef.current = true;
      };

      // Domain events arrive as named events; the server also sends unnamed
      // frames, so listen on the generic channel too.
      es.onmessage = handleEvent;
      for (const name of [
        "payment_received",
        "plan_updated",
        "quota_reset",
        // Consumption moved: refreshes the entitlements query only, so the
        // quota bars stay live without touching the data lists.
        "usage_changed",
        "ai_endpoint_down",
        "ai_route_quarantined",
        "manual_approval_needed",
        // Notification-inbox lifecycle: the notifications page patches its
        // cache in place from these (no full refetch).
        "notification_created",
        "notification_read",
        "notification_deleted",
      ]) {
        es.addEventListener(name, handleEvent as EventListener);
      }

      es.onerror = () => {
        // ReadyState CLOSED means the connection is gone and will not retry on
        // its own (we manage retries), so schedule a reconnect.
        es.close();
        if (stoppedRef.current) return;

        const delay = backoffDelay(attemptRef.current);
        attemptRef.current += 1;
        setStreamStatus("reconnecting");
        timerRef.current = window.setTimeout(async () => {
          if (stoppedRef.current) return;
          try {
            // Refresh token before reconnecting in case the drop was due to an expired access token.
            await refreshSession();
          } catch {
            // Squelch refresh error: if user is logged out, the next connect will fail/stop cleanly.
          }
          connect();
        }, delay);
      };
    };

    connect();

    return () => {
      stoppedRef.current = true;
      window.clearTimeout(timerRef.current);
      esRef.current?.close();
      esRef.current = null;
      setStreamStatus("closed");
    };
  }, [enabled, eventSourceFactory, handleEvent]);

  return { status };
}
