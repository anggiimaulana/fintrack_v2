'use client'

import * as React from 'react'
import { CreditCard, ExternalLink, Upload } from 'lucide-react'
import { toast } from 'sonner'
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
import { useUploadProof, type CheckoutResult } from '@/hooks/use-billing'

const MAX_PROOF_BYTES = 5 * 1024 * 1024

/**
 * The post-checkout modal. It shows the issued invoice, every configured payment
 * destination, and — for the manual gateway — the proof upload inline, so the
 * user can pay and submit proof without leaving the page.
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
  const upload = useUploadProof()
  const [file, setFile] = React.useState<File | null>(null)

  // Reset the chosen file when a new payment result arrives. Adjusting state
  // during render (rather than in an effect) avoids a cascading render; the
  // input is remounted via `key` so its DOM value clears too.
  const [lastPaymentId, setLastPaymentId] = React.useState(paymentId)
  if (paymentId !== lastPaymentId) {
    setLastPaymentId(paymentId)
    setFile(null)
  }

  const onFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null
    if (f && f.size > MAX_PROOF_BYTES) {
      toast.error('Ukuran berkas melebihi 5 MB')
      e.target.value = ''
      return
    }
    setFile(f)
  }

  const submitProof = async () => {
    if (!paymentId || !file) {
      toast.error('Pilih berkas bukti terlebih dahulu')
      return
    }
    await upload.mutateAsync({ paymentId, file })
    setFile(null)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto border-2 rounded-md border-foreground bg-card shadow-[6px_6px_0px_0px] shadow-foreground sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-lg font-black">
            <CreditCard className="h-5 w-5 text-primary" aria-hidden="true" /> Selesaikan Pembayaran
          </DialogTitle>
          <DialogDescription className="mt-2 text-xs font-bold text-muted-foreground">
            Faktur telah diterbitkan. Bayar sesuai instruksi di bawah, lalu unggah bukti transfer Anda.
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

            {isManual && paymentId && (
              <TransferInstructions paymentId={paymentId} amountIdr={result.invoice?.total_idr ?? 0} />
            )}

            {/* Upload the proof right here — no need to hunt for the payments tab. */}
            {isManual && paymentId && (
              <div className="space-y-2 border-2 rounded-md border-foreground bg-muted/40 p-3">
                <p className="text-xs font-black uppercase tracking-wide text-foreground">
                  Unggah Bukti Transfer
                </p>
                <input
                  key={paymentId ?? 'none'}
                  type="file"
                  accept="image/png,image/jpeg,image/webp"
                  onChange={onFileChange}
                  aria-label="Unggah bukti transfer"
                  className="block w-full cursor-pointer border-2 rounded-md border-foreground bg-background px-2 py-1.5 text-xs font-bold file:mr-2 file:border-0 file:bg-primary file:px-2 file:py-1 file:text-xs file:font-bold file:text-primary-foreground"
                />
                <Button
                  onClick={submitProof}
                  disabled={!file || upload.isPending}
                  size="sm"
                  className="w-full cursor-pointer"
                >
                  <Upload className="mr-2 h-4 w-4" />
                  {upload.isPending ? 'Mengunggah…' : 'Kirim Bukti'}
                </Button>
                <p className="text-[11px] font-bold text-muted-foreground">
                  Setelah bukti dikirim, admin akan memverifikasi pembayaran Anda.
                </p>
              </div>
            )}

            {!isManual && (
              <p className="text-xs font-bold text-muted-foreground">
                Lanjutkan ke laman pembayaran untuk menyelesaikan transaksi.
              </p>
            )}
          </div>
        )}

        <DialogFooter className="mt-4 gap-2 sm:gap-0">
          <Button variant="outline" className="cursor-pointer" onClick={() => onOpenChange(false)}>
            Tutup
          </Button>
          {result?.redirect_url && (
            <Button asChild className="cursor-pointer">
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
