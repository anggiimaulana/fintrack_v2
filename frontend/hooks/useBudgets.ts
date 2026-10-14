"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";

export interface Budget {
  id: string;
  user_id: string;
  category_id: string;
  category_name: string;
  amount_idr: number;
  created_at: string;
  updated_at: string;
}

export interface BudgetStatus extends Budget {
  actual: number;
  remaining: number;
  used_pct: number;
  over: boolean;
}

export interface BudgetPayload {
  category_id: string;
  amount_idr: number;
}

/**
 * useBudgets loads the user's category budgets and their actual spend for a
 * period. Gated by the `advanced_budgeting` feature; callers pass `enabled:
 * false` for a plan that lacks it so we never fire a doomed request.
 */
export function useBudgets(
  period: { startDate: string; endDate: string } | undefined,
  options?: { enabled?: boolean }
) {
  const query = period ? `?start_date=${period.startDate}&end_date=${period.endDate}` : "";
  return useQuery({
    queryKey: [...queryKeys.budgets.all, query],
    enabled: options?.enabled ?? true,
    queryFn: async (): Promise<BudgetStatus[]> => {
      const res = await api.get(`/budgets/status${query}`);
      return res.data?.data ?? [];
    },
    staleTime: staleTimeConfig.dashboard,
  });
}

export function useUpsertBudget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: BudgetPayload) => {
      const res = await api.post("/budgets", payload);
      return res.data?.data as Budget;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.budgets.all }),
  });
}

export function useDeleteBudget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/budgets/${id}`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: queryKeys.budgets.all }),
  });
}
