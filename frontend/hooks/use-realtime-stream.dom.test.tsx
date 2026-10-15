import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { useRealtimeStream } from "./use-realtime-stream";
import { subscribeDomainEvents } from "@/lib/realtime-bus";

/**
 * A fake EventSource whose lifecycle the test drives manually, so we can assert
 * on the client-synthesised `stream_reconnected` resync without a network.
 */
class FakeEventSource {
  onopen: (() => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: (() => void) | null = null;
  readyState = 0;
  url: string;

  constructor(url: string) {
    this.url = url;
  }

  addEventListener() {}
  close() {
    this.readyState = 2;
  }
  fireOpen() {
    this.readyState = 1;
    this.onopen?.();
  }
}

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

/** Builds a hook harness that records every domain event published to the bus. */
function harness() {
  const sources: FakeEventSource[] = [];
  const client = new QueryClient();
  const received: string[] = [];
  const unsub = subscribeDomainEvents((ev) => received.push(ev.event));
  const view = renderHook(
    () =>
      useRealtimeStream({
        eventSourceFactory: (url) => {
          const es = new FakeEventSource(url);
          sources.push(es);
          return es as unknown as EventSource;
        },
      }),
    { wrapper: wrapper(client) },
  );
  return { sources, received, unsub, view };
}

describe("useRealtimeStream reconnect resync", () => {
  it("does NOT emit stream_reconnected on the first open", () => {
    const { sources, received, unsub } = harness();
    sources[0].fireOpen();
    expect(received).not.toContain("stream_reconnected");
    unsub();
  });

  it("emits stream_reconnected when a dropped stream re-opens", async () => {
    const { sources, received, unsub } = harness();

    // First connection opens normally: no resync.
    sources[0].fireOpen();
    expect(received.filter((e) => e === "stream_reconnected")).toHaveLength(0);

    // Simulate a drop. The hook schedules a reconnect via setTimeout whose
    // callback awaits refreshSession() before reconnecting, so timers alone are
    // not enough — the awaited microtasks must flush too.
    vi.useFakeTimers();
    sources[0].onerror?.();
    vi.runOnlyPendingTimers();
    vi.useRealTimers();
    await vi.waitFor(() => expect(sources.length).toBeGreaterThan(1));

    // The reconnected source opens: exactly one resync is emitted.
    sources[sources.length - 1].fireOpen();
    expect(received.filter((e) => e === "stream_reconnected")).toHaveLength(1);

    unsub();
  });
});
