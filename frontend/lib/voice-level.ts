/**
 * Microphone level analysis for the VU meter (Q-05).
 *
 * Wraps the Web Audio API so the component only receives a 0..1 level. All
 * resources (AudioContext + animation frame) are released by `stop()`, which is
 * what prevents the classic "AudioContext leak" in a long-lived page.
 */

export interface LevelMeter {
  /** Begin analysing `stream`, invoking onLevel(0..1) each animation frame. */
  start: (stream: MediaStream, onLevel: (level: number) => void) => void;
  /** Release the AudioContext and stop the animation loop. Idempotent. */
  stop: () => void;
}

/**
 * rmsToLevel converts a byte time-domain buffer into a 0..1 display level.
 * Pure so the scaling is unit-testable without an AudioContext.
 */
export function rmsToLevel(samples: Uint8Array): number {
  if (samples.length === 0) return 0;
  let sum = 0;
  for (let i = 0; i < samples.length; i++) {
    const x = (samples[i] - 128) / 128; // centre the unsigned byte around 0
    sum += x * x;
  }
  const rms = Math.sqrt(sum / samples.length);
  // Amplify: speech RMS is small, so a raw value would barely move the bar.
  return Math.min(1, rms * 3);
}

type AudioContextCtor = new (options?: AudioContextOptions) => AudioContext;

function getAudioContextCtor(): AudioContextCtor | null {
  if (typeof window === "undefined") return null;
  const w = window as unknown as {
    AudioContext?: AudioContextCtor;
    webkitAudioContext?: AudioContextCtor;
  };
  return w.AudioContext ?? w.webkitAudioContext ?? null;
}

/** createLevelMeter builds a level meter bound to no stream yet. */
export function createLevelMeter(): LevelMeter {
  let ctx: AudioContext | null = null;
  let source: MediaStreamAudioSourceNode | null = null;
  let analyser: AnalyserNode | null = null;
  let raf = 0;

  const cleanup = () => {
    if (raf) {
      cancelAnimationFrame(raf);
      raf = 0;
    }
    try {
      source?.disconnect();
    } catch {
      // disconnect on a already-torn-down node throws; ignore.
    }
    source = null;
    analyser = null;
    if (ctx) {
      void ctx.close().catch(() => {});
      ctx = null;
    }
  };

  return {
    start(stream, onLevel) {
      cleanup();
      const Ctor = getAudioContextCtor();
      if (!Ctor) return;
      try {
        ctx = new Ctor();
        source = ctx.createMediaStreamSource(stream);
        analyser = ctx.createAnalyser();
        analyser.fftSize = 256;
        analyser.smoothingTimeConstant = 0.8;
        source.connect(analyser);
      } catch {
        cleanup();
        return;
      }

      const buf = new Uint8Array(analyser.fftSize);
      const tick = () => {
        if (!analyser) return;
        analyser.getByteTimeDomainData(buf);
        onLevel(rmsToLevel(buf));
        raf = requestAnimationFrame(tick);
      };
      raf = requestAnimationFrame(tick);
    },
    stop: cleanup,
  };
}
