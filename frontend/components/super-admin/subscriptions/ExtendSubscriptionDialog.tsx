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

interface ExtendSubscriptionDialogProps {
  subscription: AdminSubscription | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (payload: { days: number; reason: string }) => Promise<unknown>
  isLoading?: boolean
}

export function ExtendSubscriptionDialog({
  subscription,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: ExtendSubscriptionDialogProps) {
  const [days, setDays] = React.useState<number>(30)
  const [reason, setReason] = React.useState<string>('')

  useResetOnOpen(open, () => {
    setDays(30)
    setReason('')
  })

  const trimmedReason = reason.trim()
  const isDaysValid = Number.isInteger(days) && days >= 1
  const canSubmit = isDaysValid && trimmedReason.length > 0 && !isLoading

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    await onConfirm({ days, reason: trimmedReason })
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Perpanjang Masa Aktif</DialogTitle>
            <DialogDescription>
              Tambahkan masa aktif langganan untuk user{' '}
              <span className="font-semibold text-foreground">
                {subscription?.user_email || 'pengguna'}
              </span>
              . Tindakan ini akan dicatat ke audit log.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="extend-days">Jumlah Hari Tambahan</Label>
              <Input
                id="extend-days"
                type="number"
                min={1}
                step={1}
                value={days}
                onChange={(e) => setDays(parseInt(e.target.value, 10) || 0)}
                placeholder="Contoh: 30"
                required
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">Minimal 1 hari integer.</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="extend-reason">
                Alasan Administratif <span className="text-destructive">*</span>
              </Label>
              <Input
                id="extend-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Contoh: Kompensasi insiden layanan"
                required
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Alasan wajib diisi untuk kepatuhan audit.
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
              Batal
            </Button>
            <Button type="submit" disabled={!canSubmit}>
              {isLoading ? 'Menyimpan...' : 'Perpanjang'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
