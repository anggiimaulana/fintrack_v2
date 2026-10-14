'use client'

import * as React from 'react'
import { CreditCard, ExternalLink } from 'lucide-react'
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
import { TransferInstructions } from './TransferInstructions'
import type { CheckoutResult } from '@/hooks/use-billing'

/**
 * The post-checkout modal. It shows the issued invoice and — for the manual
 * gateway — the transfer instructions plus a pointer to the proof upload, so
 * the user can complete payment without leaving the page.
 */
export function CheckoutDialog({
  open,
  onOpenChange,
  result,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  result: CheckoutResult | null
}) {
  const paymentId = result?.payment?.id
  const isManual = result?.gateway === 'manual'
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="border-2 border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-lg font-black">
            <CreditCard className="h-5 w-5 text-primary" aria-hidden="true" /> Sesi Pembayaran Dibuat
          </DialogTitle>
          <DialogDescription className="mt-2 text-xs font-bold text-muted-foreground">
            Faktur telah diterbitkan. Selesaikan pembayaran untuk mengaktifkan paket Anda.
          </DialogDescription>
        </DialogHeader>

        {result && (
          <div className="space-y-3 py-2 text-sm">
            <div className="flex justify-between border-b pb-2">
              <span className="font-medium text-muted-foreground">Nomor Faktur</span>
              <span className="font-mono font-bold">{result.invoice?.invoice_number || '—'}</span>
            </div>
            <div className="flex justify-between border-b pb-2">
              <span className="font-medium text-muted-foreground">Total Pembayaran</span>
              <Money value={result.invoice?.total_idr ?? 0} className="font-black text-primary" />
            </div>
            <div className="flex justify-between border-b pb-2">
              <span className="font-medium text-muted-foreground">Metode Gateway</span>
              <span className="font-bold uppercase">{result.gateway || '—'}</span>
            </div>

            {isManual && paymentId && (
              <TransferInstructions paymentId={paymentId} amountIdr={result.invoice?.total_idr ?? 0} />
            )}
            {isManual && (
              <p className="text-xs font-bold text-muted-foreground">
                Setelah transfer, unggah bukti pada kartu pesanan di bawah agar admin dapat memverifikasi.
              </p>
            )}
          </div>
        )}

        <DialogFooter className="mt-4 gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
          {result?.redirect_url && (
            <Button asChild>
              <a href={result.redirect_url} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="mr-2 h-4 w-4" /> Buka Laman Pembayaran
              </a>
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
