'use client'

import * as React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { CreditCard, Landmark } from 'lucide-react'
import { SecretField } from './SecretField'
import { PaymentAccountsManager } from './PaymentAccountsManager'
import { BillingSettingsCard } from './BillingSettingsCard'
import { useConfigStatus } from '@/hooks/use-config-status'

interface PaymentSettingsTabProps {
  isLoading: boolean
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
}

/** Editable payment fields, seeded from the effective config. */
interface PaymentForm {
  midtrans_enabled: boolean
  midtrans_server_key: string
  midtrans_client_key: string
  midtrans_is_production: boolean
  manual_bank_name: string
  manual_account_number: string
  manual_account_holder: string
}

const EMPTY: PaymentForm = {
  midtrans_enabled: false,
  midtrans_server_key: '',
  midtrans_client_key: '',
  midtrans_is_production: false,
  manual_bank_name: '',
  manual_account_number: '',
  manual_account_holder: '',
}

function fromStatus(effective: Record<string, unknown>): PaymentForm {
  const s = (k: string) => (typeof effective[k] === 'string' ? (effective[k] as string) : '')
  const b = (k: string) => effective[k] === true
  return {
    midtrans_enabled: b('midtrans_enabled'),
    midtrans_server_key: '', // write-only
    midtrans_client_key: s('midtrans_client_key'),
    midtrans_is_production: b('midtrans_is_production'),
    manual_bank_name: s('manual_bank_name'),
    manual_account_number: s('manual_account_number'),
    manual_account_holder: s('manual_account_holder'),
  }
}

/**
 * Payment gateway configuration.
 *
 * Midtrans credentials and the manual bank-transfer instructions are resolved at
 * runtime with DB-over-env precedence, so saving here applies without a
 * redeploy. Secret keys are write-only: an empty field leaves the stored
 * (encrypted) value untouched.
 */
export function PaymentSettingsTab({ isLoading, onSave, isSaving }: PaymentSettingsTabProps) {
  const { data: status, isLoading: statusLoading } = useConfigStatus()
  const [edits, setEdits] = React.useState<Partial<PaymentForm>>({})
  const [touched, setTouched] = React.useState(false)

  const baseline = status ? fromStatus(status.effective) : EMPTY
  const form: PaymentForm = { ...baseline, ...edits }
  const patch = (next: Partial<PaymentForm>) => {
    setEdits((prev) => ({ ...prev, ...next }))
    setTouched(true)
  }

  const handleSave = async () => {
    const payload: Record<string, unknown> = {
      midtrans_enabled: form.midtrans_enabled,
      midtrans_client_key: form.midtrans_client_key.trim(),
      midtrans_is_production: form.midtrans_is_production,
      manual_bank_name: form.manual_bank_name.trim(),
      manual_account_number: form.manual_account_number.trim(),
      manual_account_holder: form.manual_account_holder.trim(),
    }
    // Only send secrets the operator actually typed (empty = leave unchanged).
    if (form.midtrans_server_key.trim() !== '') payload.midtrans_server_key = form.midtrans_server_key
    await onSave(payload)
    setEdits({})
    setTouched(false)
  }

  if (isLoading || statusLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-12" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <CreditCard className="size-4" /> Midtrans
          </CardTitle>
          <CardDescription>
            Gateway pembayaran online. Kredensial di sini menimpa default environment.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <input
              id="midtrans_enabled"
              type="checkbox"
              checked={form.midtrans_enabled}
              onChange={(e) => patch({ midtrans_enabled: e.target.checked })}
              className="size-4"
            />
            <Label htmlFor="midtrans_enabled" className="text-sm font-bold">
              Aktifkan Midtrans
            </Label>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <SecretField
              id="midtrans_server_key"
              label="Server Key"
              value={form.midtrans_server_key}
              onChange={(v) => patch({ midtrans_server_key: v })}
              masked={status?.secrets?.midtrans_server_key}
              hasSecret={status?.has_secret?.midtrans_server_key}
              source={status?.source?.midtrans_server_key}
            />
            <div className="space-y-1.5">
              <Label htmlFor="midtrans_client_key">Client Key (publik)</Label>
              <Input
                id="midtrans_client_key"
                value={form.midtrans_client_key}
                onChange={(e) => patch({ midtrans_client_key: e.target.value })}
              />
            </div>
          </div>

          <div className="flex items-center gap-3">
            <input
              id="midtrans_is_production"
              type="checkbox"
              checked={form.midtrans_is_production}
              onChange={(e) => patch({ midtrans_is_production: e.target.checked })}
              className="size-4"
            />
            <Label htmlFor="midtrans_is_production" className="text-sm font-bold">
              Mode Produksi (nonaktif = sandbox)
            </Label>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Landmark className="size-4" /> Transfer Manual
          </CardTitle>
          <CardDescription>Instruksi rekening yang ditampilkan ke pengguna saat checkout manual.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label htmlFor="manual_bank_name">Nama Bank</Label>
              <Input
                id="manual_bank_name"
                value={form.manual_bank_name}
                placeholder="BCA"
                onChange={(e) => patch({ manual_bank_name: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="manual_account_number">Nomor Rekening</Label>
              <Input
                id="manual_account_number"
                value={form.manual_account_number}
                placeholder="1234567890"
                onChange={(e) => patch({ manual_account_number: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="manual_account_holder">Nama Pemilik</Label>
              <Input
                id="manual_account_holder"
                value={form.manual_account_holder}
                placeholder="PT FinTrack"
                onChange={(e) => patch({ manual_account_holder: e.target.value })}
              />
            </div>
          </div>
          <p className="text-xs font-bold text-muted-foreground">
            Rekening tunggal ini dipakai sebagai cadangan. Untuk beberapa tujuan (bank, e-wallet, QRIS), kelola
            daftar akun di bawah.
          </p>
        </CardContent>
      </Card>

      {/* Multiple payment destinations (bank / e-wallet / QRIS). */}
      <PaymentAccountsManager />

      {/* PPN toggle + invoice expiry window. */}
      <BillingSettingsCard />

      <Button onClick={handleSave} disabled={isSaving || !touched}>
        {isSaving ? 'Menyimpan...' : 'Simpan Konfigurasi Pembayaran'}
      </Button>
    </div>
  )
}
