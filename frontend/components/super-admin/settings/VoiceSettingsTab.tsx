'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Mic, Save, RotateCcw, AlertCircle } from 'lucide-react'
import type { SettingRow } from '@/hooks/useSuperAdminSettings'
import { SettingsConfirmDialog, type SettingDiff } from './SettingsConfirmDialog'

interface VoiceSettingsTabProps {
  settings: SettingRow[]
  isLoading: boolean
  error: unknown
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
  onRefresh: () => void
}

/** The dynamic voice keys this tab edits (K-VOI-08..19). */
const VOICE_KEYS = {
  maxDuration: 'voice_max_duration_seconds',
  languageMode: 'voice_language_mode',
  defaultLanguage: 'voice_default_language',
  languages: 'voice_languages',
  retention: 'voice_history_retention_days',
  resetTimezone: 'voice_daily_reset_timezone',
  allowEdit: 'voice_allow_edit_before_send',
  vuMeter: 'voice_vu_meter_enabled',
  fallback: 'voice_fallback_to_browser',
  serverEnabled: 'voice_server_enabled',
} as const

/** Human labels for the confirm-dialog diff rows. */
const VOICE_LABELS: Record<string, string> = {
  [VOICE_KEYS.maxDuration]: 'Durasi rekam maksimum',
  [VOICE_KEYS.languageMode]: 'Mode bahasa',
  [VOICE_KEYS.defaultLanguage]: 'Bahasa default',
  [VOICE_KEYS.languages]: 'Daftar bahasa',
  [VOICE_KEYS.retention]: 'Retensi riwayat (hari)',
  [VOICE_KEYS.resetTimezone]: 'Zona reset harian',
  [VOICE_KEYS.allowEdit]: 'Izinkan edit sebelum kirim',
  [VOICE_KEYS.vuMeter]: 'Tampilkan VU meter',
  [VOICE_KEYS.fallback]: 'Fallback ke mode dasar',
  [VOICE_KEYS.serverEnabled]: 'Aktifkan jalur server',
}

interface FormState {
  maxDuration: string
  languageMode: string
  defaultLanguage: string
  languages: string
  retention: string
  resetTimezone: string
  allowEdit: boolean
  vuMeter: boolean
  fallback: boolean
  serverEnabled: boolean
}

const DEFAULT_FORM: FormState = {
  maxDuration: '60',
  languageMode: 'id_only',
  defaultLanguage: 'id',
  languages: 'id',
  retention: '90',
  resetTimezone: 'Asia/Jakarta',
  allowEdit: true,
  vuMeter: true,
  fallback: true,
  serverEnabled: true,
}

/** Reads a setting row's raw value, unwrapping the JSON-ish shapes the API may send. */
function readString(rows: SettingRow[], key: string, fallback: string): string {
  const row = rows.find((r) => r.key === key)
  if (!row || row.value === null || row.value === undefined) return fallback
  if (typeof row.value === 'string') return row.value
  if (typeof row.value === 'number') return String(row.value)
  return fallback
}

function readBool(rows: SettingRow[], key: string, fallback: boolean): boolean {
  const row = rows.find((r) => r.key === key)
  if (!row || row.value === null || row.value === undefined) return fallback
  if (typeof row.value === 'boolean') return row.value
  if (row.value === 'true') return true
  if (row.value === 'false') return false
  return fallback
}

function formFromRows(rows: SettingRow[]): FormState {
  return {
    maxDuration: readString(rows, VOICE_KEYS.maxDuration, DEFAULT_FORM.maxDuration),
    languageMode: readString(rows, VOICE_KEYS.languageMode, DEFAULT_FORM.languageMode),
    defaultLanguage: readString(rows, VOICE_KEYS.defaultLanguage, DEFAULT_FORM.defaultLanguage),
    languages: readString(rows, VOICE_KEYS.languages, DEFAULT_FORM.languages),
    retention: readString(rows, VOICE_KEYS.retention, DEFAULT_FORM.retention),
    resetTimezone: readString(rows, VOICE_KEYS.resetTimezone, DEFAULT_FORM.resetTimezone),
    allowEdit: readBool(rows, VOICE_KEYS.allowEdit, DEFAULT_FORM.allowEdit),
    vuMeter: readBool(rows, VOICE_KEYS.vuMeter, DEFAULT_FORM.vuMeter),
    fallback: readBool(rows, VOICE_KEYS.fallback, DEFAULT_FORM.fallback),
    serverEnabled: readBool(rows, VOICE_KEYS.serverEnabled, DEFAULT_FORM.serverEnabled),
  }
}

