import { describe, it, expect } from "vitest";
import { backoffDelay, BACKOFF_STEPS_MS } from "./use-realtime-stream";

describe("backoffDelay", () => {
  // Jitter of 1 returns the full base delay, so the schedule is deterministic.
  const fullJitter = () => 1;

  it("follows the documented 1s→16s progression", () => {
    expect(backoffDelay(0, fullJitter)).toBe(1000);
    expect(backoffDelay(1, fullJitter)).toBe(2000);
    expect(backoffDelay(2, fullJitter)).toBe(4000);
    expect(backoffDelay(3, fullJitter)).toBe(8000);
    expect(backoffDelay(4, fullJitter)).toBe(16000);
  });

  it("caps at the final step instead of growing without bound", () => {
    expect(backoffDelay(5, fullJitter)).toBe(16000);
    expect(backoffDelay(50, fullJitter)).toBe(16000);
  });

  it("never exceeds the cap even with maximum jitter", () => {
    const max = BACKOFF_STEPS_MS[BACKOFF_STEPS_MS.length - 1];
    for (let attempt = 0; attempt < 10; attempt++) {
      expect(backoffDelay(attempt, fullJitter)).toBeLessThanOrEqual(max);
    }
  });

  it("applies jitter within [half, full] of the base delay", () => {
    // Jitter of 0 halves the delay: spread so a fleet of clients does not
    // reconnect in lockstep after a restart.
    expect(backoffDelay(0, () => 0)).toBe(500);
    expect(backoffDelay(2, () => 0)).toBe(2000);
    // Mid jitter lands between the two bounds.
    const mid = backoffDelay(2, () => 0.5);
    expect(mid).toBeGreaterThanOrEqual(2000);
    expect(mid).toBeLessThanOrEqual(4000);
  });

  it("treats a negative attempt as the first step", () => {
    expect(backoffDelay(-3, fullJitter)).toBe(1000);
  });

  it("uses Math.random by default and stays within bounds", () => {
    for (let i = 0; i < 50; i++) {
      const d = backoffDelay(3);
      expect(d).toBeGreaterThanOrEqual(4000);
      expect(d).toBeLessThanOrEqual(8000);
    }
  });
});
