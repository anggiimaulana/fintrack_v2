/**
 * A tiny module-level pub/sub for realtime domain events.
 *
 * The SSE connection is owned by a single hook (`use-realtime-stream`, mounted
 * once by `RealtimeProvider`). Consumers that also want to react to events —
 * such as the notification bell — subscribe here instead of opening a second
 * EventSource, so the browser holds exactly one stream per tab.
 *
 * Deliberately dependency-free and framework-free so it is unit-testable and
 * importable from any layer.
 */
import type { DomainEvent } from "@/lib/realtime-dispatcher";

type Listener = (event: DomainEvent) => void;

const listeners = new Set<Listener>();

/** subscribe registers a listener and returns an unsubscribe function. */
export function subscribeDomainEvents(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** publishDomainEvent fans one event out to every current listener with error isolation. */
export function publishDomainEvent(event: DomainEvent): void {
  for (const listener of listeners) {
    try {
      listener(event);
    } catch (err) {
      console.error("[RealtimeBus] Listener callback error:", err);
    }
  }
}
