'use client'

import { useState, useId } from 'react'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  useAdjustTokens,
  type TokenBalanceRow,
} from '@/hooks/useSuperAdminTokens'

interface AdjustTokensDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: TokenBalanceRow | null
}

export function AdjustTokensDialog({
  open,
  onOpenChange,
  user,
}: AdjustTokensDialogProps) {
  const [amountStr, setAmountStr] = useState('')
  const [reason, setReason] = useState('')
  const [idempotencyKey, setIdempotencyKey] = useState('')
  const [apiError, setApiError] = useState<string | null>(null)

  const adjustMutation = useAdjustTokens()

  const amountHelpId = useId()
  const errorHelpId = useId()

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setIdempotencyKey(crypto.randomUUID())
      setAmountStr('')
      setReason('')
      setApiError(null)
    } else {
      setApiError(null)
    }
    onOpenChange(nextOpen)
  }

  const parsedAmount = parseInt(amountStr, 10)
  const isAmountValid = !isNaN(parsedAmount) && parsedAmount !== 0
  const isReasonValid = reason.trim().length > 0
  const canSubmit = isAmountValid && isReasonValid && !adjustMutation.isPending

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!user || !canSubmit) return

    setApiError(null)

    try {
      const res = await adjustMutation.mutateAsync({
        user_id: user.user_id,
        amount: parsedAmount,
        reason: reason.trim(),
        idempotency_key: idempotencyKey,
      })

      if (res.replayed) {
        toast.success('Penyesuaian sudah pernah diterapkan (replay)')
      } else {
        toast.success('Penyesuaian token berhasil diterapkan')
      }

      onOpenChange(false)
    } catch (err: unknown) {
      const axiosErr = err as {
        response?: {
          status?: number
          data?: {
            message?: string
            error?: string
          }
        }
      }

      const msg =
        axiosErr.response?.data?.message ||
        'Gagal memproses penyesuaian token'

      // Contract §5: 409 NO_ACTIVE_PERIOD & 400 validation inline under amount
      if (
        axiosErr.response?.status === 409 ||
        axiosErr.response?.status === 400
      ) {
        setApiError(msg)
      } else {
        toast.error(msg)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Sesuaikan Saldo Token</DialogTitle>
            <DialogDescription>
              {user ? (
                <span>
                  Pengguna: <strong className="text-foreground">{user.user_full_name}</strong> ({user.user_email})
                  <br />
                  Saldo tersedia saat ini: <strong className="text-foreground">{user.available_tokens.toLocaleString('id-ID')}</strong> token
                </span>
              ) : (
                'Pilih pengguna untuk penyesuaian token'
              )}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-1.5">
              <Label htmlFor="adjust-amount">Jumlah Token</Label>
              <Input
                id="adjust-amount"
                type="number"
                step="1"
                placeholder="Contoh: 5000 atau -5000"
                value={amountStr}
                onChange={(e) => {
                  setAmountStr(e.target.value)
                  if (apiError) setApiError(null)
                }}
                aria-describedby={apiError ? errorHelpId : amountHelpId}
                aria-invalid={!!apiError || (amountStr !== '' && !isAmountValid)}
                required
              />
              <p id={amountHelpId} className="text-xs text-muted-foreground">
                Nilai bertanda: positif (+) = kredit, negatif (−) = debit. Tidak boleh 0.
              </p>
              {apiError && (
                <p id={errorHelpId} className="text-xs font-semibold text-destructive">
                  {apiError}
                </p>
              )}
            </div>

            <div className="grid gap-1.5">
              <Label htmlFor="adjust-reason">Alasan Administratif</Label>
              <Input
                id="adjust-reason"
                placeholder="Contoh: Koreksi audit, kompensasi gangguan sistem"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                required
              />
            </div>

            <div className="grid gap-1.5">
              <Label htmlFor="adjust-idempotency" className="text-xs text-muted-foreground">
                Idempotency Key (Otomatis per sesi dialog)
              </Label>
              <Input
                id="adjust-idempotency"
                value={idempotencyKey}
                readOnly
                className="bg-muted font-mono text-xs text-muted-foreground cursor-not-allowed select-all"
                tabIndex={-1}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={adjustMutation.isPending}
            >
              Batal
            </Button>
            <Button
              type="submit"
              disabled={!canSubmit}
            >
              {adjustMutation.isPending ? 'Memproses...' : 'Terapkan Penyesuaian'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
