"use client"

import * as React from "react"
import { Clock } from "lucide-react"
import { cn } from "@/lib/utils"

export function formatQuotaRemaining(ms: number): string {
  if (ms <= 0) return "mereset..."
  const totalMinutes = Math.floor(ms / 60_000)
  const days = Math.floor(totalMinutes / (60 * 24))
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60)
  const minutes = totalMinutes % 60
  if (days > 0) return `${days} hari ${hours} jam`
  if (hours > 0) return `${hours} jam ${minutes} menit`
  return `${minutes} menit`
}

interface QuotaCountdownProps {
  weeklyResetAt?: string | null
  monthlyResetAt?: string | null
  className?: string
}

export function QuotaCountdown({ weeklyResetAt, monthlyResetAt, className }: QuotaCountdownProps) {
  const [now, setNow] = React.useState<number>(() => Date.now())

  React.useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 60_000)
    return () => clearInterval(timer)
  }, [])

  const weeklyMs = weeklyResetAt ? Date.parse(weeklyResetAt) - now : null
  const monthlyMs = monthlyResetAt ? Date.parse(monthlyResetAt) - now : null

  return (
    <div className={cn("flex flex-wrap items-center gap-2 text-xs text-muted-foreground", className)}>
      <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" aria-hidden="true" />
      {weeklyMs !== null && (
        <span className={cn(weeklyMs > 0 && weeklyMs < 3600_000 && "text-destructive font-bold")}>
          Reset mingguan: {formatQuotaRemaining(weeklyMs)}
        </span>
      )}
      {weeklyMs !== null && monthlyMs !== null && <span aria-hidden="true">•</span>}
      {monthlyMs !== null && (
        <span className={cn(monthlyMs > 0 && monthlyMs < 3600_000 && "text-destructive font-bold")}>
          Reset bulanan: {formatQuotaRemaining(monthlyMs)}
        </span>
      )}
    </div>
  )
}
