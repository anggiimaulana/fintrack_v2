"use client";

import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export interface ReportSummary {
  total_income: number;
  total_expense: number;
  total_savings: number;
  surplus: number;
}

export interface CategoryReport {
  category_name: string;
  total: number;
  percentage: number;
  trend: number;
  count: number;
}

export interface MonthlyComparison {
  month: string;
  income: number;
  expense: number;
  savings: number;
}

export interface ReportResponse {
  summary: ReportSummary;
  categories: CategoryReport[];
  monthly: MonthlyComparison[];
}

const emptySummary: ReportSummary = { total_income: 0, total_expense: 0, total_savings: 0, surplus: 0 };

/**
 * Parse the raw `/reports` payload. The Go backend marshals nil slices to JSON
 * `null` (empty period, no transactions), so `categories`/`monthly` can arrive
 * as `null`. Callers render these with `.map()`/`.length`, so coerce them to
 * arrays here instead of letting a null crash the reports page.
 */
export function normalizeReport(data: unknown): ReportResponse | undefined {
  if (!data || typeof data !== "object") return undefined;
  const d = data as Partial<ReportResponse>;
  return {
    summary: d.summary ?? emptySummary,
    categories: Array.isArray(d.categories) ? d.categories : [],
    monthly: Array.isArray(d.monthly) ? d.monthly : [],
  };
}

export function useReports(
  params?: { period?: string; type?: string; startDate?: string; endDate?: string; compareStart?: string; compareEnd?: string },
  options?: { enabled?: boolean }
) {
  const period = params?.period || "current_month";
  const qKey = `${params?.type || "expense"}|${params?.startDate ? `${params.startDate}|${params.endDate}|${params.compareStart}|${params.compareEnd}` : period}`;
  return useQuery({
    queryKey: [...queryKeys.reports.byPeriod(period), qKey],
    // The `/reports` endpoint is gated by the `reports_export` plan feature and
    // returns 403 for Free accounts. Callers that already know the plan lacks
    // the feature pass `enabled: false` so we never fire a doomed request (and
    // never show an error state for something the plan simply does not include).
    enabled: options?.enabled ?? true,
    queryFn: async () => {
      const url = "/reports";
      const queryParams = new URLSearchParams();
      if (params?.type) queryParams.set("type", params.type);
      if (params?.startDate && params?.endDate) {
        queryParams.set("start_date", params.startDate);
        queryParams.set("end_date", params.endDate);
        if (params.compareStart && params.compareEnd) {
          queryParams.set("compare_start", params.compareStart);
          queryParams.set("compare_end", params.compareEnd);
        }
      } else {
        queryParams.set("period", period);
      }
      const res = await api.get(`${url}?${queryParams.toString()}`);
      return normalizeReport(res.data?.data);
    },
    staleTime: staleTimeConfig.reports,
    placeholderData: (prev) => prev,
  });
}
