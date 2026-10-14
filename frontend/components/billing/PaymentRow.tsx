'use client'

import * as React from 'react'
import { Upload, CheckCircle2, Clock, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Money } from '@/components/shared/Money'
import { TransferInstructions } from './TransferInstructions'
import { useUploadProof, type UserPayment } from '@/hooks/use-billing'
import { formatTanggal } from '@/lib/format'

const MAX_PROOF_BYTES = 5 * 1024 * 1024

const STATUS_LABEL: Record<UserPayment['status'], string> = {
  pending: 'Menunggu pembayaran',
  pending_review: 'Menunggu verifikasi',
  settled: 'Berhasil',
  failed: 'Gagal',
  expired: 'Kedaluwarsa',
  refunded: 'Dikembalikan',
}

const STATUS_VARIANT: Record<UserPayment['status'], 'default' | 'secondary' | 'destructive' | 'outline'> = {
  pending: 'secondary',
  pending_review: 'outline',
  settled: 'default',
  failed: 'destructive',
  expired: 'destructive',
  refunded: 'secondary',
}

/**
 * One payment row: amount, status, transfer instructions, and the proof-upload
 * control that unblocks admin review. Upload is only offered while a payment is
 * `pending` — a settled/failed payment is terminal.
 */
export function PaymentRow({ payment }: { payment: UserPayment }) {
  const [file, setFile] = React.useState<File | null>(null)
  const upload = useUploadProof()
  const inputRef = React.useRef<HTMLInputElement>(null)

  const canUpload = payment.status === 'pending'
  const planLabel = payment.plan_name || payment.plan_code || '—'

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null
    if (f && f.size > MAX_PROOF_BYTES) {
      toast.error('Ukuran berkas melebihi 5 MB')
      e.target.value = ''
      return
    }
    setFile(f)
  }

  const submit = async () => {
    if (!file) {
      toast.error('Pilih berkas bukti terlebih dahulu')
      return
    }
    await upload.mutateAsync({ paymentId: payment.id, file })
    setFile(null)
    if (inputRef.current) inputRef.current.value = ''
  }

  const statusIcon =
    payment.status === 'settled' ? (
      <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
    ) : payment.status === 'pending' || payment.status === 'pending_review' ? (
      <Clock className="h-3.5 w-3.5" aria-hidden="true" />
    ) : (
      <XCircle className="h-3.5 w-3.5" aria-hidden="true" />
    )

  return (
    <div className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b-2 border-foreground p-4">
        <div className="min-w-0">
          <p className="truncate text-sm font-black uppercase tracking-wide text-foreground">{planLabel}</p>
          <p className="font-mono text-xs font-bold text-muted-foreground">{payment.invoice_number || '—'}</p>
        </div>
        <div className="flex items-center gap-3">
          <Money value={payment.amount_idr} className="text-sm font-black" />
          <Badge variant={STATUS_VARIANT[payment.status]} className="gap-1">
            {statusIcon}
            {STATUS_LABEL[payment.status]}
          </Badge>
        </div>
      </div>

      <div className="space-y-3 p-4">
        <div className="grid grid-cols-2 gap-2 text-xs font-bold text-muted-foreground sm:grid-cols-4">
          <span>Dibuat: {formatTanggal(payment.created_at)}</span>
          <span>Jatuh tempo: {payment.invoice_due_at ? formatTanggal(payment.invoice_due_at) : '—'}</span>
          <span>Gateway: {payment.gateway}</span>
          <span>Subtotal: <Money value={payment.subtotal_idr} /></span>
        </div>

        {canUpload && <TransferInstructions paymentId={payment.id} amountIdr={payment.amount_idr} />}

        {canUpload && (
          <div className="flex flex-wrap items-center gap-3">
            <input
              ref={inputRef}
              type="file"
              accept="image/png,image/jpeg,image/webp"
              onChange={onFileChange}
              aria-label={`Unggah bukti transfer untuk ${payment.invoice_number}`}
              className="block w-full max-w-xs cursor-pointer border-2 border-foreground bg-background px-2 py-1.5 text-xs font-bold file:mr-2 file:border-0 file:bg-primary file:px-2 file:py-1 file:text-xs file:font-bold file:text-primary-foreground"
            />
            <Button onClick={submit} disabled={!file || upload.isPending} size="sm">
              <Upload className="mr-2 h-4 w-4" />
              {upload.isPending ? 'Mengunggah…' : 'Unggah Bukti'}
            </Button>
          </div>
        )}

        {payment.status === 'pending_review' && (
          <p className="text-xs font-bold text-amber-600 dark:text-amber-400">
            Bukti terkirim. Admin akan memverifikasi pembayaran Anda.
          </p>
        )}
        {payment.status === 'settled' && (
          <p className="text-xs font-bold text-emerald-600 dark:text-emerald-400">
            Pembayaran berhasil — paket Anda sudah aktif.
          </p>
        )}
      </div>
    </div>
  )
}
