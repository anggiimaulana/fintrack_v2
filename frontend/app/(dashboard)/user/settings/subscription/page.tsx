'use client'

import * as React from 'react'
import { ShieldCheck, Sparkles, Receipt, FileText, CalendarClock, Undo2 } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { EmptyState } from '@/components/shared/EmptyState'
import { ErrorState } from '@/components/shared/ErrorState'
import { Money } from '@/components/shared/Money'
import { ConfirmDialog } from '@/components/shared/ConfirmDialog'
import { useEntitlements } from '@/hooks/use-entitlements'
import {
  usePublicPlans,
  useMyPayments,
  useMyInvoices,
  useMySubscription,
  useCheckout,
  useScheduleDowngrade,
  useCancelDowngrade,
  type CheckoutResult,
  type Invoice,
} from '@/hooks/use-billing'
import { PlanCard } from '@/components/billing/PlanCard'
import { QuotaUsageCard } from '@/components/billing/QuotaUsageCard'
import { PaymentRow } from '@/components/billing/PaymentRow'
import { CheckoutDialog } from '@/components/billing/CheckoutDialog'
import { formatTanggal } from '@/lib/format'

const SUB_STATUS_LABELS: Record<string, string> = {
  active: 'Aktif',
  trialing: 'Uji coba',
  past_due: 'Menunggu pembayaran',
  grace: 'Masa tenggang',
  canceled: 'Dibatalkan',
  expired: 'Kedaluwarsa',
}

const INVOICE_STATUS_LABELS: Record<Invoice['status'], string> = {
  draft: 'Draf',
  issued: 'Terbit',
  paid: 'Lunas',
  past_due: 'Jatuh tempo',
  voided: 'Dibatalkan',
}

