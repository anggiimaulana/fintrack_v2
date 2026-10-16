import { describe, it, expect } from "vitest";
import {
  REPORT_TABS,
  DEFAULT_REPORT_TAB,
  parseTab,
  parseReportType,
  typeLabel,
  trendIsGood,
  isReportTabId,
  parseReportFilterParams,
  reportFilterToParams,
} from "./report-tabs";

describe("parseTab", () => {
  it("accepts every declared tab", () => {
    for (const t of REPORT_TABS) {
      expect(parseTab(t.id)).toBe(t.id);
    }
  });

  it("resolves English aliases to declared tab IDs", () => {
    expect(parseTab("summary")).toBe("ringkasan");
    expect(parseTab("calendar")).toBe("kalender");
    expect(parseTab("category")).toBe("kategori");
    expect(parseTab("trend")).toBe("tren");
    expect(parseTab("budget")).toBe("budget");
    expect(parseTab("spk")).toBe("spk");
  });

  it("falls back to the default for unknown/empty input", () => {
    expect(parseTab(null)).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab(undefined)).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab("")).toBe(DEFAULT_REPORT_TAB);
    expect(parseTab("nope")).toBe(DEFAULT_REPORT_TAB);
  });
});

describe("parseReportFilterParams and reportFilterToParams", () => {
  const fixedNow = new Date("2026-10-16T12:00:00Z");

  it("parses preset param in English", () => {
    const sp = new URLSearchParams("preset=today");
    const res = parseReportFilterParams(sp, fixedNow);
    expect(res.preset).toBe("today");
    expect(res.startDate).toBe("2026-10-16");
    expect(res.endDate).toBe("2026-10-16");
  });

  it("parses custom date range in English params start and end", () => {
    const sp = new URLSearchParams("start=2026-09-01&end=2026-09-30");
    const res = parseReportFilterParams(sp, fixedNow);
    expect(res.preset).toBe("custom");
    expect(res.startDate).toBe("2026-09-01");
    expect(res.endDate).toBe("2026-09-30");
  });

  it("serializes filter to English URL params", () => {
    const presetFilter = {
      preset: "this_week" as const,
      startDate: "2026-10-12",
      endDate: "2026-10-18",
      compareStart: "",
      compareEnd: "",
      enableCompare: false,
    };
    expect(reportFilterToParams(presetFilter)).toEqual({
      preset: "this_week",
      start: null,
      end: null,
      cmp: null,
      cs: null,
      ce: null,
    });

    const customFilter = {
      preset: "custom" as const,
      startDate: "2026-09-01",
      endDate: "2026-09-30",
      compareStart: "2026-08-01",
      compareEnd: "2026-08-31",
      enableCompare: true,
    };
    expect(reportFilterToParams(customFilter)).toEqual({
      preset: "custom",
      start: "2026-09-01",
      end: "2026-09-30",
      cmp: "1",
      cs: "2026-08-01",
      ce: "2026-08-31",
    });
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
