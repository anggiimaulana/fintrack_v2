'use client'

import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

/** One user's consumption against a single numeric plan limit. */
export interface QuotaMonitorRow {
  user_id: string
  email: string
  full_name: string
  plan_code: string
  plan_name: string
  limit_key: string
  used: number
  limit: number
  used_percent: number
  remaining: number
}

export const quotaMonitorKeys = {
  all: ['admin', 'quota-monitor'] as const,
  byThreshold: (threshold: number) => ['admin', 'quota-monitor', threshold] as const,
}

/**
 * Users whose consumption of a numeric limit has crossed `threshold` percent.
 *
 * Backed by GET /admin/quota-monitor?threshold=N, which reads the same counters
 * the enforcement middleware uses — so this view reflects exactly what will
 * block a user.
 */
export function useSuperAdminQuotaMonitor(threshold: number) {
  const query = useQuery({
    queryKey: quotaMonitorKeys.byThreshold(threshold),
    queryFn: async (): Promise<QuotaMonitorRow[]> => {
      const res = await api.get('/admin/quota-monitor', { params: { threshold } })
      const data = res.data?.data
      return Array.isArray(data) ? (data as QuotaMonitorRow[]) : []
    },
    staleTime: staleTimeConfig.admin,
  })
  return {
    rows: query.data ?? [],
    isLoading: query.isLoading,
    isError: query.isError,
    refetch: query.refetch,
  }
}
