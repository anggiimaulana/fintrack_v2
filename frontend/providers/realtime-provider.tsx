"use client";

import type { ReactNode } from "react";
import { useSession } from "@/lib/session-context";
import { useRealtimeStream } from "@/hooks/use-realtime-stream";

/**
 * Keeps the realtime SSE stream connected for the signed-in user.
 *
 * Mounted once near the app root. The stream opens only when a session exists
 * and closes on logout, so an anonymous visitor never holds a connection (and
 * never triggers a 401 loop).
 */
export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useSession();

  // The hook's return value (status) is intentionally unused here: this provider
  // exists to own the connection lifecycle. Components that need the status can
  // call useRealtimeStream themselves, but the connection is driven from here so
  // there is exactly one stream per tab.
  useRealtimeStream({ enabled: isAuthenticated });

  return <>{children}</>;
}
