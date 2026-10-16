"use client";

import type { ReactNode } from "react";
import { useSession } from "@/lib/session-context";
import { useRealtimeStream } from "@/hooks/use-realtime-stream";
import { useNotificationsRealtime } from "@/hooks/useNotifications";
import { useReportsRealtime } from "@/hooks/useReports";
import { useAIConfig } from "@/hooks/useAIConfig";

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

  // Subscribe the notification cache to the stream ONCE for the whole app (not
  // per-page), so the header bell and the notifications page both stay live
  // without polling. Safe when unauthenticated: no events arrive.
  useNotificationsRealtime();
  // Keep reports, budgets, SPK, and dashboard caches live on SSE events.
  useReportsRealtime();

  // Install the Super-Admin-configured AI request timeout for the whole app, so
  // a slow receipt/voice call is not aborted by the browser before the server
  // would. Safe when unauthenticated: the fetch is skipped and the built-in
  // fallback stands.
  useAIConfig();

  return <>{children}</>;
}
