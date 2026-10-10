"use client";

import { useQuery } from "@tanstack/react-query";
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

export function mapOverview(data: any): DashboardOverview {
  if (!data) {
    return { totalIncome: 0, totalExpense: 0, totalSavings: 0, netBalance: 0, savings: 0, incomeTrend: 0, expenseTrend: 0, savingsTrend: 0 };
  }
  const income = data.current_month?.income || 0;
  const expense = data.current_month?.expense || 0;
  const savings = data.current_month?.savings || 0;
  return {
    totalIncome: income,
    totalExpense: expense,
    totalSavings: savings,
    netBalance: data.current_month?.balance || (income - expense),
    savings: savings,
    incomeTrend: data.change?.income_change_pct || 0,
    expenseTrend: data.change?.expense_change_pct || 0,
    savingsTrend: data.change?.savings_change_pct || 0,
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

function mapChartData(data: any): ChartDataPoint[] {
  if (!data?.labels || !data?.series) return [];
  const incomeSeries = data.series.find((s: any) => s.name === "Pemasukan");
  const expenseSeries = data.series.find((s: any) => s.name === "Pengeluaran");
  const savingsSeries = data.series.find((s: any) => s.name === "Tabungan");
  return data.labels.map((label: string, i: number) => ({
    name: label,
    income: incomeSeries?.data?.[i] || 0,
    expense: expenseSeries?.data?.[i] || 0,
    savings: savingsSeries?.data?.[i] || 0,
  }));
}

export function useCategoryChart(startDate: string, endDate: string) {
  return useQuery({
    queryKey: ["dashboard", "category-chart", startDate, endDate],
    queryFn: async () => {
      const res = await api.get(`/dashboard/charts/categories?start_date=${startDate}&end_date=${endDate}`);
      return (res.data?.data as { labels: string[]; series: Array<{ name: string; data: number[]; color: string }> }) || null;
    },
    staleTime: staleTimeConfig.dashboard,
    placeholderData: (prev: any) => prev,
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
