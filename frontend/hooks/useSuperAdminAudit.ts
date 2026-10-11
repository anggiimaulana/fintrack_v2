'use client'

import { useQuery } from '@tanstack/react-query'
import api from '@/lib/api'
import { staleTimeConfig } from '@/lib/cache-invalidation'

export interface AuditLogItem {
  id: string
  actor_id?: string | null
  actor_name?: string | null
  actor_email?: string | null
  action: string
  entity: string
  entity_id?: string | null
  before?: Record<string, unknown> | null
  after?: Record<string, unknown> | null
  ip?: string | null
  user_agent?: string | null
  request_id?: string | null
  created_at: string
}

export interface AuditLogFilters {
  page?: number
  per_page?: number
  sort?: string
  order?: 'asc' | 'desc'
  start_date?: string
  end_date?: string
  email?: string
  user_id?: string
  entity?: string
  action?: string
  entity_id?: string
}

export interface AuditLogMeta {
  total: number
  page: number
  per_page: number
}

export function useSuperAdminAudit(filters: AuditLogFilters) {
  const query = useQuery({
    queryKey: ['super-admin', 'audit-logs', filters],
    queryFn: async () => {
      const params = new URLSearchParams()

      if (filters.page) params.set('page', String(filters.page))
      if (filters.per_page) params.set('per_page', String(filters.per_page))
      if (filters.sort) params.set('sort', filters.sort)
      if (filters.order) params.set('order', filters.order)
      if (filters.start_date) params.set('start_date', filters.start_date)
      if (filters.end_date) params.set('end_date', filters.end_date)
      if (filters.email) params.set('email', filters.email)
      if (filters.user_id) params.set('user_id', filters.user_id)
      if (filters.entity) params.set('entity', filters.entity)
      if (filters.action) params.set('action', filters.action)
      if (filters.entity_id) params.set('entity_id', filters.entity_id)

      const response = await api.get(`/admin/audit-logs?${params.toString()}`)
      const data: AuditLogItem[] = response.data?.data || []
      const meta: AuditLogMeta = response.data?.meta || {
        total: 0,
        page: filters.page || 1,
        per_page: filters.per_page || 20,
      }

      return { data, meta }
    },
    staleTime: staleTimeConfig.admin,
    placeholderData: (prev) => prev,
  })

  return query
}
