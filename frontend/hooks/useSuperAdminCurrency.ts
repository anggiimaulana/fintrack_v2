"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import type { CurrencyMeta } from "@/lib/currency";

export type AdminCurrency = CurrencyMeta;

export type CurrencyPayload = {
  code: string;
  name: string;
  symbol?: string;
  locale?: string;
  decimal_digits?: number;
  is_active?: boolean;
  sort_order?: number;
  rate_source?: string;
};

export type RateRow = {
  id?: string;
  currency_code: string;
  rate_to_base: number;
  effective_at: string;
  source?: string;
  note?: string | null;
};

/** Full catalog (including inactive) for the Super Admin console. */
export function useAdminCurrencies() {
  return useQuery<AdminCurrency[]>({
    queryKey: queryKeys.currency.admin,
    queryFn: async () => {
      const res = await api.get("/admin/currencies");
      return (res.data?.data ?? []) as AdminCurrency[];
    },
  });
}

/** Append-only rate history for a currency. */
export function useCurrencyRates(code: string | null) {
  return useQuery<RateRow[]>({
    queryKey: queryKeys.currency.rates(code ?? ""),
    queryFn: async () => {
      const res = await api.get(`/admin/currency-rates?code=${encodeURIComponent(code!)}`);
      return (res.data?.data ?? []) as RateRow[];
    },
    enabled: Boolean(code),
  });
}

export function useUpsertCurrency() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: CurrencyPayload) => {
      const res = await api.post("/admin/currencies", payload);
      return res.data?.data as AdminCurrency;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.admin });
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.all });
    },
  });
}

export function useAddCurrencyRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: RateRow) => {
      const res = await api.post("/admin/currency-rates", payload);
      return res.data?.data;
    },
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.rates(variables.currency_code) });
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.admin });
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.all });
    },
  });
}

export function useSyncCurrencyRates() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post("/admin/currency-sync");
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.admin });
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.all });
    },
  });
}
