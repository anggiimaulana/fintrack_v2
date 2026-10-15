'use client'

import * as React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Save, AlertCircle } from 'lucide-react'
import { useSettingDefinitions, type SettingDefinition } from '@/hooks/useConfigCatalog'
import { SettingRow } from '@/hooks/useSuperAdminSettings'

/**
 * Quota settings tab (P-13).
 *
 * Unlike the older tabs, this one renders whatever the `kuota` group of the
 * setting catalog defines — no field list is hardcoded in the frontend. Adding
 * a quota setting in the DB makes it appear here automatically.
 */

interface QuotaSettingsTabProps {
  settings: SettingRow[]
  isLoading: boolean
  onSave: (dirty: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
}

/** Coerce a setting's current/definition value to a string for the input. */
function toInputString(v: unknown): string {
  if (v === null || v === undefined) return ''
  if (typeof v === 'string') return v
  if (typeof v === 'number' || typeof v === 'boolean') return String(v)
  return JSON.stringify(v)
}

export function QuotaSettingsTab({ settings, isLoading, onSave, isSaving }: QuotaSettingsTabProps) {
  const { definitions, isLoading: defsLoading } = useSettingDefinitions()
  const quotaDefs = React.useMemo(
    () =>
      definitions.filter(
        (d) =>
          (d.group_key === 'kuota' || d.group_key === 'ai_estimate') &&
          d.is_active &&
          d.is_writable &&
          !d.is_secret,
      ),
    [definitions],
  )

  // Current values keyed by setting key.
  const current = React.useMemo(() => {
    const m: Record<string, unknown> = {}
    for (const s of settings) m[s.key] = s.value
    return m
  }, [settings])

  const [draft, setDraft] = React.useState<Record<string, string>>({})
  const [dirty, setDirty] = React.useState<Record<string, unknown>>({})

  const valueOf = (d: SettingDefinition): string =>
    d.key in draft ? draft[d.key] : toInputString(current[d.key] ?? d.default_value)

  const onChange = (d: SettingDefinition, raw: string) => {
    setDraft((prev) => ({ ...prev, [d.key]: raw }))
    // Coerce to the definition's kind so the backend receives the right type.
    // An empty int field must NOT become NaN (which serializes to JSON `null`
    // and is rejected by the backend): fall back to the definition default, and
    // ultimately to 0, so the payload is always a number.
    let value: unknown = raw
    if (d.kind === 'int') {
      const parsed = parseInt(raw, 10)
      if (Number.isNaN(parsed)) {
        const def = typeof d.default_value === 'number' ? d.default_value : 0
        value = def
      } else {
        value = parsed
      }
    }
    if (d.kind === 'bool') value = raw === 'true'
    setDirty((prev) => ({ ...prev, [d.key]: value }))
  }

  const handleSave = async () => {
    if (Object.keys(dirty).length === 0) return
    await onSave(dirty)
    setDirty({})
    setDraft({})
  }

  const loading = isLoading || defsLoading

  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="text-lg font-black">Kuota & Estimasi AI (Chat, Voice, Vision)</CardTitle>
        <CardDescription>
          Nilai di bawah berasal dari katalog setting (database) — dapat diubah tanpa deploy.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {loading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-12 w-full border-2 rounded-md border-foreground" />
            ))}
          </div>
        ) : quotaDefs.length === 0 ? (
          <p className="text-sm text-muted-foreground flex items-center gap-2">
            <AlertCircle className="w-4 h-4" /> Tidak ada setting grup &quot;kuota&quot; di katalog.
          </p>
        ) : (
          quotaDefs.map((d) => (
            <div key={d.key} className="space-y-1.5">
              <Label htmlFor={`qs-${d.key}`}>{d.label || d.key}</Label>
              {d.kind === 'enum' ? (
                <select
                  id={`qs-${d.key}`}
                  value={valueOf(d)}
                  onChange={(e) => onChange(d, e.target.value)}
                  className="h-9 w-full rounded-md border-2 border-foreground bg-background px-3 text-sm font-medium"
                >
                  {(d.enum_values ?? []).map((opt) => (
                    <option key={opt} value={opt}>{opt}</option>
                  ))}
                </select>
              ) : d.kind === 'bool' ? (
                <select
                  id={`qs-${d.key}`}
                  value={valueOf(d)}
                  onChange={(e) => onChange(d, e.target.value)}
                  className="h-9 w-full rounded-md border-2 border-foreground bg-background px-3 text-sm font-medium"
                >
                  <option value="true">Aktif</option>
                  <option value="false">Nonaktif</option>
                </select>
              ) : (
                <Input
                  id={`qs-${d.key}`}
                  type={d.kind === 'int' ? 'number' : 'text'}
                  min={d.min_value}
                  max={d.max_value}
                  value={valueOf(d)}
                  onChange={(e) => onChange(d, e.target.value)}
                />
              )}
              {d.description && (
                <p className="text-xs text-muted-foreground">{d.description}</p>
              )}
            </div>
          ))
        )}

        <div className="flex justify-end">
          <Button onClick={handleSave} disabled={isSaving || Object.keys(dirty).length === 0}>
            <Save className="w-4 h-4 mr-2" />
            {isSaving ? 'Menyimpan…' : 'Simpan'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
