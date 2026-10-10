import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";
import { TransactionData } from "@/components/transactions/TransactionRow";

interface TransactionFilter {
  type?: "income" | "expense" | "savings" | "all";
  page?: number;
  perPage?: number;
  search?: string;
  dateFrom?: string;
  dateTo?: string;
  sortBy?: "date" | "amount" | "category";
  sortOrder?: "asc" | "desc";
}

export function useTransactions(filter: TransactionFilter = {}) {
  const params = new URLSearchParams();
  if (filter.type && filter.type !== "all") params.set("type", filter.type);
  if (filter.page) params.set("page", filter.page.toString());
  if (filter.perPage) params.set("perPage", filter.perPage.toString());
  if (filter.search) params.set("search", filter.search);
  if (filter.dateFrom) params.set("date_from", filter.dateFrom);
  if (filter.dateTo) params.set("date_to", filter.dateTo);
  if (filter.sortBy) params.set("sortBy", filter.sortBy);
  if (filter.sortOrder) params.set("sortOrder", filter.sortOrder);

  return useQuery({
    queryKey: [...queryKeys.transactions.all, filter],
    queryFn: async () => {
      try {
        const res = await api.get(`/transactions?${params.toString()}`);
        return {
          data: res.data?.data || ([] as TransactionData[]),
          total: res.data?.total || 0,
          page: res.data?.page || 1,
          perPage: res.data?.per_page || 20,
        };
      } catch {
        return { data: [] as TransactionData[], total: 0, page: 1, perPage: 20 };
      }
    },
    staleTime: staleTimeConfig.transactions,
    placeholderData: (previousData) => previousData,
  });
}

export function useTransactionStats(period: string = "current_month") {
  return useQuery({
    queryKey: queryKeys.transactions.stats(period),
    queryFn: async () => {
      try {
        const res = await api.get(`/transactions/stats?period=${period}`);
        return res.data?.data;
      } catch {
        return null;
      }
    },
    staleTime: staleTimeConfig.transactions,
  });
}

export function useCreateTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Partial<TransactionData>) => {
      const res = await api.post("/transactions", data);
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

export function useUpdateTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...data }: { id: string } & Partial<TransactionData>) => {
      const res = await api.put(`/transactions/${id}`, data);
      return res.data?.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/transactions/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
    },
  });
}
