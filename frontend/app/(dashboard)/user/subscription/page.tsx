"use client";

import * as React from "react";
import { Suspense } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import {
  ShieldCheck,
  Sparkles,
  Receipt,
  History as HistoryIcon,
  CalendarClock,
  Undo2,
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorState } from "@/components/shared/ErrorState";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePagination } from "@/components/data-table/data-table-pagination";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTableUrlState } from "@/hooks/use-data-table-url-state";
import { useEntitlements } from "@/hooks/use-entitlements";
import {
  usePublicPlans,
  useMyPaymentsPaged,
  useMyInvoicesPaged,
  useMySubscription,
  useMySubscriptionEvents,
  useCheckout,
  useScheduleDowngrade,
  useCancelDowngrade,
  useCancelInvoice,
  type CheckoutResult,
  type PublicPlan,
} from "@/hooks/use-billing";
import { PlanCard } from "@/components/billing/PlanCard";
import { CheckoutDialog } from "@/components/billing/CheckoutDialog";
import { CheckoutConfirmDialog } from "@/components/billing/CheckoutConfirmDialog";
import {
  paymentColumns,
  invoiceColumns,
  subscriptionEventColumns,
  PAYMENT_STATUS_OPTIONS,
  INVOICE_STATUS_OPTIONS,
} from "@/components/billing/SubscriptionColumns";
import { formatTanggal } from "@/lib/format";

const SUB_STATUS_LABELS: Record<string, string> = {
  active: "Aktif",
  trialing: "Uji coba",
  past_due: "Menunggu pembayaran",
  grace: "Masa tenggang",
  canceled: "Dibatalkan",
  expired: "Kedaluwarsa",
};

type TabKey = "plan" | "payment" | "history";

/** Accept the current tab names plus legacy ones, defaulting to "plan". */
function normalizeTab(raw: string | null): TabKey {
  switch (raw) {
    case "payment":
    case "payments":
    case "invoice":
    case "invoices":
      return "payment";
    case "history":
      return "history";
    default:
      return "plan";
  }
}

export default function SubscriptionPage() {
  return (
    <Suspense fallback={<div className="h-40 animate-pulse rounded-md border-2 border-foreground bg-muted" />}>
      <SubscriptionContent />
    </Suspense>
  );
}

function SubscriptionContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();

  const tab = normalizeTab(searchParams.get("tab"));
  const setTab = (next: TabKey) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set("tab", next);
    router.replace(`${pathname}?${params.toString()}`, { scroll: false });
  };

  const { planCode, planName, status, isLoading: entLoading, isError: entError } = useEntitlements();
  const { plans, isLoading: plansLoading, isError: plansError } = usePublicPlans();
  const { subscription, scheduledPlanCode, refetch: refetchSub } = useMySubscription();
  const checkout = useCheckout();
  const downgrade = useScheduleDowngrade();
  const cancelDowngrade = useCancelDowngrade();

  const [checkoutResult, setCheckoutResult] = React.useState<CheckoutResult | null>(null);
  const [checkoutOpen, setCheckoutOpen] = React.useState(false);
  const [downgradeTarget, setDowngradeTarget] = React.useState<string | null>(null);
  const [confirmPlan, setConfirmPlan] = React.useState<PublicPlan | null>(null);

  const currentSortOrder = React.useMemo(() => {
    const cur = plans.find((p) => p.code === planCode);
    return cur?.sort_order ?? 0;
  }, [plans, planCode]);

  const handleUpgrade = (targetPlanCode: string) => {
    const target = plans.find((p) => p.code === targetPlanCode);
    if (target) setConfirmPlan(target);
  };

  const confirmUpgrade = async () => {
    if (!confirmPlan) return;
    try {
      const res = await checkout.mutateAsync(confirmPlan.code);
      setConfirmPlan(null);
      setCheckoutResult(res);
      setCheckoutOpen(true);
      if (res.redirect_url) window.open(res.redirect_url, "_blank", "noopener,noreferrer");
    } catch (e) {
      const httpStatus = (e as { response?: { status?: number } })?.response?.status;
      if (httpStatus === 409) {
        setConfirmPlan(null);
        setTab("payment");
        refetchSub();
      }
    }
  };

  const handleDowngrade = (targetPlanCode: string) => setDowngradeTarget(targetPlanCode);

  const confirmDowngrade = async () => {
    if (!downgradeTarget) return;
    await downgrade.mutateAsync(downgradeTarget);
    setDowngradeTarget(null);
    refetchSub();
  };

  const handleCancelDowngrade = async () => {
    await cancelDowngrade.mutateAsync();
    refetchSub();
  };

  return (
    <div className="mx-auto flex w-full max-w-7xl flex-col gap-6">
      <div>
        <h1 className="text-xl font-black uppercase tracking-wide text-foreground">Langganan &amp; Paket</h1>
        <p className="mt-1 text-xs font-bold text-muted-foreground">
          Kelola paket, pantau pembayaran &amp; tagihan, dan lihat riwayat langganan Anda.
        </p>
      </div>

      {/* Current plan summary */}
      <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg font-black">
            <ShieldCheck className="h-5 w-5" aria-hidden="true" />
            Paket Anda
          </CardTitle>
          <CardDescription>Perubahan paket oleh admin berlaku seketika, tanpa perlu masuk ulang.</CardDescription>
        </CardHeader>
        <CardContent>
          {entLoading ? (
            <Skeleton className="h-6 w-40" />
          ) : entError ? (
            <p className="text-sm text-muted-foreground">Gagal memuat data paket. Coba muat ulang halaman ini.</p>
          ) : (
            <div className="space-y-3">
              <div className="flex flex-wrap items-center gap-3">
                <span className="text-lg font-black">{planName || "—"}</span>
                <Badge variant="outline" className="font-mono text-xs">{planCode || "—"}</Badge>
                <Badge variant={status === "active" ? "default" : "secondary"}>
                  {SUB_STATUS_LABELS[status] ?? status ?? "—"}
                </Badge>
              </div>
              {subscription?.current_period_end && (
                <p className="flex items-center gap-1.5 text-xs font-bold text-muted-foreground">
                  <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
                  Periode berakhir {formatTanggal(subscription.current_period_end)}
                  {subscription.cancel_at_period_end && !scheduledPlanCode && " · tidak diperpanjang"}
                </p>
              )}
              {scheduledPlanCode && (
                <div className="flex flex-wrap items-center gap-3 rounded-md border-2 border-amber-500 bg-amber-50 p-2 dark:bg-amber-950/30">
                  <p className="text-xs font-bold text-amber-700 dark:text-amber-300">
                    Penurunan ke paket <span className="uppercase">{scheduledPlanCode}</span> dijadwalkan pada akhir periode.
                  </p>
                  <Button size="sm" variant="outline" onClick={handleCancelDowngrade} disabled={cancelDowngrade.isPending}>
                    <Undo2 className="mr-1.5 h-3.5 w-3.5" /> Batalkan
                  </Button>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      <Tabs value={tab} onValueChange={(v) => setTab(normalizeTab(v))} className="space-y-4">
        <TabsList>
          <TabsTrigger value="plan">
            <Sparkles className="mr-1.5 h-4 w-4" aria-hidden="true" /> Paket
          </TabsTrigger>
          <TabsTrigger value="payment">
            <Receipt className="mr-1.5 h-4 w-4" aria-hidden="true" /> Pembayaran &amp; Tagihan
          </TabsTrigger>
          <TabsTrigger value="history">
            <HistoryIcon className="mr-1.5 h-4 w-4" aria-hidden="true" /> Riwayat
          </TabsTrigger>
        </TabsList>

        <TabsContent value="plan" className="space-y-4">
          {plansLoading ? (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-80 w-full" />
              ))}
            </div>
          ) : plansError ? (
            <ErrorState description="Gagal memuat katalog paket." />
          ) : plans.length === 0 ? (
            <EmptyState title="Belum ada paket" description="Katalog paket belum tersedia." />
          ) : (
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              {plans.map((plan) => (
                <PlanCard
                  key={plan.code}
                  plan={plan}
                  currentPlanCode={planCode}
                  currentSortOrder={currentSortOrder}
                  onUpgrade={handleUpgrade}
                  onDowngrade={handleDowngrade}
                  isBusy={checkout.isPending || downgrade.isPending}
                  scheduledPlanCode={scheduledPlanCode}
                />
              ))}
            </div>
          )}
        </TabsContent>

        <TabsContent value="payment">
          <PaymentTab active={tab === "payment"} />
        </TabsContent>

        <TabsContent value="history">
          <HistoryTab active={tab === "history"} />
        </TabsContent>
      </Tabs>

      <CheckoutDialog open={checkoutOpen} onOpenChange={setCheckoutOpen} result={checkoutResult} />

      <CheckoutConfirmDialog
        open={confirmPlan !== null}
        onOpenChange={(o) => !o && setConfirmPlan(null)}
        plan={confirmPlan}
        currentPlanName={planName}
        onConfirm={confirmUpgrade}
        loading={checkout.isPending}
      />

      <ConfirmDialog
        open={downgradeTarget !== null}
        onOpenChange={(o) => !o && setDowngradeTarget(null)}
        onConfirm={confirmDowngrade}
        title="Turunkan Paket"
        message={`Paket akan diturunkan ke ${downgradeTarget ?? ""} pada akhir periode berjalan. Fitur paket saat ini tetap aktif sampai periode berakhir dan data Anda tidak dihapus.`}
        confirmText="Ya, Turunkan"
        variant="primary"
        loading={downgrade.isPending}
      />
    </div>
  );
}

