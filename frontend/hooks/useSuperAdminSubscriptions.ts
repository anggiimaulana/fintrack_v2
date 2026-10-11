'use client'

import axios from 'axios'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface AdminSubscription {
  id: string
  user_id: string
  user_email: string
  user_full_name: string
  plan_id: string
  plan_code: string
  plan_name: string
  status: 'active' | 'past_due' | 'canceled' | 'expired' | string
  current_period_start: string
  current_period_end?: string | null
  cancel_at_period_end: boolean
  gateway: string
  created_at: string
}

export interface AdminPlanOption {
  id: string
  code: string
  name: string
  price_monthly_idr: number
  is_active: boolean
}

export interface SubscriptionListParams {
  page?: number
  per_page?: number
  q?: string
  sort?: string
  order?: 'asc' | 'desc'
  status?: string
  plan_code?: string
  user_id?: string
}

export interface ExtendSubscriptionPayload {
  days: number
  reason: string
}

export interface ForceDowngradePayload {
  plan_code: string
  reason: string
}

export interface CancelSubscriptionPayload {
  reason: string
  immediate: boolean
}

export const superAdminSubscriptionKeys = {
  all: ['admin', 'subscriptions'] as const,
  list: (params: SubscriptionListParams) => ['admin', 'subscriptions', 'list', params] as const,
  plans: ['admin', 'plans'] as const,
}

function getErrorMessage(err: unknown, fallback: string): string {
  if (axios.isAxiosError(err)) {
    const data = err.response?.data
    if (data && typeof data === 'object') {
      if ('message' in data && typeof data.message === 'string') return data.message
      if ('error' in data && typeof data.error === 'string') return data.error
    }
    if (err.message) return err.message
  } else if (err instanceof Error) {
    return err.message
  }
  return fallback
}

export function useSuperAdminSubscriptions(params: SubscriptionListParams = {}) {
  const queryClient = useQueryClient()

  const listQuery = useQuery({
    queryKey: superAdminSubscriptionKeys.list(params),
    queryFn: async () => {
      const cleanParams: Record<string, string | number> = {}
      if (params.page) cleanParams.page = params.page
      if (params.per_page) cleanParams.per_page = params.per_page
      if (params.q) cleanParams.q = params.q
      if (params.sort) cleanParams.sort = params.sort
      if (params.order) cleanParams.order = params.order
      if (params.status) cleanParams.status = params.status
      if (params.plan_code) cleanParams.plan_code = params.plan_code
      if (params.user_id) cleanParams.user_id = params.user_id

      const res = await api.get('/admin/subscriptions', { params: cleanParams })
      const data: AdminSubscription[] = res.data?.data || []
      const meta = res.data?.meta || {
        page: params.page || 1,
        per_page: params.per_page || 20,
        total: data.length,
        total_pages: Math.ceil(data.length / (params.per_page || 20)) || 1,
      }
      return { data, meta }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const plansQuery = useQuery({
    queryKey: superAdminSubscriptionKeys.plans,
    queryFn: async () => {
      const res = await api.get('/admin/plans')
      const data: AdminPlanOption[] = res.data?.data || []
      return data
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const handle409AndToast = (err: unknown, fallback: string) => {
    if (axios.isAxiosError(err) && err.response?.status === 409) {
      queryClient.invalidateQueries({ queryKey: superAdminSubscriptionKeys.all })
    }
    toast.error(getErrorMessage(err, fallback))
  }

  const extendMutation = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: ExtendSubscriptionPayload }) => {
      const res = await api.post(`/admin/subscriptions/${id}/extend`, payload)
      return res.data
    },
    onSuccess: (res) => {
      toast.success(res?.message || 'Langganan berhasil diperpanjang')
      queryClient.invalidateQueries({ queryKey: superAdminSubscriptionKeys.all })
    },
    onError: (err: unknown) => {
      handle409AndToast(err, 'Gagal memperpanjang langganan')
    },
  })

  const forceDowngradeMutation = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: ForceDowngradePayload }) => {
      const res = await api.post(`/admin/subscriptions/${id}/force-downgrade`, payload)
      return res.data
    },
    onSuccess: (res) => {
      toast.success(res?.message || 'Paket berhasil diturunkan')
      queryClient.invalidateQueries({ queryKey: superAdminSubscriptionKeys.all })
    },
    onError: (err: unknown) => {
      handle409AndToast(err, 'Gagal menurunkan paket')
    },
  })

  const cancelMutation = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: CancelSubscriptionPayload }) => {
      const res = await api.post(`/admin/subscriptions/${id}/cancel`, payload)
      return res.data
    },
    onSuccess: (res) => {
      toast.success(res?.message || 'Langganan berhasil dibatalkan')
      queryClient.invalidateQueries({ queryKey: superAdminSubscriptionKeys.all })
    },
    onError: (err: unknown) => {
      handle409AndToast(err, 'Gagal membatalkan langganan')
    },
  })

  return {
    subscriptions: listQuery.data?.data || [],
    meta: listQuery.data?.meta || { page: 1, per_page: 20, total: 0, total_pages: 0 },
    isLoading: listQuery.isLoading,
    isFetching: listQuery.isFetching,
    error: listQuery.error ? getErrorMessage(listQuery.error, 'Gagal memuat daftar langganan') : null,
    refetch: listQuery.refetch,
    plans: plansQuery.data || [],
    isLoadingPlans: plansQuery.isLoading,
    extendSubscription: extendMutation.mutateAsync,
    isExtending: extendMutation.isPending,
    forceDowngradeSubscription: forceDowngradeMutation.mutateAsync,
    isDowngrading: forceDowngradeMutation.isPending,
    cancelSubscription: cancelMutation.mutateAsync,
    isCanceling: cancelMutation.isPending,
  }
}
