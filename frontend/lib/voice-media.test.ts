import { describe, it, expect } from "vitest";
import { pickAudioMimeType } from "./voice-recorder";
import { rmsToLevel } from "./voice-level";
import { isAudioTooLarge, MAX_AUDIO_BASE64 } from "./voice-audio";

describe("pickAudioMimeType", () => {
  it("prefers opus webm when supported", () => {
    expect(pickAudioMimeType((t) => t === "audio/webm;codecs=opus")).toBe("audio/webm;codecs=opus");
  });

  it("falls through to the next supported type", () => {
    expect(pickAudioMimeType((t) => t === "audio/mp4")).toBe("audio/mp4");
  });

  it("returns empty string when nothing is supported (browser default)", () => {
    expect(pickAudioMimeType(() => false)).toBe("");
  });
});

describe("rmsToLevel", () => {
  it("returns 0 for silence (all samples centred at 128)", () => {
    expect(rmsToLevel(new Uint8Array(256).fill(128))).toBe(0);
  });

  it("returns 1 (clamped) for a full-scale signal", () => {
    const loud = new Uint8Array(256);
    for (let i = 0; i < loud.length; i++) loud[i] = i % 2 === 0 ? 0 : 255;
    expect(rmsToLevel(loud)).toBe(1);
  });

  it("returns 0 for an empty buffer", () => {
    expect(rmsToLevel(new Uint8Array(0))).toBe(0);
  });

  it("produces a level between 0 and 1 for a moderate signal", () => {
    const buf = new Uint8Array(256);
    for (let i = 0; i < buf.length; i++) buf[i] = 128 + Math.round(30 * Math.sin(i));
    const level = rmsToLevel(buf);
    expect(level).toBeGreaterThan(0);
    expect(level).toBeLessThanOrEqual(1);
  });
});

describe("isAudioTooLarge", () => {
  it("accepts a payload at the limit", () => {
    expect(isAudioTooLarge("a".repeat(MAX_AUDIO_BASE64))).toBe(false);
  });

  it("rejects a payload over the limit", () => {
    expect(isAudioTooLarge("a".repeat(MAX_AUDIO_BASE64 + 1))).toBe(true);
  });
});
