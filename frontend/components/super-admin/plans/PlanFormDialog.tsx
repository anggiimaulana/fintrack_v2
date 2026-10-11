'use client'

import { useState, useId } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { RupiahInput } from '@/components/shared/RupiahInput'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { PlanDTO, PlanPayload } from '@/hooks/useSuperAdminPlans'

interface EntitlementEntry {
  id: string
  key: string
  enabled: boolean
}

interface LimitEntry {
  id: string
  key: string
  value: number
}

interface PlanFormDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  plan?: PlanDTO | null
  onSubmit: (payload: PlanPayload) => Promise<void>
  isSubmitting?: boolean
  codeConflictError?: string | null
  onClearCodeConflict?: () => void
}

export function PlanFormDialog({
  open,
  onOpenChange,
  plan,
  onSubmit,
  isSubmitting = false,
  codeConflictError = null,
  onClearCodeConflict,
}: PlanFormDialogProps) {
  const isEdit = Boolean(plan)

  // Basic fields state
  const [code, setCode] = useState(plan?.code ?? '')
  const [name, setName] = useState(plan?.name ?? '')
  const [description, setDescription] = useState(plan?.description ?? '')
  const [priceMonthly, setPriceMonthly] = useState<number | undefined>(plan?.price_monthly_idr ?? 0)
  const [priceYearly, setPriceYearly] = useState<number | undefined>(plan?.price_yearly_idr ?? 0)
  const [billingPeriodDays, setBillingPeriodDays] = useState<number>(plan?.billing_period_days ?? 30)
  const [isActive, setIsActive] = useState<boolean>(plan?.is_active ?? true)
  const [isPublic, setIsPublic] = useState<boolean>(plan?.is_public ?? true)
  const [fallbackGateway, setFallbackGateway] = useState<string>(plan?.fallback_gateway ?? 'midtrans')
  const [sortOrder, setSortOrder] = useState<number>(plan?.sort_order ?? 0)

  // Entitlements list
  const [entitlements, setEntitlements] = useState<EntitlementEntry[]>(() => {
    if (!plan?.entitlements) return []
    return Object.entries(plan.entitlements).map(([k, v], idx) => ({
      id: `${k}-${idx}`,
      key: k,
      enabled: v,
    }))
  })

  // Limits list
  const [limits, setLimits] = useState<LimitEntry[]>(() => {
    if (!plan?.limits) return []
    return Object.entries(plan.limits).map(([k, v], idx) => ({
      id: `${k}-${idx}`,
      key: k,
      value: v,
    }))
  })

  // Validation state
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  const codeInputId = useId()
  const nameInputId = useId()
  const descInputId = useId()
  const periodInputId = useId()
  const sortInputId = useId()
  const gatewayInputId = useId()

  const handleAddEntitlement = () => {
    setEntitlements((prev) => [
      ...prev,
      { id: `ent-${Date.now()}-${Math.random()}`, key: '', enabled: true },
    ])
  }

  const handleRemoveEntitlement = (id: string) => {
    setEntitlements((prev) => prev.filter((item) => item.id !== id))
  }

  const handleUpdateEntitlementKey = (id: string, newKey: string) => {
    setEntitlements((prev) =>
      prev.map((item) => (item.id === id ? { ...item, key: newKey } : item))
    )
  }

  const handleUpdateEntitlementValue = (id: string, enabled: boolean) => {
    setEntitlements((prev) =>
      prev.map((item) => (item.id === id ? { ...item, enabled } : item))
    )
  }

  const handleAddLimit = () => {
    setLimits((prev) => [
      ...prev,
      { id: `lim-${Date.now()}-${Math.random()}`, key: '', value: 0 },
    ])
  }

  const handleRemoveLimit = (id: string) => {
    setLimits((prev) => prev.filter((item) => item.id !== id))
  }

  const handleUpdateLimitKey = (id: string, newKey: string) => {
    setLimits((prev) =>
      prev.map((item) => (item.id === id ? { ...item, key: newKey } : item))
    )
  }

  const handleUpdateLimitValue = (id: string, val: number) => {
    setLimits((prev) =>
      prev.map((item) => (item.id === id ? { ...item, value: val } : item))
    )
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    const errors: Record<string, string> = {}
    const trimmedCode = code.trim().toLowerCase()
    const trimmedName = name.trim()

    if (!trimmedCode) {
      errors.code = 'Kode paket wajib diisi'
    }
    if (!trimmedName) {
      errors.name = 'Nama paket wajib diisi'
    }
    if ((priceMonthly ?? 0) < 0) {
      errors.price_monthly = 'Harga bulanan tidak boleh negatif'
    }
    if ((priceYearly ?? 0) < 0) {
      errors.price_yearly = 'Harga tahunan tidak boleh negatif'
    }
    if (billingPeriodDays <= 0) {
      errors.billing_period_days = 'Periode tagihan minimal 1 hari'
    }

    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors)
      return
    }

    setValidationErrors({})

    // Pack entitlements object
    const entitlementsObj: Record<string, boolean> = {}
    for (const item of entitlements) {
      const k = item.key.trim()
      if (k) {
        entitlementsObj[k] = item.enabled
      }
    }

    // Pack limits object
    const limitsObj: Record<string, number> = {}
    for (const item of limits) {
      const k = item.key.trim()
      if (k) {
        limitsObj[k] = Math.max(0, item.value || 0)
      }
    }

    const payload: PlanPayload = {
      code: trimmedCode,
      name: trimmedName,
      description: description.trim() || null,
      price_monthly_idr: priceMonthly ?? 0,
      price_yearly_idr: priceYearly ?? 0,
      billing_period_days: billingPeriodDays,
      is_active: isActive,
      is_public: isPublic,
      fallback_gateway: fallbackGateway.trim() || 'midtrans',
      sort_order: sortOrder,
      entitlements: entitlementsObj,
      limits: limitsObj,
    }

    await onSubmit(payload)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? `Edit Paket: ${plan?.name}` : 'Buat Paket Baru'}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Perbarui konfigurasi harga, fitur, dan limit paket langganan.'
              : 'Tambahkan paket langganan baru ke dalam sistem FinTrack.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Identitas Paket */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor={codeInputId}>
                Kode Paket <span className="text-destructive">*</span>
              </Label>
              <Input
                id={codeInputId}
                value={code}
                onChange={(e) => {
                  setCode(e.target.value)
                  if (validationErrors.code) {
                    setValidationErrors((prev) => ({ ...prev, code: '' }))
                  }
                  onClearCodeConflict?.()
                }}
                placeholder="contoh: pro, enterprise"
                disabled={isSubmitting}
                className={
                  validationErrors.code || codeConflictError
                    ? 'border-destructive focus-visible:ring-destructive'
                    : ''
                }
              />
              {validationErrors.code && (
                <p className="text-xs font-semibold text-destructive">
                  {validationErrors.code}
                </p>
              )}
              {codeConflictError && (
                <p className="text-xs font-semibold text-destructive">
                  {codeConflictError}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={nameInputId}>
                Nama Paket <span className="text-destructive">*</span>
              </Label>
              <Input
                id={nameInputId}
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  if (validationErrors.name) {
                    setValidationErrors((prev) => ({ ...prev, name: '' }))
                  }
                }}
                placeholder="contoh: Pro, Enterprise"
                disabled={isSubmitting}
                className={
                  validationErrors.name
                    ? 'border-destructive focus-visible:ring-destructive'
                    : ''
                }
              />
              {validationErrors.name && (
                <p className="text-xs font-semibold text-destructive">
                  {validationErrors.name}
                </p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor={descInputId}>Deskripsi</Label>
            <Input
              id={descInputId}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Deskripsi singkat paket..."
              disabled={isSubmitting}
            />
          </div>

          {/* Harga & Periode */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label>Harga Bulanan</Label>
              <RupiahInput
                value={priceMonthly}
                onChange={(val) => setPriceMonthly(val ?? 0)}
                placeholder="0"
                disabled={isSubmitting}
              />
              {validationErrors.price_monthly && (
                <p className="text-xs font-semibold text-destructive">
                  {validationErrors.price_monthly}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label>Harga Tahunan</Label>
              <RupiahInput
                value={priceYearly}
                onChange={(val) => setPriceYearly(val ?? 0)}
                placeholder="0"
                disabled={isSubmitting}
              />
              {validationErrors.price_yearly && (
                <p className="text-xs font-semibold text-destructive">
                  {validationErrors.price_yearly}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={periodInputId}>Periode (Hari)</Label>
              <Input
                id={periodInputId}
                type="number"
                min={1}
                value={billingPeriodDays}
                onChange={(e) => setBillingPeriodDays(parseInt(e.target.value, 10) || 30)}
                disabled={isSubmitting}
              />
              {validationErrors.billing_period_days && (
                <p className="text-xs font-semibold text-destructive">
                  {validationErrors.billing_period_days}
                </p>
              )}
            </div>
          </div>

          {/* Konfigurasi & Visibilitas */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor={gatewayInputId}>Fallback Gateway</Label>
              <Input
                id={gatewayInputId}
                value={fallbackGateway}
                onChange={(e) => setFallbackGateway(e.target.value)}
                placeholder="midtrans / manual"
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={sortInputId}>Urutan Tampilan (sort_order)</Label>
              <Input
                id={sortInputId}
                type="number"
                value={sortOrder}
                onChange={(e) => setSortOrder(parseInt(e.target.value, 10) || 0)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-6 p-4 rounded-md border border-border bg-card">
            <div className="flex items-center space-x-2">
              <Switch
                id="plan-is-active"
                checked={isActive}
                onCheckedChange={setIsActive}
                disabled={isSubmitting}
              />
              <Label htmlFor="plan-is-active" className="cursor-pointer font-medium">
                Aktif (is_active)
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="plan-is-public"
                checked={isPublic}
                onCheckedChange={setIsPublic}
                disabled={isSubmitting}
              />
              <Label htmlFor="plan-is-public" className="cursor-pointer font-medium">
                Publik (is_public)
              </Label>
            </div>
          </div>

          {/* Entitlements Editor */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm font-bold">Hak Akses Fitur (Entitlements)</Label>
                <p className="text-xs text-muted-foreground">
                  Fitur boolean yang aktif untuk pengguna dengan paket ini.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddEntitlement}
                disabled={isSubmitting}
              >
                <Plus className="h-4 w-4 mr-1" />
                Tambah Fitur
              </Button>
            </div>

            {entitlements.length === 0 ? (
              <p className="text-xs italic text-muted-foreground py-2">
                Belum ada hak akses khusus yang ditentukan.
              </p>
            ) : (
              <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                {entitlements.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center gap-3 p-2 rounded border border-border bg-card"
                  >
                    <Input
                      value={item.key}
                      onChange={(e) => handleUpdateEntitlementKey(item.id, e.target.value)}
                      placeholder="kunci fitur (contoh: ai_chat)"
                      className="flex-1 font-mono text-xs"
                      disabled={isSubmitting}
                    />
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={item.enabled}
                        onCheckedChange={(val) => handleUpdateEntitlementValue(item.id, val)}
                        disabled={isSubmitting}
                        aria-label={`Aktifkan fitur ${item.key || 'baru'}`}
                      />
                      <span className="text-xs font-mono text-muted-foreground w-12">
                        {item.enabled ? 'true' : 'false'}
                      </span>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveEntitlement(item.id)}
                      disabled={isSubmitting}
                      className="text-destructive hover:text-destructive"
                      aria-label={`Hapus fitur ${item.key || 'baru'}`}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Limits Editor */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div>
                <Label className="text-sm font-bold">Batas Penggunaan (Limits)</Label>
                <p className="text-xs text-muted-foreground">
                  Batas kuota numerik (integer) per periode tagihan.
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleAddLimit}
                disabled={isSubmitting}
              >
                <Plus className="h-4 w-4 mr-1" />
                Tambah Limit
              </Button>
            </div>

            {limits.length === 0 ? (
              <p className="text-xs italic text-muted-foreground py-2">
                Belum ada batasan numerik yang ditentukan.
              </p>
            ) : (
              <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                {limits.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center gap-3 p-2 rounded border border-border bg-card"
                  >
                    <Input
                      value={item.key}
                      onChange={(e) => handleUpdateLimitKey(item.id, e.target.value)}
                      placeholder="kunci batas (contoh: ai_chat_monthly)"
                      className="flex-1 font-mono text-xs"
                      disabled={isSubmitting}
                    />
                    <Input
                      type="number"
                      min={0}
                      value={item.value}
                      onChange={(e) =>
                        handleUpdateLimitValue(item.id, parseInt(e.target.value, 10) || 0)
                      }
                      placeholder="0"
                      className="w-28 font-mono text-xs"
                      disabled={isSubmitting}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveLimit(item.id)}
                      disabled={isSubmitting}
                      className="text-destructive hover:text-destructive"
                      aria-label={`Hapus limit ${item.key || 'baru'}`}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
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
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Menyimpan...' : isEdit ? 'Simpan Perubahan' : 'Buat Paket'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
