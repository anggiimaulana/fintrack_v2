"use client"

import * as React from "react"
import { CheckCircle2, AlertTriangle, Gauge, ArrowDownCircle, XCircle } from "lucide-react"
import { Badge } from "@/components/ui/badge"

export type QuotaStatus = "normal" | "warning" | "degraded_throttle" | "degraded_downgrade" | "blocked"

interface QuotaStatusBadgeProps {
  status: QuotaStatus
  className?: string
}

export function QuotaStatusBadge({ status, className }: QuotaStatusBadgeProps) {
  switch (status) {
    case "warning":
      return (
        <Badge
          variant="outline"
          className={`border-amber-500 text-amber-600 bg-amber-500/10 font-bold gap-1 text-[11px] ${className ?? ""}`}
        >
          <AlertTriangle className="h-3 w-3" aria-hidden="true" />
          Peringatan
        </Badge>
      )
    case "degraded_throttle":
      return (
        <Badge
          variant="outline"
          className={`border-orange-500 text-orange-600 bg-orange-500/10 font-bold gap-1 text-[11px] ${className ?? ""}`}
        >
          <Gauge className="h-3 w-3" aria-hidden="true" />
          Dibatasi
        </Badge>
      )
    case "degraded_downgrade":
      return (
        <Badge
          variant="outline"
          className={`border-orange-500 text-orange-600 bg-orange-500/10 font-bold gap-1 text-[11px] ${className ?? ""}`}
        >
          <ArrowDownCircle className="h-3 w-3" aria-hidden="true" />
          Mode Hemat
        </Badge>
      )
    case "blocked":
      return (
        <Badge
          variant="outline"
          className={`border-destructive text-destructive bg-destructive/10 font-bold gap-1 text-[11px] ${className ?? ""}`}
        >
          <XCircle className="h-3 w-3" aria-hidden="true" />
          Terblokir
        </Badge>
      )
    default:
      return (
        <Badge
          variant="outline"
          className={`border-emerald-500 text-emerald-600 bg-emerald-500/10 font-bold gap-1 text-[11px] ${className ?? ""}`}
        >
          <CheckCircle2 className="h-3 w-3" aria-hidden="true" />
          Normal
        </Badge>
      )
  }
}
