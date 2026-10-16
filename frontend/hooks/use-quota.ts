"use client"

import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"
import { queryKeys } from "@/lib/cache-keys"
import type { UsageCounter, PlanLimitCycle } from "@/hooks/usePlanCycles"

export interface UserQuotaSummary {
  warning_threshold: number
  weekly_anchor?: string | null
  counters: UsageCounter[]
  cycles: Record<string, PlanLimitCycle>
}

export function useQuota() {
  return useQuery<UserQuotaSummary>({
    queryKey: queryKeys.quota.snapshot,
    queryFn: async () => {
      const res = await api.get("/me/quota")
      return res.data?.data ?? { warning_threshold: 80, counters: [], cycles: {} }
    },
    staleTime: 30_000,
  })
}
