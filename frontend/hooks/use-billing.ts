"use client";

import { useQuery, useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import api from "@/lib/api";
import { queryKeys } from "@/lib/cache-keys";

/**
 * User-facing billing hooks (P0/P1).
 *
 * The backend is authoritative for every money decision; these hooks only
 * mirror state and trigger the documented endpoints. All amounts are integer
 * Rupiah (never floats).
 */

// ─── Types ────────────────────────────────────────────────────

export interface PublicPlan {
  code: string;
  name: string;
  description?: string;
  price_monthly_idr: number;
  price_yearly_idr: number;
  billing_period_days: number;
  is_public: boolean;
  sort_order: number;
  fallback_gateway: string;
  features: Record<string, boolean>;
  limits: Record<string, number>;
}

export interface UserPayment {
  id: string;
  invoice_id: string;
  invoice_number: string;
  invoice_status: string;
  plan_code?: string;
  plan_name?: string;
  gateway: string;
  gateway_ref?: string;
  amount_idr: number;
  subtotal_idr: number;
  tax_idr: number;
  total_idr: number;
  status: "pending" | "pending_review" | "settled" | "failed" | "expired" | "refunded";
  payment_method?: string;
  proof_url?: string;
  invoice_due_at?: string;
  created_at: string;
  updated_at: string;
}

export interface Invoice {
  id: string;
  invoice_number: string;
  status: "draft" | "issued" | "paid" | "past_due" | "voided";
  subtotal_idr: number;
  tax_idr: number;
  total_idr: number;
  issued_at: string;
  due_at?: string;
  paid_at?: string;
  voided_at?: string;
  notes?: string;
}

export interface SubscriptionInfo {
  id: string;
  plan_id: string;
  status: string;
  current_period_start: string;
  current_period_end?: string;
  cancel_at_period_end: boolean;
  plan?: { code: string; name: string; price_monthly_idr: number };
}

export interface MySubscription {
  subscription: SubscriptionInfo | null;
  scheduled_plan_code: string;
}

export interface CheckoutResult {
  invoice?: { invoice_number?: string; total_idr?: number };
  payment?: { id?: string };
  gateway?: string;
  redirect_url?: string;
  snap_token?: string;
  instructions?: Record<string, unknown>;
}

export interface PaymentInstructions {
  bank_name?: string;
  account_number?: string;
  account_holder?: string;
  amount_idr?: number;
  order_id?: string;
  qris_url?: string;
}

// ─── Error helper ─────────────────────────────────────────────

/** A backend error envelope: `{ response: { data: { message } } }`. */
function messageFromAxiosError(err: unknown): string | null {
  if (typeof err !== "object" || err === null || !("response" in err)) return null;
  const response = err.response;
  if (typeof response !== "object" || response === null || !("data" in response)) return null;
  const data = response.data;
  if (typeof data !== "object" || data === null || !("message" in data)) return null;
  const message = data.message;
  return typeof message === "string" && message.length > 0 ? message : null;
}

export function billingErrorMessage(err: unknown, fallback: string): string {
  const fromApi = messageFromAxiosError(err);
  if (fromApi) return fromApi;
  if (err instanceof Error) return err.message;
  return fallback;
}

// ─── Queries ──────────────────────────────────────────────────

export function usePublicPlans() {
  const query = useQuery({
    queryKey: queryKeys.billing.plans,
    queryFn: async (): Promise<PublicPlan[]> => {
      const res = await api.get("/plans");
      return (res.data?.data ?? []) as PublicPlan[];
    },
    staleTime: 1000 * 60 * 10,
  });
  return { plans: query.data ?? [], isLoading: query.isLoading, isError: query.isError };
}

export function useMyPayments() {
  const query = useQuery({
    queryKey: queryKeys.billing.payments,
    queryFn: async (): Promise<UserPayment[]> => {
      const res = await api.get("/payments");
      return (res.data?.data ?? []) as UserPayment[];
    },
    staleTime: 1000 * 30,
  });
  return { payments: query.data ?? [], isLoading: query.isLoading, isError: query.isError, refetch: query.refetch };
}

export function useMyInvoices() {
  const query = useQuery({
    queryKey: queryKeys.billing.invoices,
    queryFn: async (): Promise<Invoice[]> => {
      const res = await api.get("/invoices");
      return (res.data?.data ?? []) as Invoice[];
    },
    staleTime: 1000 * 30,
  });
  return { invoices: query.data ?? [], isLoading: query.isLoading, isError: query.isError };
}

export function useMySubscription() {
  const query = useQuery({
    queryKey: queryKeys.billing.subscription,
    queryFn: async (): Promise<MySubscription> => {
      const res = await api.get("/subscriptions/me");
      const data = res.data?.data;
      return {
        subscription: data?.subscription ?? null,
        scheduled_plan_code: data?.scheduled_plan_code ?? "",
      };
    },
    staleTime: 1000 * 30,
  });
  return {
    subscription: query.data?.subscription ?? null,
    scheduledPlanCode: query.data?.scheduled_plan_code ?? "",
    isLoading: query.isLoading,
    refetch: query.refetch,
  };
}

// ─── Mutations ────────────────────────────────────────────────

/** Refreshes everything a plan change touches. */
function invalidateBilling(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: queryKeys.billing.payments });
  qc.invalidateQueries({ queryKey: queryKeys.billing.invoices });
  qc.invalidateQueries({ queryKey: queryKeys.billing.subscription });
  qc.invalidateQueries({ queryKey: queryKeys.entitlements.me });
}

