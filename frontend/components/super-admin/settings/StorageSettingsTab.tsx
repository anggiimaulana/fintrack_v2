'use client'

import * as React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { HardDrive, Brain } from 'lucide-react'
import { SecretField } from './SecretField'
import { useConfigStatus } from '@/hooks/use-config-status'

interface StorageSettingsTabProps {
  isLoading: boolean
  onSave: (dirtySettings: Record<string, unknown>) => Promise<unknown>
  isSaving: boolean
}

interface StorageForm {
  storage_driver: string
  r2_account_id: string
  r2_access_key_id: string
  r2_secret_access_key: string
  r2_bucket: string
  r2_region: string
  r2_endpoint: string
  r2_prefix: string
  r2_public_base_url: string
  ai_reasoning_effort: string
  gemini_embedding_api_key: string
}

const EMPTY: StorageForm = {
  storage_driver: 'local',
  r2_account_id: '',
  r2_access_key_id: '',
  r2_secret_access_key: '',
  r2_bucket: '',
  r2_region: 'auto',
  r2_endpoint: '',
  r2_prefix: '',
  r2_public_base_url: '',
  ai_reasoning_effort: '',
  gemini_embedding_api_key: '',
}

function fromStatus(effective: Record<string, unknown>): StorageForm {
  const s = (k: string) => (typeof effective[k] === 'string' ? (effective[k] as string) : '')
  return {
    storage_driver: s('storage_driver') || 'local',
    r2_account_id: s('r2_account_id'),
    r2_access_key_id: s('r2_access_key_id'),
    r2_secret_access_key: '', // write-only
    r2_bucket: s('r2_bucket'),
    r2_region: s('r2_region') || 'auto',
    r2_endpoint: s('r2_endpoint'),
    r2_prefix: s('r2_prefix'),
    r2_public_base_url: s('r2_public_base_url'),
    ai_reasoning_effort: s('ai_reasoning_effort'),
    gemini_embedding_api_key: '', // write-only
  }
}

const REASONING_OPTIONS = [
  { value: '', label: 'Tidak dikirim (default)' },
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
]

/**
 * Object storage (R2) and AI knobs. R2 credentials resolve at runtime, so a
 * change applies without a restart; the `storage_driver` setting selects whether
 * R2 is actually used (local remains the safe fallback).
 */
export function StorageSettingsTab({ isLoading, onSave, isSaving }: StorageSettingsTabProps) {
  const { data: status, isLoading: statusLoading } = useConfigStatus()
  const [edits, setEdits] = React.useState<Partial<StorageForm>>({})
  const [touched, setTouched] = React.useState(false)

  const baseline = status ? fromStatus(status.effective) : EMPTY
  const form: StorageForm = { ...baseline, ...edits }
  const patch = (next: Partial<StorageForm>) => {
    setEdits((prev) => ({ ...prev, ...next }))
    setTouched(true)
  }

  const handleSave = async () => {
    const payload: Record<string, unknown> = {
      storage_driver: form.storage_driver,
      r2_account_id: form.r2_account_id.trim(),
      r2_access_key_id: form.r2_access_key_id.trim(),
      r2_bucket: form.r2_bucket.trim(),
      r2_region: form.r2_region.trim(),
      r2_endpoint: form.r2_endpoint.trim(),
      r2_prefix: form.r2_prefix.trim(),
      r2_public_base_url: form.r2_public_base_url.trim(),
      ai_reasoning_effort: form.ai_reasoning_effort,
    }
    // Write-only secrets: only send when the operator typed a new value.
    if (form.r2_secret_access_key.trim() !== '') payload.r2_secret_access_key = form.r2_secret_access_key
    if (form.gemini_embedding_api_key.trim() !== '') payload.gemini_embedding_api_key = form.gemini_embedding_api_key
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
            <HardDrive className="size-4" /> Penyimpanan Objek (Cloudflare R2)
          </CardTitle>
          <CardDescription>
            Kredensial R2 di sini menimpa default environment. Driver aktif dipilih di bawah.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5 sm:max-w-xs">
            <Label htmlFor="storage_driver">Driver Aktif</Label>
            <Select value={form.storage_driver} onValueChange={(v) => patch({ storage_driver: v })}>
              <SelectTrigger id="storage_driver">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="local">Local (server)</SelectItem>
                <SelectItem value="r2">Cloudflare R2</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="r2_account_id">Account ID</Label>
              <Input id="r2_account_id" value={form.r2_account_id} onChange={(e) => patch({ r2_account_id: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r2_bucket">Bucket</Label>
              <Input id="r2_bucket" value={form.r2_bucket} onChange={(e) => patch({ r2_bucket: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r2_access_key_id">Access Key ID</Label>
              <Input id="r2_access_key_id" value={form.r2_access_key_id} onChange={(e) => patch({ r2_access_key_id: e.target.value })} />
            </div>
            <SecretField
              id="r2_secret_access_key"
              label="Secret Access Key"
              value={form.r2_secret_access_key}
              onChange={(v) => patch({ r2_secret_access_key: v })}
              masked={status?.secrets?.r2_secret_access_key}
              hasSecret={status?.has_secret?.r2_secret_access_key}
              source={status?.source?.r2_secret_access_key}
            />
            <div className="space-y-1.5">
              <Label htmlFor="r2_region">Region</Label>
              <Input id="r2_region" value={form.r2_region} onChange={(e) => patch({ r2_region: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r2_endpoint">Endpoint (opsional)</Label>
              <Input id="r2_endpoint" value={form.r2_endpoint} onChange={(e) => patch({ r2_endpoint: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r2_prefix">Prefix (opsional)</Label>
              <Input id="r2_prefix" value={form.r2_prefix} onChange={(e) => patch({ r2_prefix: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="r2_public_base_url">Public Base URL (opsional)</Label>
              <Input id="r2_public_base_url" value={form.r2_public_base_url} onChange={(e) => patch({ r2_public_base_url: e.target.value })} />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Brain className="size-4" /> AI
          </CardTitle>
          <CardDescription>Parameter model dan embedding yang dipakai saat runtime.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5 sm:max-w-xs">
            <Label htmlFor="ai_reasoning_effort">Reasoning Effort (mode PENALARAN)</Label>
            <Select
              value={form.ai_reasoning_effort}
              onValueChange={(v) => patch({ ai_reasoning_effort: v })}
            >
              <SelectTrigger id="ai_reasoning_effort">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REASONING_OPTIONS.map((o) => (
                  <SelectItem key={o.value || 'none'} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-[11px] font-bold text-muted-foreground">
              Sebagian provider menolak field ini; biarkan default bila ragu.
            </p>
          </div>

          <SecretField
            id="gemini_embedding_api_key"
            label="Gemini Embedding API Key (legacy RAG)"
            value={form.gemini_embedding_api_key}
            onChange={(v) => patch({ gemini_embedding_api_key: v })}
            masked={status?.secrets?.gemini_embedding_api_key}
            hasSecret={status?.has_secret?.gemini_embedding_api_key}
            source={status?.source?.gemini_embedding_api_key}
          />
        </CardContent>
      </Card>

      <Button onClick={handleSave} disabled={isSaving || !touched}>
        {isSaving ? 'Menyimpan...' : 'Simpan Konfigurasi Penyimpanan & AI'}
      </Button>
    </div>
  )
}
