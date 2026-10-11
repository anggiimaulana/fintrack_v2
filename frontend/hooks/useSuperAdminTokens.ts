'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface TokenBalanceRow {
  user_id: string
  user_email: string
  user_full_name: string
  period_id?: string
  period_start?: string
  period_end?: string
  quota_tokens: number
  used_tokens: number
  reserved_tokens: number
  available_tokens: number
}

export interface TokenLedgerRow {
  id: string
  user_id: string
  user_email: string
  entry_type: 'commit' | 'refund' | 'adjust' | 'grant' | 'reset' | string
  amount: number
  modality: 'chat' | 'vision' | 'voice' | 'embedding' | 'admin' | string
  cost_idr: number
  idempotency_key: string
  reference_id?: string
  notes?: string
  created_at: string
  metadata?: Record<string, unknown>
}

export interface AdjustTokensPayload {
  user_id: string
  amount: number
  reason: string
  idempotency_key: string
}

export interface AdjustTokensResponse {
  ledger_id: string
  user_id: string
  amount: number
  quota_tokens: number
  replayed?: boolean
}

export interface TokenUserParams {
  page?: number
  per_page?: number
  q?: string
  sort?: string // 'user' | 'quota' | 'used'
  order?: 'asc' | 'desc'
}

export interface TokenLedgerParams {
  page?: number
  per_page?: number
  entry_type?: string
  modality?: string
  user_id?: string
  start_date?: string
  end_date?: string
  sort?: string // 'created_at' | 'amount' | 'cost' | 'user' | 'type'
  order?: 'asc' | 'desc'
}

export const superAdminTokenKeys = {
  all: ['super-admin', 'tokens'] as const,
  users: (params?: TokenUserParams) => ['super-admin', 'tokens', 'users', params] as const,
  ledger: (params?: TokenLedgerParams) => ['super-admin', 'tokens', 'ledger', params] as const,
}

export function useTokenBalances(params: TokenUserParams = {}) {
  return useQuery({
    queryKey: superAdminTokenKeys.users(params),
    queryFn: async () => {
      const queryParams: Record<string, unknown> = {}
      if (params.page) queryParams.page = params.page
      if (params.per_page) queryParams.per_page = params.per_page
      if (params.q) queryParams.q = params.q
      if (params.sort) queryParams.sort = params.sort
      if (params.order) queryParams.order = params.order

      const res = await api.get('/admin/tokens/users', { params: queryParams })
      return {
        data: (res.data?.data ?? []) as TokenBalanceRow[],
        meta: (res.data?.meta ?? {
          page: params.page ?? 1,
          per_page: params.per_page ?? 20,
          total: 0,
          total_pages: 0,
        }) as { page: number; per_page: number; total: number; total_pages: number },
      }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })
}

export function useTokenLedger(params: TokenLedgerParams = {}) {
  return useQuery({
    queryKey: superAdminTokenKeys.ledger(params),
    queryFn: async () => {
      const queryParams: Record<string, unknown> = {}
      if (params.page) queryParams.page = params.page
      if (params.per_page) queryParams.per_page = params.per_page
      if (params.entry_type) queryParams.entry_type = params.entry_type
      if (params.modality) queryParams.modality = params.modality
      if (params.user_id) queryParams.user_id = params.user_id
      if (params.start_date) queryParams.start_date = params.start_date
      if (params.end_date) queryParams.end_date = params.end_date
      if (params.sort) queryParams.sort = params.sort
      if (params.order) queryParams.order = params.order

      const res = await api.get('/admin/tokens/ledger', { params: queryParams })
      return {
        data: (res.data?.data ?? []) as TokenLedgerRow[],
        meta: (res.data?.meta ?? {
          page: params.page ?? 1,
          per_page: params.per_page ?? 20,
          total: 0,
          total_pages: 0,
        }) as { page: number; per_page: number; total: number; total_pages: number },
      }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })
}

export function useAdjustTokens() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (payload: AdjustTokensPayload) => {
      const res = await api.post('/admin/tokens/adjust', payload)
      return res.data?.data as AdjustTokensResponse
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['super-admin', 'tokens'] })
    },
  })
}
