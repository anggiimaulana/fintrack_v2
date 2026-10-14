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
  const { isAuthenticated, isLoading } = useSession();

  // Wait until session finishes loading (and auto-refreshing if expired) before
  // opening the SSE connection, so the initial connect has a fresh cookie.
  useRealtimeStream({ enabled: isAuthenticated && !isLoading });

  return <>{children}</>;
}
