'use client'

import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, useCallback } from 'react'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export type MetricsPeriod = '7d' | '30d' | '90d'

export interface OverviewMetrics {
  mrr_idr: number
  mrr_growth_percentage: number
  active_users_30d: number
  active_users_growth_percentage: number
  tokens_consumed_24h: number
  tokens_growth_percentage: number
  payment_success_rate: number
  pending_manual_payments: number
  ai_endpoints_healthy: number
  ai_endpoints_total: number
  circuit_breaker_open_count: number
  cached_at: string
}

export interface TimeseriesPoint {
  date: string
  revenue_idr: number
  payments_settled: number
  mrr_idr: number
  tokens_chat: number
  tokens_vision: number
  tokens_voice: number
  tokens_embedding: number
}

export interface PaymentMethodShare {
  gateway: string
  settled_count: number
  total_idr?: number
  amount_idr?: number
}

export const superAdminQueryKeys = {
  overview: (refresh = false) => ['admin', 'metrics', 'overview', { refresh }] as const,
  timeseries: (period: MetricsPeriod) => ['admin', 'metrics', 'timeseries', period] as const,
  paymentMethods: () => ['admin', 'metrics', 'payment-methods'] as const,
}

export function useSuperAdminMetrics(initialPeriod: MetricsPeriod = '30d') {
  const [period, setPeriod] = useState<MetricsPeriod>(initialPeriod)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const queryClient = useQueryClient()

  const overviewQuery = useQuery({
    queryKey: superAdminQueryKeys.overview(false),
    queryFn: async () => {
      const res = await api.get('/admin/metrics/overview')
      return res.data?.data as OverviewMetrics
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const timeseriesQuery = useQuery({
    queryKey: superAdminQueryKeys.timeseries(period),
    queryFn: async () => {
      const res = await api.get(`/admin/metrics/timeseries?period=${period}`)
      return (res.data?.data || []) as TimeseriesPoint[]
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const paymentMethodsQuery = useQuery({
    queryKey: superAdminQueryKeys.paymentMethods(),
    queryFn: async () => {
      const res = await api.get('/admin/metrics/payment-methods')
      return (res.data?.data || []) as PaymentMethodShare[]
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const refreshOverview = useCallback(async () => {
    setIsRefreshing(true)
    try {
      const res = await api.get('/admin/metrics/overview?refresh=true')
      const updated = res.data?.data as OverviewMetrics
      queryClient.setQueryData(superAdminQueryKeys.overview(false), updated)
      return updated
    } finally {
      setIsRefreshing(false)
    }
  }, [queryClient])

  return {
    period,
    setPeriod,
    overview: overviewQuery.data,
    isOverviewLoading: overviewQuery.isLoading,
    overviewError: overviewQuery.error,
    refreshOverview,
    isRefreshing,
    timeseries: timeseriesQuery.data,
    isTimeseriesLoading: timeseriesQuery.isLoading,
    timeseriesError: timeseriesQuery.error,
    paymentMethods: paymentMethodsQuery.data,
    isPaymentMethodsLoading: paymentMethodsQuery.isLoading,
    paymentMethodsError: paymentMethodsQuery.error,
    isLoading: overviewQuery.isLoading || timeseriesQuery.isLoading || paymentMethodsQuery.isLoading,
    refetchAll: () => {
      overviewQuery.refetch()
      timeseriesQuery.refetch()
      paymentMethodsQuery.refetch()
    },
  }
}
