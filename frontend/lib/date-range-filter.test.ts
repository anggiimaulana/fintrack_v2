import { describe, it, expect } from "vitest";
import { computePresetDates } from "@/components/shared/DateRangeFilter";

describe("computePresetDates", () => {
  it("this_month returns full month from first day to last day", () => {
    const oct16 = new Date(2026, 9, 16);
    const res = computePresetDates("this_month", oct16);
    expect(res.start).toBe("2026-10-01");
    expect(res.end).toBe("2026-10-31");
  });

  it("handles February leap year vs non-leap year correctly", () => {
    const feb2024 = new Date(2024, 1, 10);
    expect(computePresetDates("this_month", feb2024).end).toBe("2024-02-29");

    const feb2025 = new Date(2025, 1, 10);
    expect(computePresetDates("this_month", feb2025).end).toBe("2025-02-28");
  });

  it("last_month returns full previous month", () => {
    const nov = new Date(2026, 10, 5);
    const res = computePresetDates("last_month", nov);
    expect(res.start).toBe("2026-10-01");
    expect(res.end).toBe("2026-10-31");
  });
});
