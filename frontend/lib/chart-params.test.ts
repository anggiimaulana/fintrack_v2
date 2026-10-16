import { describe, it, expect } from "vitest";
import { parseChartFromParams, chartToParams, DEFAULT_CHART_STATE } from "./chart-params";

describe("chart URL params", () => {
  it("defaults to bar/summary/month when the URL has no chart params", () => {
    expect(parseChartFromParams(new URLSearchParams(""))).toEqual(DEFAULT_CHART_STATE);
    expect(DEFAULT_CHART_STATE.chartType).toBe("bar");
  });

  it("reads chart, period and group from the URL", () => {
    const state = parseChartFromParams(new URLSearchParams("chart=line&period=year&group=category"));
    expect(state).toEqual({ chartType: "line", period: "year", groupBy: "category" });
  });

  it("ignores unknown values and falls back to defaults", () => {
    const state = parseChartFromParams(new URLSearchParams("chart=doughnut&period=decade&group=foo"));
    expect(state).toEqual(DEFAULT_CHART_STATE);
  });

  it("forces summary grouping for bar and pie charts", () => {
    expect(parseChartFromParams(new URLSearchParams("chart=bar&group=category")).groupBy).toBe("summary");
    expect(parseChartFromParams(new URLSearchParams("chart=pie&group=category")).groupBy).toBe("summary");
  });

  it("serializes every chart switch into the URL", () => {
    expect(chartToParams({ chartType: "bar", period: "month", groupBy: "summary" })).toEqual({
      chart: "bar",
      period: "month",
      group: "summary",
    });
  });

  it("round-trips state through params", () => {
    const state = { chartType: "area", period: "6months", groupBy: "category" } as const;
    expect(parseChartFromParams(new URLSearchParams(chartToParams(state)))).toEqual(state);
  });
});
