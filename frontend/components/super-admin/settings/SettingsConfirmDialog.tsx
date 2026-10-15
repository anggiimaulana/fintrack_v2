'use client'

import * as React from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

export interface SettingDiff {
  key: string
  label: string
  before: unknown
  after: unknown
}

interface SettingsConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  diffs: SettingDiff[]
  onConfirm: () => void
  isSubmitting?: boolean
}

export function SettingsConfirmDialog({
  open,
  onOpenChange,
  diffs,
  onConfirm,
  isSubmitting = false,
}: SettingsConfirmDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Konfirmasi Perubahan Pengaturan</DialogTitle>
          <DialogDescription>
            Pastikan perubahan berikut sudah sesuai sebelum disimpan ke sistem.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 py-2">
          <div className="overflow-hidden rounded-md border border-border divide-y divide-border text-sm">
            {diffs.map((diff) => (
              <div key={diff.key} className="p-3 bg-card space-y-1">
                <div className="font-semibold text-foreground flex items-center justify-between">
                  <span>{diff.label}</span>
                  <code className="text-xs text-muted-foreground font-mono">{diff.key}</code>
                </div>
                <div className="flex items-center gap-2 text-xs">
                  <span className="line-through text-muted-foreground">
                    {String(diff.before ?? '(kosong)')}
                  </span>
                  <span className="text-muted-foreground">→</span>
                  <span className="font-bold text-foreground bg-primary/10 px-1 py-0.5 border rounded-md border-primary/20">
                    {String(diff.after)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isSubmitting}
          >
            Batal
          </Button>
          <Button
            type="button"
            onClick={onConfirm}
            disabled={isSubmitting}
          >
            {isSubmitting ? 'Menyimpan...' : 'Simpan Perubahan'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
