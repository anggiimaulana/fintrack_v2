import api from "@/lib/api";

/** One recorded transcription as returned by GET /me/voice-history. */
export interface VoiceHistoryEntry {
  id: string;
  transcript: string;
  language: string;
  mode: "server" | "browser";
  duration_ms: number;
  tokens_used: number;
  created_at: string;
}

/**
 * recordBrowserHistory reports an on-device transcription so the daily cap
 * counts it (server-mode rows are recorded by the backend). Best-effort: the
 * caller should not fail the UX when this throws.
 */
export async function recordBrowserHistory(input: {
  transcript: string;
  language: string;
  durationMs: number;
}): Promise<VoiceHistoryEntry> {
  const res = await api.post("/me/voice-history", {
    transcript: input.transcript,
    language: input.language,
    mode: "browser",
    duration_ms: input.durationMs,
  });
  return res.data?.data as VoiceHistoryEntry;
}