export default function SubscriptionPage() {
  const { planCode, planName, status, isLoading: entLoading, isError: entError, usage, period } = useEntitlements()
  const { plans, isLoading: plansLoading, isError: plansError } = usePublicPlans()
  const { payments, isLoading: paymentsLoading, isError: paymentsError, refetch: refetchPayments } = useMyPayments()
  const { invoices, isLoading: invoicesLoading } = useMyInvoices()
  const { subscription, scheduledPlanCode, refetch: refetchSub } = useMySubscription()

  const checkout = useCheckout()
  const downgrade = useScheduleDowngrade()
  const cancelDowngrade = useCancelDowngrade()

  const [checkoutResult, setCheckoutResult] = React.useState<CheckoutResult | null>(null)
  const [checkoutOpen, setCheckoutOpen] = React.useState(false)
  const [downgradeTarget, setDowngradeTarget] = React.useState<string | null>(null)

  const currentSortOrder = React.useMemo(() => {
    const cur = plans.find((p) => p.code === planCode)
    return cur?.sort_order ?? 0
  }, [plans, planCode])

  const handleUpgrade = async (targetPlanCode: string) => {
    const res = await checkout.mutateAsync(targetPlanCode)
    setCheckoutResult(res)
    setCheckoutOpen(true)
    if (res.redirect_url) window.open(res.redirect_url, '_blank', 'noopener,noreferrer')
    refetchPayments()
    refetchSub()
  }

  const handleDowngrade = (targetPlanCode: string) => setDowngradeTarget(targetPlanCode)

  const confirmDowngrade = async () => {
    if (!downgradeTarget) return
    await downgrade.mutateAsync(downgradeTarget)
    setDowngradeTarget(null)
    refetchSub()
  }

  const handleCancelDowngrade = async () => {
    await cancelDowngrade.mutateAsync()
    refetchSub()
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Langganan &amp; Paket</h1>
        <p className="text-sm text-muted-foreground">
          Kelola paket, lihat riwayat pembayaran, dan unggah bukti transfer Anda.
        </p>
      </div>

      {/* Current plan summary */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
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
                <span className="text-lg font-bold">{planName || '—'}</span>
                <Badge variant="outline" className="font-mono text-xs">{planCode || '—'}</Badge>
                <Badge variant={status === 'active' ? 'default' : 'secondary'}>
                  {SUB_STATUS_LABELS[status] ?? status ?? '—'}
                </Badge>
              </div>
              {subscription?.current_period_end && (
                <p className="flex items-center gap-1.5 text-xs font-bold text-muted-foreground">
                  <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
                  Periode berakhir {formatTanggal(subscription.current_period_end)}
                  {subscription.cancel_at_period_end && !scheduledPlanCode && ' · tidak diperpanjang'}
                </p>
              )}
              {scheduledPlanCode && (
                <div className="flex flex-wrap items-center gap-3 border-2 border-amber-500 bg-amber-50 p-2 dark:bg-amber-950/30">
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

      {/* Real-time quota consumption (F6) */}
      <QuotaUsageCard usage={usage} period={period} />

      <Tabs defaultValue="plans" className="space-y-4">
        <TabsList>
          <TabsTrigger value="plans">
            <Sparkles className="mr-1.5 h-4 w-4" aria-hidden="true" /> Paket
          </TabsTrigger>
          <TabsTrigger value="payments">
            <Receipt className="mr-1.5 h-4 w-4" aria-hidden="true" /> Pembayaran
          </TabsTrigger>
          <TabsTrigger value="invoices">
            <FileText className="mr-1.5 h-4 w-4" aria-hidden="true" /> Tagihan
          </TabsTrigger>
        </TabsList>

        {/* Plans (dynamic from the backend catalog) */}
        <TabsContent value="plans" className="space-y-4">
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

        {/* Payment history + proof upload */}
        <TabsContent value="payments" className="space-y-4">
          {paymentsLoading ? (
            <div className="space-y-4">
              {[1, 2].map((i) => (
                <Skeleton key={i} className="h-40 w-full" />
              ))}
            </div>
          ) : paymentsError ? (
            <ErrorState description="Gagal memuat riwayat pembayaran." onRetry={() => refetchPayments()} />
          ) : payments.length === 0 ? (
            <EmptyState
              title="Belum ada pembayaran"
              description="Pembayaran akan muncul di sini setelah Anda memilih paket berbayar."
            />
          ) : (
            <div className="space-y-4">
              {payments.map((p) => (
                <PaymentRow key={p.id} payment={p} />
              ))}
            </div>
          )}
        </TabsContent>

        {/* Invoices */}
        <TabsContent value="invoices" className="space-y-4">
          {invoicesLoading ? (
            <Skeleton className="h-40 w-full" />
          ) : invoices.length === 0 ? (
            <EmptyState title="Belum ada tagihan" description="Tagihan akan muncul di sini setelah Anda melakukan checkout." />
          ) : (
            <div className="overflow-x-auto border-2 border-foreground">
              <table className="w-full text-sm">
                <thead className="border-b-2 border-foreground bg-muted">
                  <tr>
                    <th scope="col" className="p-3 text-left font-black uppercase tracking-wide">No. Faktur</th>
                    <th scope="col" className="p-3 text-left font-black uppercase tracking-wide">Terbit</th>
                    <th scope="col" className="p-3 text-left font-black uppercase tracking-wide">Jatuh Tempo</th>
                    <th scope="col" className="p-3 text-right font-black uppercase tracking-wide">Total</th>
                    <th scope="col" className="p-3 text-left font-black uppercase tracking-wide">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {invoices.map((inv) => (
                    <tr key={inv.id} className="border-b border-foreground/20 last:border-b-0">
                      <td className="p-3 font-mono font-bold">{inv.invoice_number}</td>
                      <td className="p-3 text-muted-foreground">{formatTanggal(inv.issued_at)}</td>
                      <td className="p-3 text-muted-foreground">{inv.due_at ? formatTanggal(inv.due_at) : '—'}</td>
                      <td className="p-3 text-right"><Money value={inv.total_idr} /></td>
                      <td className="p-3">
                        <Badge variant={inv.status === 'paid' ? 'default' : inv.status === 'voided' ? 'destructive' : 'secondary'}>
                          {INVOICE_STATUS_LABELS[inv.status]}
                        </Badge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </TabsContent>
      </Tabs>

      <CheckoutDialog open={checkoutOpen} onOpenChange={setCheckoutOpen} result={checkoutResult} />

      <ConfirmDialog
        open={downgradeTarget !== null}
        onOpenChange={(o) => !o && setDowngradeTarget(null)}
        onConfirm={confirmDowngrade}
        title="Turunkan Paket"
        message={`Paket akan diturunkan ke ${downgradeTarget ?? ''} pada akhir periode berjalan. Fitur paket saat ini tetap aktif sampai periode berakhir dan data Anda tidak dihapus.`}
        confirmText="Ya, Turunkan"
        variant="primary"
        loading={downgrade.isPending}
      />
    </div>
  )
}
