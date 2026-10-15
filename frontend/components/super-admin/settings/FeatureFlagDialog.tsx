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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { FeatureFlag, UpdateFeatureFlagPayload } from '@/hooks/useSuperAdminSettings'
import { useResetOnOpen } from '@/hooks/use-reset-on-open'

interface FeatureFlagDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  flag: FeatureFlag | null
  onSave: (payload: UpdateFeatureFlagPayload) => Promise<unknown>
  isSaving: boolean
}

export function FeatureFlagDialog({
  open,
  onOpenChange,
  flag,
  onSave,
  isSaving,
}: FeatureFlagDialogProps) {
  const isEditing = flag !== null

  const [key, setKey] = React.useState('')
  const [name, setName] = React.useState('')
  const [description, setDescription] = React.useState('')
  const [enabled, setEnabled] = React.useState(false)
  const [rollout, setRollout] = React.useState(100)
  const [rulesText, setRulesText] = React.useState('')
  const [jsonError, setJsonError] = React.useState<string | null>(null)
  const [keyError, setKeyError] = React.useState<string | null>(null)
  const [rolloutError, setRolloutError] = React.useState<string | null>(null)

  useResetOnOpen(
    open,
    () => {
      if (flag) {
        setKey(flag.key)
        setName(flag.name)
        setDescription(flag.description)
        setEnabled(flag.enabled ?? flag.is_enabled ?? false)
        setRollout(flag.rollout ?? 0)
        setRulesText(flag.rules ? JSON.stringify(flag.rules, null, 2) : '')
      } else {
        setKey('')
        setName('')
        setDescription('')
        setEnabled(false)
        setRollout(100)
        setRulesText('')
      }
      setJsonError(null)
      setKeyError(null)
      setRolloutError(null)
    },
    flag?.key ?? 'new',
  )

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    let hasError = false
    setJsonError(null)
    setKeyError(null)
    setRolloutError(null)

    // Key validation
    const trimmedKey = key.trim()
    if (!trimmedKey) {
      setKeyError('Key feature flag wajib diisi')
      hasError = true
    } else if (!/^[a-z0-9_-]+$/i.test(trimmedKey)) {
      setKeyError('Key hanya boleh berupa huruf, angka, underscore, atau hyphen')
      hasError = true
    }

    // Rollout validation
    if (rollout < 0 || rollout > 100 || Number.isNaN(rollout)) {
      setRolloutError('Rollout harus bernilai 0 sampai 100')
      hasError = true
    }

    // Rules JSON validation
    let parsedRules: Record<string, unknown> | null = null
    const trimmedRules = rulesText.trim()
    if (trimmedRules.length > 0) {
      try {
        const parsed: unknown = JSON.parse(trimmedRules)
        if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
          setJsonError('Rules harus berupa JSON object (contoh: {"tier": "pro"})')
          hasError = true
        } else {
          parsedRules = parsed as Record<string, unknown>
        }
      } catch (err: unknown) {
        let msg = 'Format JSON tidak valid'
        if (err instanceof Error) {
          msg = err.message
        }
        setJsonError(msg)
        hasError = true
      }
    }

    if (hasError) return

    try {
      await onSave({
        key: trimmedKey,
        name: name.trim() || trimmedKey,
        description: description.trim(),
        enabled,
        rollout,
        rules: parsedRules,
      })
      onOpenChange(false)
    } catch {
      // Error handling is toast-driven in mutation hook
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? 'Ubah Feature Flag' : 'Tambah Feature Flag'}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Perbarui rollout percentage, aturan aktivasi, dan toggle fitur.'
              : 'Daftarkan flag baru untuk mengontrol peluncuran fitur secara bertahap.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {/* Key */}
          <div className="space-y-1">
            <Label htmlFor="flag-key" className="text-xs font-semibold">
              Key <span className="text-destructive">*</span>
            </Label>
            <Input
              id="flag-key"
              value={key}
              onChange={(e) => {
                setKey(e.target.value)
                setKeyError(null)
              }}
              placeholder="e.g. ai_voice_transcription"
              disabled={isEditing}
              className="font-mono text-xs"
              aria-label="Flag Key"
            />
            {keyError && (
              <p className="text-xs text-destructive font-medium">{keyError}</p>
            )}
          </div>

          {/* Name */}
          <div className="space-y-1">
            <Label htmlFor="flag-name" className="text-xs font-semibold">
              Nama Tampilan
            </Label>
            <Input
              id="flag-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. AI Voice Transcription"
              aria-label="Flag Name"
            />
          </div>

          {/* Description */}
          <div className="space-y-1">
            <Label htmlFor="flag-desc" className="text-xs font-semibold">
              Deskripsi
            </Label>
            <Input
              id="flag-desc"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Keterangan fungsi flag ini..."
              aria-label="Flag Description"
            />
          </div>

          {/* Enabled switch & Rollout */}
          <div className="grid grid-cols-2 gap-4 pt-2">
            <div className="flex flex-col justify-between p-3 border rounded-md border-border bg-card">
              <span className="text-xs font-semibold mb-2">Status Flag</span>
              <div className="flex items-center gap-3">
                <Switch
                  id="flag-enabled"
                  checked={enabled}
                  onCheckedChange={setEnabled}
                  aria-label="Toggle status flag"
                />
                <Label htmlFor="flag-enabled" className="text-xs cursor-pointer">
                  {enabled ? 'Aktif (Enabled)' : 'Nonaktif (Disabled)'}
                </Label>
              </div>
            </div>

            <div className="p-3 border rounded-md border-border bg-card space-y-1">
              <Label htmlFor="flag-rollout" className="text-xs font-semibold">
                Rollout ({rollout}%)
              </Label>
              <Input
                id="flag-rollout"
                type="number"
                min={0}
                max={100}
                step={1}
                value={rollout}
                onChange={(e) => {
                  setRollout(parseInt(e.target.value || '0', 10))
                  setRolloutError(null)
                }}
                className="h-8 text-xs"
                aria-label="Rollout percentage"
              />
              {rolloutError && (
                <p className="text-[11px] text-destructive font-medium">{rolloutError}</p>
              )}
            </div>
          </div>

          {/* Rules (JSON Textarea) */}
          <div className="space-y-1 pt-1">
            <div className="flex items-center justify-between">
              <Label htmlFor="flag-rules" className="text-xs font-semibold">
                Aturan Khusus (JSON Rules)
              </Label>
              <span className="text-[11px] text-muted-foreground">
                Kosongkan bila tidak ada aturan
              </span>
            </div>
            <textarea
              id="flag-rules"
              rows={4}
              value={rulesText}
              onChange={(e) => {
                setRulesText(e.target.value)
                setJsonError(null)
              }}
              placeholder='{\n  "plan": "pro",\n  "min_version": "2.1.0"\n}'
              className="w-full min-h-[90px] border-2 rounded-md border-foreground bg-background px-3 py-2 text-xs font-mono text-foreground outline-none focus:shadow-[2px_2px_0px_0px] focus:shadow-foreground"
              aria-label="Flag Rules JSON"
            />
            {jsonError && (
              <p className="text-xs text-destructive font-medium">{jsonError}</p>
            )}
          </div>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSaving}
            >
              Batal
            </Button>
            <Button type="submit" disabled={isSaving}>
              {isSaving ? 'Menyimpan...' : isEditing ? 'Perbarui Flag' : 'Tambah Flag'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
