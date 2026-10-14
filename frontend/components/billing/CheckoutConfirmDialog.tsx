'use client'

import * as React from 'react'
import { Receipt, AlertCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Money } from '@/components/shared/Money'
import type { PublicPlan } from '@/hooks/use-billing'

/**
 * Pre-checkout confirmation. Shows exactly what will happen — the plan, the
 * price breakdown including PPN, and that an invoice will be issued — before
 * any invoice is created.
 *
 * Checkout is a side effect with real consequences (an issued invoice with a
 * due date), so it must never fire on a single unconfirmed click.
 */
export function CheckoutConfirmDialog({
  open,
  onOpenChange,
  plan,
  currentPlanName,
  onConfirm,
  loading,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan: PublicPlan | null
  currentPlanName: string
  onConfirm: () => void
  loading: boolean
}) {
  if (!plan) return null

  const subtotal = plan.price_monthly_idr
  const tax = plan.tax_idr ?? 0
  const total = plan.total_idr ?? subtotal
  const taxRate = plan.tax_rate_percent ?? 0

  return (
    <Dialog open={open} onOpenChange={(o) => !loading && onOpenChange(o)}>
      <DialogContent className="border-2 border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-lg font-black">
            <Receipt className="h-5 w-5 text-primary" aria-hidden="true" />
            Konfirmasi Peningkatan Paket
          </DialogTitle>
          <DialogDescription className="mt-2 text-xs font-bold text-muted-foreground">
            Tinjau rincian berikut. Faktur baru akan diterbitkan saat Anda menekan &ldquo;Lanjutkan&rdquo;.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 py-2 text-sm">
          <div className="flex justify-between border-b pb-2">
            <span className="font-medium text-muted-foreground">Paket saat ini</span>
            <span className="font-bold">{currentPlanName || 'Free'}</span>
          </div>
          <div className="flex justify-between border-b pb-2">
            <span className="font-medium text-muted-foreground">Paket tujuan</span>
            <span className="font-bold text-primary">{plan.name}</span>
          </div>
          <div className="flex justify-between border-b pb-2">
            <span className="font-medium text-muted-foreground">Durasi</span>
            <span className="font-bold">{plan.billing_period_days} hari</span>
          </div>

          <div className="space-y-1.5 border-2 border-foreground bg-muted/40 p-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Harga paket</span>
              <Money value={subtotal} />
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">PPN {taxRate}%</span>
              <Money value={tax} />
            </div>
            <div className="mt-1 flex justify-between border-t-2 border-foreground pt-2">
              <span className="font-black uppercase tracking-wide">Total bayar</span>
              <Money value={total} className="text-base font-black text-primary" />
            </div>
          </div>

          <p className="flex items-start gap-2 text-xs font-bold text-muted-foreground">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            Faktur akan diterbitkan dengan batas waktu pembayaran. Anda belum dikenakan biaya sampai
            pembayaran diselesaikan dan diverifikasi.
          </p>
        </div>

        <DialogFooter className="mt-4 gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Batal
          </Button>
          <Button onClick={onConfirm} disabled={loading}>
            {loading ? 'Membuat faktur…' : 'Lanjutkan'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
