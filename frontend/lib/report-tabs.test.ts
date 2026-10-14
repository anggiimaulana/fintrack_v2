import { describe, it, expect } from "vitest";
import {
  REPORT_TABS,
  DEFAULT_REPORT_TAB,
  parseTab,
  parseReportType,
  typeLabel,
  trendIsGood,
  isReportTabId,
} from "./report-tabs";

describe("parseTab", () => {
  it("accepts every declared tab", () => {
    for (const t of REPORT_TABS) {
      expect(parseTab(t.id)).toBe(t.id);
    }
  });

  it("falls back to the default for unknown/empty input", () => {
    expect(parseTab(null)).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab(undefined)).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab("")).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab("nope")).toBe(DEFAULT_REPORT_TAB);
  });
});

describe("isReportTabId", () => {
  it("narrows correctly", () => {
    expect(isReportTabId("spk")).toBe(true);
    expect(isReportTabId("bogus")).toBe(false);
    expect(isReportTabId(null)).toBe(false);
  });
});

describe("parseReportType", () => {
  it("accepts known types and defaults to expense", () => {
    expect(parseReportType("income")).toBe("income");
    expect(parseReportType("savings")).toBe("savings");
    expect(parseReportType("expense")).toBe("expense");
    expect(parseReportType("garbage")).toBe("expense");
    expect(parseReportType(null)).toBe("expense");
  });
});

describe("typeLabel", () => {
  it("renders Indonesian labels", () => {
    expect(typeLabel("income")).toBe("Pemasukan");
    expect(typeLabel("expense")).toBe("Pengeluaran");
    expect(typeLabel("savings")).toBe("Tabungan");
  });
});

describe("trendIsGood (the income-increase-is-green fix)", () => {
  it("treats rising income as good and rising expense as bad", () => {
    expect(trendIsGood("income", 10)).toBe(true);
    expect(trendIsGood("income", -10)).toBe(false);
    expect(trendIsGood("expense", 10)).toBe(false);
    expect(trendIsGood("expense", -10)).toBe(true);
  });

  it("treats rising savings as good", () => {
    expect(trendIsGood("savings", 5)).toBe(true);
    expect(trendIsGood("savings", -5)).toBe(false);
  });

  it("treats no change as neutral-good", () => {
    expect(trendIsGood("expense", 0)).toBe(true);
    expect(trendIsGood("income", 0)).toBe(true);
  });
});
