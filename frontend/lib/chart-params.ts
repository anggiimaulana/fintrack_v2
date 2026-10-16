export type ChartType = "line" | "bar" | "pie" | "area";
export type PeriodType = "week" | "month" | "3months" | "6months" | "year";
export type GroupBy = "summary" | "category";

export interface ChartState {
  chartType: ChartType;
  period: PeriodType;
  groupBy: GroupBy;
}

export const CHART_TYPES: ChartType[] = ["line", "bar", "pie", "area"];
export const PERIODS: PeriodType[] = ["week", "month", "3months", "6months", "year"];

export const DEFAULT_CHART_STATE: ChartState = {
  chartType: "bar",
  period: "month",
  groupBy: "summary",
};

/** Parse chart controls from the URL so every switch is shareable/bookmarkable. */
export function parseChartFromParams(sp: URLSearchParams): ChartState {
  const rawType = sp.get("chart");
  const rawPeriod = sp.get("period");
  const rawGroup = sp.get("group");
  const chartType = CHART_TYPES.includes(rawType as ChartType) ? (rawType as ChartType) : DEFAULT_CHART_STATE.chartType;
  const period = PERIODS.includes(rawPeriod as PeriodType) ? (rawPeriod as PeriodType) : DEFAULT_CHART_STATE.period;
  let groupBy: GroupBy = rawGroup === "category" ? "category" : "summary";
  // Bar & pie only support the summary view.
  if ((chartType === "bar" || chartType === "pie") && groupBy !== "summary") groupBy = "summary";
  return { chartType, period, groupBy };
}

export function chartToParams(c: ChartState): Record<string, string> {
  return { chart: c.chartType, period: c.period, group: c.groupBy };
}
