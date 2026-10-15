'use client'

import * as React from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Settings, Flag, Mail, CreditCard, HardDrive, Mic, Timer, Gauge } from 'lucide-react'
import { useSuperAdminSettings } from '@/hooks/useSuperAdminSettings'
import { useSuperAdminStorage } from '@/hooks/useSuperAdminStorage'
import { useHash } from '@/hooks/use-hash'
import { GlobalSettingsTab } from '@/components/super-admin/settings/GlobalSettingsTab'
import { QuotaSettingsTab } from '@/components/super-admin/settings/QuotaSettingsTab'
import { AITimeoutSettingsTab } from '@/components/super-admin/settings/AITimeoutSettingsTab'
import { FeatureFlagsTab } from '@/components/super-admin/settings/FeatureFlagsTab'
import { MailSettingsTab } from '@/components/super-admin/settings/MailSettingsTab'
import { PaymentSettingsTab } from '@/components/super-admin/settings/PaymentSettingsTab'
import { StorageSettingsTab } from '@/components/super-admin/settings/StorageSettingsTab'
import { VoiceSettingsTab } from '@/components/super-admin/settings/VoiceSettingsTab'

// Valid tab ids, in display order. A deep link such as
// /super-admin/settings#kuota opens the matching tab directly.
const SETTINGS_TABS = [
  'settings',
  'kuota',
  'ai-timeout',
  'voice',
  'payment',
  'storage',
  'mail',
  'flags',
] as const

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

  // Deep-link support: /super-admin/settings#kuota opens the "Kuota" tab. The
  // tab is controlled state so the hash drives it, and switching a tab writes
  // the hash back (so the URL is shareable and the back button works).
  const hash = useHash()
  const [tab, setTab] = React.useState<string>(
    (SETTINGS_TABS as readonly string[]).includes(hash) ? hash : 'settings',
  )
  React.useEffect(() => {
    // A valid hash opens that tab; an empty/invalid hash falls back to the
    // default tab so clicking the bare "Pengaturan" entry (which clears the
    // hash) returns the user to the Global tab.
    if (hash && (SETTINGS_TABS as readonly string[]).includes(hash)) {
      setTab(hash)
    } else if (!hash) {
      setTab('settings')
    }
  }, [hash])

  const onTabChange = (value: string) => {
    setTab(value)
    if (typeof window !== 'undefined') {
      window.history.replaceState(null, '', `#${value}`)
    }
  }

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

      <Tabs value={tab} onValueChange={onTabChange} className="space-y-6">
        <TabsList className="grid w-full grid-cols-2 sm:grid-cols-3 lg:grid-cols-7 max-w-5xl">
          <TabsTrigger value="settings" className="flex items-center gap-2">
            <Settings className="size-4" />
            <span>Global</span>
          </TabsTrigger>
          <TabsTrigger value="kuota" className="flex items-center gap-2">
            <Gauge className="size-4" />
            <span>Kuota</span>
          </TabsTrigger>
          <TabsTrigger value="ai-timeout" className="flex items-center gap-2">
            <Timer className="size-4" />
            <span>Batas Waktu AI</span>
          </TabsTrigger>
          <TabsTrigger value="voice" className="flex items-center gap-2">
            <Mic className="size-4" />
            <span>Suara</span>
          </TabsTrigger>
          <TabsTrigger value="payment" className="flex items-center gap-2">
            <CreditCard className="size-4" />
            <span>Pembayaran</span>
          </TabsTrigger>
          <TabsTrigger value="storage" className="flex items-center gap-2">
            <HardDrive className="size-4" />
            <span>Penyimpanan</span>
          </TabsTrigger>
          <TabsTrigger value="mail" className="flex items-center gap-2">
            <Mail className="size-4" />
            <span>Email</span>
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

        <TabsContent value="kuota" className="space-y-4 focus-visible:outline-none">
          <QuotaSettingsTab
            settings={settings}
            isLoading={isLoadingSettings}
            onSave={updateSettings}
            isSaving={isSavingSettings}
          />
        </TabsContent>
        <TabsContent value="ai-timeout" className="space-y-4 focus-visible:outline-none">
          <AITimeoutSettingsTab
            settings={settings}
            isLoading={isLoadingSettings}
            error={settingsError}
            onSave={updateSettings}
            isSaving={isSavingSettings}
            onRefresh={refetchSettings}
          />
        </TabsContent>

        <TabsContent value="voice" className="space-y-4 focus-visible:outline-none">
          <VoiceSettingsTab
            settings={settings}
            isLoading={isLoadingSettings}
            error={settingsError}
            onSave={updateSettings}
            isSaving={isSavingSettings}
            onRefresh={refetchSettings}
          />
        </TabsContent>

        <TabsContent value="payment" className="space-y-4 focus-visible:outline-none">
          <PaymentSettingsTab
            isLoading={isLoadingSettings}
            onSave={updateSettings}
            isSaving={isSavingSettings}
          />
        </TabsContent>

        <TabsContent value="storage" className="space-y-4 focus-visible:outline-none">
          <StorageSettingsTab
            isLoading={isLoadingSettings}
            onSave={updateSettings}
            isSaving={isSavingSettings}
          />
        </TabsContent>

        <TabsContent value="mail" className="space-y-4 focus-visible:outline-none">
          <MailSettingsTab
            settings={settings}
            isLoading={isLoadingSettings}
            onSave={updateSettings}
            isSaving={isSavingSettings}
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
