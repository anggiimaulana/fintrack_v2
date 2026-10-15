import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as React from "react";
import { axe } from "@/test/setup";
import { VoiceInput } from "./VoiceInput";

// ─── module mocks ──────────────────────────────────────────────────

const postMock = vi.fn();

vi.mock("@/lib/api", () => ({
  default: { get: vi.fn(), post: (...args: unknown[]) => postMock(...args), delete: vi.fn() },
}));

const modeState = { mode: "browser" as string, canRecord: true, dailyRemaining: -1 };
vi.mock("@/hooks/useVoiceMode", () => ({
  useVoiceMode: () => ({ ...modeState, isPremium: modeState.mode === "server", capabilities: {} }),
}));

vi.mock("@/hooks/useVoiceConfig", () => ({
  useVoiceConfig: () => ({
    config: {
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
      server_available: true,
    },
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useVoiceLanguage", () => ({
  useVoiceLanguage: () => ({ language: "id", canChoose: false, options: ["id"], setLanguage: vi.fn() }),
}));

const historyMutate = vi.fn();
vi.mock("@/hooks/useVoiceHistory", () => ({
  useRecordBrowserHistory: () => ({ mutateAsync: historyMutate }),
}));

const startRecordingMock = vi.fn();
vi.mock("@/lib/voice-recorder", () => ({
  startRecording: (...args: unknown[]) => startRecordingMock(...args),
}));

vi.mock("@/lib/voice-audio", () => ({
  blobToBase64: vi.fn(async () => "BASE64AUDIO"),
  isAudioTooLarge: () => false,
}));

vi.mock("@/lib/voice-level", () => ({
  createLevelMeter: () => ({ start: vi.fn(), stop: vi.fn() }),
}));

// ─── helpers ───────────────────────────────────────────────────────

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
}

/** A minimal SpeechRecognition stub the component can drive. */
class FakeRecognition {
  lang = "";
  continuous = false;
  interimResults = false;
  maxAlternatives = 1;
  onstart: (() => void) | null = null;
  onresult: ((e: unknown) => void) | null = null;
  onerror: ((e: unknown) => void) | null = null;
  onend: (() => void) | null = null;
  start() {
    this.onstart?.();
  }
  stop() {
    this.onend?.();
  }
  abort() {}
  /** Emit a final transcript. */
  emit(text: string) {
    this.onresult?.({
      resultIndex: 0,
      results: Object.assign([{ isFinal: true, 0: { transcript: text } }], { length: 1 }),
    });
  }
}

let lastRecognition: FakeRecognition | null = null;

/** The window shape we augment for the test (named, not an inline cast). */
interface TestWindow {
  SpeechRecognition?: new () => FakeRecognition;
}

beforeEach(() => {
  vi.clearAllMocks();
  modeState.mode = "browser";
  modeState.canRecord = true;
  lastRecognition = null;
  historyMutate.mockResolvedValue({});
  // Assign through a typed reference so no unchecked cast is needed.
  const w = window as unknown as TestWindow;
  w.SpeechRecognition = class extends FakeRecognition {
    constructor() {
      super();
      lastRecognition = this;
    }
  };
});

// ─── tests ─────────────────────────────────────────────────────────

describe("VoiceInput", () => {
  it("emits the transcript from on-device mode without calling the server", async () => {
    const onTranscript = vi.fn();
    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    fireEvent.click(screen.getByLabelText(/Input suara/));
    expect(lastRecognition).not.toBeNull();
    lastRecognition!.emit("beli kopi 25 ribu");

    await waitFor(() => expect(onTranscript).toHaveBeenCalledWith("beli kopi 25 ribu"));
    // Browser mode must never hit /ai/voice.
    expect(postMock).not.toHaveBeenCalledWith("/ai/voice", expect.anything(), expect.anything());
  });

  it("records the on-device transcript to the history endpoint (daily cap accounting)", async () => {
    historyMutate.mockResolvedValue({});
    const onTranscript = vi.fn();
    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    fireEvent.click(screen.getByLabelText(/Input suara/));
    lastRecognition!.emit("catat makan siang");

    await waitFor(() => expect(historyMutate).toHaveBeenCalled());
    expect(historyMutate.mock.calls[0][0]).toMatchObject({ transcript: "catat makan siang" });
  });

  it("does not duplicate a repeated final result (protects the daily cap)", async () => {
    historyMutate.mockResolvedValue({});
    const onTranscript = vi.fn();
    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    fireEvent.click(screen.getByLabelText(/Input suara/));
    // The Web Speech API can fire the SAME final result more than once.
    lastRecognition!.emit("beli kopi");
    lastRecognition!.emit("beli kopi");

    await waitFor(() => expect(onTranscript).toHaveBeenCalled());
    expect(onTranscript).toHaveBeenCalledTimes(1);
    expect(historyMutate).toHaveBeenCalledTimes(1);
  });

  it("emits only the new suffix when the final result grows", async () => {
    historyMutate.mockResolvedValue({});
    const onTranscript = vi.fn();
    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    fireEvent.click(screen.getByLabelText(/Input suara/));
    lastRecognition!.emit("beli kopi");
    lastRecognition!.emit("beli kopi susu");

    await waitFor(() => expect(onTranscript).toHaveBeenCalledTimes(2));
    expect(onTranscript).toHaveBeenNthCalledWith(1, "beli kopi");
    expect(onTranscript).toHaveBeenNthCalledWith(2, "susu");
  });

  it("sends audio to /ai/voice in server mode and emits the returned transcript", async () => {
    modeState.mode = "server";
    const onTranscript = vi.fn();
    const stop = vi.fn(async () => ({ blob: new Blob(["x"]), durationMs: 1200 }));
    startRecordingMock.mockResolvedValue({ stream: {}, stop, cancel: vi.fn() });
    postMock.mockResolvedValue({ data: { data: { transcript: "halo dunia" } } });

    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    fireEvent.click(screen.getByLabelText(/Input suara/));
    await waitFor(() => expect(startRecordingMock).toHaveBeenCalled());

    // Press again to stop & submit.
    fireEvent.click(screen.getByLabelText(/Berhenti merekam/));
    await waitFor(() => expect(onTranscript).toHaveBeenCalledWith("halo dunia"));
    expect(postMock).toHaveBeenCalledWith("/ai/voice", expect.objectContaining({ audio: "BASE64AUDIO" }), expect.anything());
  });

  it("blocks the mic when the daily cap is exhausted", () => {
    modeState.mode = "daily_limit";
    modeState.canRecord = false;
    const onTranscript = vi.fn();
    render(<VoiceInput onTranscript={onTranscript} />, { wrapper: wrapper(newClient()) });

    const btn = screen.getByLabelText(/Batas suara harian tercapai/);
    expect(btn).toBeDisabled();
    fireEvent.click(btn);
    expect(onTranscript).not.toHaveBeenCalled();
  });

  it("shows a disabled mic when voice is unavailable", () => {
    modeState.mode = "unavailable";
    render(<VoiceInput onTranscript={vi.fn()} />, { wrapper: wrapper(newClient()) });
    expect(screen.getByLabelText("Input suara tidak tersedia")).toBeDisabled();
  });

  it("has no accessibility violations", async () => {
    const { container } = render(<VoiceInput onTranscript={vi.fn()} />, { wrapper: wrapper(newClient()) });
    expect(await axe(container)).toHaveNoViolations();
  });
});