export function useCheckout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (planCode: string): Promise<CheckoutResult> => {
      const res = await api.post("/payments/checkout", { plan_code: planCode });
      return (res.data?.data ?? {}) as CheckoutResult;
    },
    onSuccess: () => {
      invalidateBilling(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal membuat sesi pembayaran"));
    },
  });
}

export function useUploadProof() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ paymentId, file }: { paymentId: string; file: File }) => {
      const form = new FormData();
      form.append("file", file);
      const res = await api.post(`/payments/${paymentId}/proof`, form, {
        headers: { "Content-Type": "multipart/form-data" },
      });
      return res.data;
    },
    onSuccess: () => {
      toast.success("Bukti terkirim. Menunggu verifikasi admin.");
      invalidateBilling(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal mengunggah bukti"));
    },
  });
}

export function usePaymentInstructions(paymentId: string | null) {
  const query = useQuery({
    queryKey: ["billing", "instructions", paymentId],
    queryFn: async (): Promise<PaymentInstructions> => {
      const res = await api.get(`/payments/${paymentId}/instructions`);
      return (res.data?.data ?? {}) as PaymentInstructions;
    },
    enabled: Boolean(paymentId),
    staleTime: 1000 * 60 * 5,
  });
  return { instructions: query.data ?? null, isLoading: query.isLoading };
}

export function useScheduleDowngrade() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (planCode: string) => {
      const res = await api.post("/subscriptions/me/downgrade", { plan_code: planCode });
      return res.data;
    },
    onSuccess: () => {
      toast.success("Penurunan paket dijadwalkan pada akhir periode.");
      invalidateBilling(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal menjadwalkan penurunan paket"));
    },
  });
}

export function useCancelDowngrade() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.delete("/subscriptions/me/downgrade");
      return res.data;
    },
    onSuccess: () => {
      toast.success("Penurunan paket dibatalkan.");
      invalidateBilling(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal membatalkan penurunan paket"));
    },
  });
}

export function useCancelSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const res = await api.post("/subscriptions/me/cancel");
      return res.data;
    },
    onSuccess: () => {
      toast.success("Langganan akan berakhir pada akhir periode.");
      invalidateBilling(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal membatalkan langganan"));
    },
  });
}
