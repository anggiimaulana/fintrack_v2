'use client'

import * as React from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Settings, Flag } from 'lucide-react'
import { useSuperAdminSettings } from '@/hooks/useSuperAdminSettings'
import { useSuperAdminStorage } from '@/hooks/useSuperAdminStorage'
import { GlobalSettingsTab } from '@/components/super-admin/settings/GlobalSettingsTab'
import { FeatureFlagsTab } from '@/components/super-admin/settings/FeatureFlagsTab'

export default function SuperAdminSettingsPage() {
  const {
    // Global Settings
    settings,
    isLoadingSettings,
    settingsError,
    refetchSettings,
    updateSettings,
    isSavingSettings,

    // Feature Flags
    featureFlags,
    isLoadingFeatureFlags,
    featureFlagsError,
    refetchFeatureFlags,
    updateFeatureFlag,
    isSavingFeatureFlag,
  } = useSuperAdminSettings()

  const { storageStatus } = useSuperAdminStorage()

  return (
    <div className="space-y-6">
      <div className="border-b border-border pb-5">
        <h1 className="text-2xl font-bold tracking-tight text-foreground">
          Pengaturan Sistem
        </h1>
        <p className="text-sm text-muted-foreground mt-1">
          Kelola parameter operasional global dan status rollout feature flags platform FinTrack.
        </p>
      </div>

      <Tabs defaultValue="settings" className="space-y-6">
        <TabsList className="grid w-full grid-cols-2 max-w-xs">
          <TabsTrigger value="settings" className="flex items-center gap-2">
            <Settings className="size-4" />
            <span>Pengaturan Global</span>
          </TabsTrigger>
          <TabsTrigger value="flags" className="flex items-center gap-2">
            <Flag className="size-4" />
            <span>Feature Flags</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="settings" className="space-y-4 focus-visible:outline-none">
          <GlobalSettingsTab
            settings={settings}
            isLoading={isLoadingSettings}
            error={settingsError}
            onSave={updateSettings}
            isSaving={isSavingSettings}
            onRefresh={refetchSettings}
            storageStatus={storageStatus}
          />
        </TabsContent>

        <TabsContent value="flags" className="space-y-4 focus-visible:outline-none">
          <FeatureFlagsTab
            flags={featureFlags}
            isLoading={isLoadingFeatureFlags}
            error={featureFlagsError}
            onSave={updateFeatureFlag}
            isSaving={isSavingFeatureFlag}
            onRefresh={refetchFeatureFlags}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
