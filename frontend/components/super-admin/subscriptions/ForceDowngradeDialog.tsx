'use client'

import * as React from 'react'
import { AlertTriangle } from 'lucide-react'
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { AdminPlanOption, AdminSubscription } from '@/hooks/useSuperAdminSubscriptions'
import { useResetOnOpen } from '@/hooks/use-reset-on-open'

interface ForceDowngradeDialogProps {
  subscription: AdminSubscription | null
  plans: AdminPlanOption[]
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (payload: { plan_code: string; reason: string }) => Promise<unknown>
  isLoading?: boolean
}

const DEFAULT_CHEAPEST_VALUE = '__cheapest__'

export function ForceDowngradeDialog({
  subscription,
  plans,
  open,
  onOpenChange,
  onConfirm,
  isLoading = false,
}: ForceDowngradeDialogProps) {
  const [selectedPlanCode, setSelectedPlanCode] = React.useState<string>(DEFAULT_CHEAPEST_VALUE)
  const [reason, setReason] = React.useState<string>('')

  useResetOnOpen(open, () => {
    setSelectedPlanCode(DEFAULT_CHEAPEST_VALUE)
    setReason('')
  })

  const trimmedReason = reason.trim()
  const canSubmit = trimmedReason.length > 0 && !isLoading

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSubmit) return
    const plan_code = selectedPlanCode === DEFAULT_CHEAPEST_VALUE ? '' : selectedPlanCode
    await onConfirm({ plan_code, reason: trimmedReason })
    onOpenChange(false)
  }

  const activePlans = plans.filter((p) => p.is_active)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Paksa Turunkan Paket</DialogTitle>
            <DialogDescription>
              Turunkan paket langganan untuk user{' '}
              <span className="font-semibold text-foreground">
                {subscription?.user_email || 'pengguna'}
              </span>
              . Paket saat ini:{' '}
              <span className="font-semibold text-foreground">
                {subscription?.plan_name || subscription?.plan_code}
              </span>
              .
            </DialogDescription>
          </DialogHeader>

          <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-600 dark:text-amber-400 flex items-start gap-2">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold">Peringatan Entitlement Cache</p>
              <p className="mt-0.5">
                Hak akses (entitlement cache) pengguna akan langsung berubah seketika. Riwayat
                transaksi finansial dan invoice tidak dihapus.
              </p>
            </div>
          </div>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="downgrade-plan">Pilih Paket Tujuan</Label>
              <Select
                value={selectedPlanCode}
                onValueChange={setSelectedPlanCode}
                disabled={isLoading}
              >
                <SelectTrigger id="downgrade-plan" className="w-full">
                  <SelectValue placeholder="Pilih paket..." />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={DEFAULT_CHEAPEST_VALUE}>
                    Otomatis: Paket termurah aktif
                  </SelectItem>
                  {activePlans.map((p) => (
                    <SelectItem key={p.id} value={p.code}>
                      {p.name} ({p.code})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                Jika dibiarkan default, sistem akan otomatis memilih paket aktif dengan harga
                termurah.
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="downgrade-reason">
                Alasan Administratif <span className="text-destructive">*</span>
              </Label>
              <Input
                id="downgrade-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Contoh: Permintaan pembatalan sepihak atau dispute pembayaran"
                required
                disabled={isLoading}
              />
              <p className="text-xs text-muted-foreground">
                Alasan wajib diisi untuk catatan event dan audit log.
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
            <Button type="submit" variant="destructive" disabled={!canSubmit}>
              {isLoading ? 'Memproses...' : 'Turunkan Paket'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
