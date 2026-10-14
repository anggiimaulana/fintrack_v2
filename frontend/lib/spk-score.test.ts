import { describe, it, expect } from "vitest";
import {
  scoreIndicator,
  weightedScore,
  statusFor,
  invertSubScore,
  projectScore,
} from "./spk-score";
import type { SpkMetricModel } from "@/hooks/useSpkMetrics";

const model: SpkMetricModel = {
  indicators: [
    { key: "saving_ratio", label: "Rasio Tabungan", weight: 0.5, target: 0.2, type: "saving_ratio" },
    { key: "expense_trend", label: "Tren Pengeluaran", weight: 0.5, target: 1.0, type: "expense_trend" },
  ],
  status_thresholds: [
    { min: 80, label: "Sangat Baik" },
    { min: 50, label: "Baik" },
    { min: 0, label: "Kritis" },
  ],
};

describe("scoreIndicator", () => {
  it("matches the backend for higher_better", () => {
    expect(scoreIndicator({ ...model.indicators[0] }, 0.2)).toBe(100);
    expect(scoreIndicator({ ...model.indicators[0] }, 0.5)).toBe(100);
    expect(scoreIndicator({ ...model.indicators[0] }, 0.1)).toBe(50);
    expect(scoreIndicator({ ...model.indicators[0] }, 0)).toBe(0);
  });

  it("matches the backend for lower_better", () => {
    const ind = { key: "x", label: "X", weight: 1, target: 0.1, type: "expense_trend" as const, direction: "lower_better" as const };
    expect(scoreIndicator(ind, 0.1)).toBe(100);
    expect(scoreIndicator(ind, 0.05)).toBe(100);
    expect(scoreIndicator(ind, 0.2)).toBe(50);
    expect(scoreIndicator(ind, 0)).toBe(100);
  });

  it("is safe with a zero target", () => {
    expect(scoreIndicator({ key: "x", label: "X", weight: 1, target: 0, type: "consistency" }, 5)).toBe(0);
  });
});

describe("weightedScore", () => {
  it("applies the weights", () => {
    // 100*0.5 + 0*0.5 = 50
    expect(weightedScore(model, { saving_ratio: 100, expense_trend: 0 })).toBe(50);
  });

  it("treats a missing sub-score as zero", () => {
    expect(weightedScore(model, { saving_ratio: 100 })).toBe(50);
  });

  it("clamps to 0..100", () => {
    expect(weightedScore(model, { saving_ratio: 100, expense_trend: 100 })).toBe(100);
  });
});

describe("statusFor", () => {
  it("picks the highest threshold the score reaches", () => {
    expect(statusFor(model.status_thresholds, 95)).toBe("Sangat Baik");
    expect(statusFor(model.status_thresholds, 80)).toBe("Sangat Baik");
    expect(statusFor(model.status_thresholds, 79)).toBe("Baik");
    expect(statusFor(model.status_thresholds, 50)).toBe("Baik");
    expect(statusFor(model.status_thresholds, 10)).toBe("Kritis");
  });

  it("falls back to the lowest threshold below all of them", () => {
    const th = [{ min: 90, label: "A" }, { min: 60, label: "B" }];
    expect(statusFor(th, 10)).toBe("B");
  });

  it("handles an empty threshold list", () => {
    expect(statusFor([], 50)).toBe("—");
  });
});

describe("invertSubScore", () => {
  it("recovers the raw value for higher_better", () => {
    // 50% of a target-0.2 indicator => raw 0.1
    expect(invertSubScore(model.indicators[0], 50)).toBeCloseTo(0.1, 6);
  });

  it("returns null for a capped sub-score", () => {
    expect(invertSubScore(model.indicators[0], 0)).toBeNull();
    const lower = { key: "x", label: "X", weight: 1, target: 0.1, type: "expense_trend" as const, direction: "lower_better" as const };
    expect(invertSubScore(lower, 100)).toBeNull();
  });
});

describe("projectScore", () => {
  it("improves the score when a weakness is reduced", () => {
    // Saving ratio at 50 (raw 0.1), trend perfect.
    const current = { saving_ratio: 50, expense_trend: 100 };
    // Boost the raw saving ratio by 20% => raw 0.12 => sub-score 60.
    const p = projectScore(model, current, { saving_ratio: 20 });
    expect(p.current).toBe(75); // 50*0.5 + 100*0.5
    expect(p.projected).toBe(80); // 60*0.5 + 100*0.5
    expect(p.delta).toBe(5);
    expect(p.projectedStatus).toBe("Sangat Baik");
  });

  it("is a no-op with no changes", () => {
    const current = { saving_ratio: 50, expense_trend: 100 };
    const p = projectScore(model, current, {});
    expect(p.projected).toBe(p.current);
    expect(p.delta).toBe(0);
  });

  it("reports per-indicator before/after", () => {
    const current = { saving_ratio: 50, expense_trend: 100 };
    const p = projectScore(model, current, { saving_ratio: 20 });
    const sr = p.perIndicator.find((x) => x.key === "saving_ratio");
    expect(sr?.before).toBe(50);
    expect(sr?.after).toBe(60);
  });

  it("can lower the score with a negative change", () => {
    const current = { saving_ratio: 100, expense_trend: 100 };
    const p = projectScore(model, current, { saving_ratio: -50 });
    expect(p.projected).toBeLessThan(p.current);
    expect(p.delta).toBeLessThan(0);
  });
});
