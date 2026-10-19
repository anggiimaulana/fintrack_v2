"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import type { CurrencyMeta } from "@/lib/currency";

/** The active, selectable currency catalog (with the current rate). */
export function useCurrencies(options?: { enabled?: boolean }) {
  return useQuery<CurrencyMeta[]>({
    queryKey: queryKeys.currency.all,
    queryFn: async () => {
      const res = await api.get("/currencies");
      return (res.data?.data ?? []) as CurrencyMeta[];
    },
    // Rates are managed dynamically by a Super Admin; keep the window short so a
    // new rate is picked up within a minute (a currency change invalidates
    // everything anyway).
    staleTime: 1000 * 60,
    enabled: options?.enabled ?? true,
  });
}

/** Change the signed-in user's recording/display currency. */
export function useSetDisplayCurrency() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (code: string) => {
      const res = await api.put("/me/preferences/currency", { currency_code: code });
      return res.data?.data;
    },
    onSuccess: () => {
      // Reports are converted server-side for the display currency, so swap the
      // cached catalog-derived views. Invalidating everything is the safe, small
      // move: currency changes are rare and every money view must be recomputed.
      queryClient.invalidateQueries({ queryKey: ["session", "me"] });
      queryClient.invalidateQueries({ queryKey: queryKeys.currency.all });
      queryClient.invalidateQueries();
    },
  });
}
