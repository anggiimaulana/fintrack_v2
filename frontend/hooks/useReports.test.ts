import { describe, it, expect } from "vitest";
import { normalizeReport } from "./useReports";

describe("normalizeReport", () => {
  it("returns undefined for a missing payload", () => {
    expect(normalizeReport(null)).toBeUndefined();
    expect(normalizeReport(undefined)).toBeUndefined();
    expect(normalizeReport("nope")).toBeUndefined();
  });

  it("coerces null categories/monthly (Go nil slice) to empty arrays", () => {
    // The reports page renders `report.categories.map(...)` and
    // `report.monthly.map(...)`; a null here used to crash the page.
    const out = normalizeReport({ summary: null, categories: null, monthly: null });
    expect(out?.categories).toEqual([]);
    expect(out?.monthly).toEqual([]);
    expect(out?.summary).toEqual({ total_income: 0, total_expense: 0, total_savings: 0, surplus: 0 });
  });

  it("keeps a valid payload intact", () => {
    const payload = {
      summary: { total_income: 5_000_000, total_expense: 2_000_000, total_savings: 1_000_000, surplus: 3_000_000 },
      categories: [{ category_name: "Makanan", total: 500_000, percentage: 25, trend: 10, count: 12 }],
      monthly: [{ month: "Jan 2026", income: 5_000_000, expense: 2_000_000, savings: 1_000_000 }],
      currency_code: "JPY",
      rate_used: 110,
      rate_at: "2026-10-31T00:00:00Z",
    };
    expect(normalizeReport(payload)).toEqual(payload);
  });

  it("defaults currency metadata to base IDR when absent", () => {
    const out = normalizeReport({ categories: [], monthly: [] });
    expect(out?.currency_code).toBe("IDR");
    expect(out?.rate_used).toBe(1);
    expect(out?.rate_at).toBeNull();
  });

  it("defaults a missing summary to zeros", () => {
    const out = normalizeReport({ categories: [], monthly: [] });
    expect(out?.summary.total_income).toBe(0);
    expect(out?.summary.surplus).toBe(0);
  });
});
