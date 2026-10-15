import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { queryKeys } from "./cache-keys";

export function useInvalidateQueries() {
  const queryClient = useQueryClient();

  const invalidateTransactions = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
    queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
    queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    queryClient.invalidateQueries({ queryKey: queryKeys.reports.all });
    // A transaction create/delete moves the `max_transactions` usage counter, so
    // the quota bars must refresh too (the SSE usage_changed event covers other
    // tabs; this covers the tab that made the change).
    queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
  }, [queryClient]);

  /**
   * invalidateEntitlements refreshes only the quota/entitlements query. Use it
   * after an action that consumes quota without changing the transaction lists
   * (e.g. an AI chat turn charging the token quota).
   */
  const invalidateEntitlements = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
  }, [queryClient]);

  const invalidateDashboard = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
  }, [queryClient]);

  const invalidateSpk = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
  }, [queryClient]);

  const invalidateCategories = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.categories.all });
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories("income") });
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories("expense") });
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories("savings") });
  }, [queryClient]);

  const invalidateAdmin = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.users });
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.stats });
    queryClient.invalidateQueries({ queryKey: queryKeys.admin.recentUsers });
  }, [queryClient]);

  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries();
  }, [queryClient]);

  return {
    invalidateTransactions,
    invalidateEntitlements,
    invalidateDashboard,
    invalidateSpk,
    invalidateCategories,
    invalidateAdmin,
    invalidateAll,
    queryClient,
  };
}

export const staleTimeConfig = {
  transactions: 1000 * 60 * 5,
  dashboard: 1000 * 60 * 2,
  categories: 1000 * 60 * 30,
  spk: 1000 * 60 * 10,
  ai: 1000 * 60 * 1,
  admin: 1000 * 60 * 5,
  profile: 1000 * 60 * 60,
};
