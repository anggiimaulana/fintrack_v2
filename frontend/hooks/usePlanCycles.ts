import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"
import { queryKeys } from "@/lib/cache-keys"

export interface UsageCounter {
  id: string
  user_id: string
  limit_key: string
  cycle_type: "weekly" | "monthly"
  cycle_start: string
  cycle_end: string
  used: number
  degraded_used: number
  created_at: string
  updated_at: string
}

export interface PlanLimitCycle {
  plan_id?: string
  limit_key: string
  cycle_type: "monthly_only" | "weekly_monthly"
  weekly_limit?: number | null
  degraded_mode?: "hard_block" | "throttle" | "model_downgrade" | null
  throttle_rate_amount?: number | null
  throttle_rate_window_secs?: number | null
  fallback_model?: string | null
  max_degraded_usage?: number | null
}
const EMPTY_CYCLES: PlanLimitCycle[] = []


export function usePlanCycles(planId?: string) {
  const queryClient = useQueryClient()

  const query = useQuery<PlanLimitCycle[]>({
    queryKey: queryKeys.superAdmin.planCycles(planId ?? ""),
    queryFn: async () => {
      if (!planId) return []
      const res = await api.get(`/admin/plans/${planId}/cycles`)
      return res.data?.data ?? []
    },
    enabled: !!planId,
  })

  const mutation = useMutation({
    mutationFn: async (cycles: PlanLimitCycle[]) => {
      if (!planId) throw new Error("planId is required")
      const res = await api.put(`/admin/plans/${planId}/cycles`, { cycles })
      return res.data
    },
    onSuccess: () => {
      if (planId) {
        queryClient.invalidateQueries({ queryKey: queryKeys.superAdmin.planCycles(planId) })
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.entitlements.me })
      queryClient.invalidateQueries({ queryKey: queryKeys.quota.snapshot })
      queryClient.invalidateQueries({ queryKey: queryKeys.billing.plans })
      queryClient.invalidateQueries({ queryKey: queryKeys.superAdmin.plans })
    },
  })

  return {
    cycles: query.data ?? EMPTY_CYCLES,
    isLoading: query.isLoading,
    error: query.error,
    saveCycles: mutation.mutateAsync,
    isSaving: mutation.isPending,
  }
}
