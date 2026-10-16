"use client"

import * as React from "react"
import { Progress } from "@/components/ui/progress"
import { QuotaStatusBadge, type QuotaStatus } from "./QuotaStatusBadge"
import { QuotaCountdown } from "./QuotaCountdown"

export interface QuotaMetricItem {
  key: string
  label: string
  unit?: string
  status: QuotaStatus
  monthlyUsed: number
  monthlyLimit: number
  monthlyUnlimited: boolean
  weeklyUsed?: number | null
  weeklyLimit?: number | null
  effectiveRemaining: number
  weeklyResetAt?: string | null
  monthlyResetAt?: string | null
}

interface DualQuotaBarProps {
  item: QuotaMetricItem
}

export function DualQuotaBar({ item }: DualQuotaBarProps) {
  const hasWeekly = item.weeklyLimit !== null && item.weeklyLimit !== undefined && item.weeklyLimit > 0

  const monthlyPct = item.monthlyUnlimited
    ? 0
    : item.monthlyLimit > 0
    ? Math.min(100, Math.round((item.monthlyUsed / item.monthlyLimit) * 100))
    : 100

  const weeklyPct =
    hasWeekly && (item.weeklyLimit ?? 0) > 0
      ? Math.min(100, Math.round(((item.weeklyUsed ?? 0) / (item.weeklyLimit ?? 1)) * 100))
      : 0

  return (
    <div className="p-3 border rounded-md bg-card/50 space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-bold text-sm">{item.label}</span>
          {item.unit && <span className="text-xs text-muted-foreground font-mono">({item.unit})</span>}
        </div>
        <QuotaStatusBadge status={item.status} />
      </div>

      {/* Monthly Bar */}
      <div className="space-y-1">
        <div className="flex justify-between text-xs font-medium">
          <span className="text-muted-foreground">Siklus Bulanan</span>
          <span>
            {item.monthlyUnlimited ? (
              <span className="font-bold">Tanpa Batas</span>
            ) : (
              `${item.monthlyUsed.toLocaleString("id-ID")} / ${item.monthlyLimit.toLocaleString("id-ID")} (${monthlyPct}%)`
            )}
          </span>
        </div>
        {!item.monthlyUnlimited && <Progress value={monthlyPct} className="h-2" />}
      </div>

      {/* Weekly Bar (if weekly configured) */}
      {hasWeekly && (
        <div className="space-y-1 pt-1">
          <div className="flex justify-between text-xs font-medium">
            <span className="text-muted-foreground">Siklus Mingguan</span>
            <span>
              {(item.weeklyUsed ?? 0).toLocaleString("id-ID")} / {(item.weeklyLimit ?? 0).toLocaleString("id-ID")} ({weeklyPct}%)
            </span>
          </div>
          <Progress value={weeklyPct} className="h-2 bg-muted [&>div]:bg-chart-2" />
        </div>
      )}

      {/* Bottom Summary: Effective Remaining + Countdown */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pt-1 border-t text-xs">
        <div className="text-muted-foreground">
          Sisa Efektif:{" "}
          <span className="font-bold text-foreground">
            {item.monthlyUnlimited
              ? "Tanpa Batas"
              : `${Math.max(0, item.effectiveRemaining).toLocaleString("id-ID")} ${item.unit || "unit"}`}
          </span>
        </div>
        <QuotaCountdown
          weeklyResetAt={item.weeklyResetAt}
          monthlyResetAt={item.monthlyResetAt}
        />
      </div>
    </div>
  )
}
