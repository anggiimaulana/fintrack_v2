import { describe, it, expect } from "vitest";
import { formatRupiah, formatTanggal } from "./format";

describe("formatRupiah", () => {
  it("formats 1000 as Rp1.000", () => {
    // Intl uses a non-breaking space after "Rp"; normalize before comparing.
    expect(formatRupiah(1000).replace(/\u00a0/g, "")).toBe("Rp1.000");
  });

  it("formats large amounts with dot separators", () => {
    expect(formatRupiah(1234567).replace(/\u00a0/g, "")).toBe("Rp1.234.567");
  });

  it("returns Rp0 for null/undefined/NaN", () => {
    expect(formatRupiah(null).replace(/\u00a0/g, "")).toBe("Rp0");
    expect(formatRupiah(undefined).replace(/\u00a0/g, "")).toBe("Rp0");
    expect(formatRupiah("abc").replace(/\u00a0/g, "")).toBe("Rp0");
  });

  it("accepts numeric strings", () => {
    expect(formatRupiah("5000").replace(/\u00a0/g, "")).toBe("Rp5.000");
  });
});

describe("formatTanggal", () => {
  it("formats an ISO date in Indonesian", () => {
    expect(formatTanggal("2026-04-15")).toBe("15 Apr 2026");
  });

  it("returns '-' for empty or invalid input", () => {
    expect(formatTanggal(null)).toBe("-");
    expect(formatTanggal("not-a-date")).toBe("-");
  });
});
