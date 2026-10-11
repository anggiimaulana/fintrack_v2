'use client'

import { useQuery, useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export type PaymentStatus =
  | 'pending'
  | 'pending_review'
  | 'settled'
  | 'failed'
  | 'expired'
  | 'refunded'

export interface PaymentRow {
  id: string
  invoice_id: string
  invoice_number?: string | null
  user_id: string
  user_email: string
  gateway: string
  gateway_ref?: string | null
  amount_idr: number
  status: PaymentStatus | string
  payment_method?: string | null
  proof_url?: string | null
  review_reason?: string | null
  reviewed_by?: string | null
  reviewed_at?: string | null
  refund_amount_idr: number
  refund_reason?: string | null
  refund_ref?: string | null
  created_at: string
}

export interface PaymentListFilters {
  page?: number
  per_page?: number
  sort?: string
  order?: 'asc' | 'desc'
  status?: string
  gateway?: string
  user_id?: string
  q?: string
  start_date?: string
  end_date?: string
}

export interface PaymentReviewFilters {
  page?: number
  per_page?: number
}

export interface PaymentListMeta {
  total: number
  page: number
  per_page: number
}

export interface RejectPaymentPayload {
  id: string
  reason: string
}

export interface RefundPaymentPayload {
  id: string
  amount_idr?: number
  reason: string
  reference?: string
}

export const paymentKeys = {
  all: ['super-admin', 'payments'] as const,
  list: (filters: PaymentListFilters) => ['super-admin', 'payments', 'list', filters] as const,
  review: (filters: PaymentReviewFilters) => ['super-admin', 'payments', 'review', filters] as const,
}

function getErrorMessage(err: unknown, fallback: string): string {
  const axiosErr = err as {
    response?: {
      data?: {
        message?: string
        error?: string
      }
    }
    message?: string
  }
  return (
    axiosErr?.response?.data?.message ||
    axiosErr?.response?.data?.error ||
    axiosErr?.message ||
    fallback
  )
}

function handlePaymentError(
  err: unknown,
  queryClient: QueryClient,
  fallbackMessage: string
) {
  const axiosErr = err as {
    response?: {
      status?: number
      data?: {
        message?: string
        error?: string
      }
    }
  }

  const is409 =
    axiosErr?.response?.status === 409 ||
    axiosErr?.response?.data?.error === 'INVALID_PAYMENT_STATE'

  if (is409) {
    const message =
      axiosErr?.response?.data?.message ||
      'Transaksi ini sudah diproses oleh admin lain.'
    toast.error(message)
    // Refetch immediately on race condition
    queryClient.invalidateQueries({ queryKey: paymentKeys.all })
    return
  }

  toast.error(getErrorMessage(err, fallbackMessage))
}

export function useSuperAdminPayments(filters: PaymentListFilters = {}) {
  const query = useQuery({
    queryKey: paymentKeys.list(filters),
    queryFn: async () => {
      const params = new URLSearchParams()
      if (filters.page) params.set('page', String(filters.page))
      if (filters.per_page) params.set('per_page', String(filters.per_page))
      if (filters.sort) params.set('sort', filters.sort)
      if (filters.order) params.set('order', filters.order)
      if (filters.status) params.set('status', filters.status)
      if (filters.gateway) params.set('gateway', filters.gateway)
      if (filters.user_id) params.set('user_id', filters.user_id)
      if (filters.q) params.set('q', filters.q)
      if (filters.start_date) params.set('start_date', filters.start_date)
      if (filters.end_date) params.set('end_date', filters.end_date)

      const response = await api.get(`/admin/payments?${params.toString()}`)
      const data: PaymentRow[] = response.data?.data || []
      const meta: PaymentListMeta = response.data?.meta || {
        total: 0,
        page: filters.page || 1,
        per_page: filters.per_page || 20,
      }

      return { data, meta }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  return {
    ...query,
    payments: query.data?.data || [],
    meta: query.data?.meta,
    refetch: query.refetch,
  }
}

export function useSuperAdminPaymentReview(filters: PaymentReviewFilters = {}) {
  const query = useQuery({
    queryKey: paymentKeys.review(filters),
    queryFn: async () => {
      const params = new URLSearchParams()
      if (filters.page) params.set('page', String(filters.page))
      if (filters.per_page) params.set('per_page', String(filters.per_page))

      const response = await api.get(`/admin/payments/review?${params.toString()}`)
      const data: PaymentRow[] = response.data?.data || []
      const meta: PaymentListMeta = response.data?.meta || {
        total: 0,
        page: filters.page || 1,
        per_page: filters.per_page || 20,
      }

      return { data, meta }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  return {
    ...query,
    payments: query.data?.data || [],
    meta: query.data?.meta,
    refetch: query.refetch,
  }
}

export function useSuperAdminPaymentMutations() {
  const queryClient = useQueryClient()

  const approveMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await api.post(`/admin/payments/${id}/approve`)
      return response.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: paymentKeys.all })
      toast.success(res?.message || 'Pembayaran berhasil disetujui')
    },
    onError: (err) => {
      handlePaymentError(err, queryClient, 'Gagal menyetujui pembayaran')
    },
  })

  const rejectMutation = useMutation({
    mutationFn: async ({ id, reason }: RejectPaymentPayload) => {
      const response = await api.post(`/admin/payments/${id}/reject`, { reason })
      return response.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: paymentKeys.all })
      toast.success(res?.message || 'Pembayaran berhasil ditolak')
    },
    onError: (err) => {
      handlePaymentError(err, queryClient, 'Gagal menolak pembayaran')
    },
  })

  const refundMutation = useMutation({
    mutationFn: async ({ id, amount_idr, reason, reference }: RefundPaymentPayload) => {
      const payload: { amount_idr?: number; reason: string; reference?: string } = {
        reason,
      }
      if (typeof amount_idr === 'number' && amount_idr > 0) {
        payload.amount_idr = amount_idr
      }
      if (reference && reference.trim()) {
        payload.reference = reference.trim()
      }
      const response = await api.post(`/admin/payments/${id}/refund`, payload)
      return response.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: paymentKeys.all })
      toast.success(res?.message || 'Refund berhasil diproses')
    },
    onError: (err) => {
      handlePaymentError(err, queryClient, 'Gagal memproses refund')
    },
  })

  return {
    approvePayment: approveMutation.mutateAsync,
    rejectPayment: rejectMutation.mutateAsync,
    refundPayment: refundMutation.mutateAsync,
    isApproving: approveMutation.isPending,
    isRejecting: rejectMutation.isPending,
    isRefunding: refundMutation.isPending,
  }
}
