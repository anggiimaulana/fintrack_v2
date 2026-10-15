"use client";

import { useQuery, keepPreviousData } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import type { SortSpec } from "@/lib/table-params";

/**
 * User-facing usage analytics (`GET /api/v1/me/usage/*`).
 *
 * The backend derives both the chart series and the history ledger from the
 * append-only token_ledger plus transactions, so these hooks only mirror state —
 * they never compute consumption.
 */

// ─── Types ────────────────────────────────────────────────────

/** One named data series aligned to the shared label axis. */
export interface UsageSeries {
  key: string;
  label: string;
  unit: string;
  data: number[];
}

export interface UsageTimeseries {
  range: string;
  granularity: "hour" | "day" | "month" | string;
  start: string;
  end: string;
  labels: string[];
  series: UsageSeries[];
}

/** One row of the consumption history ledger. */
export interface UsageEvent {
  id: string;
  kind: string;
  kind_label: string;
  description: string;
  amount: number;
  unit: string;
  cost_idr: number;
  created_at: string;
}

export interface UsageHistoryParams {
  page: number;
  perPage: number;
  sorts: SortSpec[];
  q: string;
  filters: Record<string, string>;
  startDate?: string;
  endDate?: string;
}

export interface UsageHistoryResult {
  items: UsageEvent[];
  total: number;
  page: number;
  perPage: number;
}

/** Serialise the table state into a stable cache key + request query string. */
function historyQuery(p: UsageHistoryParams): string {
  const params = new URLSearchParams();
  params.set("page", String(p.page));
  params.set("per_page", String(p.perPage));
  if (p.sorts.length > 0) {
    params.set("sort", p.sorts.map((s) => s.column).join(","));
    params.set("order", p.sorts.map((s) => s.order).join(","));
  }
  if (p.q) params.set("q", p.q);
  for (const [k, v] of Object.entries(p.filters)) {
    if (v) params.set(`filter_${k}`, v);
  }
  if (p.startDate) params.set("start_date", p.startDate);
  if (p.endDate) params.set("end_date", p.endDate);
  return params.toString();
}

// ─── Queries ──────────────────────────────────────────────────

export function useUsageTimeseries(
  range: string,
  opts?: { start?: string; end?: string; enabled?: boolean }
) {
  const params = new URLSearchParams();
  params.set("range", range);
  if (range === "custom" && opts?.start && opts?.end) {
    params.set("start", opts.start);
    params.set("end", opts.end);
  }
  const key = params.toString();

  const query = useQuery({
    queryKey: queryKeys.usage.timeseries(key),
    queryFn: async (): Promise<UsageTimeseries> => {
      const res = await api.get(`/me/usage/timeseries?${key}`);
      return res.data?.data as UsageTimeseries;
    },
    enabled: opts?.enabled ?? true,
    staleTime: 1000 * 60,
    placeholderData: keepPreviousData,
  });

  return {
    timeseries: query.data ?? null,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
  };
}

export function useUsageHistory(p: UsageHistoryParams, opts?: { enabled?: boolean }) {
  const qs = historyQuery(p);

  const query = useQuery({
    queryKey: queryKeys.usage.history(qs),
    queryFn: async (): Promise<UsageHistoryResult> => {
      const res = await api.get(`/me/usage/history?${qs}`);
      return {
        items: (res.data?.data ?? []) as UsageEvent[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? p.page,
        perPage: res.data?.meta?.per_page ?? p.perPage,
      };
    },
    enabled: opts?.enabled ?? true,
    staleTime: 1000 * 30,
    placeholderData: keepPreviousData,
  });

  return {
    events: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    refetch: query.refetch,
  };
}
