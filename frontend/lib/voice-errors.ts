/**
 * Classification of errors returned by POST /ai/voice — pure and unit-testable.
 *
 * The component reacts differently depending on the kind:
 *   - "fallback"   → retry the same recording on-device (browser mode)
 *   - "daily_limit"→ block: the daily cap is exhausted, no path is available
 *   - "permission" → stop: the microphone was denied, retrying is pointless
 *   - "fatal"      → a client bug (400); surface it rather than loop
 *
 * Anything unrecognised is treated as "fallback": a transient/unknown failure
 * should never strand the user without a way to dictate.
 */
export type VoiceErrorKind = "fallback" | "daily_limit" | "permission" | "fatal";

interface AxiosishError {
  response?: { status?: number; data?: { error?: string } };
  name?: string;
}

/** classifyVoiceError maps an error thrown by the /ai/voice request to an action. */
export function classifyVoiceError(err: unknown): VoiceErrorKind {
  const e = (err ?? {}) as AxiosishError;

  // A denied microphone surfaces from getUserMedia, not the HTTP call.
  if (e.name === "NotAllowedError" || e.name === "SecurityError") return "permission";
  if (e.name === "NotFoundError" || e.name === "NotSupportedError") return "permission";

  const status = e.response?.status;
  const code = e.response?.data?.error;

  // The daily cap is a hard block, distinct from a soft quota fallback.
  if (code === "daily_limit_exceeded" || code === "voice_daily_limit") return "daily_limit";

  if (status === 400) return "fatal";

  if (code === "insufficient_quota" || code === "limit_exceeded") return "fallback";
  if (code === "feature_not_in_plan") return "fallback";
  if (code === "rate_limited") return "fallback";

  if (status === 402 || status === 403 || status === 429) return "fallback";
  if (typeof status === "number" && status >= 500) return "fallback";

  // No HTTP response at all → network error / timeout.
  if (status === undefined) return "fallback";

  return "fatal";
}
