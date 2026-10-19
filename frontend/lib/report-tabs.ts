import { computePresetDates, type DateRangeValue, type PresetKey } from "@/components/shared/DateRangeFilter";

/**
 * The report page's tab structure and URL contract.
 *
 * Every tab is addressable via `?tab=<id>` so a tab is bookmarkable and
 * shareable, and switching tabs is a shallow URL update (no reload). Table
 * state (page/per_page/sort/order/q/filter_*) lives in the same query string,
 * so a link can reproduce a tab *and* its table view.
 */

// IDs are the URL slugs (English); labels are the visible Indonesian text.
export const REPORT_TABS = [
  { id: "summary", label: "Ringkasan" },
  { id: "spk", label: "Skor SPK" },
  { id: "category", label: "Kategori" },
  { id: "trend", label: "Tren" },
  { id: "budget", label: "Anggaran" },
  { id: "calendar", label: "Kalender" },
] as const;

export type ReportTabId = (typeof REPORT_TABS)[number]["id"];

export const DEFAULT_REPORT_TAB: ReportTabId = "summary";

/** The report "type" dimension (which slice of transactions is analysed). */
export const REPORT_TYPES = [
  { id: "expense", label: "Pengeluaran" },
  { id: "income", label: "Pemasukan" },
  { id: "savings", label: "Tabungan" },
] as const;

export type ReportTypeId = (typeof REPORT_TYPES)[number]["id"];

/** isReportTabId narrows an arbitrary string to a known tab. */
export function isReportTabId(v: string | null | undefined): v is ReportTabId {
  return !!v && REPORT_TABS.some((t) => t.id === v);
}

/** parseTab resolves a raw query value to a valid tab, defaulting safely. */
export function parseTab(v: string | null | undefined): ReportTabId {
  return isReportTabId(v) ? v : DEFAULT_REPORT_TAB;
}

/** parseReportType resolves a raw query value to a valid type. */
export function parseReportType(v: string | null | undefined): ReportTypeId {
  return REPORT_TYPES.some((t) => t.id === v) ? (v as ReportTypeId) : "expense";
}

/** typeLabel returns the Indonesian label for a report type. */
export function typeLabel(t: ReportTypeId): string {
  return REPORT_TYPES.find((x) => x.id === t)?.label ?? "Pengeluaran";
}

/**
 * trendIsGood says whether a positive change is a *good* thing for the given
 * report type. This is the fix for the original bug where an income increase
 * was coloured red: for income and savings, "up" is good; for expense, "up" is
 * bad.
 */
export function trendIsGood(type: ReportTypeId, change: number): boolean {
  if (change === 0) return true;
  const upIsGood = type === "income" || type === "savings";
  return upIsGood ? change > 0 : change < 0;
}

const PRESET_ALIASES: Record<string, PresetKey> = {
  today: "today",
  hari_ini: "today",
  this_week: "this_week",
  minggu_ini: "this_week",
  this_month: "this_month",
  bulan_ini: "this_month",
  last_month: "last_month",
  bulan_lalu: "last_month",
  "3months": "3months",
  "3_bulan": "3months",
  this_year: "this_year",
  tahun_ini: "this_year",
  custom: "custom",
  kustom: "custom",
};

export function parseReportFilterParams(
  sp: URLSearchParams | { get: (k: string) => string | null },
  now: Date
): DateRangeValue {
  const presetParam = sp.get("preset");
  const startDate = sp.get("start") || "";
  const endDate = sp.get("end") || "";
  const compareStart = sp.get("cs") || "";
  const compareEnd = sp.get("ce") || "";
  const enableCompare = sp.get("cmp") === "1";

  const presetKey = presetParam ? PRESET_ALIASES[presetParam.toLowerCase()] : undefined;

  if (presetKey && presetKey !== "custom") {
    const dates = computePresetDates(presetKey, now);
    return {
      preset: presetKey,
      startDate: dates.start,
      endDate: dates.end,
      compareStart,
      compareEnd,
      enableCompare,
    };
  }

  if (startDate && endDate) {
    return {
      preset: "custom",
      startDate,
      endDate,
      compareStart,
      compareEnd,
      enableCompare,
    };
  }

  const fallback = computePresetDates("this_month", now);
  return {
    preset: "this_month",
    startDate: fallback.start,
    endDate: fallback.end,
    compareStart,
    compareEnd,
    enableCompare,
  };
}

export function reportFilterToParams(f: DateRangeValue): Record<string, string | null> {
  const p: Record<string, string | null> = {};
  if (f.preset !== "custom") {
    p.preset = f.preset;
    p.start = null;
    p.end = null;
  } else {
    p.preset = "custom";
    p.start = f.startDate || null;
    p.end = f.endDate || null;
  }
  if (f.enableCompare) {
    p.cmp = "1";
    p.cs = f.compareStart || null;
    p.ce = f.compareEnd || null;
  } else {
    p.cmp = null;
    p.cs = null;
    p.ce = null;
  }
  return p;
}
