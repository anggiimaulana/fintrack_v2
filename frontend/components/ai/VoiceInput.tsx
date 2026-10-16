"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Mic, MicOff, Square, Loader2 } from "lucide-react";
import { toast } from "sonner";
import api from "@/lib/api";
import { Button } from "@/components/ui/button";
import { VoiceLevelMeter } from "@/components/ai/VoiceLevelMeter";
import { useVoiceMode } from "@/hooks/useVoiceMode";
import { useVoiceConfig } from "@/hooks/useVoiceConfig";
import { useVoiceLanguage } from "@/hooks/useVoiceLanguage";
import { useRecordBrowserHistory } from "@/hooks/useVoiceHistory";
import { startRecording, type RecorderHandle } from "@/lib/voice-recorder";
import { blobToBase64, isAudioTooLarge } from "@/lib/voice-audio";
import { classifyVoiceError } from "@/lib/voice-errors";


// ─── Web Speech API typings (browser-only, not in the TS lib) ──────

interface SpeechRecognitionEvent {
  resultIndex: number;
  results: {
    length: number;
    [index: number]: { isFinal: boolean; [index: number]: { transcript: string } };
  };
}

interface SpeechRecognitionErrorEvent {
  error: string;
}

interface SpeechRecognition {
  lang: string;
  continuous: boolean;
  interimResults: boolean;
  maxAlternatives: number;
  onstart: () => void;
  onresult: (event: SpeechRecognitionEvent) => void;
  onerror: (event: SpeechRecognitionErrorEvent) => void;
  onend: () => void;
  start: () => void;
  stop: () => void;
  abort: () => void;
}

type WindowWithSpeech = Window &
  typeof globalThis & {
    SpeechRecognition?: new () => SpeechRecognition;
    webkitSpeechRecognition?: new () => SpeechRecognition;
  };

/** BCP-47 tag for a bare language code (e.g. "id" → "id-ID"). */
function speechLang(lang: string): string {
  if (lang.includes("-")) return lang;
  return lang === "id" ? "id-ID" : lang;
}

/**
 * Dev-only telemetry for the voice router. Logs the chosen mode and the reason
 * a fallback happened so the behaviour is observable during development.
 * Deliberately logs NO transcript/audio (PII-free) and is a no-op in production.
 */
function voiceDebug(event: string, detail?: Record<string, unknown>): void {
  if (process.env.NODE_ENV === "production") return;

  console.debug(`[voice] ${event}`, detail ?? "");
}

export interface VoiceInputProps {
  /** Receives the final transcript from whichever mode produced it. */
  onTranscript: (text: string) => void;
  disabled?: boolean;
  /** Notified whenever the resolved mode changes (for external indicators). */
  onModeChange?: (mode: ReturnType<typeof useVoiceMode>["mode"]) => void;
}

/**
 * VoiceInput is a smart router: it transcribes via the server (/ai/voice) when
 * the plan allows and quota remains, and otherwise falls back to on-device
 * transcription. Both paths emit a plain transcript through `onTranscript`, so
 * callers never need to know which one ran.
 *
 * Resilience rules:
 *   - a server failure degrades to on-device mode exactly once (no loops),
 *   - the microphone is always released,
 *   - the daily cap blocks every path with a friendly message,
 *   - a denied microphone stops rather than retrying.
 */
