"use client";

import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import api, { applyAIRequestTimeout } from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { useSession } from "@/lib/session-context";

/**
 * The effective AI timeout budget, resolved server-side from the dynamic
 * settings. A Super Admin can retune these without a redeploy, so the client
 * must never hardcode them.
 */
export interface AIConfig {
  request_timeout_ms: number;
  /** The outer ceiling imposed by the Next.js rewrite proxy (ms). */
  proxy_ceiling_ms: number;
  timeouts_ms: {
    chat: number;
    vision: number;
    voice: number;
    embedding: number;
    reasoning: number;
  };
}

/**
 * Fallback used when the config endpoint is unavailable, mirroring the backend
 * defaults so AI requests still work (degraded) instead of failing closed.
 */
export const FALLBACK_AI_CONFIG: AIConfig = {
  request_timeout_ms: 0,
  proxy_ceiling_ms: 180_000,
  timeouts_ms: { chat: 60000, vision: 120000, voice: 120000, embedding: 60000, reasoning: 120000 },
};

async function fetchAIConfig(): Promise<AIConfig> {
  const res = await api.get("/me/ai-config");
  return (res.data?.data ?? FALLBACK_AI_CONFIG) as AIConfig;
}

/**
 * useAIConfig loads the dynamic AI timeout budget and installs the
 * client-side request timeout so the browser waits exactly as long as the
 * server allows. Mount it once near the app root.
 *
 * The query is gated on an authenticated session: the provider sits above the
 * public routes (login/register), and an unauthenticated call would 401 and
 * trigger the axios interceptor's redirect-to-/login — an infinite loop on the
 * login page itself. While unauthenticated the built-in fallback stands, which
 * is correct because no AI request can be made without a session anyway.
 *
 * It is intentionally long-lived (long staleTime): the value rarely changes,
 * and a Super Admin change is picked up on the next natural refetch or reload
 * rather than hammering the endpoint.
 */
export function useAIConfig() {
  const { isAuthenticated, isLoading: sessionLoading } = useSession();

  const query = useQuery({
    queryKey: queryKeys.ai.config,
    queryFn: fetchAIConfig,
    staleTime: 1000 * 60 * 10,
    retry: false,
    enabled: isAuthenticated && !sessionLoading,
  });

  const timeoutMs = query.data?.request_timeout_ms ?? 0;

  // Push the server-reported budget into the shared axios timeout. Runs whenever
  // the value changes, so a Super Admin retune reaches an open tab without a
  // rebuild.
  useEffect(() => {
    if (timeoutMs > 0) applyAIRequestTimeout(timeoutMs);
  }, [timeoutMs]);

  return {
    config: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}
