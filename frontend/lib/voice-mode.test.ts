import { describe, it, expect } from "vitest";
import { decideVoiceMode, hasHeadroom, type VoiceCapabilities, type VoiceEntitlement } from "./voice-mode";

const cap = (over: Partial<VoiceCapabilities> = {}): VoiceCapabilities => ({
  browserSTT: false,
  recorder: false,
  ...over,
});

const ent = (over: Partial<VoiceEntitlement> = {}): VoiceEntitlement => ({
  hasVoiceFeature: false,
  hasQuota: true,
  hasDailyQuota: true,
  ...over,
});

describe("decideVoiceMode", () => {
  it("picks server when the plan, quota and recorder all allow it", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true }), cap({ recorder: true, browserSTT: true }))).toBe("server");
  });

  it("falls back to browser when the recorder is unavailable", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true }), cap({ recorder: false, browserSTT: true }))).toBe("browser");
  });

  it("falls back to browser when the token quota is exhausted", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true, hasQuota: false }), cap({ recorder: true, browserSTT: true }))).toBe("browser");
  });

  it("falls back to browser when the plan lacks the voice feature", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: false }), cap({ recorder: true, browserSTT: true }))).toBe("browser");
  });

  it("prefers server when both paths are available", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true }), cap({ recorder: true, browserSTT: true }))).toBe("server");
  });

  it("is unavailable when neither path exists", () => {
    expect(decideVoiceMode(ent(), cap())).toBe("unavailable");
  });

  it("is unavailable when the feature is present but no recorder or STT exists", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true }), cap())).toBe("unavailable");
  });

  it("is unavailable when quota is exhausted and there is no browser STT", () => {
    expect(decideVoiceMode(ent({ hasVoiceFeature: true, hasQuota: false }), cap())).toBe("unavailable");
  });

  it("blocks every path with daily_limit when the daily cap is exhausted", () => {
    // The daily gate wins over every other signal, even a fully-capable server.
    expect(
      decideVoiceMode(ent({ hasVoiceFeature: true, hasDailyQuota: false }), cap({ recorder: true, browserSTT: true })),
    ).toBe("daily_limit");
    expect(decideVoiceMode(ent({ hasDailyQuota: false }), cap({ browserSTT: true }))).toBe("daily_limit");
    expect(decideVoiceMode(ent({ hasDailyQuota: false }), cap())).toBe("daily_limit");
  });
});

describe("hasHeadroom", () => {
  it("treats an absent entry as NO headroom (matches the fail-closed gate)", () => {
    expect(hasHeadroom(undefined)).toBe(false);
  });

  it("treats a zero limit (not in plan) as no headroom", () => {
    expect(hasHeadroom({ used: 0, limit: 0, remaining: 0, unlimited: false })).toBe(false);
  });

  it("treats unlimited as headroom", () => {
    expect(hasHeadroom({ used: 999, limit: -1, remaining: -1, unlimited: true })).toBe(true);
  });

  it("has headroom while at least one use remains", () => {
    expect(hasHeadroom({ used: 4, limit: 5, remaining: 1, unlimited: false })).toBe(true);
  });

  it("has no headroom at zero remaining", () => {
    expect(hasHeadroom({ used: 5, limit: 5, remaining: 0, unlimited: false })).toBe(false);
  });
});
