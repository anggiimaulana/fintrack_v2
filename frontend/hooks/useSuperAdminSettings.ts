'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface SettingRow {
  key: string
  value: unknown
  description?: string
  updated_by?: string | null
  updated_at: string
}

export interface FeatureFlag {
  key: string
  name: string
  description: string
  is_enabled?: boolean
  enabled?: boolean
  rollout: number
  rules: Record<string, unknown> | null
}

export interface UpdateFeatureFlagPayload {
  key: string
  name: string
  description: string
  enabled: boolean
  rollout: number
  rules: Record<string, unknown> | null
}

export const settingsQueryKeys = {
  settings: ['admin', 'settings'] as const,
  featureFlags: ['admin', 'feature-flags'] as const,
}

export function useSuperAdminSettings() {
  const queryClient = useQueryClient()

  const settingsQuery = useQuery({
    queryKey: settingsQueryKeys.settings,
    queryFn: async (): Promise<SettingRow[]> => {
      const res = await api.get('/admin/settings')
      const payload: unknown = res.data?.data
      if (Array.isArray(payload)) {
        const rows: SettingRow[] = []
        for (const item of payload) {
          if (item && typeof item === 'object' && 'key' in item && 'updated_at' in item) {
            const key = typeof item.key === 'string' ? item.key : ''
            const updatedAt = typeof item.updated_at === 'string' ? item.updated_at : ''
            const desc = 'description' in item && typeof item.description === 'string' ? item.description : undefined
            const updatedBy = 'updated_by' in item && typeof item.updated_by === 'string' ? item.updated_by : null
            const value = 'value' in item ? item.value : null
            rows.push({
              key,
              value,
              description: desc,
              updated_by: updatedBy,
              updated_at: updatedAt,
            })
          }
        }
        return rows
      }
      return []
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const updateSettingsMutation = useMutation({
    mutationFn: async (dirtySettings: Record<string, unknown>) => {
      const res = await api.put('/admin/settings', { settings: dirtySettings })
      return res.data
    },
    onSuccess: (res) => {
      let msg = 'Pengaturan berhasil disimpan'
      if (res && typeof res === 'object' && 'message' in res && typeof res.message === 'string') {
        msg = res.message
      }
      toast.success(msg)
      queryClient.invalidateQueries({ queryKey: settingsQueryKeys.settings })
      // A storage_driver change alters the active driver, so refresh its status.
      queryClient.invalidateQueries({ queryKey: ['admin', 'storage-status'] })
      // Runtime-resolved config (payment, R2, mail, AI) changed: refresh status.
      queryClient.invalidateQueries({ queryKey: ['admin', 'config-status'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'mail-status'] })
    },
    onError: (err: unknown) => {
      let msg = 'Gagal menyimpan pengaturan'
      if (err && typeof err === 'object' && 'response' in err) {
        const resp = err.response
        if (resp && typeof resp === 'object' && 'data' in resp) {
          const data = resp.data
          if (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string') {
            msg = data.message
          }
        }
      } else if (err instanceof Error) {
        msg = err.message
      }
      toast.error(msg)
    },
  })

  const featureFlagsQuery = useQuery({
    queryKey: settingsQueryKeys.featureFlags,
    queryFn: async (): Promise<FeatureFlag[]> => {
      const res = await api.get('/admin/feature-flags')
      const payload: unknown = res.data?.data
      if (!Array.isArray(payload)) return []
      const flags: FeatureFlag[] = []
      for (const item of payload) {
        if (item && typeof item === 'object') {
          const key = 'key' in item && typeof item.key === 'string' ? item.key : ''
          const name = 'name' in item && typeof item.name === 'string' && item.name.length > 0 ? item.name : key
          const desc = 'description' in item && typeof item.description === 'string' ? item.description : ''
          const isEnabled = 'is_enabled' in item && typeof item.is_enabled === 'boolean'
            ? item.is_enabled
            : 'enabled' in item && typeof item.enabled === 'boolean'
              ? item.enabled
              : false
          const rollout = 'rollout' in item && typeof item.rollout === 'number' ? item.rollout : 0
          let rules: Record<string, unknown> | null = null
          if ('rules' in item && item.rules && typeof item.rules === 'object' && !Array.isArray(item.rules)) {
            rules = item.rules as Record<string, unknown>
          }
          flags.push({
            key,
            name,
            description: desc,
            enabled: isEnabled,
            is_enabled: isEnabled,
            rollout,
            rules,
          })
        }
      }
      return flags
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const updateFeatureFlagMutation = useMutation({
    mutationFn: async (payload: UpdateFeatureFlagPayload) => {
      const res = await api.put('/admin/feature-flags', payload)
      return res.data
    },
    onSuccess: (res) => {
      let msg = 'Feature flag berhasil disimpan'
      if (res && typeof res === 'object' && 'message' in res && typeof res.message === 'string') {
        msg = res.message
      }
      toast.success(msg)
      queryClient.invalidateQueries({ queryKey: settingsQueryKeys.featureFlags })
    },
    onError: (err: unknown) => {
      let msg = 'Gagal menyimpan feature flag'
      if (err && typeof err === 'object' && 'response' in err) {
        const resp = err.response
        if (resp && typeof resp === 'object' && 'data' in resp) {
          const data = resp.data
          if (data && typeof data === 'object' && 'message' in data && typeof data.message === 'string') {
            msg = data.message
          }
        }
      } else if (err instanceof Error) {
        msg = err.message
      }
      toast.error(msg)
    },
  })

  return {
    // Settings
    settings: settingsQuery.data || [],
    isLoadingSettings: settingsQuery.isLoading,
    isFetchingSettings: settingsQuery.isFetching,
    settingsError: settingsQuery.error,
    refetchSettings: settingsQuery.refetch,
    updateSettings: updateSettingsMutation.mutateAsync,
    isSavingSettings: updateSettingsMutation.isPending,

    // Feature Flags
    featureFlags: featureFlagsQuery.data || [],
    isLoadingFeatureFlags: featureFlagsQuery.isLoading,
    isFetchingFeatureFlags: featureFlagsQuery.isFetching,
    featureFlagsError: featureFlagsQuery.error,
    refetchFeatureFlags: featureFlagsQuery.refetch,
    updateFeatureFlag: updateFeatureFlagMutation.mutateAsync,
    isSavingFeatureFlag: updateFeatureFlagMutation.isPending,
  }
}
