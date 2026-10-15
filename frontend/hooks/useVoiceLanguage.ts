"use client";

import { useCallback, useMemo, useState } from "react";
import type { VoiceConfig } from "@/hooks/useVoiceConfig";

const STORAGE_KEY = "fintrack.voice.language";

export interface UseVoiceLanguageResult {
  /** The language to use for the next recording. */
  language: string;
  /** Whether the user may pick a language (only when language_mode = multi). */
  canChoose: boolean;
  /** The selectable languages (a single entry under id_only). */
  options: string[];
  /** Choose a language for subsequent recordings; remembered locally. */
  setLanguage: (lang: string) => void;
}

function readStored(): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

/**
 * useVoiceLanguage resolves the per-recording language (N-02).
 *
 * Under `id_only` the language is fixed to the configured default. Under `multi`
 * the user may choose, and the last choice is remembered locally so it carries
 * across recordings without another server round-trip.
 */
export function useVoiceLanguage(config: VoiceConfig | null): UseVoiceLanguageResult {
  const mode = config?.language_mode ?? "id_only";
  const canChoose = mode === "multi";
  const fallback = config?.default_language ?? "id";

  // Stable array identity: a fresh array each render would make `setLanguage`
  // (and any effect depending on it) change on every render.
  const options = useMemo(
    () => config?.languages ?? [config?.default_language ?? "id"],
    [config?.languages, config?.default_language],
  );

  const [chosen, setChosen] = useState<string | null>(() => readStored());

  // A remembered language that is no longer offered (an operator removed it) must
  // fall back rather than being sent to the transcriber.
  const effectiveChosen = chosen && options.includes(chosen) ? chosen : null;
  const language = canChoose ? effectiveChosen ?? fallback : fallback;

  const setLanguage = useCallback(
    (lang: string) => {
      if (!canChoose) return;
      if (!options.includes(lang)) return;
      setChosen(lang);
      try {
        window.localStorage.setItem(STORAGE_KEY, lang);
      } catch {
        // Storage may be unavailable (private mode); the in-memory value stands.
      }
    },
    [canChoose, options],
  );

  return { language, canChoose, options, setLanguage };
}
