import { describe, it, expect } from "vitest";
import { quotaSeverity, summariseQuota } from "./quota-monitor";

describe("quotaSeverity", () => {
  it("bands >= 90% as critical", () => {
    expect(quotaSeverity(90)).toBe("critical");
    expect(quotaSeverity(100)).toBe("critical");
    expect(quotaSeverity(150)).toBe("critical");
  });

  it("bands < 90% as warning", () => {
    expect(quotaSeverity(80)).toBe("warning");
    expect(quotaSeverity(89.9)).toBe("warning");
    expect(quotaSeverity(0)).toBe("warning");
  });
});

describe("summariseQuota", () => {
  it("counts critical and warning rows", () => {
    const rows = [
      { used_percent: 95 },
      { used_percent: 91 },
      { used_percent: 82 },
      { used_percent: 80 },
    ];
    expect(summariseQuota(rows)).toEqual({ total: 4, critical: 2, warning: 2 });
  });

  it("handles an empty list", () => {
    expect(summariseQuota([])).toEqual({ total: 0, critical: 0, warning: 0 });
  });
});
