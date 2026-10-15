'use client'

import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

/**
 * Dynamic configuration catalog (P-13).
 *
 * The settings/metrics/features catalogs are DATA (three DB tables), not a
 * hardcoded list compiled into the frontend. These hooks fetch them so the
 * Super Admin UI renders whatever the backend defines — adding a setting,
 * metric, or feature needs no frontend change.
 *
 * Freshness is event-driven: a `config_changed` SSE event invalidates these
 * queries immediately, so an operator sees their own change without a refresh.
 */

/** One setting definition, as returned by GET /admin/setting-definitions. */
export interface SettingDefinition {
  key: string
  kind: 'string' | 'int' | 'bool' | 'enum' | 'secret'
  min_value?: number
  max_value?: number
  enum_values?: string[]
  max_len: number
  default_value?: unknown
  group_key: string
  label: string
  description: string
  is_writable: boolean
  is_secret: boolean
  sort_order: number
  is_active: boolean
}

/** One metric definition, as returned by GET /admin/metric-definitions. */
export interface MetricDefinition {
  key: string
  label: string
  unit: string
  description: string
  is_enforced: boolean
  resolver_key?: string
  ratio_adjusted: boolean
  per_cycle: boolean
  group_key: string
  sort_order: number
  is_active: boolean
}

/** One feature definition, as returned by GET /admin/feature-definitions. */
export interface FeatureDefinition {
  key: string
  label: string
  description: string
  group_key: string
  sort_order: number
  is_active: boolean
}

/** One resolver capability, as returned by GET /admin/resolvers. */
export interface ResolverInfo {
  key: string
  label: string
  unit: string
  requires_period: boolean
}

export const configCatalogKeys = {
  settings: ['admin', 'setting-definitions'] as const,
  metrics: ['admin', 'metric-definitions'] as const,
  features: ['admin', 'feature-definitions'] as const,
  resolvers: ['admin', 'resolvers'] as const,
}

export function useSettingDefinitions() {
  const query = useQuery({
    queryKey: configCatalogKeys.settings,
    queryFn: async (): Promise<SettingDefinition[]> => {
      const res = await api.get('/admin/setting-definitions')
      return (res.data?.data ?? []) as SettingDefinition[]
    },
    staleTime: staleTimeConfig.admin,
  })
  return { definitions: query.data ?? [], isLoading: query.isLoading, isError: query.isError }
}

export function useMetricDefinitions() {
  const query = useQuery({
    queryKey: configCatalogKeys.metrics,
    queryFn: async (): Promise<MetricDefinition[]> => {
      const res = await api.get('/admin/metric-definitions')
      return (res.data?.data ?? []) as MetricDefinition[]
    },
    staleTime: staleTimeConfig.admin,
  })
  return { metrics: query.data ?? [], isLoading: query.isLoading, isError: query.isError }
}

export function useFeatureDefinitions() {
  const query = useQuery({
    queryKey: configCatalogKeys.features,
    queryFn: async (): Promise<FeatureDefinition[]> => {
      const res = await api.get('/admin/feature-definitions')
      return (res.data?.data ?? []) as FeatureDefinition[]
    },
    staleTime: staleTimeConfig.admin,
  })
  return { features: query.data ?? [], isLoading: query.isLoading, isError: query.isError }
}

/** useResolvers lists the usage-resolver capabilities the backend provides. */
export function useResolvers() {
  const query = useQuery({
    queryKey: configCatalogKeys.resolvers,
    queryFn: async (): Promise<ResolverInfo[]> => {
      const res = await api.get('/admin/resolvers')
      return (res.data?.data?.resolvers ?? []) as ResolverInfo[]
    },
    staleTime: staleTimeConfig.admin,
  })
  return { resolvers: query.data ?? [], isLoading: query.isLoading }
}
