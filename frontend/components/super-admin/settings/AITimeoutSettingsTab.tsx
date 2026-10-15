'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Timer, Save, RotateCcw, AlertCircle, Info } from 'lucide-react'
import type { SettingRow } from '@/hooks/useSuperAdminSettings'
import { useAIConfig, FALLBACK_AI_CONFIG } from '@/hooks/useAIConfig'
import { SettingsConfirmDialog, type SettingDiff } from './SettingsConfirmDialog'

interface AITimeoutSettingsTabProps {
  settings: SettingRow[]
  isLoading: boolean
  error: unknown
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
  onRefresh: () => void
}

/**
 * The dynamic AI timeout keys this tab edits. Every value is a millisecond
 * budget applied to the running fleet within seconds — no redeploy, no restart.
 */
const TIMEOUT_KEYS = {
  chat: 'ai_timeout_chat_ms',
  vision: 'ai_timeout_vision_ms',
  voice: 'ai_timeout_voice_ms',
  embedding: 'ai_timeout_embedding_ms',
  reasoning: 'ai_timeout_reasoning_ms',
  client: 'ai_timeout_client_ms',
} as const

const TIMEOUT_LABELS: Record<string, string> = {
  [TIMEOUT_KEYS.chat]: 'Batas waktu: Chat (ms)',
  [TIMEOUT_KEYS.vision]: 'Batas waktu: Vision / OCR (ms)',
  [TIMEOUT_KEYS.voice]: 'Batas waktu: Suara (ms)',
  [TIMEOUT_KEYS.embedding]: 'Batas waktu: Embedding (ms)',
  [TIMEOUT_KEYS.reasoning]: 'Batas waktu: Penalaran (ms)',
  [TIMEOUT_KEYS.client]: 'Batas waktu klien / browser (ms)',
}

interface FormState {
  chat: string
  vision: string
  voice: string
  embedding: string
  reasoning: string
  client: string
}

/** Mirrors the backend seed values so an unset row shows a sensible default. */
const DEFAULT_FORM: FormState = {
  chat: '60000',
  vision: '120000',
  voice: '120000',
  embedding: '60000',
  reasoning: '120000',
  client: '135000',
}

/** Reads a setting row's numeric value as a string, tolerating the JSON shapes the API may send. */
function readString(rows: SettingRow[], key: string, fallback: string): string {
  const row = rows.find((r) => r.key === key)
  if (!row || row.value === null || row.value === undefined) return fallback
  if (typeof row.value === 'number') return String(row.value)
  if (typeof row.value === 'string' && row.value.trim() !== '') return row.value
  return fallback
}

function formFromRows(rows: SettingRow[]): FormState {
  return {
    chat: readString(rows, TIMEOUT_KEYS.chat, DEFAULT_FORM.chat),
    vision: readString(rows, TIMEOUT_KEYS.vision, DEFAULT_FORM.vision),
    voice: readString(rows, TIMEOUT_KEYS.voice, DEFAULT_FORM.voice),
    embedding: readString(rows, TIMEOUT_KEYS.embedding, DEFAULT_FORM.embedding),
    reasoning: readString(rows, TIMEOUT_KEYS.reasoning, DEFAULT_FORM.reasoning),
    client: readString(rows, TIMEOUT_KEYS.client, DEFAULT_FORM.client),
  }
}

/** One field's numeric bounds (milliseconds). 0 means "auto" for the client budget. */
interface FieldSpec {
  key: string
  label: string
  value: keyof FormState
  min: number
  max: number
  hint: string
}

const FIELDS: FieldSpec[] = [
  {
    key: TIMEOUT_KEYS.chat,
    label: 'Chat (streaming)',
    value: 'chat',
    min: 0,
    max: 1_800_000,
    hint: 'Chat mengalir token demi token, jadi 60 detik biasanya cukup. 0 = pakai nilai rute/default gateway.',
  },
  {
    key: TIMEOUT_KEYS.vision,
    label: 'Vision / OCR (scan struk)',
    value: 'vision',
    min: 0,
    max: 1_800_000,
    hint: 'Model harus membaca gambar lalu menjawab — naikkan bila struk sering timeout. 0 = pakai nilai rute/default.',
  },
  {
    key: TIMEOUT_KEYS.voice,
    label: 'Suara (transkripsi)',
    value: 'voice',
    min: 0,
    max: 1_800_000,
    hint: 'Transkripsi audio bisa memakan waktu lama untuk rekaman panjang. 0 = pakai nilai rute/default.',
  },
  {
    key: TIMEOUT_KEYS.embedding,
    label: 'Embedding (RAG)',
    value: 'embedding',
    min: 0,
    max: 1_800_000,
    hint: 'Pencarian semantik untuk konteks chat. 0 = pakai nilai rute/default.',
  },
  {
    key: TIMEOUT_KEYS.reasoning,
    label: 'Penalaran (reasoning)',
    value: 'reasoning',
    min: 0,
    max: 1_800_000,
    hint: 'Mode penalaran berpikir lebih lama sebelum menjawab. 0 = pakai nilai rute/default.',
  },
  {
    key: TIMEOUT_KEYS.client,
    label: 'Klien / browser',
    value: 'client',
    min: 0,
    max: 1_800_000,
    hint: 'Batas waktu yang diberikan ke browser. Isi 0 agar dihitung otomatis (use case terlama + margin). Nilai di atas plafon proxy Next.js akan dipangkas otomatis.',
  },
]


