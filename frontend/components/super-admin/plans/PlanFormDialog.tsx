'use client'

import { useState, useId, useEffect, useMemo } from 'react'
import { Plus, Trash2, Infinity as InfinityIcon, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { RupiahInput } from '@/components/shared/RupiahInput'
import { NumberInput } from '@/components/shared/NumberInput'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { PlanDTO, PlanPayload } from '@/hooks/useSuperAdminPlans'
import { useEntitlementCatalog } from '@/hooks/useEntitlementCatalog'
import { usePlanCycles, type PlanLimitCycle } from '@/hooks/usePlanCycles'
import { CycleConfigMatrix } from './CycleConfigMatrix'

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

/** Predefined standard features with human-readable Indonesian labels and descriptions. */
const DEFAULT_SYSTEM_FEATURES: Array<{ key: string; label: string; description: string }> = [
  { key: 'ai_chat', label: 'Asisten AI (Chat)', description: 'Akses ke percakapan asisten keuangan berbasis AI.' },
  { key: 'ocr_receipt', label: 'Pindai Struk (OCR)', description: 'Ekstraksi transaksi otomatis dari foto bukti struk.' },
  { key: 'voice_input', label: 'Input Suara', description: 'Transkripsi perintah suara untuk mencatat transaksi.' },
  { key: 'rag_search', label: 'Pencarian Cerdas (RAG)', description: 'Pencarian semantik cerdas atas seluruh riwayat transaksi.' },
  { key: 'reports_export', label: 'Ekspor Laporan Keuangan', description: 'Unduh laporan berkala format PDF atau spreadsheet.' },
  { key: 'advanced_budgeting', label: 'Anggaran & Analisis Lanjutan', description: 'Fitur budgeting cerdas dan SPK evaluasi keuangan.' },
  { key: 'api_access', label: 'Akses API Pengembang', description: 'Token akses API untuk integrasi sistem pihak ketiga.' },
  { key: 'bank_sync', label: 'Sinkronisasi Rekening Bank', description: 'Koneksi otomatis mutasi rekening bank.' },
]

/** Predefined standard metrics with clean Indonesian labels and units. */
const DEFAULT_SYSTEM_METRICS: Array<{ key: string; label: string; unit: string; description: string }> = [
  { key: 'ai_tokens_per_cycle', label: 'Token AI per Siklus', unit: 'token', description: 'Batas kuota token AI per siklus penagihan (-1 = tak terbatas).' },
  { key: 'ai_requests_per_cycle', label: 'Permintaan AI per Siklus', unit: 'request', description: 'Batas pemanggilan request AI per siklus penagihan.' },
  { key: 'max_transactions', label: 'Maksimal Transaksi', unit: 'transaksi', description: 'Batas pencatatan transaksi pengguna.' },
  { key: 'ocr_scans_per_cycle', label: 'Pindai Struk OCR per Siklus', unit: 'scan', description: 'Batas pemindaian struk berbasis OCR.' },
  { key: 'voice_requests_per_day', label: 'Permintaan Suara per Hari', unit: 'request', description: 'Batas penggunaan perintah suara harian.' },
  { key: 'max_categories', label: 'Maksimal Kategori', unit: 'kategori', description: 'Batas jumlah kategori khusus pengguna.' },
  { key: 'max_wallets', label: 'Maksimal Dompet / Rekening', unit: 'dompet', description: 'Batas jumlah dompet/rekening pengguna.' },
]

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

  // Authoritative catalog from server
  const { catalog } = useEntitlementCatalog()
  const allFeatures = useMemo(() => {
    const record: Record<string, { key: string; label: string; description: string }> = {}
    for (const f of DEFAULT_SYSTEM_FEATURES) {
      record[f.key] = f
    }
    for (const f of catalog.features) {
      record[f.key] = {
        key: f.key,
        label: f.label || f.key,
        description: f.description || '',
      }
    }
    return Object.values(record)
  }, [catalog.features])

  // Combined metrics list
  const allMetrics = useMemo(() => {
    const record: Record<string, { key: string; label: string; unit: string; description: string; enforced?: boolean }> = {}
    for (const m of DEFAULT_SYSTEM_METRICS) {
      record[m.key] = { ...m, enforced: true }
    }
    for (const m of catalog.limits) {
      record[m.key] = {
        key: m.key,
        label: m.label || m.key,
        unit: m.unit || 'unit',
        description: m.description || '',
        enforced: m.enforced,
      }
    }
    return Object.values(record)
  }, [catalog.limits])

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
  const [tokenQuotaRatio, setTokenQuotaRatio] = useState<number>(plan?.token_quota_ratio ?? 1)

  // Entitlements state: map of featureKey -> boolean
  const [entitlementsMap, setEntitlementsMap] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {}
    if (plan?.entitlements) {
      for (const [k, v] of Object.entries(plan.entitlements)) {
        initial[k] = Boolean(v)
      }
    }
    return initial
  })

  // Custom features added manually (outside standard catalog)
  const [customFeatures, setCustomFeatures] = useState<string[]>(() => {
    if (!plan?.entitlements) return []
    const standardKeys: Record<string, true> = {}
    for (const f of DEFAULT_SYSTEM_FEATURES) {
      standardKeys[f.key] = true
    }
    return Object.keys(plan.entitlements).filter((k) => !standardKeys[k])
  })
  const [newCustomFeatureKey, setNewCustomFeatureKey] = useState('')
  const [showAddCustomFeature, setShowAddCustomFeature] = useState(false)


  // Limits list state
  const [limits, setLimits] = useState<LimitEntry[]>(() => {
    if (!plan?.limits) return []
    return Object.entries(plan.limits).map(([k, v], idx) => ({
      id: `${k}-${idx}`,
      key: k,
      value: v,
    }))
  })

  // Dual-cycle quota configuration
  const { cycles: serverCycles, saveCycles } = usePlanCycles(plan?.id)
  const [cycles, setCycles] = useState<PlanLimitCycle[]>([])

  useEffect(() => {
    if (serverCycles && serverCycles.length > 0) {
      setCycles(serverCycles)
    }
  }, [serverCycles])

  // Validation state
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({})

  const codeInputId = useId()
  const nameInputId = useId()
  const descInputId = useId()
  const periodInputId = useId()
  const sortInputId = useId()
  const ratioInputId = useId()
  const gatewayInputId = useId()

  // Handlers for features
  const handleToggleFeature = (key: string, enabled: boolean) => {
    setEntitlementsMap((prev) => ({ ...prev, [key]: enabled }))
  }

  const handleAddCustomFeature = () => {
    const trimmed = newCustomFeatureKey.trim().toLowerCase().replace(/\s+/g, '_')
    if (!trimmed) return
    if (!customFeatures.includes(trimmed)) {
      setCustomFeatures((prev) => [...prev, trimmed])
      setEntitlementsMap((prev) => ({ ...prev, [trimmed]: true }))
    }
    setNewCustomFeatureKey('')
    setShowAddCustomFeature(false)
  }

  const handleRemoveCustomFeature = (key: string) => {
    setCustomFeatures((prev) => prev.filter((k) => k !== key))
    setEntitlementsMap((prev) => {
      const next = { ...prev }
      delete next[key]
      return next
    })
  }

  // Handlers for limits
  const handleAddMetricLimit = (metricKey: string) => {
    if (!metricKey) return
    const existing = limits.find((l) => l.key === metricKey)
    if (existing) return

    setLimits((prev) => [
      ...prev,
      {
        id: `lim-${Date.now()}-${Math.random()}`,
        key: metricKey,
        value: metricKey.includes('token') ? 10000 : 1000,
      },
    ])
  }

  const handleRemoveLimit = (id: string) => {
    setLimits((prev) => prev.filter((item) => item.id !== id))
  }

  const handleUpdateLimitValue = (id: string, val: number) => {
    setLimits((prev) =>
      prev.map((item) => (item.id === id ? { ...item, value: val } : item))
    )
  }

  const handleToggleUnlimited = (id: string) => {
    setLimits((prev) =>
      prev.map((item) => {
        if (item.id !== id) return item
        return { ...item, value: item.value === -1 ? 1000 : -1 }
      })
    )
  }

  // Available metrics that haven't been added yet
  const availableMetricsToAdd = useMemo(() => {
    const configuredKeys: Record<string, true> = {}
    for (const l of limits) {
      configuredKeys[l.key] = true
    }
    return allMetrics.filter((m) => !configuredKeys[m.key])
  }, [limits, allMetrics])
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
    if (!Number.isFinite(tokenQuotaRatio) || tokenQuotaRatio < 1 || tokenQuotaRatio > 1000) {
      errors.token_quota_ratio = 'Rasio kuota harus antara 1 dan 1000'
    }

    if (Object.keys(errors).length > 0) {
      setValidationErrors(errors)
      return
    }

    setValidationErrors({})

    // Pack entitlements
    const entitlementsObj: Record<string, boolean> = {}
    for (const [k, v] of Object.entries(entitlementsMap)) {
      if (k.trim()) {
        entitlementsObj[k.trim()] = v
      }
    }

    // Pack limits object
    const limitsObj: Record<string, number> = {}
    for (const item of limits) {
      const k = item.key.trim()
      if (k) {
        const v = item.value ?? 0
        limitsObj[k] = v < 0 ? -1 : v
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
      token_quota_ratio: tokenQuotaRatio,
      entitlements: entitlementsObj,
      limits: limitsObj,
    }

    await onSubmit(payload)
    if (isEdit && plan?.id && cycles.length > 0) {
      try {
        const sanitizedCycles: PlanLimitCycle[] = cycles.map((c) => {
          if (c.cycle_type === "weekly_monthly") {
            const isToken = c.limit_key.includes("token");
            const degraded = c.degraded_mode || (isToken ? "model_downgrade" : "hard_block");
            return {
              ...c,
              weekly_limit:
                c.weekly_limit && c.weekly_limit > 0
                  ? c.weekly_limit
                  : isToken
                  ? 50000
                  : c.limit_key.includes("scan")
                  ? 25
                  : 500,
              degraded_mode: degraded,
              fallback_model: degraded === "model_downgrade" ? c.fallback_model || "gpt-4o-mini" : null,
            };
          }
          return {
            ...c,
            weekly_limit: null,
            degraded_mode: null,
            fallback_model: null,
            max_degraded_usage: null,
          };
        });
        await saveCycles(sanitizedCycles)
      } catch (err) {
        console.error('Failed to save cycles:', err)
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[95vw] sm:max-w-3xl md:max-w-4xl max-h-[90vh] overflow-y-auto no-scrollbar p-6 sm:p-8 border-2 border-foreground shadow-[6px_6px_0px_0px] shadow-foreground rounded-lg bg-card">
        <DialogHeader>
          <DialogTitle className="text-xl font-black uppercase tracking-wide">
            {isEdit ? `Edit Paket: ${plan?.name}` : 'Buat Paket Baru'}
          </DialogTitle>
          <DialogDescription className="text-xs font-bold text-muted-foreground">
            {isEdit
              ? 'Perbarui konfigurasi harga, fitur, batasan metrik, dan status paket langganan.'
              : 'Tambahkan paket langganan baru dengan hak akses dan batas metrik siap pakai.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 pt-2">
          {/* Identitas Paket */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor={codeInputId} className="text-xs font-black uppercase tracking-wide">
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
                <p className="text-xs font-bold text-destructive">
                  {validationErrors.code}
                </p>
              )}
              {codeConflictError && (
                <p className="text-xs font-bold text-destructive">
                  {codeConflictError}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={nameInputId} className="text-xs font-black uppercase tracking-wide">
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
                <p className="text-xs font-bold text-destructive">
                  {validationErrors.name}
                </p>
              )}
            </div>
          </div>

          {/* Deskripsi Paket Menggunakan Textarea */}
          <div className="space-y-1.5">
            <Label htmlFor={descInputId} className="text-xs font-black uppercase tracking-wide">
              Deskripsi Paket
            </Label>
            <Textarea
              id={descInputId}
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Deskripsi singkat mengenai paket langganan dan target pengguna..."
              disabled={isSubmitting}
            />
          </div>

          {/* Harga & Periode */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label className="text-xs font-black uppercase tracking-wide">Harga Bulanan</Label>
              <RupiahInput
                value={priceMonthly}
                onChange={(val) => setPriceMonthly(val ?? 0)}
                placeholder="0"
                disabled={isSubmitting}
              />
              {validationErrors.price_monthly && (
                <p className="text-xs font-bold text-destructive">
                  {validationErrors.price_monthly}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label className="text-xs font-black uppercase tracking-wide">Harga Tahunan</Label>
              <RupiahInput
                value={priceYearly}
                onChange={(val) => setPriceYearly(val ?? 0)}
                placeholder="0"
                disabled={isSubmitting}
              />
              {validationErrors.price_yearly && (
                <p className="text-xs font-bold text-destructive">
                  {validationErrors.price_yearly}
                </p>
              )}
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={periodInputId} className="text-xs font-black uppercase tracking-wide">
                Periode (Hari)
              </Label>
              <NumberInput
                id={periodInputId}
                min={1}
                value={billingPeriodDays}
                onChange={(val) => setBillingPeriodDays(val || 30)}
                disabled={isSubmitting}
                suffix="Hari"
              />
              {validationErrors.billing_period_days && (
                <p className="text-xs font-bold text-destructive">
                  {validationErrors.billing_period_days}
                </p>
              )}
            </div>
          </div>

          {/* Konfigurasi Gateway, Urutan, dan Rasio Kuota Token */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor={gatewayInputId} className="text-xs font-black uppercase tracking-wide">
                Fallback Gateway
              </Label>
              <Select
                value={fallbackGateway}
                onValueChange={setFallbackGateway}
                disabled={isSubmitting}
              >
                <SelectTrigger id={gatewayInputId} className="w-full">
                  <SelectValue placeholder="Pilih Gateway" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="midtrans">Midtrans (Payment Gateway)</SelectItem>
                  <SelectItem value="manual">Transfer Manual (Bukti Transfer)</SelectItem>
                  <SelectItem value="xendit">Xendit</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={sortInputId} className="text-xs font-black uppercase tracking-wide">
                Urutan Tampilan
              </Label>
              <NumberInput
                id={sortInputId}
                value={sortOrder}
                onChange={(val) => setSortOrder(val ?? 0)}
                placeholder="0"
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor={ratioInputId} className="text-xs font-black uppercase tracking-wide">
                Rasio Kuota Token
              </Label>
              <NumberInput
                id={ratioInputId}
                min={1}
                max={1000}
                value={tokenQuotaRatio}
                onChange={(val) => setTokenQuotaRatio(val && val >= 1 ? val : 1)}
                disabled={isSubmitting}
              />
            </div>

            <div className="sm:col-span-3">
              <p className="text-xs text-muted-foreground font-medium">
                Rasio pembagi token AI sebelum memotong kuota paket (1 = 1 token per 1 kuota. Contoh: rasio 5 membuat 10.000 token hanya memotong 2.000 kuota).
              </p>
              {validationErrors.token_quota_ratio && (
                <p className="text-xs font-bold text-destructive" role="alert">
                  {validationErrors.token_quota_ratio}
                </p>
              )}
            </div>
          </div>

          {/* Status Aktif & Publik */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 p-4 rounded-md border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground">
            <div className="flex items-center justify-between gap-3">
              <div className="space-y-0.5">
                <Label htmlFor="plan-is-active" className="cursor-pointer text-sm font-bold">
                  Status Aktif
                </Label>
                <p className="text-xs text-muted-foreground">
                  Paket dapat digunakan dan dipilih oleh pengguna.
                </p>
              </div>
              <Switch
                id="plan-is-active"
                checked={isActive}
                onCheckedChange={setIsActive}
                disabled={isSubmitting}
              />
            </div>

            <div className="flex items-center justify-between gap-3">
              <div className="space-y-0.5">
                <Label htmlFor="plan-is-public" className="cursor-pointer text-sm font-bold">
                  Tampilkan ke Publik
                </Label>
                <p className="text-xs text-muted-foreground">
                  Tampil di halaman langganan dan pilihan upgrade pengguna.
                </p>
              </div>
              <Switch
                id="plan-is-public"
                checked={isPublic}
                onCheckedChange={setIsPublic}
                disabled={isSubmitting}
              />
            </div>
          </div>

          {/* Hak Akses & Batas Penggunaan */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 pt-2">
            {/* Hak Akses Fitur (Entitlements) - Siap Sedia Checklist */}
            <div className="space-y-3 p-4 rounded-md border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground flex flex-col">
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-sm font-black uppercase tracking-wide">
                    Hak Akses Fitur (Entitlements)
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    Pilih fitur siap pakai yang aktif untuk paket ini.
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => setShowAddCustomFeature(!showAddCustomFeature)}
                  disabled={isSubmitting}
                  className="font-bold"
                >
                  <Plus className="h-3.5 w-3.5 mr-1" />
                  Kustom
                </Button>
              </div>

              {/* Form Tambah Fitur Kustom */}
              {showAddCustomFeature && (
                <div className="flex items-center gap-2 p-2 border-2 rounded-md border-foreground bg-muted/30">
                  <Input
                    value={newCustomFeatureKey}
                    onChange={(e) => setNewCustomFeatureKey(e.target.value)}
                    placeholder="nama_kunci_fitur_baru"
                    className="h-8 text-xs font-mono"
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        handleAddCustomFeature()
                      }
                    }}
                  />
                  <Button
                    type="button"
                    size="xs"
                    onClick={handleAddCustomFeature}
                    disabled={!newCustomFeatureKey.trim()}
                    className="font-bold shrink-0"
                  >
                    Tambah
                  </Button>
                </div>
              )}

              {/* Checklist Fitur Siap Sedia */}
              <div className="space-y-2 max-h-72 overflow-y-auto no-scrollbar pr-1">
                {allFeatures.map((item) => {
                  const isChecked = Boolean(entitlementsMap[item.key])
                  return (
                    <div
                      key={item.key}
                      className="flex items-start justify-between gap-3 p-2.5 rounded-md border border-border bg-background hover:bg-muted/30 transition-colors"
                    >
                      <div className="space-y-0.5 min-w-0 pr-1">
                        <Label
                          htmlFor={`feat-${item.key}`}
                          className="text-xs font-bold cursor-pointer text-foreground block truncate"
                        >
                          {item.label}
                        </Label>
                        {item.description && (
                          <p className="text-[11px] text-muted-foreground line-clamp-2">
                            {item.description}
                          </p>
                        )}
                      </div>
                      <Switch
                        id={`feat-${item.key}`}
                        checked={isChecked}
                        onCheckedChange={(val) => handleToggleFeature(item.key, val)}
                        disabled={isSubmitting}
                        aria-label={`Aktifkan fitur ${item.label}`}
                      />
                    </div>
                  )
                })}

                {/* Fitur Kustom */}
                {customFeatures.map((key) => {
                  const isChecked = Boolean(entitlementsMap[key])
                  return (
                    <div
                      key={key}
                      className="flex items-start justify-between gap-3 p-2.5 rounded-md border-2 border-dashed border-border bg-background"
                    >
                      <div className="space-y-0.5 min-w-0 pr-1">
                        <div className="flex items-center gap-1.5">
                          <Label
                            htmlFor={`feat-${key}`}
                            className="text-xs font-mono font-bold cursor-pointer text-foreground block truncate"
                          >
                            {key}
                          </Label>
                          <Badge variant="outline" className="text-[9px] uppercase px-1 py-0">
                            Kustom
                          </Badge>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <Switch
                          id={`feat-${key}`}
                          checked={isChecked}
                          onCheckedChange={(val) => handleToggleFeature(key, val)}
                          disabled={isSubmitting}
                        />
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-xs"
                          onClick={() => handleRemoveCustomFeature(key)}
                          className="text-destructive hover:text-destructive"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Batas Penggunaan (Limits) - Dropdown Pilihan Metrik Siap Sedia */}
            <div className="space-y-3 p-4 rounded-md border-2 border-foreground bg-card shadow-[2px_2px_0px_0px] shadow-foreground flex flex-col">
              <div className="space-y-1">
                <Label className="text-sm font-black uppercase tracking-wide">
                  Batas Penggunaan (Limits)
                </Label>
                <p className="text-xs text-muted-foreground">
                  Pilih metrik sistem dari dropdown untuk menetapkan kuota (-1 = tak terbatas).
                </p>
              </div>

              {/* Dropdown Pemilihan Metrik Siap Sedia */}
              {availableMetricsToAdd.length > 0 && (
                <div className="pt-1">
                  <Select
                    key={availableMetricsToAdd.length}
                    onValueChange={handleAddMetricLimit}
                    disabled={isSubmitting}
                  >
                    <SelectTrigger className="w-full text-xs font-bold border-2 border-foreground">
                      <SelectValue placeholder="+ Pilih Metrik untuk Dibatasi..." />
                    </SelectTrigger>
                    <SelectContent>
                      {availableMetricsToAdd.map((m) => (
                        <SelectItem key={m.key} value={m.key} className="text-xs font-medium">
                          {m.label} ({m.unit})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}

              {limits.length === 0 ? (
                <div className="p-4 rounded-md border border-dashed border-border text-center space-y-1 my-auto">
                  <p className="text-xs font-bold text-muted-foreground">
                    Belum ada batasan metrik khusus.
                  </p>
                  <p className="text-[11px] text-muted-foreground">
                    Gunakan dropdown di atas untuk memilih metrik yang ingin dibatasi (misal Token AI, Transaksi, Scan OCR).
                  </p>
                </div>
              ) : (
                <div className="space-y-2 max-h-72 overflow-y-auto no-scrollbar pr-1">
                  {limits.map((item) => {
                    const metricMeta = allMetrics.find((m) => m.key === item.key)
                    const label = metricMeta?.label || item.key
                    const unit = metricMeta?.unit || 'unit'
                    const isUnlimited = item.value === -1

                    return (
                      <div
                        key={item.id}
                        className="p-2.5 rounded-md border border-border bg-background space-y-2"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span className="text-xs font-bold text-foreground truncate">
                              {label}
                            </span>
                            <Badge variant="outline" className="text-[10px] font-mono px-1 py-0 uppercase shrink-0">
                              {unit}
                            </Badge>
                          </div>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon-xs"
                            onClick={() => handleRemoveLimit(item.id)}
                            disabled={isSubmitting}
                            className="text-destructive hover:text-destructive shrink-0"
                            aria-label={`Hapus batas ${label}`}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>

                        <div className="flex items-center gap-2">
                          <NumberInput
                            allowNegative
                            value={item.value}
                            onChange={(val) => handleUpdateLimitValue(item.id, val ?? 0)}
                            placeholder="0"
                            className="h-8 text-xs font-mono flex-1 border-2 border-foreground"
                            disabled={isSubmitting || isUnlimited}
                            aria-label={`Nilai batas ${label}`}
                          />
                          <Button
                            type="button"
                            variant={isUnlimited ? 'default' : 'outline'}
                            size="xs"
                            onClick={() => handleToggleUnlimited(item.id)}
                            disabled={isSubmitting}
                            className="shrink-0 text-xs font-bold gap-1"
                            title="Set batas menjadi tak terbatas (-1)"
                          >
                            <InfinityIcon className="h-3.5 w-3.5" />
                            {isUnlimited ? 'Unlimited' : 'Tak Terbatas'}
                          </Button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>

          {/* Siklus & Kuota Mingguan jika Edit Plan */}
          {isEdit && plan?.id && (
            <div className="space-y-3 pt-4 border-t-2 border-border">
              <div className="flex items-center justify-between">
                <div>
                  <Label className="text-base font-black uppercase tracking-wide">
                    Siklus &amp; Kuota Mingguan
                  </Label>
                  <p className="text-xs text-muted-foreground font-medium">
                    Konfigurasi reset kuota dwi-siklus dan mode pembatasan (degraded mode) per metrik.
                  </p>
                </div>
              </div>
              <CycleConfigMatrix
                limits={catalog.limits.filter((l) => l.enforced)}
                cycles={cycles}
                onChange={setCycles}
              />
            </div>
          )}

          {/* Dialog Footer Actions */}
          <DialogFooter className="gap-2 sm:gap-0 pt-4 border-t-2 border-border">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
              className="font-bold"
            >
              Batal
            </Button>
            <Button
              type="submit"
              disabled={
                !code.trim() ||
                !name.trim() ||
                billingPeriodDays <= 0 ||
                (priceMonthly ?? 0) < 0 ||
                (priceYearly ?? 0) < 0 ||
                isSubmitting
              }
              className="disabled:opacity-50 disabled:cursor-not-allowed font-black"
            >
              {isSubmitting ? 'Menyimpan...' : isEdit ? 'Simpan Perubahan' : 'Buat Paket'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
