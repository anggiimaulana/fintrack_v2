'use client'

import { useMutation } from '@tanstack/react-query'
import api from '@/lib/api'

/**
 * Quota estimator (P-9).
 *
 * Calls POST /admin/plans/estimate — a read-only calculator that answers "how
 * many chats does this quota allow, and what does it cost?" before an operator
 * commits a ratio/quota change. Modelled as a mutation (not a query) because it
 * is an explicit, parameterised calculation rather than cacheable server state.
 */

export interface QuotaEstimateInput {
  token_quota_ratio: number
  ai_tokens_per_cycle: number
  avg_tokens_per_chat?: number
  voice_seconds?: number
  price_rule_id?: string
}

export interface QuotaEstimateResult {
  input: {
    token_quota_ratio: number
    ai_tokens_per_cycle: number
    avg_tokens_per_chat: number
    rounding_mode: string
    min_charge_units: number
  }
  quota: {
    unit_per_chat: number
    chats_per_cycle: number
    chats_per_cycle_exact: number
    leftover_units: number
  }
  cost: {
    price_unit: string
    sell_per_1k_idr: number
    provider_cost_per_1k_idr: number
    sell_per_chat_idr: number
    provider_cost_per_chat_idr: number
    margin_per_chat_idr: number
    margin_percent: number
    cost_per_full_cycle_idr: number
    revenue_per_full_cycle_idr: number
    margin_per_full_cycle_idr: number
  }
  comparison: {
    without_ratio: { chats_per_cycle: number }
    with_ratio: { chats_per_cycle: number }
  }
  warnings: string[]
  modalities?: {
    voice: {
      avg_seconds: number
      raw_tokens: number
      unit_per_call: number
      calls_per_cycle: number
      sell_idr: number
      cost_idr: number
    }
    vision: {
      raw_tokens: number
      unit_per_call: number
      calls_per_cycle: number
      sell_idr: number
      cost_idr: number
    }
  }
}

export function useQuotaEstimate() {
  return useMutation({
    mutationFn: async (input: QuotaEstimateInput): Promise<QuotaEstimateResult> => {
      const res = await api.post('/admin/plans/estimate', input)
      return res.data?.data as QuotaEstimateResult
    },
  })
}
