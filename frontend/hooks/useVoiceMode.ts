"use client";

import { useEffect, useState } from "react";
import { useEntitlements, type Entitlements } from "@/hooks/use-entitlements";
import { useVoiceConfig } from "@/hooks/useVoiceConfig";
import {
  decideVoiceMode,
  hasHeadroom,
  type VoiceCapabilities,
  type VoiceMode,
} from "@/lib/voice-mode";

export interface UseVoiceModeResult {
  /** The transcriber to use for the next recording. */
  mode: VoiceMode;
  /** Whether the mic may be pressed at all. */
  canRecord: boolean;
  /** Whether the premium (server) path is active. */
  isPremium: boolean;
  /** Remaining daily voice uses (0 when exhausted; -1 when unlimited). */
  dailyRemaining: number;
  /** Browser capabilities (false during SSR, resolved on mount). */
  capabilities: VoiceCapabilities;
}

/** Detects the browser's transcription capabilities. Client-only. */
function detectCapabilities(): VoiceCapabilities {
  if (typeof window === "undefined") return { browserSTT: false, recorder: false };
  const w = window as Window & { webkitSpeechRecognition?: unknown; SpeechRecognition?: unknown };
  const browserSTT = Boolean(w.SpeechRecognition || w.webkitSpeechRecognition);
  const recorder = Boolean(
    typeof MediaRecorder !== "undefined" &&
      navigator.mediaDevices?.getUserMedia,
  );
  return { browserSTT, recorder };
}

/**
 * useVoiceMode chooses the transcriber from the user's entitlement + the live
 * voice config + the browser's capabilities. It recomputes automatically when
 * entitlements or config change (e.g. after a `usage_changed` event), so the
 * mode degrades the moment a quota is exhausted — no reload required.
 */
export function useVoiceMode(): UseVoiceModeResult {
  const { entitlements } = useEntitlements();
  const { config } = useVoiceConfig();

  // Capabilities are only knowable on the client; resolving them in an effect
  // avoids an SSR/hydration mismatch.
  const [capabilities, setCapabilities] = useState<VoiceCapabilities>({
    browserSTT: false,
    recorder: false,
  });
  useEffect(() => {
    setCapabilities(detectCapabilities());
  }, []);

  const ent: Entitlements | null = entitlements;
  const hasVoiceFeature = ent?.features?.voice_input === true;
  const hasQuota =
    hasHeadroom(ent?.usage?.ai_tokens_per_cycle) &&
    hasHeadroom(ent?.usage?.ai_requests_per_cycle);
  // The daily entry comes from the voice config first, then entitlements. While
  // BOTH are still loading we must not force "daily_limit" (that would flash a
  // blocked mic on every mount) — treat "unknown" as headroom and let the
  // backend gate be authoritative. Once a config/usage entry arrives, it governs.
  const dailyEntry = config?.daily_limit ?? ent?.usage?.voice_requests_per_day;
  const hasDailyQuota = dailyEntry === undefined ? true : hasHeadroom(dailyEntry);

  // A kill switch forces the non-server path regardless of the plan.
  const serverAllowed = config?.server_enabled !== false;

  const mode = decideVoiceMode(
    { hasVoiceFeature: hasVoiceFeature && serverAllowed, hasQuota, hasDailyQuota },
    capabilities,
  );

  const dailyRemaining = dailyEntry
    ? dailyEntry.unlimited
      ? -1
      : dailyEntry.remaining
    : -1;

  return {
    mode,
    canRecord: mode !== "unavailable" && mode !== "daily_limit",
    isPremium: mode === "server",
    dailyRemaining,
    capabilities,
  };
}