/**
 * AITimeoutSettingsTab edits the AI timeout budget. Every value is persisted via
 * PUT /admin/settings and applied to the running gateway + HTTP layer
 * immediately (and fanned out to every instance over Pub/Sub), so an operator
 * can widen the budget during a slow-provider incident and tighten it again
 * without a redeploy.
 */
export function AITimeoutSettingsTab({
  settings,
  isLoading,
  error,
  onSave,
  isSaving,
  onRefresh,
}: AITimeoutSettingsTabProps) {
  const [form, setForm] = React.useState<FormState>(DEFAULT_FORM)
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [diffs, setDiffs] = React.useState<SettingDiff[]>([])
  const [errors, setErrors] = React.useState<Partial<Record<keyof FormState, string>>>({})

  // The effective outer ceiling, reported by the backend so the operator sees
  // why a larger value would be clamped instead of silently ignored.
  const { config: aiConfig } = useAIConfig()
  const proxyCeilingMs = aiConfig?.proxy_ceiling_ms ?? FALLBACK_AI_CONFIG.proxy_ceiling_ms

  // Hydrate the form once the server settings arrive, without a
  // setState-in-effect: the signature changes only when the server values
  // actually change, so local edits are not clobbered on every render.
  const serverSignature = React.useMemo(() => JSON.stringify(formFromRows(settings)), [settings])
  const [hydratedSignature, setHydratedSignature] = React.useState<string | null>(null)
  if (settings.length > 0 && !isLoading && hydratedSignature !== serverSignature) {
    setHydratedSignature(serverSignature)
    setForm(formFromRows(settings))
    setErrors({})
  }

  const set = (key: keyof FormState, value: string) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  /** Validate every field; returns the payload or null when something is invalid. */
  const buildPayload = (): Record<string, unknown> | null => {
    const next: Record<string, unknown> = {}
    const nextErrors: Partial<Record<keyof FormState, string>> = {}
    for (const f of FIELDS) {
      const raw = form[f.value]
      const n = Number(raw)
      if (raw.trim() === '' || !Number.isFinite(n)) {
        nextErrors[f.value] = 'Harus berupa angka.'
        continue
      }
      if (n < f.min || n > f.max) {
        nextErrors[f.value] = `Rentang ${f.min}–${f.max} ms.`
        continue
      }
      next[f.key] = Math.round(n)
    }
    setErrors(nextErrors)
    return Object.keys(nextErrors).length > 0 ? null : next
  }

  const handleSave = () => {
    const payload = buildPayload()
    if (!payload) return
    setDiffs(
      Object.entries(payload).map(([key, value]) => ({
        key,
        label: TIMEOUT_LABELS[key] ?? key,
        before: settings.find((s) => s.key === key)?.value ?? null,
        after: value,
      })),
    )
    setConfirmOpen(true)
  }

  const confirmSave = async () => {
    const payload = buildPayload()
    if (!payload) return
    await onSave(payload)
    setConfirmOpen(false)
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-6 space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardContent className="p-6 flex items-center gap-3 text-sm text-muted-foreground">
          <AlertCircle className="w-5 h-5" />
          Gagal memuat pengaturan batas waktu AI.
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RotateCcw className="w-4 h-4 mr-1.5" /> Coba lagi
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Timer className="w-5 h-5" /> Batas Waktu AI (Timeout)
          </CardTitle>
          <CardDescription>
            Semua nilai bersifat dinamis dan langsung diterapkan ke seluruh instance tanpa deploy ulang
            (zero downtime). Perubahan aktif dalam hitungan detik.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-start gap-2 rounded-md border border-border bg-muted/40 p-3 text-xs text-muted-foreground">
            <Info className="w-4 h-4 mt-0.5 shrink-0" />
            <div className="space-y-1">
              <p>
                Nilai dalam milidetik. Isi <span className="font-mono">0</span> untuk memakai nilai
                rute/default gateway (per use case). Untuk <strong>Klien / browser</strong>, isi{' '}
                <span className="font-mono">0</span> agar dihitung otomatis.
              </p>
              <p>
                Plafon efektif proxy (Next.js):{' '}
                <span className="font-mono font-bold">{proxyCeilingMs.toLocaleString('id-ID')} ms</span>
                . Nilai di atas plafon ini otomatis dipangkas — naikkan{' '}
                <span className="font-mono">API_PROXY_TIMEOUT_MS</span> bila perlu lebih lama.
              </p>
            </div>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            {FIELDS.map((f) => (
              <div key={f.key} className="space-y-2">
                <Label htmlFor={f.key}>{f.label}</Label>
                <Input
                  id={f.key}
                  type="number"
                  min={f.min}
                  max={f.max}
                  step={1000}
                  value={form[f.value]}
                  aria-invalid={errors[f.value] ? true : undefined}
                  onChange={(e) => set(f.value, e.target.value)}
                />
                {errors[f.value] ? (
                  <p className="text-xs text-destructive">{errors[f.value]}</p>
                ) : (
                  <p className="text-xs text-muted-foreground">{f.hint}</p>
                )}
              </div>
            ))}
          </div>

          <div className="flex flex-wrap gap-2">
            <Button onClick={handleSave} disabled={isSaving}>
              <Save className="w-4 h-4 mr-1.5" />
              {isSaving ? 'Menyimpan...' : 'Simpan Batas Waktu'}
            </Button>
            <Button variant="outline" onClick={() => setForm(formFromRows(settings))} disabled={isSaving}>
              <RotateCcw className="w-4 h-4 mr-1.5" /> Reset
            </Button>
          </div>
        </CardContent>
      </Card>

      <SettingsConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        diffs={diffs}
        isSubmitting={isSaving}
        onConfirm={confirmSave}
      />
    </div>
  )
}
