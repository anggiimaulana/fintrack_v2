import { useQuery, useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";
import { TransactionData } from "@/components/transactions/TransactionRow";
import { notifyEntitlementError } from "@/lib/entitlement-errors";
import { toast } from "sonner";

/**
 * Error handling for transaction mutations: a plan gate (402/403) explains
 * itself and links to the subscription page; a 409 optimistic-lock conflict
 * needs its own wording; everything else gets the generic message.
 */
function notifyTransactionError(err: unknown, fallback: string) {
  if (notifyEntitlementError(err)) return;
  const status = (err as { response?: { status?: number } })?.response?.status;
  if (status === 409) {
    toast.error("Transaksi ini sudah diubah di tempat lain. Muat ulang lalu coba lagi.");
    return;
  }
  toast.error(fallback);
}

interface TransactionFilter {
  type?: "income" | "expense" | "savings" | "all";
  page?: number;
  perPage?: number;
  search?: string;
  dateFrom?: string;
  dateTo?: string;
  sortBy?: string;
  sortOrder?: "asc" | "desc";
  sort?: string;
  order?: "asc" | "desc";
}

export function useTransactions(filter: TransactionFilter = {}, options?: { enabled?: boolean }) {
  const params = new URLSearchParams();
  if (filter.type && filter.type !== "all") params.set("type", filter.type);
  if (filter.page) params.set("page", filter.page.toString());
  if (filter.perPage) params.set("perPage", filter.perPage.toString());
  if (filter.search) params.set("search", filter.search);
  if (filter.dateFrom) params.set("date_from", filter.dateFrom);
  if (filter.dateTo) params.set("date_to", filter.dateTo);
  const sort = filter.sort || filter.sortBy;
  if (sort) {
    params.set("sort", sort);
    params.set("sortBy", sort);
  }
  const order = filter.order || filter.sortOrder;
  if (order) {
    params.set("order", order);
    params.set("sortOrder", order);
  }

  return useQuery({
    queryKey: [...queryKeys.transactions.all, filter],
    queryFn: async () => {
      const res = await api.get(`/transactions?${params.toString()}`);
      return {
        data: (res.data?.data ?? []) as TransactionData[],
        total: res.data?.meta?.total ?? res.data?.total ?? 0,
        page: res.data?.meta?.page ?? res.data?.page ?? 1,
        perPage: res.data?.meta?.per_page ?? res.data?.per_page ?? 20,
      };
    },
    staleTime: staleTimeConfig.transactions,
    placeholderData: (previousData) => previousData,
    enabled: options?.enabled,
  });
}
export interface TransactionStatsData {
  total_income: number;
  total_expense: number;
  total_savings: number;
  net_balance: number;
  income_count: number;
  expense_count: number;
  savings_count: number;
}

export function useTransactionStats(params?: string | { startDate?: string; endDate?: string; period?: string }) {
  const query = new URLSearchParams();
  let key = "default";
  if (typeof params === "string") {
    query.set("period", params);
    key = params;
  } else if (params) {
    if (params.startDate) query.set("date_from", params.startDate);
    if (params.endDate) query.set("date_to", params.endDate);
    if (params.period) query.set("period", params.period);
    key = `${params.startDate || ""}_${params.endDate || ""}_${params.period || ""}`;
  }
  const qStr = query.toString();
  return useQuery<TransactionStatsData>({
    queryKey: [...queryKeys.transactions.stats(key)],
    queryFn: async () => {
      const res = await api.get(qStr ? `/transactions/stats?${qStr}` : `/transactions/stats`);
      return res.data?.data;
    },
    staleTime: staleTimeConfig.transactions,
  });
}

/** The shape the list query caches: `{ data, total, page, perPage }`. */
export interface TransactionListPage {
  data: TransactionData[];
  total: number;
  page: number;
  perPage: number;
}

/** Every cached transactions list page, for optimistic patching. */
type ListSnapshot = [readonly unknown[], TransactionListPage][];

function snapshotLists(queryClient: QueryClient): ListSnapshot {
  return queryClient.getQueriesData<TransactionListPage>({ queryKey: queryKeys.transactions.all }) as ListSnapshot;
}

function restoreLists(queryClient: QueryClient, snapshot: ListSnapshot) {
  for (const [key, value] of snapshot) queryClient.setQueryData(key, value);
}

/** Patch one transaction in every cached list page. */
function patchList(
  queryClient: QueryClient,
  id: string,
  patch: Partial<TransactionData>,
) {
  for (const [key, value] of snapshotLists(queryClient)) {
    if (!value?.data) continue;
    queryClient.setQueryData(key, {
      ...value,
      data: value.data.map((tx) => (tx.id === id ? { ...tx, ...patch } : tx)),
    });
  }
}

/** Remove one transaction from every cached list page. */
function removeFromList(queryClient: QueryClient, id: string) {
  for (const [key, value] of snapshotLists(queryClient)) {
    if (!value?.data) continue;
    const next = value.data.filter((tx) => tx.id !== id);
    queryClient.setQueryData(key, {
      ...value,
      data: next,
      total: Math.max(0, (value.total ?? next.length) - 1),
    });
  }
}

function invalidateTransactionViews(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.reports.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.budgets.all });
  // A create/delete moves the `max_transactions` usage counter, so refresh the
  // quota bars immediately in this tab (the SSE usage_changed event covers
  // other tabs).
  queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me });
  queryClient.invalidateQueries({ queryKey: queryKeys.usage.all });
}

export function useCreateTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: Partial<TransactionData>) => {
      const res = await api.post("/transactions", data);
      return res.data?.data;
    },
    // Deliberately NOT optimistic: the list is server-paginated and
    // server-sorted, so a new row has no known position. Showing it in the wrong
    // place and then jumping is worse than a brief spinner.
    onSuccess: () => invalidateTransactionViews(queryClient),
  });
}

export function useUpdateTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...data }: { id: string } & Partial<TransactionData>) => {
      const res = await api.put(`/transactions/${id}`, data);
      return res.data?.data;
    },
    // Optimistic: the row updates immediately; on failure the snapshot restores
    // exactly what was there (including page/total), so a failed edit never
    // leaves a half-applied state behind.
    onMutate: async ({ id, ...data }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.transactions.all });
      const snapshot = snapshotLists(queryClient);
      patchList(queryClient, id, data);
      return { snapshot };
    },
    onError: (err, _vars, context) => {
      if (context?.snapshot) restoreLists(queryClient, context.snapshot);
      notifyTransactionError(err, "Gagal menyimpan perubahan transaksi.");
    },
    onSettled: () => invalidateTransactionViews(queryClient),
  });
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/transactions/${id}`);
    },
    onMutate: async (id: string) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.transactions.all });
      const snapshot = snapshotLists(queryClient);
      removeFromList(queryClient, id);
      return { snapshot };
    },
    onError: (err, _id, context) => {
      if (context?.snapshot) restoreLists(queryClient, context.snapshot);
      notifyTransactionError(err, "Gagal menghapus transaksi.");
    },
    onSettled: () => invalidateTransactionViews(queryClient),
  });
}
