"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import type { UsageEntry } from "@/hooks/use-entitlements";

/**
 * The effective voice configuration, resolved server-side from the dynamic
 * settings (K-VOI-19). Every field here is tunable by a Super Admin without a
 * redeploy, so the client must never hardcode any of these values.
 */
export interface VoiceConfig {
  max_duration_seconds: number;
  language_mode: "id_only" | "multi";
  default_language: string;
  languages: string[];
  language_per_recording: boolean;
  daily_limit: UsageEntry;
  daily_reset_timezone: string;
  daily_resets_at: string;
  history_retention_days: number;
  allow_edit_before_send: boolean;
  vu_meter_enabled: boolean;
  fallback_to_browser: boolean;
  server_enabled: boolean;
  server_available: boolean;
}

/**
 * Sensible fallback used when the config endpoint is unavailable, so the voice
 * UI still works (degraded) instead of failing closed. Mirrors the backend
 * defaults.
 */
export const FALLBACK_VOICE_CONFIG: VoiceConfig = {
  max_duration_seconds: 60,
  language_mode: "id_only",
  default_language: "id",
  languages: ["id"],
  language_per_recording: false,
  daily_limit: { used: 0, limit: -1, remaining: -1, unlimited: true },
  daily_reset_timezone: "Asia/Jakarta",
  daily_resets_at: "",
  history_retention_days: 90,
  allow_edit_before_send: true,
  vu_meter_enabled: true,
  fallback_to_browser: true,
  server_enabled: true,
  server_available: false,
};

async function fetchVoiceConfig(): Promise<VoiceConfig> {
  const res = await api.get("/me/voice-config");
  return (res.data?.data ?? FALLBACK_VOICE_CONFIG) as VoiceConfig;
}

/**
 * useVoiceConfig loads the dynamic voice configuration.
 *
 * It is invalidated by the `usage_changed` SSE event (via the dispatcher's
 * entitlements key) and by the entitlements query, so a Super Admin change or a
 * consumed recording is reflected without a manual reload.
 */
export function useVoiceConfig() {
  const query = useQuery({
    queryKey: queryKeys.voice.config,
    queryFn: fetchVoiceConfig,
    staleTime: 1000 * 60 * 2,
    retry: false,
  });

  return {
    config: query.data ?? null,
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  };
}