// ─── Payment & invoice tab ────────────────────────────────────

function PaymentTab({ active }: { active: boolean }) {
  // Namespaced params so the payments/invoices/history tables can coexist.
  const table = useDataTableUrlState({ perPage: 20, prefix: "payment" });
  const statusFilter = table.filters.status || "";

  const { payments, total, isLoading, isError, refetch } = useMyPaymentsPaged(
    {
      page: table.page,
      perPage: table.perPage,
      sorts: table.sorts,
      q: table.q,
      filters: table.filters,
    },
    { enabled: active }
  );

  const { invoices, total: invoiceTotal, isLoading: invoicesLoading, refetch: refetchInvoices } =
    useMyInvoicesPaged(
      {
        page: table.page,
        perPage: table.perPage,
        sorts: table.sorts,
        q: table.q,
        filters: table.filters,
      },
      { enabled: active }
    );

  const cancelInvoice = useCancelInvoice();
  const [cancelId, setCancelId] = React.useState<string | null>(null);

  const handleCancelConfirm = async () => {
    if (!cancelId) return;
    await cancelInvoice.mutateAsync(cancelId);
    setCancelId(null);
    refetchInvoices();
  };

  const paymentCols = React.useMemo<ColumnDef<import("@/hooks/use-billing").UserPayment, unknown>[]>(
    () => paymentColumns({ page: table.page, perPage: table.perPage }),
    [table.page, table.perPage]
  );

  const invoiceCols = React.useMemo<ColumnDef<import("@/hooks/use-billing").Invoice, unknown>[]>(
    () => invoiceColumns({ page: table.page, perPage: table.perPage, onCancel: (id) => setCancelId(id) }),
    [table.page, table.perPage]
  );

  return (
    <div className="space-y-6">
      {/* Payments */}
      <div className="flex flex-col overflow-hidden rounded-md border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <div className="flex flex-col justify-between gap-4 border-b-2 border-foreground bg-muted p-4 md:flex-row md:items-center">
          <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Pembayaran</h3>
          <DataTableToolbar
            search={table.q}
            onSearch={(q) => table.setSearch(q)}
            placeholder="Cari faktur atau paket..."
          >
            <select
              value={statusFilter}
              onChange={(e) => table.setFilter("status", e.target.value)}
              aria-label="Filter status pembayaran"
              className="cursor-pointer rounded-md border-2 border-foreground bg-background px-2 py-1.5 text-xs font-bold focus:outline-none"
            >
              {PAYMENT_STATUS_OPTIONS.map((o) => (
                <option key={o.value || "all"} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </DataTableToolbar>
        </div>

        <DataTable
          columns={paymentCols}
          data={payments}
          isLoading={isLoading}
          error={isError ? "Gagal memuat riwayat pembayaran." : null}
          emptyMessage="Belum ada pembayaran."
          sort={table.sort}
          order={table.order}
          sorts={table.sorts}
          onSort={table.setSort}
        />

        <div className="border-t-2 border-foreground bg-muted/40 p-4">
          <DataTablePagination
            page={table.page}
            perPage={table.perPage}
            total={total}
            totalPages={Math.max(1, Math.ceil(total / table.perPage))}
            onPageChange={table.setPage}
            onPerPageChange={table.setPerPage}
          />
        </div>
        {isError && (
          <div className="border-t-2 border-foreground p-4">
            <Button variant="outline" className="cursor-pointer" onClick={() => refetch()}>
              Coba lagi
            </Button>
          </div>
        )}
      </div>

      {/* Invoices */}
      <div className="flex flex-col overflow-hidden rounded-md border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
        <div className="flex flex-col justify-between gap-4 border-b-2 border-foreground bg-muted p-4 md:flex-row md:items-center">
          <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Tagihan</h3>
          <div className="flex items-center gap-2">
            <select
              value={table.filters.invoice_status || ""}
              onChange={(e) => table.setFilter("invoice_status", e.target.value)}
              aria-label="Filter status tagihan"
              className="cursor-pointer rounded-md border-2 border-foreground bg-background px-2 py-1.5 text-xs font-bold focus:outline-none"
            >
              {INVOICE_STATUS_OPTIONS.map((o) => (
                <option key={o.value || "all"} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <DataTable
          columns={invoiceCols}
          data={invoices}
          isLoading={invoicesLoading}
          emptyMessage="Belum ada tagihan."
          sort={table.sort}
          order={table.order}
          sorts={table.sorts}
          onSort={table.setSort}
        />

        <div className="border-t-2 border-foreground bg-muted/40 p-4">
          <DataTablePagination
            page={table.page}
            perPage={table.perPage}
            total={invoiceTotal}
            totalPages={Math.max(1, Math.ceil(invoiceTotal / table.perPage))}
            onPageChange={table.setPage}
            onPerPageChange={table.setPerPage}
          />
        </div>
      </div>

      <ConfirmDialog
        open={cancelId !== null}
        onOpenChange={(o) => !o && setCancelId(null)}
        onConfirm={handleCancelConfirm}
        title="Batalkan Tagihan?"
        message="Tagihan ini akan dibatalkan dan tidak dapat dibayar lagi. Anda dapat membuat tagihan baru setelahnya."
        confirmText="Ya, Batalkan"
        variant="danger"
        loading={cancelInvoice.isPending}
      />
    </div>
  );
}

// ─── Subscription history tab ─────────────────────────────────

function HistoryTab({ active }: { active: boolean }) {
  const table = useDataTableUrlState({ perPage: 20, prefix: "history" });

  const { events, total, isLoading, isError, refetch } = useMySubscriptionEvents(
    {
      page: table.page,
      perPage: table.perPage,
      sorts: table.sorts,
      q: table.q,
      filters: table.filters,
    },
    { enabled: active }
  );

  const columns = React.useMemo<ColumnDef<import("@/hooks/use-billing").SubscriptionEvent, unknown>[]>(
    () => subscriptionEventColumns({ page: table.page, perPage: table.perPage }),
    [table.page, table.perPage]
  );

  return (
    <div className="flex flex-col overflow-hidden rounded-md border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
      <div className="flex flex-col justify-between gap-4 border-b-2 border-foreground bg-muted p-4 md:flex-row md:items-center">
        <h3 className="text-sm font-black uppercase tracking-wide text-foreground">Riwayat Langganan</h3>
        <DataTableToolbar
          search={table.q}
          onSearch={(q) => table.setSearch(q)}
          placeholder="Cari riwayat..."
        />
      </div>

      <DataTable
        columns={columns}
        data={events}
        isLoading={isLoading}
        error={isError ? "Gagal memuat riwayat langganan." : null}
        emptyMessage="Belum ada riwayat langganan."
        sort={table.sort}
        order={table.order}
        sorts={table.sorts}
        onSort={table.setSort}
      />

      <div className="border-t-2 border-foreground bg-muted/40 p-4">
        <DataTablePagination
          page={table.page}
          perPage={table.perPage}
          total={total}
          totalPages={Math.max(1, Math.ceil(total / table.perPage))}
          onPageChange={table.setPage}
          onPerPageChange={table.setPerPage}
        />
      </div>
      {isError && (
        <div className="border-t-2 border-foreground p-4">
          <Button variant="outline" className="cursor-pointer" onClick={() => refetch()}>
            Coba lagi
          </Button>
        </div>
      )}
    </div>
  );
}
