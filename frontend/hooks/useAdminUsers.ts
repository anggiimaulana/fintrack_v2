'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { queryKeys } from '@/lib/cache-keys'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export function useAdminUsers() {
  const queryClient = useQueryClient()

  const { data: users, isLoading } = useQuery({
    queryKey: queryKeys.admin.users,
    queryFn: async () => {
      try {
        const response = await api.get('/admin/users')
        return response.data?.data || []
      } catch {
        return []
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
    users,
    isLoading,
    toggleUserStatus: toggleStatusMutation.mutateAsync,
    isMutating: toggleStatusMutation.isPending,
  }
}
