"use client";

import { useQuery, useMutation, useQueryClient, keepPreviousData, type QueryClient } from "@tanstack/react-query";
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

/** Display metadata for a plan limit, including whether it is enforced. */
export interface PlanLimitMeta {
  key: string;
  label: string;
  unit: string;
  enforced: boolean;
}

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
  /** Per-limit label/unit/enforced, from the backend entitlement catalog. */
  limit_meta?: Record<string, PlanLimitMeta>;
  /** PPN rate applied at checkout (0 when tax is off). */
  tax_rate_percent?: number;
  /** Tax amount for one billing period at the current rate. */
  tax_idr?: number;
  /** Grand total (subtotal + tax) for one billing period. */
  total_idr?: number;
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
  status: "draft" | "issued" | "paid" | "past_due" | "voided" | "expired";
  subtotal_idr: number;
  tax_idr: number;
  total_idr: number;
  issued_at: string;
  due_at?: string;
  paid_at?: string;
  voided_at?: string;
  /** Why a non-paid invoice closed: cancelled_by_user | expired | superseded. */
  closed_reason?: string;
  expired_at?: string;
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
  /** All active destinations (bank/e-wallet/QRIS). Preferred over the legacy
   *  single bank_* fields, which remain as a fallback. */
  accounts?: PaymentAccount[];
}

export type PaymentAccountKind = "bank" | "ewallet" | "qris";

