"use client";

import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";
import { staleTimeConfig } from "@/lib/cache-invalidation";
import type { PaymentAccount, PaymentAccountKind } from "@/hooks/use-billing";

/** The wire payload for creating/updating a payment account. */
export interface PaymentAccountPayload {
  kind: PaymentAccountKind;
  name: string;
  account_number: string;
  account_holder: string;
  qris_url: string;
  sort_order: number;
  is_active?: boolean;
}

/**
 * usePaymentAccounts loads the active destinations users may pay to. Used by the
 * checkout/instructions UI so it can list every configured account.
 */
export function usePaymentAccounts(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.paymentAccounts.active,
    enabled: options?.enabled ?? true,
    queryFn: async (): Promise<PaymentAccount[]> => {
      const res = await api.get("/payment-accounts");
      return (res.data?.data ?? []) as PaymentAccount[];
    },
    staleTime: staleTimeConfig.admin,
  });
}

/** useAdminPaymentAccounts loads every account (including inactive). */
export function useAdminPaymentAccounts() {
  return useQuery({
    queryKey: queryKeys.paymentAccounts.all,
    queryFn: async (): Promise<PaymentAccount[]> => {
      const res = await api.get("/admin/payment-accounts");
      return (res.data?.data ?? []) as PaymentAccount[];
    },
    staleTime: staleTimeConfig.admin,
  });
}

function invalidateAccounts(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: queryKeys.paymentAccounts.all });
  qc.invalidateQueries({ queryKey: queryKeys.paymentAccounts.active });
}

export function useCreatePaymentAccount() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: PaymentAccountPayload) => {
      const res = await api.post("/admin/payment-accounts", payload);
      return res.data?.data as PaymentAccount;
    },
    onSuccess: () => invalidateAccounts(qc),
  });
}

export function useUpdatePaymentAccount() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: PaymentAccountPayload }) => {
      const res = await api.put(`/admin/payment-accounts/${id}`, payload);
      return res.data?.data as PaymentAccount;
    },
    onSuccess: () => invalidateAccounts(qc),
  });
}

export function useDeletePaymentAccount() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/admin/payment-accounts/${id}`);
    },
    onSuccess: () => invalidateAccounts(qc),
  });
}
