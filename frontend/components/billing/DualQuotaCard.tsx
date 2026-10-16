"use client"

import * as React from "react"
import { Gauge } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DualQuotaBar, type QuotaMetricItem } from "./DualQuotaBar"
import { useEntitlements } from "@/hooks/use-entitlements"
import { useQuota } from "@/hooks/use-quota"
import type { QuotaStatus } from "./QuotaStatusBadge"

const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: "Kuota AI",
  ai_requests_per_cycle: "Permintaan AI",
  ocr_scans_per_cycle: "Pindai Struk (OCR)",
  voice_requests_per_day: "Input Suara",
  max_transactions: "Transaksi",
}

const LIMIT_ORDER = [
  "ai_tokens_per_cycle",
  "ai_requests_per_cycle",
  "ocr_scans_per_cycle",
  "voice_requests_per_day",
  "max_transactions",
]

export function DualQuotaCard() {
  const { entitlements } = useEntitlements()
  const { data: quotaData } = useQuota()

  const items: QuotaMetricItem[] = React.useMemo(() => {
    if (!entitlements || !entitlements.limits) return []

    const result: QuotaMetricItem[] = []
    const usageMap = entitlements.usage || {}

    // Find weekly counters from useQuota
    const weeklyCounters: Record<string, { used: number; start: string; end: string }> = {}
    if (quotaData?.counters) {
      for (const c of quotaData.counters) {
        if (c.cycle_type === "weekly") {
          weeklyCounters[c.limit_key] = {
            used: c.used,
            start: c.cycle_start,
            end: c.cycle_end,
          }
        }
      }
    }

    const keys = Object.keys(entitlements.limits).sort((a, b) => {
      const idxA = LIMIT_ORDER.indexOf(a)
      const idxB = LIMIT_ORDER.indexOf(b)
      if (idxA === -1 && idxB === -1) return a.localeCompare(b)
      if (idxA === -1) return 1
      if (idxB === -1) return -1
      return idxA - idxB
    })

    for (const key of keys) {
      const limit = entitlements.limits[key]
      if (limit === 0) continue // not in plan

      const usage = usageMap[key]
      const monthlyUsed = usage?.used ?? 0
      const monthlyUnlimited = limit < 0
      const monthlyRemaining = monthlyUnlimited ? -1 : Math.max(0, limit - monthlyUsed)

      const cycle = quotaData?.cycles?.[key]
      const wCounter = weeklyCounters[key]

      const hasWeekly = Boolean(cycle?.cycle_type === "weekly_monthly" && cycle.weekly_limit && cycle.weekly_limit > 0)
      const weeklyLimit = hasWeekly ? cycle!.weekly_limit : null
      let weeklyUsed = hasWeekly ? (wCounter?.used ?? 0) : null

      // In the first 7 days of the billing cycle, weekly usage encompasses all period usage so far
      if (hasWeekly && entitlements?.period?.start) {
        const pStart = new Date(entitlements.period.start).getTime()
        const now = Date.now()
        if (now >= pStart && now - pStart < 7 * 24 * 60 * 60 * 1000) {
          if (monthlyUsed > (weeklyUsed ?? 0)) {
            weeklyUsed = monthlyUsed
          }
        }
      }

      const weeklyRemaining = hasWeekly ? Math.max(0, (weeklyLimit ?? 0) - (weeklyUsed ?? 0)) : null

      let effective = monthlyRemaining
      if (hasWeekly && weeklyRemaining !== null && !monthlyUnlimited) {
        effective = Math.min(monthlyRemaining, weeklyRemaining)
      } else if (hasWeekly && weeklyRemaining !== null) {
        effective = weeklyRemaining
      }

      // Compute status
      let status: QuotaStatus = "normal"
      if (!monthlyUnlimited && monthlyRemaining <= 0) {
        status = "blocked"
      } else if (hasWeekly && weeklyRemaining !== null && weeklyRemaining <= 0) {
        if (cycle?.degraded_mode === "throttle") {
          status = "degraded_throttle"
        } else if (cycle?.degraded_mode === "model_downgrade") {
          status = "degraded_downgrade"
        } else {
          status = "blocked"
        }
      } else if (!monthlyUnlimited && (monthlyUsed / limit) >= 0.8) {
        status = "warning"
      }

      result.push({
        key,
        label: LIMIT_LABELS[key] || key,
        unit: usage?.unit,
        status,
        monthlyUsed,
        monthlyLimit: limit,
        monthlyUnlimited,
        weeklyUsed,
        weeklyLimit,
        effectiveRemaining: effective,
        weeklyResetAt: wCounter?.end,
        monthlyResetAt: entitlements.period?.end,
      })
    }

    return result
  }, [entitlements, quotaData])

  if (items.length === 0) return null

  return (
    <Card className="border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <div className="flex items-center gap-2">
          <Gauge className="h-5 w-5" aria-hidden="true" />
          <CardTitle className="text-lg">Penggunaan Kuota Anda</CardTitle>
        </div>
        <CardDescription>
          Pantau konsumsi kuota mingguan dan bulanan paket {entitlements?.plan_name || "aktif"}.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {items.map((item) => (
          <DualQuotaBar key={item.key} item={item} />
        ))}
      </CardContent>
    </Card>
  )
}
