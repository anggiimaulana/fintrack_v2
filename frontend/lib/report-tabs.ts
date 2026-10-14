/**
 * The report page's tab structure and URL contract.
 *
 * Every tab is addressable via `?tab=<id>` so a tab is bookmarkable and
 * shareable, and switching tabs is a shallow URL update (no reload). Table
 * state (page/per_page/sort/order/q/filter_*) lives in the same query string,
 * so a link can reproduce a tab *and* its table view.
 */

export const REPORT_TABS = [
  { id: "ringkasan", label: "Ringkasan" },
  { id: "spk", label: "Skor SPK" },
  { id: "kategori", label: "Kategori" },
  { id: "tren", label: "Tren" },
  { id: "budget", label: "Anggaran" },
  { id: "kalender", label: "Kalender" },
] as const;

export type ReportTabId = (typeof REPORT_TABS)[number]["id"];

export const DEFAULT_REPORT_TAB: ReportTabId = "ringkasan";

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
