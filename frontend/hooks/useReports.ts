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

export function useReports(params?: { period?: string; type?: string; startDate?: string; endDate?: string; compareStart?: string; compareEnd?: string }) {
  const period = params?.period || "current_month";
  const qKey = `${params?.type || "expense"}|${params?.startDate ? `${params.startDate}|${params.endDate}|${params.compareStart}|${params.compareEnd}` : period}`;
  return useQuery({
    queryKey: [...queryKeys.reports.byPeriod(period), qKey],
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
      return res.data?.data as ReportResponse;
    },
    staleTime: staleTimeConfig.dashboard,
  });
}
