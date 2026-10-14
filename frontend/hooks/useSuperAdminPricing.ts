'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface PricingRuleDTO {
  id: string
  rule_name: string
  base_per_1k_idr: number
  markup_multiplier: number
  markup_percent: number
  modality_multipliers: Record<string, number>
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface PricingRulePayload {
  rule_name?: string
  base_per_1k_idr?: number
  markup_multiplier?: number
  markup_percent?: number
  modality_multipliers?: Record<string, number>
  is_active?: boolean
}

export const pricingQueryKeys = {
  all: ['admin', 'pricing-rules'] as const,
  active: ['admin', 'pricing-rules', 'active'] as const,
}

/** Modality keys the pricing rule may scale (mirrors the backend catalog). */
export const PRICING_MODALITIES = ['chat', 'vision', 'voice', 'reasoning', 'embedding'] as const

export function usePricingRules() {
  const queryClient = useQueryClient()

  const rulesQuery = useQuery({
    queryKey: pricingQueryKeys.all,
    queryFn: async () => {
      const res = await api.get('/admin/pricing-rules')
      return (res.data?.data || []) as PricingRuleDTO[]
    },
    staleTime: staleTimeConfig.admin,
  })

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: pricingQueryKeys.all })
  }

  const createRule = useMutation({
    mutationFn: async (payload: PricingRulePayload) => {
      const res = await api.post('/admin/pricing-rules', payload)
      return res.data
    },
    onSuccess: invalidate,
  })

  const updateRule = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: PricingRulePayload }) => {
      const res = await api.put(`/admin/pricing-rules/${id}`, payload)
      return res.data
    },
    onSuccess: invalidate,
  })

  const activateRule = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.post(`/admin/pricing-rules/${id}/activate`)
      return res.data
    },
    onSuccess: invalidate,
  })

  const deleteRule = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.delete(`/admin/pricing-rules/${id}`)
      return res.data
    },
    onSuccess: invalidate,
  })

  return {
    rules: rulesQuery.data ?? [],
    isLoading: rulesQuery.isLoading,
    isError: rulesQuery.isError,
    refetch: rulesQuery.refetch,
    createRule,
    updateRule,
    activateRule,
    deleteRule,
  }
}
