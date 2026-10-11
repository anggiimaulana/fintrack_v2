'use client'

import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Money } from '@/components/shared/Money'
import { formatTanggal } from '@/lib/format'
import { PaymentStatusBadge } from './PaymentStatusBadge'
import { PaymentProofDialog } from './PaymentProofDialog'
import type { PaymentRow } from '@/hooks/useSuperAdminPayments'

interface PaymentDetailDialogProps {
  payment: PaymentRow | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function PaymentDetailDialog({
  payment,
  open,
  onOpenChange,
}: PaymentDetailDialogProps) {
  if (!payment) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl sm:max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-3">
            <span>Detail Pembayaran</span>
            <PaymentStatusBadge status={payment.status} />
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 text-sm">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 rounded border border-border bg-card p-3">
            <div>
              <p className="text-xs text-muted-foreground">ID Pembayaran</p>
              <p className="font-mono text-xs font-semibold text-foreground break-all">{payment.id}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Nomor Invoice</p>
              <p className="font-mono text-xs font-semibold text-foreground">
                {payment.invoice_number || payment.invoice_id || '-'}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Email User</p>
              <p className="font-medium text-foreground">{payment.user_email}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">User ID</p>
              <p className="font-mono text-xs text-foreground break-all">{payment.user_id}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Gateway / Metode</p>
              <p className="font-medium text-foreground">
                {payment.gateway} {payment.payment_method ? `(${payment.payment_method})` : ''}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Gateway Ref</p>
              <p className="font-mono text-xs text-foreground">{payment.gateway_ref || '-'}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Nominal</p>
              <p className="text-base font-bold text-foreground">
                <Money value={payment.amount_idr} />
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Tanggal Dibuat</p>
              <p className="font-medium text-foreground">{formatTanggal(payment.created_at)}</p>
            </div>
          </div>

          <div className="space-y-2 rounded border border-border bg-muted/30 p-3">
            <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
              Review Verifikasi Manual
            </h4>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
              <div>
                <span className="text-muted-foreground">Bukti Transfer: </span>
                <div className="mt-1">
                  <PaymentProofDialog
                    proofUrl={payment.proof_url}
                    invoiceNumber={payment.invoice_number}
                  />
                </div>
              </div>
              <div>
                <span className="text-muted-foreground">Direview Oleh: </span>
                <span className="font-mono text-foreground">{payment.reviewed_by || '-'}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Waktu Review: </span>
                <span className="text-foreground">{formatTanggal(payment.reviewed_at)}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Catatan Review: </span>
                <span className="text-foreground">{payment.review_reason || '-'}</span>
              </div>
            </div>
          </div>

          {(payment.refund_amount_idr > 0 || payment.refund_reason) && (
            <div className="space-y-2 rounded border border-border bg-muted/30 p-3">
              <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                Informasi Refund
              </h4>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-muted-foreground">Total Refund: </span>
                  <span className="font-bold text-foreground">
                    <Money value={payment.refund_amount_idr} />
                  </span>
                </div>
                <div>
                  <span className="text-muted-foreground">Ref Refund: </span>
                  <span className="font-mono text-foreground">{payment.refund_ref || '-'}</span>
                </div>
                <div className="sm:col-span-2">
                  <span className="text-muted-foreground">Alasan: </span>
                  <span className="text-foreground">{payment.refund_reason || '-'}</span>
                </div>
              </div>
            </div>
          )}

          <div className="space-y-1">
            <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
              Raw JSON Data
            </p>
            <pre className="max-h-40 overflow-auto rounded border border-border bg-muted p-2 font-mono text-[11px] text-muted-foreground">
              {JSON.stringify(payment, null, 2)}
            </pre>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
