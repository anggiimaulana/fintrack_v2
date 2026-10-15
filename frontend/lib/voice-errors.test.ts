import { describe, it, expect } from "vitest";
import { classifyVoiceError } from "./voice-errors";

/** Builds an axios-like error. */
function http(status: number, error?: string) {
  return { response: { status, data: error ? { error } : undefined } };
}

describe("classifyVoiceError", () => {
  it("maps quota errors to fallback", () => {
    expect(classifyVoiceError(http(402, "limit_exceeded"))).toBe("fallback");
    expect(classifyVoiceError(http(429, "insufficient_quota"))).toBe("fallback");
  });

  it("maps the plan gate to fallback", () => {
    expect(classifyVoiceError(http(403, "feature_not_in_plan"))).toBe("fallback");
  });

  it("maps rate limiting to fallback", () => {
    expect(classifyVoiceError(http(429, "rate_limited"))).toBe("fallback");
  });

  it("maps server errors to fallback", () => {
    expect(classifyVoiceError(http(500))).toBe("fallback");
    expect(classifyVoiceError(http(503))).toBe("fallback");
  });

  it("maps a network error (no response) to fallback", () => {
    expect(classifyVoiceError(new Error("Network Error"))).toBe("fallback");
    expect(classifyVoiceError({})).toBe("fallback");
  });

  it("maps the daily cap to daily_limit, distinct from fallback", () => {
    expect(classifyVoiceError(http(402, "daily_limit_exceeded"))).toBe("daily_limit");
    expect(classifyVoiceError(http(429, "voice_daily_limit"))).toBe("daily_limit");
  });

  it("maps a 400 (client bug) to fatal", () => {
    expect(classifyVoiceError(http(400, "file_too_large"))).toBe("fatal");
  });

  it("maps microphone denial to permission", () => {
    expect(classifyVoiceError({ name: "NotAllowedError" })).toBe("permission");
    expect(classifyVoiceError({ name: "NotFoundError" })).toBe("permission");
  });

  it("never throws on odd inputs", () => {
    expect(classifyVoiceError(null)).toBe("fallback");
    expect(classifyVoiceError(undefined)).toBe("fallback");
    expect(classifyVoiceError("boom")).toBe("fallback");
  });
});
