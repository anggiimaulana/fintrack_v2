'use client'

import * as React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Lock, AlertCircle, Save, RotateCcw } from 'lucide-react'
import { SettingRow } from '@/hooks/useSuperAdminSettings'
import type { StorageStatusInfo } from '@/hooks/useSuperAdminStorage'
import { SettingsConfirmDialog, SettingDiff } from './SettingsConfirmDialog'
import { formatTanggal } from '@/lib/format'

interface GlobalSettingsTabProps {
  settings: SettingRow[]
  isLoading: boolean
  error: unknown
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
  onRefresh: () => void
  /** Current proof-storage driver status (from GET /admin/settings/storage). */
  storageStatus?: StorageStatusInfo | null
}

interface FormState {
  payment_gateway_default: string
  grace_period_hours: string
  tax_rate_percent: string
  invoice_due_hours: string
  storage_driver: string
  ai_estimate_chat: string
  ai_estimate_reasoning: string
  ai_estimate_vision: string
  ai_estimate_voice: string
  ai_estimate_embedding: string
  dunning_hours_urgent: string
  dunning_hours_soon: string
  breaker_failure_threshold: string
  breaker_cooldown_seconds: string
  breaker_half_open_probes: string
  route_breaker_threshold: string
  route_breaker_cooldown_secs: string
  metering_chars_per_token: string
  metering_safety_buffer_pct: string
  metering_image_tokens: string
  metering_audio_tokens_per_sec: string
}

/** The AI reserve-estimate keys, edited as a group (P3). */
const ESTIMATE_KEYS = [
  'ai_estimate_chat',
  'ai_estimate_reasoning',
  'ai_estimate_vision',
  'ai_estimate_voice',
  'ai_estimate_embedding',
] as const

/** Dunning reminder offsets (hours before due). */
const DUNNING_KEYS = ['dunning_hours_urgent', 'dunning_hours_soon'] as const

/** AI gateway breaker tuning keys. */
const BREAKER_KEYS = [
  'breaker_failure_threshold',
  'breaker_cooldown_seconds',
  'breaker_half_open_probes',
  'route_breaker_threshold',
  'route_breaker_cooldown_secs',
] as const

/** Token metering heuristics estimation keys. */
const METERING_KEYS = [
  'metering_chars_per_token',
  'metering_safety_buffer_pct',
  'metering_image_tokens',
  'metering_audio_tokens_per_sec',
] as const

interface ValidationErrors {
  payment_gateway_default?: string
  grace_period_hours?: string
  tax_rate_percent?: string
  invoice_due_hours?: string
  storage_driver?: string
  ai_estimate_chat?: string
  ai_estimate_reasoning?: string
  ai_estimate_vision?: string
  ai_estimate_voice?: string
  ai_estimate_embedding?: string
  dunning_hours_urgent?: string
  dunning_hours_soon?: string
  breaker_failure_threshold?: string
  breaker_cooldown_seconds?: string
  breaker_half_open_probes?: string
  route_breaker_threshold?: string
  route_breaker_cooldown_secs?: string
  metering_chars_per_token?: string
  metering_safety_buffer_pct?: string
  metering_image_tokens?: string
  metering_audio_tokens_per_sec?: string
}

const SETTING_LABELS: Record<string, string> = {
  payment_gateway_default: 'Payment Gateway Default',
  grace_period_hours: 'Grace Period (Jam)',
  tax_rate_percent: 'Tarif Pajak PPN (%)',
  invoice_due_hours: 'Jatuh Tempo Invoice (Jam)',
  storage_driver: 'Penyimpanan Bukti Transfer',
  ai_estimate_chat: 'Estimasi Token: Chat',
  ai_estimate_reasoning: 'Estimasi Token: Reasoning',
  ai_estimate_vision: 'Estimasi Token: Vision (OCR)',
  ai_estimate_voice: 'Estimasi Token: Suara',
  ai_estimate_embedding: 'Estimasi Token: Embedding',
  dunning_hours_urgent: 'Dunning: Urgent (Jam)',
  dunning_hours_soon: 'Dunning: Segera (Jam)',
  breaker_failure_threshold: 'Breaker: Ambang Kegagalan',
  breaker_cooldown_seconds: 'Breaker: Cooldown (Detik)',
  breaker_half_open_probes: 'Breaker: Probe Half-Open',
  route_breaker_threshold: 'Breaker Rute: Ambang Kegagalan',
  route_breaker_cooldown_secs: 'Breaker Rute: Cooldown (Detik)',
  metering_chars_per_token: 'Heuristik: Karakter per Token',
  metering_safety_buffer_pct: 'Heuristik: Buffer Pengaman (%)',
  metering_image_tokens: 'Heuristik: Token per Gambar (OCR)',
  metering_audio_tokens_per_sec: 'Heuristik: Token per Detik Suara',
}

