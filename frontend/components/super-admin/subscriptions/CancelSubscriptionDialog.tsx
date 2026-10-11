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
import type { AdminSubscription } from '@/hooks/useSuperAdminSubscriptions'
import { useResetOnOpen } from '@/hooks/use-reset-on-open'

interface CancelSubscriptionDialogProps {
  subscription: AdminSubscription | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (payload: { reason: string; immediate: boolean }) => Promise<unknown>
  isLoading?: boolean
}

export function CancelSubscriptionDialog({
  subscription,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: CancelSubscriptionDialogProps) {
  const [immediate, setImmediate] = React.useState<boolean>(false)
  const [reason, setReason] = React.useState<string>('')

  useResetOnOpen(open, () => {
    setImmediate(false)
    setReason('')
  })

  const trimmedReason = reason.trim()
  const canSubmit = trimmedReason.length > 0 && !isLoading

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    await onConfirm({ reason: trimmedReason, immediate })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Batalkan Langganan</DialogTitle>
            <DialogDescription>
              Batalkan langganan pengguna{' '}
              <span className="font-semibold text-foreground">
                {subscription?.user_email || 'pengguna'}
              </span>
              .
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="rounded-md border p-3 bg-muted/30">
              <label
                htmlFor="cancel-immediate"
                className="flex items-start gap-3 cursor-pointer select-none"
              >
                <input
                  id="cancel-immediate"
                  aria-label="Berhenti sekarang (Immediate)"
                  type="checkbox"
                  checked={immediate}
                  onChange={(e) => setImmediate(e.target.checked)}
                  disabled={isLoading}
                  className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-ring"
                />
                <div className="space-y-0.5">
                  <span className="text-sm font-semibold text-foreground block">
                    Berhenti sekarang (Immediate)
                  </span>
                  <span className="text-xs text-muted-foreground block">
                    {immediate
                      ? 'Akses fitur langganan akan dicabut detik ini juga dan status langsung menjadi canceled.'
                      : 'Pengguna tetap dapat menikmati fitur sampai masa periode berakhir (cancel_at_period_end).'}
                  </span>
                </div>
              </label>
            </div>

            <div className="space-y-2">
              <Label htmlFor="cancel-reason">
                Alasan Pembatalan <span className="text-destructive">*</span>
              </Label>
              <Input
                id="cancel-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Contoh: Permintaan pengguna atau refund disetujui"
                required
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Alasan administratif wajib diisi untuk catatan event dan audit.
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              Kembali
            </Button>
            <Button type="submit" variant="destructive" disabled={!canSubmit}>
              {isLoading ? 'Membatalkan...' : 'Batalkan Langganan'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
