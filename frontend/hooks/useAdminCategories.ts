'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '@/lib/api'
import { queryKeys } from '@/lib/cache-keys'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface AdminCategory {
  id: string;
  name: string;
  icon: string;
  color: string;
  description?: string;
  sort_order?: number;
  is_active: boolean;
}

export function useAdminCategories(type: 'income' | 'expense' | 'savings') {
  const queryClient = useQueryClient()

  const { data: categories, isLoading } = useQuery({
    queryKey: queryKeys.admin.categories(type),
    queryFn: async () => {
      const response = await api.get(`/categories/${type}`)
      return (response.data?.data as AdminCategory[]) || []
    },
    staleTime: staleTimeConfig.categories,
    placeholderData: (prev) => prev,
  })

  const createMutation = useMutation({
    mutationFn: async (newCategory: Partial<AdminCategory>) => {
      const response = await api.post(`/admin/categories/${type}`, newCategory)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories(type) })
      queryClient.invalidateQueries({ queryKey: queryKeys.categories.all })
    },
  })

  const updateMutation = useMutation({
    mutationFn: async ({ id, ...data }: Partial<AdminCategory> & { id: string }) => {
      const response = await api.put(`/admin/categories/${type}/${id}`, data)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories(type) })
      queryClient.invalidateQueries({ queryKey: queryKeys.categories.all })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      const response = await api.delete(`/admin/categories/${type}/${id}`)
      return response.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.admin.categories(type) })
      queryClient.invalidateQueries({ queryKey: queryKeys.categories.all })
    },
  })

  return {
    categories,
    isLoading,
    createCategory: createMutation.mutateAsync,
    updateCategory: updateMutation.mutateAsync,
    deleteCategory: deleteMutation.mutateAsync,
    isMutating: createMutation.isPending || updateMutation.isPending || deleteMutation.isPending,
  }
}
