'use client'

import * as React from 'react'
import { Check, X, Undo2, Calendar, Mail, FileText, CreditCard } from 'lucide-react'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Money } from '@/components/shared/Money'
import { formatTanggal } from '@/lib/format'
import { PaymentStatusBadge } from './PaymentStatusBadge'
import { PaymentProofDialog } from './PaymentProofDialog'
import {
  PaymentApproveDialog,
  PaymentRejectDialog,
  PaymentRefundDialog,
} from './PaymentActionDialogs'
import {
  useSuperAdminPaymentReview,
  useSuperAdminPaymentMutations,
  type PaymentRow,
} from '@/hooks/useSuperAdminPayments'

export function PaymentReviewQueue() {
  const { payments, meta, isLoading, isError, error } = useSuperAdminPaymentReview({
    page: 1,
    per_page: 50,
  })

  const {
    approvePayment,
    rejectPayment,
    refundPayment,
    isApproving,
    isRejecting,
    isRefunding,
  } = useSuperAdminPaymentMutations()

  const [selectedApprove, setSelectedApprove] = React.useState<PaymentRow | null>(null)
  const [selectedReject, setSelectedReject] = React.useState<PaymentRow | null>(null)
  const [selectedRefund, setSelectedRefund] = React.useState<PaymentRow | null>(null)

  const isMutating = isApproving || isRejecting || isRefunding

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <Card key={i} className="border-2 rounded-md border-border bg-card">
            <CardHeader className="space-y-2 pb-2">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-48" />
            </CardHeader>
            <CardContent className="space-y-3">
              <Skeleton className="h-24 w-full" />
              <Skeleton className="h-6 w-24" />
            </CardContent>
            <CardFooter className="gap-2 pt-2">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
            </CardFooter>
          </Card>
        ))}
      </div>
    )
  }

  if (isError) {
    return (
      <div className="rounded-md border-2 border-destructive bg-destructive/10 p-6 text-sm text-destructive font-bold">
        Gagal memuat antrean verifikasi pembayaran: {(error as Error)?.message || 'Terjadi kesalahan sistem'}
      </div>
    )
  }

  if (!payments || payments.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-border bg-card/50 p-12 text-center">
        <div className="rounded-full border-2 border-border bg-muted p-4 mb-3">
          <Check className="h-8 w-8 text-emerald-600 dark:text-emerald-400" />
        </div>
        <h3 className="text-base font-bold text-foreground">Semua Pembayaran Bersih</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground">
          Tidak ada pembayaran manual yang menunggu review. Transaksi baru akan masuk ke antrean ini secara FIFO.
        </p>
      </div>
    )
  }

  return (
    <>
      <div className="mb-3 flex items-center justify-between">
        <p className="text-xs font-bold text-muted-foreground">
          Menampilkan {payments.length} dari total {meta?.total ?? payments.length} antrean verifikasi (FIFO)
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {payments.map((p) => (
          <Card
            key={p.id}
            className="flex flex-col justify-between border-2 rounded-md border-border bg-card transition hover:border-foreground shadow-sm"
          >
            <CardHeader className="space-y-2 pb-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <FileText className="h-3.5 w-3.5 shrink-0" />
                    <span className="font-mono font-bold text-foreground truncate">
                      {p.invoice_number || 'INV-PENDING'}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground mt-1">
                    <Mail className="h-3.5 w-3.5 shrink-0" />
                    <span className="truncate" title={p.user_email}>
                      {p.user_email}
                    </span>
                  </div>
                </div>
                <PaymentStatusBadge status={p.status} />
              </div>
            </CardHeader>

            <CardContent className="space-y-3 py-2 flex-1">
              <div className="rounded-md border border-border bg-muted/30 p-2.5">
                <div className="flex items-baseline justify-between">
                  <span className="text-xs text-muted-foreground">Nominal Transfer</span>
                  <span className="text-base font-bold text-foreground">
                    <Money value={p.amount_idr} />
                  </span>
                </div>
                <div className="mt-2 flex items-center justify-between text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <CreditCard className="h-3 w-3" />
                    {p.payment_method || p.gateway || '-'}
                  </span>
                  <span className="flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    {formatTanggal(p.created_at)}
                  </span>
                </div>
              </div>

              <div className="space-y-1.5">
                <span className="text-xs font-semibold text-muted-foreground block">
                  Bukti Transfer:
                </span>
                <PaymentProofDialog
                  proofUrl={p.proof_url}
                  invoiceNumber={p.invoice_number}
                />
              </div>
            </CardContent>

            <CardFooter className="flex flex-wrap items-center gap-2 pt-3 border-t border-border bg-muted/10">
              <Button
                type="button"
                size="sm"
                className="flex-1 gap-1.5 bg-emerald-600 hover:bg-emerald-700 text-white font-bold"
                onClick={() => setSelectedApprove(p)}
                disabled={isMutating}
              >
                <Check className="h-4 w-4" />
                Setujui
              </Button>
              <Button
                type="button"
                size="sm"
                variant="destructive"
                className="flex-1 gap-1.5 font-bold"
                onClick={() => setSelectedReject(p)}
                disabled={isMutating}
              >
                <X className="h-4 w-4" />
                Tolak
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="gap-1 text-xs"
                title="Refund pembayaran"
                onClick={() => setSelectedRefund(p)}
                disabled={isMutating}
                aria-label="Refund pembayaran"
              >
                <Undo2 className="h-3.5 w-3.5" />
                Refund
              </Button>
            </CardFooter>
          </Card>
        ))}
      </div>

      <PaymentApproveDialog
        payment={selectedApprove}
        open={Boolean(selectedApprove)}
        onOpenChange={(open) => !open && setSelectedApprove(null)}
        onConfirm={async (id) => {
          await approvePayment(id)
        }}
        isLoading={isApproving}
      />

      <PaymentRejectDialog
        payment={selectedReject}
        open={Boolean(selectedReject)}
        onOpenChange={(open) => !open && setSelectedReject(null)}
        onConfirm={async (id, reason) => {
          await rejectPayment({ id, reason })
        }}
        isLoading={isRejecting}
      />

      <PaymentRefundDialog
        payment={selectedRefund}
        open={Boolean(selectedRefund)}
        onOpenChange={(open) => !open && setSelectedRefund(null)}
        onConfirm={async (payload) => {
          await refundPayment(payload)
        }}
        isLoading={isRefunding}
      />
    </>
  )
}
