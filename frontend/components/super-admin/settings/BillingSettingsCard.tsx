'use client'

import * as React from 'react'
import { toast } from 'sonner'
import { Receipt, Clock, Percent, Save } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useConfigStatus } from '@/hooks/use-config-status'
import { useSuperAdminSettings } from '@/hooks/useSuperAdminSettings'

/** Numeric string state for the form (empty = unset). */
interface BillingForm {
  tax_enabled: boolean
  tax_label: string
  tax_rate_percent: string
  invoice_expiry_hours: string
  invoice_expiry_minutes: string
}

const DEFAULTS: BillingForm = {
  tax_enabled: false,
  tax_label: 'PPN',
  tax_rate_percent: '0',
  invoice_expiry_hours: '24',
  invoice_expiry_minutes: '0',
}

/** Reads the effective values into the form shape. */
function fromEffective(effective: Record<string, string | number | boolean>): BillingForm {
  const str = (k: string, def: string) => (effective[k] != null ? String(effective[k]) : def)
  return {
    tax_enabled: effective.tax_enabled === true || effective.tax_enabled === 'true',
    tax_label: str('tax_label', 'PPN'),
    tax_rate_percent: str('tax_rate_percent', '0'),
    invoice_expiry_hours: str('invoice_expiry_hours', '24'),
    invoice_expiry_minutes: str('invoice_expiry_minutes', '0'),
  }
}

/**
 * Billing settings: PPN (on/off, dynamic rate + label) and the invoice expiry
 * window (hours + minutes). Both are read by the backend at invoice time, so a
 * change applies to the next invoice without a redeploy.
 */
export function BillingSettingsCard() {
  const { data: status, isLoading } = useConfigStatus()
  const { updateSettings, isSavingSettings } = useSuperAdminSettings()

  const [edits, setEdits] = React.useState<Partial<BillingForm>>({})
  const [touched, setTouched] = React.useState(false)

  const baseline = status ? fromEffective(status.effective) : DEFAULTS
  const form: BillingForm = { ...baseline, ...edits }
  const patch = (next: Partial<BillingForm>) => {
    setEdits((prev) => ({ ...prev, ...next }))
    setTouched(true)
  }

  const rateNum = Number(form.tax_rate_percent)
  const rateInvalid = form.tax_enabled && (form.tax_rate_percent.trim() === '' || Number.isNaN(rateNum) || rateNum < 0 || rateNum > 100)
  const hoursNum = Number(form.invoice_expiry_hours)
  const minsNum = Number(form.invoice_expiry_minutes)
  const hoursInvalid = form.invoice_expiry_hours.trim() === '' || !Number.isInteger(hoursNum) || hoursNum < 0 || hoursNum > 8760
  const minsInvalid = form.invoice_expiry_minutes.trim() === '' || !Number.isInteger(minsNum) || minsNum < 0 || minsNum > 59
  const windowEmpty = !hoursInvalid && !minsInvalid && hoursNum === 0 && minsNum === 0
  const canSave = !rateInvalid && !hoursInvalid && !minsInvalid && !windowEmpty && touched

  const handleSave = async () => {
    if (!canSave) return
    try {
      await updateSettings({
        tax_enabled: form.tax_enabled,
        tax_label: form.tax_label.trim() || 'PPN',
        tax_rate_percent: parseInt(form.tax_rate_percent, 10),
        invoice_expiry_hours: hoursNum,
        invoice_expiry_minutes: minsNum,
      })
      toast.success('Pengaturan tagihan disimpan')
      setEdits({})
      setTouched(false)
    } catch {
      toast.error('Gagal menyimpan pengaturan tagihan')
    }
  }

  if (isLoading) {
    return <Card><CardContent className="h-40 animate-pulse" /></Card>
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Receipt className="size-4" /> Pajak &amp; Masa Berlaku Tagihan
        </CardTitle>
        <CardDescription>
          PPN dapat dinyalakan/dimatikan dan tarifnya diubah kapan saja. Masa berlaku tagihan mengatur kapan
          tagihan yang belum dibayar otomatis kedaluwarsa.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* ── PPN ───────────────────────────────────────────── */}
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <input
              id="tax_enabled"
              type="checkbox"
              checked={form.tax_enabled}
              onChange={(e) => patch({ tax_enabled: e.target.checked })}
              className="size-4"
            />
            <Label htmlFor="tax_enabled" className="text-sm font-bold">
              Aktifkan PPN pada tagihan
            </Label>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="tax_label" className="flex items-center gap-1.5">
                <Percent className="size-3.5" /> Label Pajak
              </Label>
              <Input
                id="tax_label"
                value={form.tax_label}
                maxLength={32}
                disabled={!form.tax_enabled}
                placeholder="PPN"
                onChange={(e) => patch({ tax_label: e.target.value })}
              />
              <p className="text-xs text-muted-foreground">Nama pajak yang tampil di tagihan (mis. PPN / PB1).</p>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="tax_rate_percent">Tarif Pajak (%)</Label>
              <Input
                id="tax_rate_percent"
                type="number"
                min={0}
                max={100}
                disabled={!form.tax_enabled}
                value={form.tax_rate_percent}
                onChange={(e) => patch({ tax_rate_percent: e.target.value })}
              />
              {rateInvalid ? (
                <p className="text-xs font-bold text-red-600">Tarif harus 0–100.</p>
              ) : (
                <p className="text-xs text-muted-foreground">
                  {form.tax_enabled ? `Tagihan akan dikenai ${form.tax_label || 'PPN'} ${form.tax_rate_percent || 0}%.` : 'PPN nonaktif: tagihan tanpa pajak.'}
                </p>
              )}
            </div>
          </div>
        </div>

        {/* ── Expiry ────────────────────────────────────────── */}
        <div className="space-y-4 border-t-2 border-foreground/10 pt-4">
          <p className="flex items-center gap-1.5 text-sm font-bold">
            <Clock className="size-4" /> Masa Berlaku Tagihan
          </p>
          <div className="grid grid-cols-2 gap-4 sm:max-w-sm">
            <div className="space-y-1.5">
              <Label htmlFor="invoice_expiry_hours">Jam</Label>
              <Input
                id="invoice_expiry_hours"
                type="number"
                min={0}
                max={8760}
                value={form.invoice_expiry_hours}
                onChange={(e) => patch({ invoice_expiry_hours: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="invoice_expiry_minutes">Menit</Label>
              <Input
                id="invoice_expiry_minutes"
                type="number"
                min={0}
                max={59}
                value={form.invoice_expiry_minutes}
                onChange={(e) => patch({ invoice_expiry_minutes: e.target.value })}
              />
            </div>
          </div>
          {hoursInvalid || minsInvalid ? (
            <p className="text-xs font-bold text-red-600">
              Jam harus 0–8760 dan menit 0–59.
            </p>
          ) : windowEmpty ? (
            <p className="text-xs font-bold text-red-600">Masa berlaku tidak boleh 0 jam 0 menit.</p>
          ) : (
            <p className="text-xs text-muted-foreground">
              Tagihan yang belum dibayar otomatis <strong>kedaluwarsa</strong> setelah{' '}
              {hoursNum} jam {minsNum} menit. Pengguna dapat membatalkan tagihan lebih awal.
            </p>
          )}
        </div>

        <Button onClick={handleSave} disabled={!canSave || isSavingSettings} className="cursor-pointer">
          <Save className="mr-2 size-4" />
          {isSavingSettings ? 'Menyimpan…' : 'Simpan Pengaturan Tagihan'}
        </Button>
      </CardContent>
    </Card>
  )
}
