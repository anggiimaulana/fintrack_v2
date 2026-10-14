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
