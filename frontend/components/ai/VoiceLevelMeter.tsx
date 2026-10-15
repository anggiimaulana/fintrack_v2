"use client";

import { useEffect, useRef, useState } from "react";
import { createLevelMeter } from "@/lib/voice-level";

export interface VoiceLevelMeterProps {
  /** The live capture stream, or null when not recording. */
  stream: MediaStream | null;
  /** Whether to analyse; false tears the meter down. */
  active: boolean;
}

/**
 * A horizontal microphone level bar (Q-05). It analyses the SAME stream the
 * recorder is using — it never opens a second capture — and releases the
 * AudioContext on stop/unmount so nothing leaks.
 */
export function VoiceLevelMeter({ stream, active }: VoiceLevelMeterProps) {
  const [level, setLevel] = useState(0);
  const meterRef = useRef<ReturnType<typeof createLevelMeter> | null>(null);

  useEffect(() => {
    if (!active || !stream) {
      setLevel(0);
      return;
    }
    const meter = createLevelMeter();
    meterRef.current = meter;
    meter.start(stream, setLevel);
    return () => {
      meter.stop();
      meterRef.current = null;
    };
  }, [active, stream]);

  const pct = Math.round(level * 100);

  return (
    <div
      className="h-2.5 w-full border-2 rounded-md border-foreground bg-muted overflow-hidden"
      role="progressbar"
      aria-label="Level mikrofon"
      aria-valuenow={pct}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <div
        className="h-full bg-emerald-500 transition-[width] duration-75"
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}
