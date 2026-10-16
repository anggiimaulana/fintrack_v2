"use client"

import * as React from "react"
import Link from "next/link"
import { Sparkles, TrendingUp, TrendingDown, Minus, ArrowRight } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

export interface SmartInsightMetric {
  label: string
  value: string | number
}

export interface SmartInsightAction {
  text: string
  actionLabel?: string
  href?: string
}

export interface SmartInsightCardProps {
  title?: string
  summaryText: string
  deltaPercent?: number
  deltaDirection?: "up" | "down" | "flat"
  isPositiveDelta?: boolean
  metricsSummary?: SmartInsightMetric[]
  actionRecommendation?: SmartInsightAction
  className?: string
}

export function SmartInsightCard({
  title = "Insight Analitik Operasional",
  summaryText,
  deltaPercent,
  deltaDirection = "flat",
  isPositiveDelta = true,
  metricsSummary,
  actionRecommendation,
  className,
}: SmartInsightCardProps) {
  const renderDeltaBadge = () => {
    if (deltaPercent === undefined) return null

    const formatted = `${deltaPercent > 0 ? "+" : ""}${deltaPercent.toFixed(1)}%`
    const isUp = deltaDirection === "up"
    const isDown = deltaDirection === "down"

    const colorClasses = isPositiveDelta
      ? "bg-emerald-500/15 text-emerald-600 border-emerald-500/30 dark:text-emerald-400"
      : "bg-destructive/15 text-destructive border-destructive/30"

    return (
      <Badge variant="outline" className={cn("gap-1 font-mono font-bold px-2 py-0.5", colorClasses)}>
        {isUp && <TrendingUp className="w-3.5 h-3.5" />}
        {isDown && <TrendingDown className="w-3.5 h-3.5" />}
        {!isUp && !isDown && <Minus className="w-3.5 h-3.5" />}
        <span>{formatted}</span>
      </Badge>
    )
  }

  return (
    <div
      className={cn(
        "rounded-lg border-2 border-foreground bg-card p-5 shadow-[4px_4px_0px_0px] shadow-foreground space-y-4",
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-md border border-foreground/30 bg-primary/10 text-primary">
            <Sparkles className="w-4 h-4" />
          </div>
          <h4 className="text-xs font-black uppercase tracking-wider text-muted-foreground">
            {title}
          </h4>
        </div>
        {renderDeltaBadge()}
      </div>

      {/* Summary Narrative */}
      <p className="text-sm font-bold text-foreground leading-relaxed">
        {summaryText}
      </p>

      {/* Key Metric Highlights */}
      {metricsSummary && metricsSummary.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 pt-1">
          {metricsSummary.map((m, idx) => (
            <div
              key={idx}
              className="p-2.5 rounded-md border border-border bg-background/50 flex flex-col min-w-0"
            >
              <span className="text-[10px] font-black uppercase tracking-wider text-muted-foreground truncate">
                {m.label}
              </span>
              <span className="text-base font-black font-mono text-foreground mt-0.5 truncate">
                {m.value}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Action Recommendation */}
      {actionRecommendation && (
        <div className="pt-2 border-t border-border flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 text-xs">
          <span className="text-muted-foreground font-medium">
            <span className="font-bold text-foreground">Rekomendasi: </span>
            {actionRecommendation.text}
          </span>
          {actionRecommendation.href && (
            <Link
              href={actionRecommendation.href}
              className="inline-flex items-center gap-1 font-bold text-primary hover:underline shrink-0"
            >
              {actionRecommendation.actionLabel || "Buka Halaman"}
              <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          )}
        </div>
      )}
    </div>
  )
}
