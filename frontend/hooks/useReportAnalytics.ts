"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

// ─── Report analytics (deep-dive tabs) ────────────────────────
// Mirrors the Go domain types in internal/domain/report_analytics.go.

export interface PeriodDelta {
  current: number;
  previous: number;
  change: number;
  change_pct: number;
}

export interface PeriodComparisonSet {
  label: string;
  start: string;
  end: string;
  income: PeriodDelta;
  expense: PeriodDelta;
  savings: PeriodDelta;
  surplus: PeriodDelta;
}

export interface PeriodComparison {
  mom: PeriodComparisonSet;
  qoq: PeriodComparisonSet;
  yoy: PeriodComparisonSet;
}

export interface CategoryMover {
  category_name: string;
  current: number;
  previous: number;
  change: number;
  change_pct: number;
}

export interface TopMovers {
  up: CategoryMover[];
  down: CategoryMover[];
}

export interface DailyPoint {
  date: string;
  income: number;
  expense: number;
  savings: number;
  net: number;
}

export interface CategoryAnomaly {
  category_name: string;
  current: number;
  mean: number;
  std_dev: number;
  z_score: number;
  direction: "spike" | "drop";
}

export interface FinancialRatios {
  savings_rate: number;
  expense_ratio: number;
  runway_months: number;
  months_spanned: number;
}

export interface BestWorstPeriod {
  best_month: string;
  best_value: number;
  worst_month: string;
  worst_value: number;
}

export interface ReportAnalytics {
  comparison: PeriodComparison;
  top_movers: TopMovers;
  daily: DailyPoint[];
  anomalies: CategoryAnomaly[];
  ratios: FinancialRatios;
  best_worst: BestWorstPeriod;
}

export interface AnalyticsParams {
  type?: string;
  period?: string;
  startDate?: string;
  endDate?: string;
  compareStart?: string;
  compareEnd?: string;
}

/** Builds the query string shared by the report and analytics endpoints. */
export function buildReportQuery(params: AnalyticsParams): string {
  const q = new URLSearchParams();
  if (params.type) q.set("type", params.type);
  if (params.startDate && params.endDate) {
    q.set("start_date", params.startDate);
    q.set("end_date", params.endDate);
    if (params.compareStart && params.compareEnd) {
      q.set("compare_start", params.compareStart);
      q.set("compare_end", params.compareEnd);
    }
  } else if (params.period) {
    q.set("period", params.period);
  }
  return q.toString();
}

/**
 * useReportAnalytics loads the deep-dive analytics for the report tabs.
 *
 * `enabled` lets the caller skip the request entirely for a Free account (the
 * endpoint shares the `reports_export` gate and would 403).
 */
export function useReportAnalytics(params: AnalyticsParams, options?: { enabled?: boolean }) {
  const query = buildReportQuery(params);
  return useQuery({
    queryKey: queryKeys.reports.analytics(query),
    enabled: options?.enabled ?? true,
    queryFn: async (): Promise<ReportAnalytics | undefined> => {
      const res = await api.get(`/reports/analytics?${query}`);
      return res.data?.data as ReportAnalytics;
    },
    staleTime: staleTimeConfig.reports,
    placeholderData: (prev) => prev,
  });
}

/** useExportReport downloads a report file and triggers the browser save. */
export function useExportReport() {
  return async (format: "csv" | "xlsx" | "pdf", params: AnalyticsParams) => {
    const query = buildReportQuery(params);
    const res = await api.get(`/reports/export?format=${format}&${query}`, {
      responseType: "blob",
    });

    // Derive the filename from Content-Disposition when present, else build one.
    const disposition = (res.headers?.["content-disposition"] ?? "") as string;
    const match = /filename="?([^"]+)"?/.exec(disposition);
    const filename = match?.[1] ?? `laporan.${format}`;

    const contentType = res.headers?.["content-type"];
    const blob = new Blob([res.data as BlobPart], { type: typeof contentType === "string" ? contentType : undefined });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };
}
