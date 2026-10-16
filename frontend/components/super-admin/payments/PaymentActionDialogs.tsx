'use client'

import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Money } from '@/components/shared/Money'
import { RupiahInput } from '@/components/shared/RupiahInput'
import { Loader2 } from 'lucide-react'
import type { PaymentRow } from '@/hooks/useSuperAdminPayments'
import { useResetOnOpen } from '@/hooks/use-reset-on-open'

interface PaymentApproveDialogProps {
  payment: PaymentRow | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (id: string) => Promise<void>
  isLoading?: boolean
}

export function PaymentApproveDialog({
  payment,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: PaymentApproveDialogProps) {
  if (!payment) return null

  const handleApprove = async () => {
    await onConfirm(payment.id)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto px-6 py-6 sm:max-w-[480px] border-2 border-foreground">
        <DialogHeader>
          <DialogTitle>Setujui Pembayaran</DialogTitle>
          <DialogDescription>
            Konfirmasi persetujuan pembayaran untuk invoice{' '}
            <span className="font-semibold text-foreground">
              {payment.invoice_number || payment.id}
            </span>
            . Langganan user akan diaktifkan dan invoice diterbitkan.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 rounded border border-border bg-muted/40 p-3 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">User:</span>
            <span className="font-medium text-foreground">{payment.user_email}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Nominal:</span>
            <span className="font-medium text-foreground">
              <Money value={payment.amount_idr} />
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Metode:</span>
            <span className="font-medium text-foreground">{payment.payment_method || '-'}</span>
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Batal
          </Button>
          <Button
            type="button"
            onClick={handleApprove}
            disabled={isLoading}
            className="disabled:opacity-50 disabled:cursor-not-allowed font-bold"
          >
            {isLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Memproses...
              </>
            ) : (
              'Setujui Pembayaran'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface PaymentRejectDialogProps {
  payment: PaymentRow | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (id: string, reason: string) => Promise<void>
  isLoading?: boolean
}

export function PaymentRejectDialog({
  payment,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: PaymentRejectDialogProps) {
  const [reason, setReason] = React.useState('')

  useResetOnOpen(open, () => setReason(''))

  if (!payment) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!reason.trim()) return
    await onConfirm(payment.id, reason.trim())
    onOpenChange(false)
  }

  const isValid = reason.trim().length > 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto px-6 py-6 sm:max-w-[480px] border-2 border-foreground">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Tolak Pembayaran</DialogTitle>
            <DialogDescription>
              Tolak verifikasi pembayaran untuk invoice{' '}
              <span className="font-semibold text-foreground">
                {payment.invoice_number || payment.id}
              </span>
              . Alasan penolakan wajib diisi.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="reject-reason">
              Alasan Penolakan <span className="text-destructive">*</span>
            </Label>
            <Input
              id="reject-reason"
              placeholder="Contoh: Nominal transfer tidak sesuai mutasi rekening"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              required
              disabled={isLoading}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={!isValid || isLoading}
              className="disabled:opacity-50 disabled:cursor-not-allowed font-bold"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Menolak...
                </>
              ) : (
                'Tolak Pembayaran'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

interface PaymentRefundDialogProps {
  payment: PaymentRow | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (payload: { id: string; amount_idr?: number; reason: string; reference?: string }) => Promise<void>
  isLoading?: boolean
}

export function PaymentRefundDialog({
  payment,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: PaymentRefundDialogProps) {
  const [amountNum, setAmountNum] = React.useState<number | undefined>(undefined)
  const [reason, setReason] = React.useState('')
  const [reference, setReference] = React.useState('')

  useResetOnOpen(open, () => {
    setAmountNum(undefined)
    setReason('')
    setReference('')
  })

  if (!payment) return null

  const remaining = Math.max(0, payment.amount_idr - (payment.refund_amount_idr || 0))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!reason.trim()) return

    await onConfirm({
      id: payment.id,
      amount_idr: amountNum && amountNum > 0 ? amountNum : undefined,
      reason: reason.trim(),
      reference: reference.trim() || undefined,
    })
    onOpenChange(false)
  }

  const isValid = reason.trim().length > 0

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto px-6 py-6 sm:max-w-[500px] border-2 border-foreground">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Proses Refund Pembayaran</DialogTitle>
            <DialogDescription>
              Invoice{' '}
              <span className="font-semibold text-foreground">
                {payment.invoice_number || payment.id}
              </span>
              . Kosongkan nominal untuk refund sisa penuh.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-1 rounded border border-border bg-muted/40 p-3 text-xs">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Total Bayar:</span>
              <span className="font-medium text-foreground">
                <Money value={payment.amount_idr} />
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Sudah Direfund:</span>
              <span className="font-medium text-foreground">
                <Money value={payment.refund_amount_idr || 0} />
              </span>
            </div>
            <div className="flex justify-between font-semibold">
              <span className="text-muted-foreground">Sisa Maksimal:</span>
              <span className="text-foreground">
                <Money value={remaining} />
              </span>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="refund-amount">
              Nominal Refund (IDR) <span className="text-xs text-muted-foreground font-normal">(opsional: kosong = sisa penuh)</span>
            </Label>
            <RupiahInput
              id="refund-amount"
              value={amountNum}
              onChange={setAmountNum}
              min={1}
              placeholder={`Kosong = sisa penuh (${remaining.toLocaleString('id-ID')})`}
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="refund-reason">
              Alasan Refund <span className="text-destructive">*</span>
            </Label>
            <Input
              id="refund-reason"
              placeholder="Contoh: Permintaan pembatalan user / double transfer"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              required
              disabled={isLoading}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="refund-reference">
              Nomor Referensi Transfer Bank / Gateway <span className="text-xs text-muted-foreground font-normal">(opsional)</span>
            </Label>
            <Input
              id="refund-reference"
              placeholder="Contoh: REF-BCA-981247"
              value={reference}
              onChange={(e) => setReference(e.target.value)}
              disabled={isLoading}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Batal
            </Button>
            <Button
              type="submit"
              variant="default"
              disabled={!isValid || isLoading}
              className="disabled:opacity-50 disabled:cursor-not-allowed font-bold"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Memproses...
                </>
              ) : (
                'Kirim Refund'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
