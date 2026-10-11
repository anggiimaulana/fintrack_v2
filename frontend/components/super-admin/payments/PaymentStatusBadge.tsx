import * as React from "react"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

export type PaymentStatusType =
  | "pending"
  | "pending_review"
  | "settled"
  | "failed"
  | "expired"
  | "refunded"

interface PaymentStatusBadgeProps {
  status: string
  className?: string
}

export function PaymentStatusBadge({ status, className }: PaymentStatusBadgeProps) {
  const normStatus = status.toLowerCase()

  switch (normStatus) {
    case "pending":
    case "pending_review":
      return (
        <Badge
          variant="outline"
          className={cn(
            "border-amber-600 bg-amber-500/15 text-amber-900 dark:text-amber-200 font-bold",
            className
          )}
        >
          {normStatus === "pending_review" ? "Menunggu Verifikasi" : "Pending"}
        </Badge>
      )
    case "settled":
      return (
        <Badge
          variant="outline"
          className={cn(
            "border-emerald-600 bg-emerald-500/15 text-emerald-900 dark:text-emerald-200 font-bold",
            className
          )}
        >
          Berhasil
        </Badge>
      )
    case "failed":
    case "expired":
      return (
        <Badge
          variant="outline"
          className={cn(
            "border-rose-600 bg-rose-500/15 text-rose-900 dark:text-rose-200 font-bold",
            className
          )}
        >
          {normStatus === "expired" ? "Kedaluwarsa" : "Gagal"}
        </Badge>
      )
    case "refunded":
      return (
        <Badge
          variant="outline"
          className={cn(
            "border-purple-600 bg-purple-500/15 text-purple-900 dark:text-purple-200 font-bold",
            className
          )}
        >
          Refunded
        </Badge>
      )
    default:
      return (
        <Badge variant="outline" className={cn("font-bold", className)}>
          {status}
        </Badge>
      )
  }
}
