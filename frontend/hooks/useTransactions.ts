import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
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
  });
}

export function useTransactionStats(period: string = "current_month") {
  return useQuery({
    queryKey: queryKeys.transactions.stats(period),
    queryFn: async () => {
      const res = await api.get(`/transactions/stats?period=${period}`);
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

function snapshotLists(queryClient: ReturnType<typeof useQueryClient>): ListSnapshot {
  return queryClient.getQueriesData<TransactionListPage>({ queryKey: queryKeys.transactions.all }) as ListSnapshot;
}

function restoreLists(queryClient: ReturnType<typeof useQueryClient>, snapshot: ListSnapshot) {
  for (const [key, value] of snapshot) queryClient.setQueryData(key, value);
}

/** Patch one transaction in every cached list page. */
function patchList(
  queryClient: ReturnType<typeof useQueryClient>,
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
function removeFromList(queryClient: ReturnType<typeof useQueryClient>, id: string) {
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

function invalidateTransactionViews(queryClient: ReturnType<typeof useQueryClient>) {
  queryClient.invalidateQueries({ queryKey: queryKeys.transactions.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.dashboard.all });
  queryClient.invalidateQueries({ queryKey: queryKeys.spk.all });
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
