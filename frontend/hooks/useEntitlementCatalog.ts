'use client'

import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

/** One feature key in the entitlement catalog. */
export interface CatalogFeature {
  key: string
  label: string
  description: string
}

/** One numeric limit key in the entitlement catalog. */
export interface CatalogLimit {
  key: string
  label: string
  unit: string
  /** Whether the backend actually computes usage for this key. */
  enforced: boolean
}

export interface EntitlementCatalog {
  features: CatalogFeature[]
  limits: CatalogLimit[]
}

export const entitlementCatalogKeys = {
  all: ['admin', 'entitlement-catalog'] as const,
}

/**
 * Reads the authoritative entitlement key catalog (GET /admin/entitlement-catalog).
 *
 * The admin plan editor uses it to offer known keys and — importantly — to warn
 * when a limit the operator is configuring is NOT enforced by the backend, so a
 * plan never advertises a benefit the API cannot apply (F4).
 */
export function useEntitlementCatalog() {
  const query = useQuery({
    queryKey: entitlementCatalogKeys.all,
    queryFn: async (): Promise<EntitlementCatalog> => {
      const res = await api.get('/admin/entitlement-catalog')
      const data = res.data?.data
      const features = Array.isArray(data?.features) ? (data.features as CatalogFeature[]) : []
      const limits = Array.isArray(data?.limits) ? (data.limits as CatalogLimit[]) : []
      return { features, limits }
    },
    staleTime: staleTimeConfig.admin,
  })
  return {
    catalog: query.data ?? { features: [], limits: [] },
    isLoading: query.isLoading,
  }
}
