'use client'

import * as React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import { Plus, Flag, AlertCircle, Edit, Sliders } from 'lucide-react'
import { FeatureFlag, UpdateFeatureFlagPayload } from '@/hooks/useSuperAdminSettings'
import { FeatureFlagDialog } from './FeatureFlagDialog'

interface FeatureFlagsTabProps {
  flags: FeatureFlag[]
  isLoading: boolean
  error: unknown
  onSave: (payload: UpdateFeatureFlagPayload) => Promise<unknown>
  isSaving: boolean
  onRefresh: () => void
}

export function FeatureFlagsTab({
  flags,
  isLoading,
  error,
  onSave,
  isSaving,
  onRefresh,
}: FeatureFlagsTabProps) {
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [selectedFlag, setSelectedFlag] = React.useState<FeatureFlag | null>(null)

  const handleAdd = () => {
    setSelectedFlag(null)
    setDialogOpen(true)
  }

  const handleEdit = (flag: FeatureFlag) => {
    setSelectedFlag(flag)
    setDialogOpen(true)
  }

  const handleQuickToggle = async (flag: FeatureFlag, newEnabled: boolean) => {
    await onSave({
      key: flag.key,
      name: flag.name,
      description: flag.description,
      enabled: newEnabled,
      rollout: flag.rollout,
      rules: flag.rules,
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-9 w-32" />
        </div>
        <Card className="border-border">
          <CardContent className="p-6 space-y-4">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <Card className="border-destructive/50 bg-destructive/5 p-6 text-center">
        <AlertCircle className="mx-auto size-8 text-destructive mb-2" />
        <h3 className="font-semibold text-foreground mb-1">Gagal Memuat Feature Flags</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Terjadi kesalahan saat memuat daftar feature flag dari server.
        </p>
        <Button variant="outline" onClick={onRefresh}>
          Coba Lagi
        </Button>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-bold text-foreground">Feature Flags</h2>
          <p className="text-sm text-muted-foreground">
            Kontrol peluncuran fitur dinamis, aturan segmentasi pengguna, dan persentase rollout.
          </p>
        </div>
        <Button onClick={handleAdd}>
          <Plus className="size-4 mr-2" />
          Tambah Flag
        </Button>
      </div>

      {flags.length === 0 ? (
        <Card className="border-border p-12 text-center">
          <Flag className="mx-auto size-10 text-muted-foreground/60 mb-3" />
          <h3 className="font-semibold text-foreground mb-1">Belum ada feature flag</h3>
          <p className="text-sm text-muted-foreground max-w-sm mx-auto mb-4">
            Sistem belum memiliki flag aktif. Buat flag pertama Anda untuk mengontrol peluncuran fitur aplikasi.
          </p>
          <Button variant="outline" onClick={handleAdd}>
            <Plus className="size-4 mr-2" />
            Buat Feature Flag
          </Button>
        </Card>
      ) : (
        <div className="border rounded-md border-border divide-y divide-border bg-card">
          {flags.map((flag) => {
            const isEnabled = flag.enabled ?? flag.is_enabled ?? false
            return (
              <div
                key={flag.key}
                className="p-5 flex flex-col md:flex-row md:items-center justify-between gap-4 hover:bg-muted/10 transition-colors"
              >
                <div className="space-y-1.5 flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-bold text-foreground text-sm">
                      {flag.name}
                    </span>
                    <code className="text-xs font-mono bg-muted px-1.5 py-0.5 border rounded-md border-border text-muted-foreground">
                      {flag.key}
                    </code>
                    <span
                      className={`text-[11px] px-2 py-0.5 font-semibold border rounded-md ${
                        isEnabled
                          ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/30'
                          : 'bg-muted text-muted-foreground border-border'
                      }`}
                    >
                      {isEnabled ? 'Aktif' : 'Nonaktif'}
                    </span>
                    <span className="text-[11px] px-2 py-0.5 font-mono bg-primary/10 text-primary-strong border rounded-md border-primary/20">
                      Rollout {flag.rollout}%
                    </span>
                  </div>

                  {flag.description && (
                    <p className="text-xs text-muted-foreground">
                      {flag.description}
                    </p>
                  )}

                  {flag.rules && Object.keys(flag.rules).length > 0 && (
                    <div className="pt-1">
                      <div className="inline-flex items-center gap-1.5 text-[11px] font-mono text-muted-foreground bg-muted/40 px-2 py-1 border rounded-md border-border max-w-full overflow-hidden truncate">
                        <Sliders className="size-3 shrink-0" />
                        <span>Rules: {JSON.stringify(flag.rules)}</span>
                      </div>
                    </div>
                  )}
                </div>

                <div className="flex items-center gap-4 self-end md:self-center shrink-0">
                  <div className="flex items-center gap-2">
                    <Switch
                      id={`switch-${flag.key}`}
                      checked={isEnabled}
                      onCheckedChange={(val) => handleQuickToggle(flag, val)}
                      disabled={isSaving}
                      aria-label={`Toggle flag ${flag.key}`}
                    />
                    <label
                      htmlFor={`switch-${flag.key}`}
                      className="text-xs font-medium cursor-pointer"
                    >
                      {isEnabled ? 'ON' : 'OFF'}
                    </label>
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleEdit(flag)}
                    disabled={isSaving}
                    aria-label={`Edit flag ${flag.key}`}
                  >
                    <Edit className="size-3.5 mr-1.5" />
                    Edit
                  </Button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <FeatureFlagDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        flag={selectedFlag}
        onSave={onSave}
        isSaving={isSaving}
      />
    </div>
  )
}