/**
 * VoiceSettingsTab edits every dynamic voice knob. Nothing here is hardcoded:
 * a change is persisted via PUT /admin/settings and takes effect within the
 * resolver TTL (seconds), with no redeploy.
 */
export function VoiceSettingsTab({
  settings,
  isLoading,
  error,
  onSave,
  isSaving,
  onRefresh,
}: VoiceSettingsTabProps) {
  const [form, setForm] = React.useState<FormState>(DEFAULT_FORM)
  const [confirmOpen, setConfirmOpen] = React.useState(false)
  const [diffs, setDiffs] = React.useState<SettingDiff[]>([])

  React.useEffect(() => {
    if (!isLoading && settings.length > 0) setForm(formFromRows(settings))
  }, [settings, isLoading])

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  const handleSave = () => {
    const duration = Number(form.maxDuration)
    const retention = Number(form.retention)
    if (!Number.isFinite(duration) || duration < 1 || duration > 600) {
      return
    }
    if (!Number.isFinite(retention) || retention < 1 || retention > 3650) {
      return
    }
    const payload: Record<string, unknown> = {
      [VOICE_KEYS.maxDuration]: duration,
      [VOICE_KEYS.languageMode]: form.languageMode,
      [VOICE_KEYS.defaultLanguage]: form.defaultLanguage,
      [VOICE_KEYS.languages]: form.languages,
      [VOICE_KEYS.retention]: retention,
      [VOICE_KEYS.resetTimezone]: form.resetTimezone,
      [VOICE_KEYS.allowEdit]: form.allowEdit,
      [VOICE_KEYS.vuMeter]: form.vuMeter,
      [VOICE_KEYS.fallback]: form.fallback,
      [VOICE_KEYS.serverEnabled]: form.serverEnabled,
    }
    setDiffs(
      Object.entries(payload).map(([key, value]) => ({
        key,
        label: VOICE_LABELS[key] ?? key,
        before: settings.find((s) => s.key === key)?.value ?? null,
        after: value,
      })),
    )
    setConfirmOpen(true)
  }

  const confirmSave = async () => {
    const payload: Record<string, unknown> = {
      [VOICE_KEYS.maxDuration]: Number(form.maxDuration),
      [VOICE_KEYS.languageMode]: form.languageMode,
      [VOICE_KEYS.defaultLanguage]: form.defaultLanguage,
      [VOICE_KEYS.languages]: form.languages,
      [VOICE_KEYS.retention]: Number(form.retention),
      [VOICE_KEYS.resetTimezone]: form.resetTimezone,
      [VOICE_KEYS.allowEdit]: form.allowEdit,
      [VOICE_KEYS.vuMeter]: form.vuMeter,
      [VOICE_KEYS.fallback]: form.fallback,
      [VOICE_KEYS.serverEnabled]: form.serverEnabled,
    }
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
          Gagal memuat pengaturan suara.
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RotateCcw className="w-4 h-4 mr-1.5" /> Coba lagi
          </Button>
        </CardContent>
      </Card>
    )
  }

  const multi = form.languageMode === 'multi'

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Mic className="w-5 h-5" /> Input Suara
          </CardTitle>
          <CardDescription>
            Semua parameter bersifat dinamis dan berlaku tanpa deploy ulang. Perubahan aktif dalam
            hitungan detik.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Recording duration */}
          <div className="space-y-2">
            <Label htmlFor="voice-duration">Durasi rekam maksimum (detik)</Label>
            <Input
              id="voice-duration"
              type="number"
              min={1}
              max={600}
              value={form.maxDuration}
              onChange={(e) => set('maxDuration', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Rentang 1–600 detik. Default 60.</p>
          </div>

          {/* Language */}
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="voice-lang-mode">Mode bahasa</Label>
              <Select value={form.languageMode} onValueChange={(v) => set('languageMode', v)}>
                <SelectTrigger id="voice-lang-mode">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="id_only">Hanya Indonesia (id_only)</SelectItem>
                  <SelectItem value="multi">Multi bahasa (multi)</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {multi
                  ? 'Pengguna memilih bahasa per rekaman.'
                  : 'Bahasa dikunci ke bahasa default.'}
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="voice-default-lang">Bahasa default</Label>
              <Input
                id="voice-default-lang"
                value={form.defaultLanguage}
                onChange={(e) => set('defaultLanguage', e.target.value)}
                placeholder="id"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="voice-langs">Daftar bahasa (dipisah koma)</Label>
            <Input
              id="voice-langs"
              value={form.languages}
              onChange={(e) => set('languages', e.target.value)}
              placeholder="id,en"
              disabled={!multi}
            />
            <p className="text-xs text-muted-foreground">
              Hanya relevan saat mode <span className="font-mono">multi</span>.
            </p>
          </div>

          {/* Retention + timezone */}
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="voice-retention">Retensi riwayat (hari)</Label>
              <Input
                id="voice-retention"
                type="number"
                min={1}
                max={3650}
                value={form.retention}
                onChange={(e) => set('retention', e.target.value)}
              />
              <p className="text-xs text-muted-foreground">Rentang 1–3650 hari. Default 90.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="voice-tz">Zona reset harian</Label>
              <Select value={form.resetTimezone} onValueChange={(v) => set('resetTimezone', v)}>
                <SelectTrigger id="voice-tz">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Asia/Jakarta">Asia/Jakarta (WIB)</SelectItem>
                  <SelectItem value="UTC">UTC</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">Menentukan batas hari untuk limit harian.</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Behaviour toggles */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Perilaku</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <ToggleRow
            id="voice-allow-edit"
            label="Izinkan edit sebelum kirim"
            desc="Pengguna dapat meninjau/menyunting transkrip sebelum dikirim ke chat."
            checked={form.allowEdit}
            onChange={(v) => set('allowEdit', v)}
          />
          <ToggleRow
            id="voice-vu"
            label="Tampilkan VU meter"
            desc="Indikator level mikrofon saat merekam (disarankan aktif)."
            checked={form.vuMeter}
            onChange={(v) => set('vuMeter', v)}
          />
          <ToggleRow
            id="voice-fallback"
            label="Fallback ke mode dasar"
            desc="Turunkan ke transkripsi perangkat saat jalur premium tidak tersedia."
            checked={form.fallback}
            onChange={(v) => set('fallback', v)}
          />
          <ToggleRow
            id="voice-server"
            label="Aktifkan jalur server (premium)"
            desc="Kill switch insiden. Nonaktif = semua pengguna memakai mode dasar."
            checked={form.serverEnabled}
            onChange={(v) => set('serverEnabled', v)}
          />
        </CardContent>
      </Card>

      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={() => setForm(formFromRows(settings))} disabled={isSaving}>
          <RotateCcw className="w-4 h-4 mr-1.5" /> Reset
        </Button>
        <Button onClick={handleSave} disabled={isSaving}>
          <Save className="w-4 h-4 mr-1.5" /> Simpan
        </Button>
      </div>

      <SettingsConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        diffs={diffs}
        onConfirm={confirmSave}
        isSubmitting={isSaving}
      />
    </div>
  )
}

interface ToggleRowProps {
  id: string
  label: string
  desc: string
  checked: boolean
  onChange: (v: boolean) => void
}

function ToggleRow({ id, label, desc, checked, onChange }: ToggleRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-0.5">
        <Label htmlFor={id}>{label}</Label>
        <p className="text-xs text-muted-foreground">{desc}</p>
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
    </div>
  )
}
