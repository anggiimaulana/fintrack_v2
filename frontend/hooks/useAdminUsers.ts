'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { queryKeys } from '@/lib/cache-keys'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface AdminUsersParams {
  page?: number
  per_page?: number
  sort?: string
  order?: string
  q?: string
  status?: string
}

export function useAdminUsers(params?: AdminUsersParams) {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: [...queryKeys.admin.users, params],
    queryFn: async () => {
      const response = await api.get('/admin/users', { params })
      const raw = response.data
      return {
        users: Array.isArray(raw?.data) ? raw.data : [],
        meta: raw?.meta || {
          total: Array.isArray(raw?.data) ? raw.data.length : 0,
          page: params?.page || 1,
          per_page: params?.per_page || 20,
          total_pages: 1,
        },
      }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  const toggleStatusMutation = useMutation({
    mutationFn: async ({ id, is_active }: { id: string, is_active: boolean }) => {
      const response = await api.put(`/admin/users/${id}/status`, { is_active })
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.users })
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.stats })
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.recentUsers })
    },
  })

  return {
    users: data?.users || [],
    meta: data?.meta || { total: 0, page: 1, per_page: 20, total_pages: 1 },
    isLoading,
    toggleUserStatus: toggleStatusMutation.mutateAsync,
    isMutating: toggleStatusMutation.isPending,
  }
}