const KNOWN_KEYS: Record<string, true> = {
  payment_gateway_default: true,
  grace_period_hours: true,
  tax_rate_percent: true,
  invoice_due_hours: true,
  storage_driver: true,
  ai_estimate_chat: true,
  ai_estimate_reasoning: true,
  ai_estimate_vision: true,
  ai_estimate_voice: true,
  ai_estimate_embedding: true,
  dunning_hours_urgent: true,
  dunning_hours_soon: true,
  breaker_failure_threshold: true,
  breaker_cooldown_seconds: true,
  breaker_half_open_probes: true,
  route_breaker_threshold: true,
  route_breaker_cooldown_secs: true,
  metering_chars_per_token: true,
  metering_safety_buffer_pct: true,
  metering_image_tokens: true,
  metering_audio_tokens_per_sec: true,
}

/** Default reserve estimates, mirroring backend DefaultQuotaEstimate. */
const DEFAULT_ESTIMATES: Pick<
  FormState,
  'ai_estimate_chat' | 'ai_estimate_reasoning' | 'ai_estimate_vision' | 'ai_estimate_voice' | 'ai_estimate_embedding'
> = {
  ai_estimate_chat: '2000',
  ai_estimate_reasoning: '4000',
  ai_estimate_vision: '5000',
  ai_estimate_voice: '2000',
  ai_estimate_embedding: '1000',
}

/** Defaults for dunning + breaker, mirroring the backend seed values. */
const DEFAULT_DUNNING_BREAKER: Pick<
  FormState,
  | 'dunning_hours_urgent'
  | 'dunning_hours_soon'
  | 'breaker_failure_threshold'
  | 'breaker_cooldown_seconds'
  | 'breaker_half_open_probes'
  | 'route_breaker_threshold'
  | 'route_breaker_cooldown_secs'
> = {
  dunning_hours_urgent: '24',
  dunning_hours_soon: '72',
  breaker_failure_threshold: '5',
  breaker_cooldown_seconds: '30',
  breaker_half_open_probes: '1',
  route_breaker_threshold: '2',
  route_breaker_cooldown_secs: '60',
}

/** Defaults for token metering heuristics estimation. */
const DEFAULT_METERING: Pick<
  FormState,
  | 'metering_chars_per_token'
  | 'metering_safety_buffer_pct'
  | 'metering_image_tokens'
  | 'metering_audio_tokens_per_sec'
> = {
  metering_chars_per_token: '4',
  metering_safety_buffer_pct: '10',
  metering_image_tokens: '85',
  metering_audio_tokens_per_sec: '25',
}

