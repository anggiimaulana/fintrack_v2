import { describe, it, expect } from "vitest";
import { remainingMs, isExpired, formatCountdown, urgencyFor, isCancellable } from "./invoice-expiry";

const NOW = Date.parse("2026-10-14T12:00:00Z");

describe("remainingMs", () => {
  it("returns the gap to the due date", () => {
    expect(remainingMs("2026-10-14T13:00:00Z", NOW)).toBe(60 * 60 * 1000);
  });
  it("clamps to 0 once past due", () => {
    expect(remainingMs("2026-10-14T11:00:00Z", NOW)).toBe(0);
  });
  it("is 0 for null/invalid input", () => {
    expect(remainingMs(null, NOW)).toBe(0);
    expect(remainingMs(undefined, NOW)).toBe(0);
    expect(remainingMs("not-a-date", NOW)).toBe(0);
  });
});

describe("isExpired", () => {
  it("is true when the due date has passed", () => {
    expect(isExpired("2026-10-14T11:59:59Z", NOW)).toBe(true);
    expect(isExpired("2026-10-14T12:00:00Z", NOW)).toBe(true); // boundary
  });
  it("is false while time remains", () => {
    expect(isExpired("2026-10-14T12:00:01Z", NOW)).toBe(false);
  });
  it("is false when there is no due date", () => {
    expect(isExpired(null, NOW)).toBe(false);
  });
});

describe("formatCountdown", () => {
  it("formats hours and minutes", () => {
    expect(formatCountdown(2 * 3600_000 + 15 * 60_000)).toBe("2j 15m");
  });
  it("formats minutes and seconds", () => {
    expect(formatCountdown(45 * 60_000 + 10_000)).toBe("45m 10s");
  });
  it("formats seconds only", () => {
    expect(formatCountdown(12_000)).toBe("12s");
  });
  it("returns empty when nothing remains", () => {
    expect(formatCountdown(0)).toBe("");
    expect(formatCountdown(-5)).toBe("");
  });
});

describe("urgencyFor", () => {
  it("classifies by remaining time", () => {
    expect(urgencyFor(0)).toBe("expired");
    expect(urgencyFor(30 * 60_000)).toBe("critical");
    expect(urgencyFor(3 * 3600_000)).toBe("warning");
    expect(urgencyFor(48 * 3600_000)).toBe("normal");
  });
});

describe("isCancellable", () => {
  it("allows issued and past_due only", () => {
    expect(isCancellable("issued")).toBe(true);
    expect(isCancellable("past_due")).toBe(true);
    expect(isCancellable("paid")).toBe(false);
    expect(isCancellable("voided")).toBe(false);
    expect(isCancellable("expired")).toBe(false);
  });
});