export function VoiceInput({ onTranscript, disabled, onModeChange }: VoiceInputProps) {
  const { mode, dailyRemaining } = useVoiceMode();
  const { config } = useVoiceConfig();
  const { language } = useVoiceLanguage(config);
  const recordHistory = useRecordBrowserHistory();

  const [isListening, setIsListening] = useState(false);
  const [isTranscribing, setIsTranscribing] = useState(false);
  const [stream, setStream] = useState<MediaStream | null>(null);

  const recognitionRef = useRef<SpeechRecognition | null>(null);
  const recorderRef = useRef<RecorderHandle | null>(null);
  // Analysis-only stream used for the VU meter in browser mode (the Web Speech
  // API does not expose its own capture stream).
  const browserStreamRef = useRef<MediaStream | null>(null);
  // A fallback may fire at most once per recording session.
  const fallbackUsedRef = useRef(false);

  useEffect(() => {
    onModeChange?.(mode);
    voiceDebug("mode", { mode, dailyRemaining });
  }, [mode, onModeChange, dailyRemaining]);

  /** Releases the browser-mode analysis stream (idempotent). */
  const releaseBrowserStream = useCallback(() => {
    const s = browserStreamRef.current;
    if (s) {
      for (const track of s.getTracks()) track.stop();
      browserStreamRef.current = null;
    }
    setStream(null);
  }, []);

  // Release every capture on unmount (mic must never stay on).
  useEffect(() => {
    return () => {
      recorderRef.current?.cancel();
      recognitionRef.current?.abort();
      const s = browserStreamRef.current;
      if (s) for (const track of s.getTracks()) track.stop();
    };
  }, []);



  // ─── on-device (browser) mode ──────────────────────────────────

  const startBrowser = useCallback(() => {
    const w = window as WindowWithSpeech;
    const SpeechRecognitionCtor = w.SpeechRecognition || w.webkitSpeechRecognition;
    if (!SpeechRecognitionCtor) {
      toast.error("Browser tidak mendukung input suara. Gunakan Chrome atau Edge.");
      return;
    }

    // The Web Speech API does not expose its capture stream, so open a separate
    // analysis-only stream purely for the VU meter (Q-05). It is released on
    // stop/unmount; failure is non-fatal (the meter simply stays flat).
    const media = navigator.mediaDevices;
    if (media?.getUserMedia) {
      media
        .getUserMedia({ audio: true })
        .then((s) => {
          browserStreamRef.current = s;
          setStream(s);
        })
        .catch(() => {});
    }

    const recognition = new SpeechRecognitionCtor();
    recognition.lang = speechLang(language);
    recognition.continuous = false;
    recognition.interimResults = true;
    recognition.maxAlternatives = 1;

    const startedAt = Date.now();
    recognition.onstart = () => setIsListening(true);

    const deliver = (text: string) => {
      onTranscript(text);
      // Record on-device transcriptions so the daily cap counts them (Q-03).
      // Best-effort: a failed write must never break the transcription UX.
      try {
        void Promise.resolve(
          recordHistory.mutateAsync({ transcript: text, language, durationMs: Date.now() - startedAt }),
        ).catch(() => {});
      } catch {
        // A synchronous throw from the mutation must not escape either.
      }
    };

    // The Web Speech API may emit MULTIPLE final results for one utterance
    // (and interim results before them). Without a guard, each final would be
    // sent to chat and recorded to history again — duplicating the message and
    // burning the daily cap. Track what was already emitted this session.
    let emitted = "";
    const emitOnce = (text: string) => {
      const t = text.trim();
      if (!t) return;
      // Only the newly-finalized suffix that we have not sent yet.
      if (emitted && t.startsWith(emitted)) {
        const delta = t.slice(emitted.length).trim();
        emitted = t;
        if (delta) deliver(delta);
        return;
      }
      if (t === emitted) return;
      emitted = t;
      deliver(t);
    };

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let final = "";
      for (let i = event.resultIndex; i < event.results.length; i++) {
        if (event.results[i].isFinal) final += event.results[i][0].transcript;
      }
      emitOnce(final);
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      setIsListening(false);
      releaseBrowserStream();
      if (event.error === "not-allowed") {
        toast.error("Izin mikrofon ditolak. Aktifkan di pengaturan browser Anda.");
      } else if (event.error === "network") {
        toast.error("Gagal terhubung ke layanan suara. Periksa koneksi internet Anda.");
      } else if (event.error !== "no-speech" && event.error !== "aborted") {
        toast.error(`Error pengenalan suara: ${event.error}.`);
      }
    };

    recognition.onend = () => {
      setIsListening(false);
      releaseBrowserStream();
    };
    recognitionRef.current = recognition;
    recognition.start();
  }, [language, onTranscript, recordHistory, releaseBrowserStream]);

  // ─── server mode ───────────────────────────────────────────────
  const stopServer = useCallback(async () => {
    const handle = recorderRef.current;
    if (!handle) return;
    recorderRef.current = null;
    setIsListening(false);
    setIsTranscribing(true);

    try {
      const recording = await handle.stop();
      setStream(null);
      const base64 = await blobToBase64(recording.blob);
      if (isAudioTooLarge(base64)) {
        toast.error("Rekaman terlalu panjang. Coba rekam lebih singkat.");
        return;
      }
      const res = await api.post("/ai/voice", { audio: base64, language }, {
        headers: { "X-Request-Id": crypto.randomUUID() },
      });
      if (res.data?.data?.success === false) {
        throw new Error(res.data?.data?.message || "voice_processing_failed");
      }
      const text = (res.data?.data?.transcript ?? "").trim();
      if (!text) {
        throw new Error("empty_transcript");
      }
      onTranscript(text);
    } catch (err) {
      const kind = classifyVoiceError(err);
      if (kind === "daily_limit") {
        toast.error("Batas suara harian tercapai. Reset besok.");
        return;
      }
      if (kind === "permission") {
        toast.error("Izin mikrofon ditolak. Aktifkan di pengaturan browser Anda.");
        return;
      }
      if (kind === "fallback" && !fallbackUsedRef.current && (config?.fallback_to_browser ?? true)) {
        fallbackUsedRef.current = true;
        voiceDebug("fallback", { reason: kind });
        toast.info("Layanan suara premium tidak tersedia — beralih ke mode dasar.");
        startBrowser();
        return;
      }
      toast.error("Gagal memproses suara. Silakan coba lagi.");
    } finally {
      setIsTranscribing(false);
      setStream(null);
    }
  }, [config?.fallback_to_browser, language, onTranscript, startBrowser]);

  // ─── toggle ────────────────────────────────────────────────────

  const handleToggle = () => {
    if (isTranscribing) return;

    if (isListening) {
      if (recorderRef.current) {
        void stopServer();
      } else {
        recognitionRef.current?.stop();
        setIsListening(false);
        releaseBrowserStream();
      }
      return;
    }

    // A fresh session may fall back again.
    fallbackUsedRef.current = false;

    if (mode === "daily_limit") {
      toast.error("Batas suara harian tercapai. Reset besok.");
      return;
    }
    if (mode === "unavailable") {
      toast.error("Browser Anda tidak mendukung input suara. Gunakan Chrome atau Edge.");
      return;
    }
    if (mode === "server") {
      void (async () => {
        try {
          const handle = await startRecording((config?.max_duration_seconds ?? 60) * 1000);
          recorderRef.current = handle;
          setStream(handle.stream);
          setIsListening(true);
        } catch (err) {
          const kind = classifyVoiceError(err);
          if (kind === "permission") {
            toast.error("Izin mikrofon ditolak. Aktifkan di pengaturan browser Anda.");
            return;
          }
          if (!fallbackUsedRef.current) {
            fallbackUsedRef.current = true;
            startBrowser();
            return;
          }
          toast.error("Gagal mengakses mikrofon.");
        }
      })();
      return;
    }
    startBrowser();
  };

  if (mode === "unavailable" && !isListening) {
    return (
      <Button
        variant="ghost"
        size="icon"
        title="Browser tidak mendukung input suara"
        aria-label="Input suara tidak tersedia"
        disabled
        className="text-muted-foreground"
      >
        <MicOff className="w-4 h-4" />
      </Button>
    );
  }

  const blocked = mode === "daily_limit";
  // The VU meter must be visible while recording in BOTH modes (Q-05). In
  // browser mode the Web Speech API owns the mic, so the recorder is null and we
  // open a lightweight analysis-only stream for the meter.
  const showMeter = isListening && (config?.vu_meter_enabled ?? true);
  const title = blocked
    ? "Batas suara harian tercapai"
    : isListening
      ? "Berhenti merekam"
      : mode === "server"
        ? "Input suara (AI premium)"
        : "Input suara (mode dasar)";

  return (
    <div className="flex flex-col gap-1">
      <Button
        variant="ghost"
        size="icon"
        title={title}
        aria-label={title}
        aria-pressed={isListening}
        onClick={handleToggle}
        disabled={disabled || (blocked && !isListening) || isTranscribing}
        className={`relative border-2 rounded-md border-foreground shadow-[2px_2px_0px_0px] shadow-foreground active:shadow-none active:translate-x-[2px] active:translate-y-[2px] ${
          isListening ? "bg-red-100 text-red-700 border-red-600" : "text-muted-foreground"
        }`}
      >
        {isTranscribing ? (
          <Loader2 className="w-4 h-4 animate-spin" />
        ) : isListening ? (
          <Square className="w-4 h-4" />
        ) : (
          <Mic className="w-4 h-4" />
        )}
        {/* Mode badge (K-VOI-07): the user always sees which path is active. */}
        <span className="absolute -bottom-1 -right-1 text-[8px] font-black px-1 rounded bg-foreground text-background leading-tight">
          {mode === "server" ? "AI" : "Dasar"}
        </span>
      </Button>
      {showMeter && <VoiceLevelMeter stream={stream} active={isListening} />}
    </div>
  );
}
