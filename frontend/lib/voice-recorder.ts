/**
 * MediaRecorder wrapper for the server transcription path.
 *
 * Responsibilities kept here (so the component stays declarative):
 *   - acquire the microphone via getUserMedia,
 *   - pick a supported audio MIME type,
 *   - auto-stop at the configured duration,
 *   - ALWAYS release the microphone tracks (no "mic stays on" bug).
 *
 * The active MediaStream is exposed so the level meter can analyse it without
 * opening a second capture.
 */

export interface Recording {
  blob: Blob;
  durationMs: number;
}

export interface RecorderHandle {
  /** The live stream (used by the VU meter). */
  stream: MediaStream;
  /** Stop and resolve with the recorded audio. */
  stop: () => Promise<Recording>;
  /** Stop and discard the recording. */
  cancel: () => void;
}

/** Preferred MIME types, best first. The first supported one wins. */
const PREFERRED_MIME_TYPES = [
  "audio/webm;codecs=opus",
  "audio/webm",
  "audio/ogg;codecs=opus",
  "audio/mp4",
];

/** Picks the first MIME type the browser can record, or "" to use its default. */
export function pickAudioMimeType(isTypeSupported: (t: string) => boolean): string {
  for (const t of PREFERRED_MIME_TYPES) {
    if (isTypeSupported(t)) return t;
  }
  return "";
}

/**
 * startRecording begins capturing audio. It rejects with the raw DOMException
 * (carrying `name`) so the caller can classify a denied microphone via
 * classifyVoiceError.
 */
export async function startRecording(maxDurationMs: number): Promise<RecorderHandle> {
  if (typeof navigator === "undefined" || !navigator.mediaDevices?.getUserMedia) {
    const err = new Error("MediaRecorder is not supported");
    err.name = "NotSupportedError";
    throw err;
  }
  if (typeof MediaRecorder === "undefined") {
    const err = new Error("MediaRecorder is not supported");
    err.name = "NotSupportedError";
    throw err;
  }

  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });

  const mimeType = pickAudioMimeType((t) => MediaRecorder.isTypeSupported(t));
  const recorder = mimeType ? new MediaRecorder(stream, { mimeType }) : new MediaRecorder(stream);
  const chunks: BlobPart[] = [];
  recorder.ondataavailable = (e) => {
    if (e.data && e.data.size > 0) chunks.push(e.data);
  };

  const startedAt = Date.now();
  const { promise, resolve } = Promise.withResolvers<Recording>();
  let stopped = false;

  /** Releases the microphone — idempotent so stop/cancel/auto-stop cannot double-free. */
  const releaseStream = () => {
    for (const track of stream.getTracks()) track.stop();
  };

  recorder.onstop = () => {
    const durationMs = Date.now() - startedAt;
    releaseStream();
    resolve({ blob: new Blob(chunks, { type: recorder.mimeType || "audio/webm" }), durationMs });
  };

  recorder.start();

  const autoStop = setTimeout(() => {
    if (recorder.state !== "inactive") recorder.stop();
  }, maxDurationMs);

  return {
    stream,
    stop: () => {
      if (!stopped) {
        stopped = true;
        clearTimeout(autoStop);
        if (recorder.state !== "inactive") recorder.stop();
      }
      return promise;
    },
    cancel: () => {
      if (stopped) return;
      stopped = true;
      clearTimeout(autoStop);
      if (recorder.state !== "inactive") recorder.stop();
      // Release immediately; onstop also releases (idempotent).
      releaseStream();
    },
  };
}
