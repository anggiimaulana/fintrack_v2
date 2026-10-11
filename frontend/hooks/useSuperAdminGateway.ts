'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import axios from 'axios'
import api from '@/lib/api'

export type EndpointModality = 'chat' | 'vision' | 'voice' | 'embedding' | 'reasoning'
export type EndpointStatus = 'active' | 'degraded' | 'inactive' | 'suspended'
export type BreakerState = 'closed' | 'half_open' | 'open'
export type KillSwitchTarget = 'global' | 'chat' | 'vision' | 'voice' | 'embedding' | 'reasoning'

export interface GatewayEndpoint {
  id: string
  name: string
  base_url: string
  provider: string
  priority: number
  status: EndpointStatus
  health: string
  modalities: EndpointModality[]
  weight: number
  last_check_at?: string | null
  last_error?: string | null
  has_api_key: boolean
  breaker_state: BreakerState
}

export interface CreateEndpointPayload {
  name: string
  base_url: string
  provider?: string
  priority?: number
  weight?: number
  status?: EndpointStatus
  modalities?: EndpointModality[]
  api_key?: string
}

export interface UpdateEndpointPayload {
  name?: string
  base_url?: string
  provider?: string
  priority?: number
  weight?: number
  status?: EndpointStatus
  modalities?: EndpointModality[]
  api_key?: string
}

export interface GatewayRoute {
  id?: string
  use_case: EndpointModality
  endpoint_id: string
  model: string
  priority: number
  timeout_ms: number
  max_retries: number
  is_active: boolean
}

export type KillSwitchState = Record<KillSwitchTarget, boolean>