/** One manual payment destination shown to the user. */
export interface PaymentAccount {
  id?: string;
  kind: PaymentAccountKind;
  name: string;
  account_number?: string;
  account_holder?: string;
  qris_url?: string;
  sort_order?: number;
  is_active?: boolean;
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

export function useMyPayments(options?: { enabled?: boolean }) {
  const query = useQuery({
    queryKey: queryKeys.billing.payments,
    queryFn: async (): Promise<UserPayment[]> => {
      const res = await api.get("/payments");
      return (res.data?.data ?? []) as UserPayment[];
    },
    // Only fetch when the caller's view needs it (e.g. the payments tab is
    // open). This avoids paying for a request the user never sees.
    enabled: options?.enabled ?? true,
    staleTime: 1000 * 30,
  });
  return { payments: query.data ?? [], isLoading: query.isLoading, isError: query.isError, refetch: query.refetch };
}

export function useMyInvoices(options?: { enabled?: boolean }) {
  const query = useQuery({
    queryKey: queryKeys.billing.invoices,
    queryFn: async (): Promise<Invoice[]> => {
      const res = await api.get("/invoices");
      return (res.data?.data ?? []) as Invoice[];
    },
    enabled: options?.enabled ?? true,
    staleTime: 1000 * 30,
  });
  return { invoices: query.data ?? [], isLoading: query.isLoading, isError: query.isError };
}

/** Table query state accepted by the paginated billing hooks. */
export interface BillingTableParams {
  page: number;
  perPage: number;
  sorts: { column: string; order: "asc" | "desc" }[];
  q: string;
  filters: Record<string, string>;
}

function billingQuery(p: BillingTableParams): string {
  const params = new URLSearchParams();
  params.set("page", String(p.page));
  params.set("per_page", String(p.perPage));
  if (p.sorts.length > 0) {
    params.set("sort", p.sorts.map((s) => s.column).join(","));
    params.set("order", p.sorts.map((s) => s.order).join(","));
  }
  if (p.q) params.set("q", p.q);
  for (const [k, v] of Object.entries(p.filters)) {
    if (v) params.set(`filter_${k}`, v);
  }
  return params.toString();
}

export interface PagedResult<T> {
  items: T[];
  total: number;
  page: number;
  perPage: number;
}

/** Paginated + sortable + searchable payment history for the subscription page. */
export function useMyPaymentsPaged(p: BillingTableParams, opts?: { enabled?: boolean }) {
  const qs = billingQuery(p);
  const query = useQuery({
    queryKey: queryKeys.billing.paymentsPaged(qs),
    queryFn: async (): Promise<PagedResult<UserPayment>> => {
      const res = await api.get(`/payments?${qs}`);
      return {
        items: (res.data?.data ?? []) as UserPayment[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? p.page,
        perPage: res.data?.meta?.per_page ?? p.perPage,
      };
    },
    enabled: opts?.enabled ?? true,
    staleTime: 1000 * 30,
    placeholderData: keepPreviousData,
  });
  return {
    payments: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    refetch: query.refetch,
  };
}

/** Paginated + sortable + searchable invoice list for the subscription page. */
export function useMyInvoicesPaged(p: BillingTableParams, opts?: { enabled?: boolean }) {
  const qs = billingQuery(p);
  const query = useQuery({
    queryKey: queryKeys.billing.invoicesPaged(qs),
    queryFn: async (): Promise<PagedResult<Invoice>> => {
      const res = await api.get(`/invoices?${qs}`);
      return {
        items: (res.data?.data ?? []) as Invoice[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? p.page,
        perPage: res.data?.meta?.per_page ?? p.perPage,
      };
    },
    enabled: opts?.enabled ?? true,
    staleTime: 1000 * 30,
    placeholderData: keepPreviousData,
  });
  return {
    invoices: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    refetch: query.refetch,
  };
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

/** One subscription lifecycle event (from GET /subscriptions/me/events). */
export interface SubscriptionEvent {
  id: string;
  subscription_id: string;
  user_id: string;
  event_type: string;
  previous_state?: Record<string, unknown> | null;
  new_state?: Record<string, unknown> | null;
  reason?: string | null;
  actor_id?: string | null;
  created_at: string;
}

/** Paginated subscription lifecycle history for the "Riwayat" tab. */
export function useMySubscriptionEvents(p: BillingTableParams, opts?: { enabled?: boolean }) {
  const qs = billingQuery(p);
  const query = useQuery({
    queryKey: queryKeys.billing.subscriptionEvents(qs),
    queryFn: async (): Promise<PagedResult<SubscriptionEvent>> => {
      const res = await api.get(`/subscriptions/me/events?${qs}`);
      return {
        items: (res.data?.data ?? []) as SubscriptionEvent[],
        total: res.data?.meta?.total ?? 0,
        page: res.data?.meta?.page ?? p.page,
        perPage: res.data?.meta?.per_page ?? p.perPage,
      };
    },
    enabled: opts?.enabled ?? true,
    staleTime: 1000 * 30,
    placeholderData: keepPreviousData,
  });
  return {
    events: query.data?.items ?? [],
    total: query.data?.total ?? 0,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    isError: query.isError,
    refetch: query.refetch,
  };
}

// ─── Mutations ────────────────────────────────────────────────

/**
 * Refreshes everything a *plan change* touches (upgrade/downgrade/cancel).
 * Entitlements change only when a plan actually changes, so they belong here
 * and not on every payment mutation.
 */
function invalidateBilling(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: queryKeys.billing.payments });
  qc.invalidateQueries({ queryKey: queryKeys.billing.invoices });
  qc.invalidateQueries({ queryKey: queryKeys.billing.subscription });
  qc.invalidateQueries({ queryKey: queryKeys.entitlements.me });
}

/**
 * Refreshes only the payment history. Checkout and proof upload create/change a
 * payment and its invoice but leave the subscription and entitlements untouched
 * (a manual payment does not activate a plan until an admin verifies it), so
 * invalidating those here would refetch data that cannot have changed.
 */
function invalidatePaymentHistory(qc: QueryClient) {
  qc.invalidateQueries({ queryKey: queryKeys.billing.payments });
  qc.invalidateQueries({ queryKey: queryKeys.billing.invoices });
}

export function useCheckout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (planCode: string): Promise<CheckoutResult> => {
      const res = await api.post("/payments/checkout", { plan_code: planCode });
      return (res.data?.data ?? {}) as CheckoutResult;
    },
    onSuccess: () => {
      invalidatePaymentHistory(qc);
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
      invalidatePaymentHistory(qc);
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

export function useCancelInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (invoiceId: string) => {
      const res = await api.post(`/invoices/${invoiceId}/cancel`);
      return res.data?.data as Invoice;
    },
    onSuccess: () => {
      toast.success("Tagihan dibatalkan. Anda dapat membuat tagihan baru.");
      invalidatePaymentHistory(qc);
    },
    onError: (err: unknown) => {
      toast.error(billingErrorMessage(err, "Gagal membatalkan tagihan"));
    },
  });
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
