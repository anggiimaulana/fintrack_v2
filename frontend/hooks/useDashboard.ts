"use client";

import { useQuery } from "@tanstack/react-query";
import * as z from "zod";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export interface DashboardOverview {
  totalIncome: number;
  totalExpense: number;
  totalSavings: number;
  netBalance: number;
  savings: number;
  incomeTrend: number;
  expenseTrend: number;
  savingsTrend: number;
}

export function mapOverview(data: unknown): DashboardOverview {
  if (!data) {
    return { totalIncome: 0, totalExpense: 0, totalSavings: 0, netBalance: 0, savings: 0, incomeTrend: 0, expenseTrend: 0, savingsTrend: 0 };
  }
  const typedData = data as { current_month?: { income: number; expense: number; savings: number; balance: number }; change?: { income_change_pct: number; expense_change_pct: number; savings_change_pct: number } };
  const income = typedData.current_month?.income || 0;
  const expense = typedData.current_month?.expense || 0;
  const savings = typedData.current_month?.savings || 0;
  return {
    totalIncome: income,
    totalExpense: expense,
    totalSavings: savings,
    netBalance: typedData.current_month?.balance || (income - expense),
    savings: savings,
    incomeTrend: typedData.change?.income_change_pct || 0,
    expenseTrend: typedData.change?.expense_change_pct || 0,
    savingsTrend: typedData.change?.savings_change_pct || 0,
  };
}

export function useDashboardOverview(params?: { startDate?: string; endDate?: string; compareStart?: string; compareEnd?: string; preset?: string }) {
  const qKey = params?.preset || (params ? `${params.startDate}|${params.endDate}|${params.compareStart}|${params.compareEnd}` : "default");
  return useQuery({
    queryKey: [...queryKeys.dashboard.overview("default"), qKey],
    queryFn: async () => {
      let url = "/dashboard/overview";
      if (params?.startDate && params?.endDate) {
        url += `?start_date=${params.startDate}&end_date=${params.endDate}`;
        if (params.compareStart && params.compareEnd) {
          url += `&compare_start=${params.compareStart}&compare_end=${params.compareEnd}`;
        }
      }
      const res = await api.get(url);
      return mapOverview(res.data?.data);
    },
    staleTime: staleTimeConfig.dashboard,
  });
}

export interface ChartDataPoint {
  name: string;
  income: number;
  expense: number;
  savings: number;
}

function mapChartData(data: unknown): ChartDataPoint[] {
  if (!data || typeof data !== "object" || !("labels" in data) || !("series" in data)) return [];
  const typedData = data as { labels: string[]; series: Array<{ name: string; data: number[] }> };
  if (!typedData.labels || !typedData.series) return [];
  const incomeSeries = typedData.series.find((s) => s.name === "Pemasukan");
  const expenseSeries = typedData.series.find((s) => s.name === "Pengeluaran");
  const savingsSeries = typedData.series.find((s) => s.name === "Tabungan");
  return typedData.labels.map((label: string, i: number) => ({
    name: label,
    income: incomeSeries?.data?.[i] || 0,
    expense: expenseSeries?.data?.[i] || 0,
    savings: savingsSeries?.data?.[i] || 0,
  }));
}

export interface CategoryChartData {
  labels: string[];
  series: Array<{ name: string; data: number[]; color: string }>;
}

const categoryChartSchema = z.object({
  labels: z.array(z.string()),
  series: z.array(
    z.object({
      name: z.string(),
      data: z.array(z.number()),
      color: z.string().optional(),
    })
  ),
});

/**
 * Parse the raw `/dashboard/charts/categories` payload. The backend may emit
 * `null` for `labels`/`series` (Go marshals empty slices to null), so we treat
 * any malformed shape as "no data" rather than letting it reach the chart.
 */
export function normalizeCategoryChart(data: unknown): CategoryChartData | null {
  const parsed = categoryChartSchema.safeParse(data);
  if (!parsed.success) return null;
  return {
    labels: parsed.data.labels,
    series: parsed.data.series.map((s) => ({ name: s.name, data: s.data, color: s.color ?? "#3b5bdb" })),
  };
}

export function useCategoryChart(startDate: string, endDate: string, options?: { enabled?: boolean }) {
  return useQuery<CategoryChartData | null>({
    queryKey: ["dashboard", "category-chart", startDate, endDate],
    // Callers that will not render the chart (e.g. a plan-locked reports page)
    // pass `enabled: false` so the request is skipped entirely.
    enabled: options?.enabled ?? true,
    queryFn: async () => {
      const res = await api.get(`/dashboard/charts/categories?start_date=${startDate}&end_date=${endDate}`);
      return normalizeCategoryChart(res.data?.data);
    },
    staleTime: staleTimeConfig.dashboard,
    placeholderData: (prev) => prev,
  });
}

export function useDashboardCharts(params?: { period?: string; startDate?: string; endDate?: string; granularity?: string }) {
  const qKey = params ? `${params.period}|${params.startDate}|${params.endDate}|${params.granularity}` : "default";
  return useQuery({
    queryKey: [...queryKeys.dashboard.charts(params?.period || "default"), qKey],
    queryFn: async () => {
      let url = "/dashboard/charts?type=line";
      if (params?.startDate && params?.endDate) {
        url += `&start_date=${params.startDate}&end_date=${params.endDate}`;
      } else {
        url += `&period=${params?.period || "month"}`;
      }
      const res = await api.get(url);
      return mapChartData(res.data?.data);
    },
    staleTime: staleTimeConfig.dashboard,
    placeholderData: (prev) => prev,
  });
}