export const gatewayKeys = {
  all: ['super-admin', 'ai-gateway'] as const,
  endpoints: ['super-admin', 'ai-gateway', 'endpoints'] as const,
  routes: ['super-admin', 'ai-gateway', 'routes'] as const,
  killSwitch: ['super-admin', 'ai-gateway', 'kill-switch'] as const,
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

export function useSuperAdminGateway() {
  const queryClient = useQueryClient()

  // 1. Endpoints
  const endpointsQuery = useQuery<GatewayEndpoint[]>({
    queryKey: gatewayKeys.endpoints,
    queryFn: async () => {
      const res = await api.get('/admin/ai-gateway/endpoints')
      return res.data?.data || []
    },
    staleTime: 1000 * 30,
    placeholderData: (prev) => prev,
  })

  // 2. Routes
  const routesQuery = useQuery<GatewayRoute[]>({
    queryKey: gatewayKeys.routes,
    queryFn: async () => {
      const res = await api.get('/admin/ai-gateway/routes')
      return res.data?.data || []
    },
    staleTime: 1000 * 30,
    placeholderData: (prev) => prev,
  })

  // 3. Kill Switch
  const killSwitchQuery = useQuery<KillSwitchState>({
    queryKey: gatewayKeys.killSwitch,
    queryFn: async () => {
      const res = await api.get('/admin/ai-gateway/kill-switch')
      return (
        res.data?.data || {
          global: false,
          chat: false,
          vision: false,
          voice: false,
          embedding: false,
          reasoning: false,
        }
      )
    },
    staleTime: 1000 * 10,
    placeholderData: (prev) => prev,
  })

  // Create endpoint
  const createEndpointMutation = useMutation({
    mutationFn: async (payload: CreateEndpointPayload) => {
      const cleanPayload: Record<string, unknown> = {
        name: payload.name,
        base_url: payload.base_url,
        provider: payload.provider || '9router',
        priority: payload.priority ?? 1,
        weight: payload.weight ?? 100,
        status: payload.status || 'active',
        modalities: payload.modalities && payload.modalities.length > 0 ? payload.modalities : ['chat', 'vision', 'voice', 'embedding', 'reasoning'],
      }
      if (payload.api_key && payload.api_key.trim().length > 0) {
        cleanPayload.api_key = payload.api_key.trim()
      }
      const res = await api.post('/admin/ai-gateway/endpoints', cleanPayload)
      return res.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      toast.success(res?.message || 'Endpoint AI berhasil dibuat')
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err) && err.response?.status === 409) {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      }
      toast.error(getErrorMessage(err, 'Gagal membuat endpoint AI'))
    },
  })

  // Update endpoint
  const updateEndpointMutation = useMutation({
    mutationFn: async ({ id, payload }: { id: string; payload: UpdateEndpointPayload }) => {
      const cleanPayload: Record<string, unknown> = {}
      if (payload.name !== undefined) cleanPayload.name = payload.name
      if (payload.base_url !== undefined) cleanPayload.base_url = payload.base_url
      if (payload.provider !== undefined) cleanPayload.provider = payload.provider
      if (payload.priority !== undefined) cleanPayload.priority = payload.priority
      if (payload.weight !== undefined) cleanPayload.weight = payload.weight
      if (payload.status !== undefined) cleanPayload.status = payload.status
      if (payload.modalities !== undefined) cleanPayload.modalities = payload.modalities
      if (payload.api_key && payload.api_key.trim().length > 0) {
        cleanPayload.api_key = payload.api_key.trim()
      }
      const res = await api.put(`/admin/ai-gateway/endpoints/${id}`, cleanPayload)
      return res.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      toast.success(res?.message || 'Endpoint AI berhasil diperbarui')
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err) && err.response?.status === 409) {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      }
      toast.error(getErrorMessage(err, 'Gagal memperbarui endpoint AI'))
    },
  })

  // Delete endpoint
  const deleteEndpointMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await api.delete(`/admin/ai-gateway/endpoints/${id}`)
      return res.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      toast.success(res?.message || 'Endpoint dinonaktifkan')
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err) && err.response?.status === 409) {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
        queryClient.invalidateQueries({ queryKey: gatewayKeys.routes })
      }
      toast.error(getErrorMessage(err, 'Gagal menghapus endpoint'))
    },
  })

  // Reset circuit breaker
  const resetBreakerMutation = useMutation({
    mutationFn: async (endpointId: string) => {
      const res = await api.post('/admin/ai-gateway/circuit-breaker/reset', { endpoint_id: endpointId })
      return res.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.endpoints })
      toast.success(res?.message || 'Circuit breaker berhasil direset')
    },
    onError: (err: unknown) => {
      toast.error(getErrorMessage(err, 'Gagal mereset circuit breaker'))
    },
  })

  // Put routes (entire matrix)
  const saveRoutesMutation = useMutation({
    mutationFn: async (routes: GatewayRoute[]) => {
      const payload = routes.map((r) => {
        const item: Record<string, unknown> = {
          use_case: r.use_case,
          endpoint_id: r.endpoint_id,
          model: r.model,
          priority: Number(r.priority) || 1,
          timeout_ms: Number(r.timeout_ms) >= 0 ? Number(r.timeout_ms) : 30000,
          max_retries: Number(r.max_retries) >= 0 ? Number(r.max_retries) : 2,
          is_active: Boolean(r.is_active),
        }
        if (r.id) {
          item.id = r.id
        }
        return item
      })
      const res = await api.put('/admin/ai-gateway/routes', payload)
      return res.data
    },
    onSuccess: (res) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.routes })
      toast.success(res?.message || 'Matriks rute AI berhasil disimpan')
    },
    onError: (err: unknown) => {
      if (axios.isAxiosError(err) && err.response?.status === 409) {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.routes })
      }
      toast.error(getErrorMessage(err, 'Gagal menyimpan matriks rute AI'))
    },
  })

  // Set kill switch
  const setKillSwitchMutation = useMutation({
    mutationFn: async ({ target, enabled }: { target: KillSwitchTarget; enabled: boolean }) => {
      const res = await api.post('/admin/ai-gateway/kill-switch', { target, enabled })
      return res.data
    },
    onSuccess: (res) => {
      if (res?.data) {
        queryClient.setQueryData(gatewayKeys.killSwitch, res.data)
      } else {
        queryClient.invalidateQueries({ queryKey: gatewayKeys.killSwitch })
      }
      toast.success(res?.message || 'Status kill switch diperbarui')
    },
    onError: (err: unknown) => {
      queryClient.invalidateQueries({ queryKey: gatewayKeys.killSwitch })
      toast.error(getErrorMessage(err, 'Gagal memperbarui status kill switch'))
    },
  })

  return {
    // Queries
    endpoints: endpointsQuery.data || [],
    isLoadingEndpoints: endpointsQuery.isLoading,
    isRefetchingEndpoints: endpointsQuery.isRefetching,
    refetchEndpoints: endpointsQuery.refetch,

    routes: routesQuery.data || [],
    isLoadingRoutes: routesQuery.isLoading,
    isRefetchingRoutes: routesQuery.isRefetching,
    refetchRoutes: routesQuery.refetch,

    killSwitch: killSwitchQuery.data || {
      global: false,
      chat: false,
      vision: false,
      voice: false,
      embedding: false,
      reasoning: false,
    },
    isLoadingKillSwitch: killSwitchQuery.isLoading,
    refetchKillSwitch: killSwitchQuery.refetch,

    // Mutations
    createEndpoint: createEndpointMutation.mutateAsync,
    isCreatingEndpoint: createEndpointMutation.isPending,

    updateEndpoint: updateEndpointMutation.mutateAsync,
    isUpdatingEndpoint: updateEndpointMutation.isPending,

    deleteEndpoint: deleteEndpointMutation.mutateAsync,
    isDeletingEndpoint: deleteEndpointMutation.isPending,

    resetBreaker: resetBreakerMutation.mutateAsync,
    isResettingBreaker: resetBreakerMutation.isPending,

    saveRoutes: saveRoutesMutation.mutateAsync,
    isSavingRoutes: saveRoutesMutation.isPending,

    setKillSwitch: setKillSwitchMutation.mutateAsync,
    isSettingKillSwitch: setKillSwitchMutation.isPending,
  }
}