export function GlobalSettingsTab({
  settings,
  isLoading,
  error,
  onSave,
  isSaving,
  onRefresh,
  storageStatus,
}: GlobalSettingsTabProps) {
  const [initialForm, setInitialForm] = React.useState<FormState>({
    payment_gateway_default: 'manual',
    grace_period_hours: '72',
    tax_rate_percent: '11',
    invoice_due_hours: '24',
    storage_driver: 'local',
    ...DEFAULT_ESTIMATES,
    ...DEFAULT_DUNNING_BREAKER,
    ...DEFAULT_METERING,
  })

  const [form, setForm] = React.useState<FormState>(initialForm)
  const [clientErrors, setClientErrors] = React.useState<ValidationErrors>({})
  const [backendErrors, setBackendErrors] = React.useState<ValidationErrors>({})
  const [confirmOpen, setConfirmOpen] = React.useState(false)

  // Map settings array to lookup record
  const settingsMap = React.useMemo(() => {
    const map: Record<string, SettingRow> = {}
    for (const s of settings) {
      map[s.key] = s
    }
    return map
  }, [settings])

  // Hydrate the form once the server settings arrive (and again only if the
  // server values actually change) without a setState-in-effect.
  const serverSignature = React.useMemo(
    () =>
      JSON.stringify([
        settingsMap.payment_gateway_default?.value ?? null,
        settingsMap.grace_period_hours?.value ?? null,
        settingsMap.tax_rate_percent?.value ?? null,
        settingsMap.invoice_due_hours?.value ?? null,
        settingsMap.storage_driver?.value ?? null,
        settingsMap.ai_estimate_chat?.value ?? null,
        settingsMap.ai_estimate_reasoning?.value ?? null,
        settingsMap.ai_estimate_vision?.value ?? null,
        settingsMap.ai_estimate_voice?.value ?? null,
        settingsMap.ai_estimate_embedding?.value ?? null,
        settingsMap.dunning_hours_urgent?.value ?? null,
        settingsMap.dunning_hours_soon?.value ?? null,
        settingsMap.breaker_failure_threshold?.value ?? null,
        settingsMap.breaker_cooldown_seconds?.value ?? null,
        settingsMap.breaker_half_open_probes?.value ?? null,
        settingsMap.route_breaker_threshold?.value ?? null,
        settingsMap.route_breaker_cooldown_secs?.value ?? null,
        settingsMap.metering_chars_per_token?.value ?? null,
        settingsMap.metering_safety_buffer_pct?.value ?? null,
        settingsMap.metering_image_tokens?.value ?? null,
        settingsMap.metering_audio_tokens_per_sec?.value ?? null,
      ]),
    [settingsMap],
  )
  const [hydratedSignature, setHydratedSignature] = React.useState<string | null>(null)
  if (settings.length > 0 && hydratedSignature !== serverSignature) {
    setHydratedSignature(serverSignature)

    const gwRow = settingsMap.payment_gateway_default
    const graceRow = settingsMap.grace_period_hours
    const taxRow = settingsMap.tax_rate_percent
    const invRow = settingsMap.invoice_due_hours
    const storageRow = settingsMap.storage_driver

    const nextInit: FormState = {
      payment_gateway_default: typeof gwRow?.value === 'string' ? gwRow.value : 'manual',
      grace_period_hours: graceRow?.value != null ? String(graceRow.value) : '72',
      tax_rate_percent: taxRow?.value != null ? String(taxRow.value) : '11',
      invoice_due_hours: invRow?.value != null ? String(invRow.value) : '24',
      storage_driver: typeof storageRow?.value === 'string' ? storageRow.value : 'local',
      ai_estimate_chat: settingsMap.ai_estimate_chat?.value != null ? String(settingsMap.ai_estimate_chat.value) : DEFAULT_ESTIMATES.ai_estimate_chat,
      ai_estimate_reasoning: settingsMap.ai_estimate_reasoning?.value != null ? String(settingsMap.ai_estimate_reasoning.value) : DEFAULT_ESTIMATES.ai_estimate_reasoning,
      ai_estimate_vision: settingsMap.ai_estimate_vision?.value != null ? String(settingsMap.ai_estimate_vision.value) : DEFAULT_ESTIMATES.ai_estimate_vision,
      ai_estimate_voice: settingsMap.ai_estimate_voice?.value != null ? String(settingsMap.ai_estimate_voice.value) : DEFAULT_ESTIMATES.ai_estimate_voice,
      ai_estimate_embedding: settingsMap.ai_estimate_embedding?.value != null ? String(settingsMap.ai_estimate_embedding.value) : DEFAULT_ESTIMATES.ai_estimate_embedding,
      dunning_hours_urgent: settingsMap.dunning_hours_urgent?.value != null ? String(settingsMap.dunning_hours_urgent.value) : DEFAULT_DUNNING_BREAKER.dunning_hours_urgent,
      dunning_hours_soon: settingsMap.dunning_hours_soon?.value != null ? String(settingsMap.dunning_hours_soon.value) : DEFAULT_DUNNING_BREAKER.dunning_hours_soon,
      breaker_failure_threshold: settingsMap.breaker_failure_threshold?.value != null ? String(settingsMap.breaker_failure_threshold.value) : DEFAULT_DUNNING_BREAKER.breaker_failure_threshold,
      breaker_cooldown_seconds: settingsMap.breaker_cooldown_seconds?.value != null ? String(settingsMap.breaker_cooldown_seconds.value) : DEFAULT_DUNNING_BREAKER.breaker_cooldown_seconds,
      breaker_half_open_probes: settingsMap.breaker_half_open_probes?.value != null ? String(settingsMap.breaker_half_open_probes.value) : DEFAULT_DUNNING_BREAKER.breaker_half_open_probes,
      route_breaker_threshold: settingsMap.route_breaker_threshold?.value != null ? String(settingsMap.route_breaker_threshold.value) : DEFAULT_DUNNING_BREAKER.route_breaker_threshold,
      route_breaker_cooldown_secs: settingsMap.route_breaker_cooldown_secs?.value != null ? String(settingsMap.route_breaker_cooldown_secs.value) : DEFAULT_DUNNING_BREAKER.route_breaker_cooldown_secs,
      metering_chars_per_token: settingsMap.metering_chars_per_token?.value != null ? String(settingsMap.metering_chars_per_token.value) : DEFAULT_METERING.metering_chars_per_token,
      metering_safety_buffer_pct: settingsMap.metering_safety_buffer_pct?.value != null ? String(settingsMap.metering_safety_buffer_pct.value) : DEFAULT_METERING.metering_safety_buffer_pct,
      metering_image_tokens: settingsMap.metering_image_tokens?.value != null ? String(settingsMap.metering_image_tokens.value) : DEFAULT_METERING.metering_image_tokens,
      metering_audio_tokens_per_sec: settingsMap.metering_audio_tokens_per_sec?.value != null ? String(settingsMap.metering_audio_tokens_per_sec.value) : DEFAULT_METERING.metering_audio_tokens_per_sec,
    }
    setInitialForm(nextInit)
    setForm(nextInit)
    setClientErrors({})
    setBackendErrors({})
  }

  // Separate unknown settings
  const unknownSettings = React.useMemo(() => {
    return settings.filter((s) => !KNOWN_KEYS[s.key])
  }, [settings])

  // Compute dirty keys & diffs
  const diffs = React.useMemo<SettingDiff[]>(() => {
    const res: SettingDiff[] = []

    if (form.payment_gateway_default !== initialForm.payment_gateway_default) {
      res.push({
        key: 'payment_gateway_default',
        label: SETTING_LABELS.payment_gateway_default,
        before: initialForm.payment_gateway_default,
        after: form.payment_gateway_default,
      })
    }

    if (form.grace_period_hours !== initialForm.grace_period_hours) {
      res.push({
        key: 'grace_period_hours',
        label: SETTING_LABELS.grace_period_hours,
        before: initialForm.grace_period_hours,
        after: form.grace_period_hours,
      })
    }

    if (form.tax_rate_percent !== initialForm.tax_rate_percent) {
      res.push({
        key: 'tax_rate_percent',
        label: SETTING_LABELS.tax_rate_percent,
        before: initialForm.tax_rate_percent,
        after: form.tax_rate_percent,
      })
    }

    if (form.invoice_due_hours !== initialForm.invoice_due_hours) {
      res.push({
        key: 'invoice_due_hours',
        label: SETTING_LABELS.invoice_due_hours,
        before: initialForm.invoice_due_hours,
        after: form.invoice_due_hours,
      })
    }

    if (form.storage_driver !== initialForm.storage_driver) {
      res.push({
        key: 'storage_driver',
        label: SETTING_LABELS.storage_driver,
        before: initialForm.storage_driver,
        after: form.storage_driver,
      })
    }

    for (const key of ESTIMATE_KEYS) {
      if (form[key] !== initialForm[key]) {
        res.push({
          key,
          label: SETTING_LABELS[key],
          before: initialForm[key],
          after: form[key],
        })
      }
    }

    for (const key of [...DUNNING_KEYS, ...BREAKER_KEYS, ...METERING_KEYS]) {
      if (form[key] !== initialForm[key]) {
        res.push({
          key,
          label: SETTING_LABELS[key],
          before: initialForm[key],
          after: form[key],
        })
      }
    }

    return res
  }, [form, initialForm])

  const validate = (): boolean => {
    const errs: ValidationErrors = {}

    // Gateway default: manual | midtrans
    if (form.payment_gateway_default !== 'manual' && form.payment_gateway_default !== 'midtrans') {
      errs.payment_gateway_default = 'Harus memilih manual atau midtrans'
    }

    // Grace period: 0..720 integer
    const grace = Number(form.grace_period_hours)
    if (form.grace_period_hours.trim() === '' || Number.isNaN(grace) || !Number.isInteger(grace)) {
      errs.grace_period_hours = 'Harus berupa bilangan bulat'
    } else if (grace < 0 || grace > 720) {
      errs.grace_period_hours = 'Rentang yang diizinkan: 0 sampai 720 jam'
    }

    // Tax rate: 0..100 (contract says integer percent in backend, client UI step 0.1)
    const tax = Number(form.tax_rate_percent)
    if (form.tax_rate_percent.trim() === '' || Number.isNaN(tax)) {
      errs.tax_rate_percent = 'Harus berupa angka valid'
    } else if (tax < 0 || tax > 100) {
      errs.tax_rate_percent = 'Rentang yang diizinkan: 0 sampai 100 persen'
    }

    // Invoice due: 1..720 integer
    const inv = Number(form.invoice_due_hours)
    if (form.invoice_due_hours.trim() === '' || Number.isNaN(inv) || !Number.isInteger(inv)) {
      errs.invoice_due_hours = 'Harus berupa bilangan bulat'
    } else if (inv < 1 || inv > 720) {
      errs.invoice_due_hours = 'Rentang yang diizinkan: 1 sampai 720 jam'
    }

    // Storage driver: local | r2
    if (form.storage_driver !== 'local' && form.storage_driver !== 'r2') {
      errs.storage_driver = 'Harus memilih local atau r2'
    }

    // Reserve estimates: positive integers, 1..1.000.000 (mirrors backend spec)
    for (const key of ESTIMATE_KEYS) {
      const v = Number(form[key])
      if (form[key].trim() === '' || Number.isNaN(v) || !Number.isInteger(v)) {
        errs[key] = 'Harus berupa bilangan bulat'
      } else if (v < 1 || v > 1_000_000) {
        errs[key] = 'Rentang yang diizinkan: 1 sampai 1.000.000'
      }
    }

    // Dunning offsets + breaker tuning: positive integers within the backend
    // validator bounds (hours 1..720, threshold/probes 1..100, seconds 1..3600).
    const bounds: Record<string, [number, number]> = {
      dunning_hours_urgent: [1, 720],
      dunning_hours_soon: [1, 720],
      breaker_failure_threshold: [1, 100],
      breaker_cooldown_seconds: [1, 3600],
      breaker_half_open_probes: [1, 100],
      route_breaker_threshold: [1, 100],
      route_breaker_cooldown_secs: [1, 3600],
      metering_chars_per_token: [1, 100],
      metering_safety_buffer_pct: [0, 100],
      metering_image_tokens: [1, 10000],
      metering_audio_tokens_per_sec: [1, 1000],
    }
    for (const key of [...DUNNING_KEYS, ...BREAKER_KEYS, ...METERING_KEYS]) {
      const v = Number(form[key])
      const [lo, hi] = bounds[key]
      if (form[key].trim() === '' || Number.isNaN(v) || !Number.isInteger(v)) {
        errs[key] = 'Harus berupa bilangan bulat'
      } else if (v < lo || v > hi) {
        errs[key] = `Rentang yang diizinkan: ${lo} sampai ${hi}`
      }
    }

    setClientErrors(errs)
    return Object.keys(errs).length === 0
  }

  const handleOpenConfirm = (e: React.FormEvent) => {
    e.preventDefault()
    setBackendErrors({})
    if (!validate()) return
    if (diffs.length === 0) return
    setConfirmOpen(true)
  }

  const handleConfirmSave = async () => {
    const dirtyPayload: Record<string, unknown> = {}

    for (const d of diffs) {
      if (d.key === 'payment_gateway_default') {
        dirtyPayload[d.key] = form.payment_gateway_default
      } else if (d.key === 'grace_period_hours') {
        dirtyPayload[d.key] = parseInt(form.grace_period_hours, 10)
      } else if (d.key === 'tax_rate_percent') {
        dirtyPayload[d.key] = parseFloat(form.tax_rate_percent)
      } else if (d.key === 'invoice_due_hours') {
        dirtyPayload[d.key] = parseInt(form.invoice_due_hours, 10)
      } else if (d.key === 'storage_driver') {
        dirtyPayload[d.key] = form.storage_driver
      } else if (
        ([...ESTIMATE_KEYS, ...DUNNING_KEYS, ...BREAKER_KEYS, ...METERING_KEYS] as readonly string[]).includes(d.key)
      ) {
        dirtyPayload[d.key] = parseInt(form[d.key as keyof FormState], 10)
      }
    }

    try {
      await onSave(dirtyPayload)
      setConfirmOpen(false)
      setInitialForm({ ...form })
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const resp = err.response
        if (resp && typeof resp === 'object' && 'data' in resp) {
          const data = resp.data
          if (data && typeof data === 'object') {
            const msg = 'message' in data && typeof data.message === 'string' ? data.message : ''
            const code = 'error' in data && typeof data.error === 'string' ? data.error : ''

            // Check if specific field mentioned in message
            const newBackendErr: ValidationErrors = {}
            for (const key of Object.keys(SETTING_LABELS)) {
              if (msg.includes(key)) {
                newBackendErr[key as keyof ValidationErrors] = `${code ? `[${code}] ` : ''}${msg}`
              }
            }
            if (Object.keys(newBackendErr).length > 0) {
              setBackendErrors(newBackendErr)
            }
          }
        }
      }
    }
  }

  const handleReset = () => {
    setForm(initialForm)
    setClientErrors({})
    setBackendErrors({})
  }

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Card className="border-border">
          <CardHeader>
            <Skeleton className="h-6 w-48" />
            <Skeleton className="h-4 w-72" />
          </CardHeader>
          <CardContent className="space-y-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card className="border-destructive/50 bg-destructive/5 p-6 text-center">
        <AlertCircle className="mx-auto size-8 text-destructive mb-2" />
        <h3 className="font-semibold text-foreground mb-1">Gagal Memuat Pengaturan</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Terjadi kesalahan saat memuat data pengaturan sistem dari server.
        </p>
        <Button variant="outline" onClick={onRefresh}>
          Coba Lagi
        </Button>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleOpenConfirm}>
        <Card className="border-border">
          <CardHeader className="flex flex-row items-center justify-between pb-4">
            <div>
              <CardTitle>Pengaturan Global FinTrack</CardTitle>
              <CardDescription>
                Konfigurasi pembayaran, periode tenggang, pajak, dan faktur sistem.
              </CardDescription>
            </div>
            {diffs.length > 0 && (
              <span className="text-xs bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/30 px-2 py-1 font-medium">
                {diffs.length} perubahan belum disimpan
              </span>
            )}
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Field 1: payment_gateway_default */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="payment_gateway_default" className="font-medium">
                  {SETTING_LABELS.payment_gateway_default}
                </Label>
                {settingsMap.payment_gateway_default && (
                  <span className="text-xs text-muted-foreground">
                    Diperbarui: {formatTanggal(settingsMap.payment_gateway_default.updated_at)}
                    {settingsMap.payment_gateway_default.updated_by
                      ? ` oleh ${settingsMap.payment_gateway_default.updated_by}`
                      : ''}
                  </span>
                )}
              </div>
              <Select
                value={form.payment_gateway_default}
                onValueChange={(val) => {
                  setForm((prev) => ({ ...prev, payment_gateway_default: val }))
                  setClientErrors((prev) => ({ ...prev, payment_gateway_default: undefined }))
                  setBackendErrors((prev) => ({ ...prev, payment_gateway_default: undefined }))
                }}
              >
                <SelectTrigger id="payment_gateway_default" aria-label="Payment Gateway Default">
                  <SelectValue placeholder="Pilih gateway default" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="manual">manual (Transfer Bank Manual)</SelectItem>
                  <SelectItem value="midtrans">midtrans (Payment Gateway Otomatis)</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {settingsMap.payment_gateway_default?.description ||
                  'Gateway pembayaran default jika paket langganan tidak menentukan override.'}
              </p>
              {clientErrors.payment_gateway_default && (
                <p className="text-xs text-destructive font-medium">
                  {clientErrors.payment_gateway_default}
                </p>
              )}
              {backendErrors.payment_gateway_default && (
                <p className="text-xs text-destructive font-medium">
                  {backendErrors.payment_gateway_default}
                </p>
              )}
            </div>

            {/* Field 2: grace_period_hours */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="grace_period_hours" className="font-medium">
                  {SETTING_LABELS.grace_period_hours}
                </Label>
                {settingsMap.grace_period_hours && (
                  <span className="text-xs text-muted-foreground">
                    Diperbarui: {formatTanggal(settingsMap.grace_period_hours.updated_at)}
                    {settingsMap.grace_period_hours.updated_by
                      ? ` oleh ${settingsMap.grace_period_hours.updated_by}`
                      : ''}
                  </span>
                )}
              </div>
              <Input
                id="grace_period_hours"
                type="number"
                min={0}
                max={720}
                step={1}
                value={form.grace_period_hours}
                onChange={(e) => {
                  setForm((prev) => ({ ...prev, grace_period_hours: e.target.value }))
                  setClientErrors((prev) => ({ ...prev, grace_period_hours: undefined }))
                  setBackendErrors((prev) => ({ ...prev, grace_period_hours: undefined }))
                }}
                placeholder="72"
                aria-label="Grace Period (Jam)"
              />
              <p className="text-xs text-muted-foreground">
                {settingsMap.grace_period_hours?.description ||
                  'Durasi toleransi setelah periode langganan berakhir sebelum dinonaktifkan (rentang: 0 - 720 jam).'}
              </p>
              {clientErrors.grace_period_hours && (
                <p className="text-xs text-destructive font-medium">
                  {clientErrors.grace_period_hours}
                </p>
              )}
              {backendErrors.grace_period_hours && (
                <p className="text-xs text-destructive font-medium">
                  {backendErrors.grace_period_hours}
                </p>
              )}
            </div>

            {/* Field 3: tax_rate_percent */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="tax_rate_percent" className="font-medium">
                  {SETTING_LABELS.tax_rate_percent}
                </Label>
                {settingsMap.tax_rate_percent && (
                  <span className="text-xs text-muted-foreground">
                    Diperbarui: {formatTanggal(settingsMap.tax_rate_percent.updated_at)}
                    {settingsMap.tax_rate_percent.updated_by
                      ? ` oleh ${settingsMap.tax_rate_percent.updated_by}`
                      : ''}
                  </span>
                )}
              </div>
              <Input
                id="tax_rate_percent"
                type="number"
                min={0}
                max={100}
                step={0.1}
                value={form.tax_rate_percent}
                onChange={(e) => {
                  setForm((prev) => ({ ...prev, tax_rate_percent: e.target.value }))
                  setClientErrors((prev) => ({ ...prev, tax_rate_percent: undefined }))
                  setBackendErrors((prev) => ({ ...prev, tax_rate_percent: undefined }))
                }}
                placeholder="11"
                aria-label="Tarif Pajak PPN (%)"
              />
              <p className="text-xs text-muted-foreground">
                {settingsMap.tax_rate_percent?.description ||
                  'Tarif pajak pertambahan nilai yang dikenakan pada tagihan langganan (rentang: 0 - 100%).'}
              </p>
              {clientErrors.tax_rate_percent && (
                <p className="text-xs text-destructive font-medium">
                  {clientErrors.tax_rate_percent}
                </p>
              )}
              {backendErrors.tax_rate_percent && (
                <p className="text-xs text-destructive font-medium">
                  {backendErrors.tax_rate_percent}
                </p>
              )}
            </div>

            {/* Field 4: invoice_due_hours */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="invoice_due_hours" className="font-medium">
                  {SETTING_LABELS.invoice_due_hours}
                </Label>
                {settingsMap.invoice_due_hours && (
                  <span className="text-xs text-muted-foreground">
                    Diperbarui: {formatTanggal(settingsMap.invoice_due_hours.updated_at)}
                    {settingsMap.invoice_due_hours.updated_by
                      ? ` oleh ${settingsMap.invoice_due_hours.updated_by}`
                      : ''}
                  </span>
                )}
              </div>
              <Input
                id="invoice_due_hours"
                type="number"
                min={1}
                max={720}
                step={1}
                value={form.invoice_due_hours}
                onChange={(e) => {
                  setForm((prev) => ({ ...prev, invoice_due_hours: e.target.value }))
                  setClientErrors((prev) => ({ ...prev, invoice_due_hours: undefined }))
                  setBackendErrors((prev) => ({ ...prev, invoice_due_hours: undefined }))
                }}
                placeholder="24"
                aria-label="Jatuh Tempo Invoice (Jam)"
              />
              <p className="text-xs text-muted-foreground">
                {settingsMap.invoice_due_hours?.description ||
                  'Batas waktu pembayaran sebelum faktur kedaluwarsa otomatis (rentang: 1 - 720 jam).'}
              </p>
              {clientErrors.invoice_due_hours && (
                <p className="text-xs text-destructive font-medium">
                  {clientErrors.invoice_due_hours}
                </p>
              )}
              {backendErrors.invoice_due_hours && (
                <p className="text-xs text-destructive font-medium">
                  {backendErrors.invoice_due_hours}
                </p>
              )}
            </div>

            {/* Field 5: storage_driver */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="storage_driver" className="font-medium">
                  {SETTING_LABELS.storage_driver}
                </Label>
                {settingsMap.storage_driver && (
                  <span className="text-xs text-muted-foreground">
                    Diperbarui: {formatTanggal(settingsMap.storage_driver.updated_at)}
                    {settingsMap.storage_driver.updated_by
                      ? ` oleh ${settingsMap.storage_driver.updated_by}`
                      : ''}
                  </span>
                )}
              </div>
              <Select
                value={form.storage_driver}
                onValueChange={(val) => {
                  setForm((prev) => ({ ...prev, storage_driver: val }))
                  setClientErrors((prev) => ({ ...prev, storage_driver: undefined }))
                  setBackendErrors((prev) => ({ ...prev, storage_driver: undefined }))
                }}
              >
                <SelectTrigger id="storage_driver" aria-label="Penyimpanan Bukti Transfer">
                  <SelectValue placeholder="Pilih penyimpanan" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="local">local (Penyimpanan Lokal Server)</SelectItem>
                  <SelectItem value="r2">r2 (Cloudflare R2 Object Storage)</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {settingsMap.storage_driver?.description ||
                  'Tempat penyimpanan berkas bukti transfer. Perubahan berlaku seketika untuk unggahan baru; berkas lama tetap dapat dibaca.'}
              </p>
              {storageStatus && (
                <p className="text-xs font-medium text-muted-foreground">
                  Aktif saat ini: <span className="font-mono font-bold uppercase">{storageStatus.active}</span>
                  {' · '}
                  R2 {storageStatus.r2_configured ? (storageStatus.r2_healthy ? 'terkonfigurasi & sehat' : 'terkonfigurasi (tidak sehat)') : 'belum dikonfigurasi'}
                </p>
              )}
              {clientErrors.storage_driver && (
                <p className="text-xs text-destructive font-medium">
                  {clientErrors.storage_driver}
                </p>
              )}
              {backendErrors.storage_driver && (
                <p className="text-xs text-destructive font-medium">
                  {backendErrors.storage_driver}
                </p>
              )}
            </div>

            {/* Field 6: AI reserve estimates (P3) */}
            <div className="space-y-3 pt-4 border-t border-border">
              <div>
                <Label className="font-medium">Estimasi Reserve Token AI</Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Jumlah token yang ditahan (bukan ditagih) sebelum setiap panggilan AI. Biaya
                  sebenarnya diselesaikan saat commit. Nilai terlalu kecil dapat memicu pemotongan
                  saat pemakaian melebihi estimasi.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {ESTIMATE_KEYS.map((key) => (
                  <div key={key} className="space-y-1.5">
                    <Label htmlFor={key} className="text-xs">{SETTING_LABELS[key]}</Label>
                    <Input
                      id={key}
                      type="number"
                      min={1}
                      max={1_000_000}
                      value={form[key]}
                      onChange={(e) => {
                        setForm((prev) => ({ ...prev, [key]: e.target.value }))
                        setClientErrors((prev) => ({ ...prev, [key]: undefined }))
                        setBackendErrors((prev) => ({ ...prev, [key]: undefined }))
                      }}
                    />
                    {clientErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{clientErrors[key]}</p>
                    )}
                    {backendErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{backendErrors[key]}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Field 7: dunning reminder offsets */}
            <div className="space-y-3 pt-4 border-t border-border">
              <div>
                <Label className="font-medium">Pengingat Dunning (Jam Sebelum Jatuh Tempo)</Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Kapan email/pesan pengingat dikirim sebelum invoice jatuh tempo. Pengingat
                  mendesak dikirim saat sisa waktu ≤ nilai pertama; pengingat segera saat ≤ nilai
                  kedua. Nilai mendesak otomatis dibatasi agar tidak melebihi nilai segera.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {DUNNING_KEYS.map((key) => (
                  <div key={key} className="space-y-1.5">
                    <Label htmlFor={key} className="text-xs">{SETTING_LABELS[key]}</Label>
                    <Input
                      id={key}
                      type="number"
                      min={1}
                      max={720}
                      value={form[key]}
                      onChange={(e) => {
                        setForm((prev) => ({ ...prev, [key]: e.target.value }))
                        setClientErrors((prev) => ({ ...prev, [key]: undefined }))
                        setBackendErrors((prev) => ({ ...prev, [key]: undefined }))
                      }}
                    />
                    {clientErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{clientErrors[key]}</p>
                    )}
                    {backendErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{backendErrors[key]}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Field 8: AI gateway circuit breaker tuning */}
            <div className="space-y-3 pt-4 border-t border-border">
              <div>
                <Label className="font-medium">Tuning Circuit Breaker AI Gateway</Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Breaker <strong>endpoint</strong> melindungi seluruh upstream (kegagalan transport);
                  breaker <strong>rute</strong> mengarantina satu pasangan (endpoint, model) yang mati
                  agar model sehat lain di endpoint yang sama tetap melayani. Perubahan berlaku
                  seketika tanpa restart.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {BREAKER_KEYS.map((key) => (
                  <div key={key} className="space-y-1.5">
                    <Label htmlFor={key} className="text-xs">{SETTING_LABELS[key]}</Label>
                    <Input
                      id={key}
                      type="number"
                      min={1}
                      max={key.endsWith('seconds') || key.endsWith('secs') ? 3600 : 100}
                      value={form[key]}
                      onChange={(e) => {
                        setForm((prev) => ({ ...prev, [key]: e.target.value }))
                        setClientErrors((prev) => ({ ...prev, [key]: undefined }))
                        setBackendErrors((prev) => ({ ...prev, [key]: undefined }))
                      }}
                    />
                    {clientErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{clientErrors[key]}</p>
                    )}
                    {backendErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{backendErrors[key]}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Field 9: AI token metering heuristics estimation */}
            <div className="space-y-3 pt-4 border-t border-border">
              <div>
                <Label className="font-medium">Heuristik Estimasi Token AI (Fallback)</Label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Digunakan untuk mengestimasi pemakaian token lokal saat provider upstream tidak
                  melaporkan metrik kuota token. Perubahan langsung aktif seketika pada kalkulasi
                  turn berikutnya.
                </p>
              </div>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                {METERING_KEYS.map((key) => (
                  <div key={key} className="space-y-1.5">
                    <Label htmlFor={key} className="text-xs">{SETTING_LABELS[key]}</Label>
                    <Input
                      id={key}
                      type="number"
                      min={key === 'metering_safety_buffer_pct' ? 0 : 1}
                      max={
                        key === 'metering_image_tokens'
                          ? 10000
                          : key === 'metering_audio_tokens_per_sec'
                          ? 1000
                          : 100
                      }
                      value={form[key]}
                      onChange={(e) => {
                        setForm((prev) => ({ ...prev, [key]: e.target.value }))
                        setClientErrors((prev) => ({ ...prev, [key]: undefined }))
                        setBackendErrors((prev) => ({ ...prev, [key]: undefined }))
                      }}
                    />
                    {clientErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{clientErrors[key]}</p>
                    )}
                    {backendErrors[key] && (
                      <p className="text-xs text-destructive font-medium">{backendErrors[key]}</p>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Action buttons */}
            <div className="flex items-center justify-end gap-3 pt-4 border-t border-border">
              <Button
                type="button"
                variant="outline"
                onClick={handleReset}
                disabled={diffs.length === 0 || isSaving}
              >
                <RotateCcw className="size-4 mr-2" />
                Reset
              </Button>
              <Button
                type="submit"
                disabled={diffs.length === 0 || isSaving}
              >
                <Save className="size-4 mr-2" />
                {isSaving ? 'Menyimpan...' : 'Simpan Pengaturan'}
              </Button>
            </div>
          </CardContent>
        </Card>
      </form>

      {/* Unknown / Read-only settings section */}
      {unknownSettings.length > 0 && (
        <Card className="border-border">
          <CardHeader>
            <div className="flex items-center gap-2">
              <Lock className="size-4 text-muted-foreground" />
              <CardTitle className="text-base">Pengaturan Sistem Lainnya (Read-only)</CardTitle>
            </div>
            <CardDescription>
              Kunci konfigurasi ini dikelola langsung oleh backend atau audit endpoint khusus dan tidak dapat diubah dari UI.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="border border-border divide-y divide-border">
              {unknownSettings.map((s) => (
                <div key={s.key} className="p-4 bg-muted/30 space-y-2">
                  <div className="flex items-start justify-between">
                    <div>
                      <code className="text-sm font-semibold font-mono text-foreground">{s.key}</code>
                      {s.description && (
                        <p className="text-xs text-muted-foreground mt-0.5">{s.description}</p>
                      )}
                    </div>
                    <span className="text-[11px] bg-muted border border-border text-muted-foreground px-2 py-0.5 font-medium">
                      tidak dapat diubah dari UI
                    </span>
                  </div>
                  <div className="text-xs font-mono bg-card p-2 border border-border overflow-x-auto">
                    {typeof s.value === 'object'
                      ? JSON.stringify(s.value, null, 2)
                      : String(s.value)}
                  </div>
                  <div className="text-[11px] text-muted-foreground">
                    Diperbarui: {formatTanggal(s.updated_at)}
                    {s.updated_by ? ` oleh ${s.updated_by}` : ''}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Confirmation Dialog */}
      <SettingsConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        diffs={diffs}
        onConfirm={handleConfirmSave}
        isSubmitting={isSaving}
      />
    </div>
  )
}
