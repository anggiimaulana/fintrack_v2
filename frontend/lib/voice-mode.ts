/**
 * Voice transport selection — pure, framework-free, and unit-testable.
 *
 * The component asks this module "which transcriber should I use?" given the
 * user's entitlement and the browser's capabilities. Keeping the decision here
 * (not inside the component) makes every branch of the routing matrix testable
 * without a DOM or a network.
 */

/** How a recording will be transcribed. */
export type VoiceMode =
  /** Server-side transcription via POST /ai/voice (premium, costs quota). */
  | "server"
  /** On-device transcription via the Web Speech API (free, capped daily). */
  | "browser"
  /** The daily cap is exhausted: every path is blocked until the reset. */
  | "daily_limit"
  /** No transcriber is available in this browser/plan combination. */
  | "unavailable";

/** What the browser can do. Both flags are false during SSR. */
export interface VoiceCapabilities {
  /** The Web Speech API is available (Chrome/Edge/Safari). */
  browserSTT: boolean;
  /** MediaRecorder + getUserMedia are available (needed for server mode). */
  recorder: boolean;
}

/** The entitlement-derived inputs to the decision. */
export interface VoiceEntitlement {
  /** The plan enables the `voice_input` feature (server mode). */
  hasVoiceFeature: boolean;
  /** Token and request quota still have headroom. */
  hasQuota: boolean;
  /** The daily voice cap still has headroom. */
  hasDailyQuota: boolean;
}

/**
 * decideVoiceMode picks the best available transcriber.
 *
 * Priority: server > browser > unavailable, EXCEPT when the daily cap is
 * exhausted, which blocks every path (Q-03). The daily gate is checked first so
 * it wins over all other signals.
 */
export function decideVoiceMode(ent: VoiceEntitlement, cap: VoiceCapabilities): VoiceMode {
  if (!ent.hasDailyQuota) return "daily_limit";
  if (ent.hasVoiceFeature && ent.hasQuota && cap.recorder) return "server";
  if (cap.browserSTT) return "browser";
  return "unavailable";
}

/** A single usage entry as returned in the entitlements payload. */
export interface UsageEntryLike {
  used: number;
  limit: number;
  remaining: number;
  unlimited: boolean;
}

/**
 * hasHeadroom reports whether a usage entry still allows one more use.
 *
 * - An absent entry (the plan does not define the limit) is treated as NO
 *   headroom, matching the backend gate which fails closed on an absent key.
 * - A zero limit (also "not in plan") is no headroom.
 * - Unlimited always has headroom.
 */
export function hasHeadroom(entry: UsageEntryLike | undefined): boolean {
  if (!entry) return false;
  if (entry.unlimited) return true;
  if (entry.limit <= 0) return false;
  return entry.remaining >= 1;
}
