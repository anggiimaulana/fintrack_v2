'use client'

import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

/** Health/config of the proof-storage drivers (from GET /admin/settings/storage). */
export interface StorageStatusInfo {
  active: string
  default: string
  local_ready: boolean
  r2_configured: boolean
  r2_healthy: boolean
  r2_bucket?: string
  r2_endpoint?: string
  r2_public_url?: string
  local_root?: string
}

export interface StorageStatusResponse {
  status: StorageStatusInfo
  drivers: string[]
}

export const storageQueryKeys = {
  status: ['admin', 'storage-status'] as const,
}

function str(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function bool(value: unknown): boolean {
  return value === true
}

/** Validates the storage-status payload at the network boundary. */
function parseStorageStatus(raw: unknown): StorageStatusResponse | null {
  if (typeof raw !== 'object' || raw === null) return null
  if (!('status' in raw) || !('drivers' in raw)) return null

  const status = raw.status
  if (typeof status !== 'object' || status === null) return null
  if (!('active' in status) || typeof status.active !== 'string') return null
  if (!('default' in status) || typeof status.default !== 'string') return null

  const drivers = Array.isArray(raw.drivers) ? raw.drivers.filter((d): d is string => typeof d === 'string') : []

  return {
    status: {
      active: status.active,
      default: status.default,
      local_ready: 'local_ready' in status ? bool(status.local_ready) : false,
      r2_configured: 'r2_configured' in status ? bool(status.r2_configured) : false,
      r2_healthy: 'r2_healthy' in status ? bool(status.r2_healthy) : false,
      r2_bucket: 'r2_bucket' in status ? str(status.r2_bucket) : undefined,
      r2_endpoint: 'r2_endpoint' in status ? str(status.r2_endpoint) : undefined,
      r2_public_url: 'r2_public_url' in status ? str(status.r2_public_url) : undefined,
      local_root: 'local_root' in status ? str(status.local_root) : undefined,
    },
    drivers,
  }
}

/** Reads the current proof-storage status so the settings page can show it. */
export function useSuperAdminStorage() {
  const query = useQuery({
    queryKey: storageQueryKeys.status,
    queryFn: async (): Promise<StorageStatusResponse | null> => {
      const res = await api.get('/admin/settings/storage')
      return parseStorageStatus(res.data?.data)
    },
    staleTime: staleTimeConfig.admin,
  })
  return {
    storageStatus: query.data?.status ?? null,
    drivers: query.data?.drivers ?? [],
    isLoading: query.isLoading,
  }
}
