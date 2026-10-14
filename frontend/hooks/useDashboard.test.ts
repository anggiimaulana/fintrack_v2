import { describe, it, expect } from "vitest";
import { normalizeCategoryChart } from "./useDashboard";

describe("normalizeCategoryChart", () => {
  it("returns null when the API payload is null", () => {
    expect(normalizeCategoryChart(null)).toBeNull();
    expect(normalizeCategoryChart(undefined)).toBeNull();
  });

  it("returns null when labels/series are null (Go nil slice -> JSON null)", () => {
    // This is the exact shape that crashed MultiSeriesChart: labels present,
    // series null, so `series.forEach` threw a TypeError.
    expect(normalizeCategoryChart({ labels: ["2026-01"], series: null })).toBeNull();
    expect(normalizeCategoryChart({ labels: null, series: [] })).toBeNull();
  });

  it("returns null for malformed payloads", () => {
    expect(normalizeCategoryChart("nope")).toBeNull();
    expect(normalizeCategoryChart({ labels: ["a"], series: [{ name: "x" }] })).toBeNull();
    expect(normalizeCategoryChart({ series: [] })).toBeNull();
  });

  it("keeps a valid payload and defaults a missing color", () => {
    const out = normalizeCategoryChart({
      labels: ["2026-01", "2026-02"],
      series: [{ name: "Makanan", data: [1000, 2000], color: "#10b981" }],
    });
    expect(out).toEqual({
      labels: ["2026-01", "2026-02"],
      series: [{ name: "Makanan", data: [1000, 2000], color: "#10b981" }],
    });
  });

  it("fills a default color when the backend omits it", () => {
    const out = normalizeCategoryChart({
      labels: ["2026-01"],
      series: [{ name: "Makanan", data: [1000] }],
    });
    expect(out?.series[0].color).toBe("#3b5bdb");
  });
});
