'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface PlanDTO {
  id: string
  code: string
  name: string
  description?: string | null
  price_monthly_idr: number
  price_yearly_idr: number
  billing_period_days: number
  is_active: boolean
  is_public: boolean
  fallback_gateway: string
  sort_order: number
  entitlements: Record<string, boolean>
  limits: Record<string, number>
}

export interface PlanPayload {
  code: string
  name: string
  description?: string | null
  price_monthly_idr?: number
  price_yearly_idr?: number
  billing_period_days?: number
  is_active?: boolean
  is_public?: boolean
  fallback_gateway?: string
  sort_order?: number
  entitlements?: Record<string, boolean>
  limits?: Record<string, number>
}

export const superAdminPlansQueryKeys = {
  all: ['admin', 'plans'] as const,
}

export function useSuperAdminPlans() {
  const queryClient = useQueryClient()

  const plansQuery = useQuery({
    queryKey: superAdminPlansQueryKeys.all,
    queryFn: async () => {
      const res = await api.get('/admin/plans')
      return (res.data?.data || []) as PlanDTO[]
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const createPlanMutation = useMutation({
    mutationFn: async (payload: PlanPayload) => {
      const res = await api.post('/admin/plans', payload)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: superAdminPlansQueryKeys.all })
    },
  })

  const updatePlanMutation = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: PlanPayload }) => {
      const res = await api.put(`/admin/plans/${id}`, payload)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: superAdminPlansQueryKeys.all })
    },
  })

  const deactivatePlanMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.delete(`/admin/plans/${id}`)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: superAdminPlansQueryKeys.all })
    },
  })

  return {
    plans: plansQuery.data ?? [],
    isLoading: plansQuery.isLoading,
    isError: plansQuery.isError,
    error: plansQuery.error,
    refetch: plansQuery.refetch,
    createPlan: createPlanMutation.mutateAsync,
    isCreating: createPlanMutation.isPending,
    updatePlan: updatePlanMutation.mutateAsync,
    isUpdating: updatePlanMutation.isPending,
    deactivatePlan: deactivatePlanMutation.mutateAsync,
    isDeactivating: deactivatePlanMutation.isPending,
  }
}
