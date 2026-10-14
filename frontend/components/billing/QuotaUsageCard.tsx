'use client'

import { Gauge, AlertTriangle } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import type { UsageEntry } from '@/hooks/use-entitlements'

/** Human labels for the limit keys the backend exposes usage for. */
const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: 'Token AI',
  ai_requests_per_cycle: 'Permintaan AI',
  max_transactions: 'Transaksi',
  ocr_scans_per_cycle: 'Pindai struk',
}

/** Threshold (percent) at which a bar turns to a warning colour. */
const WARN_PERCENT = 80

function percentOf(entry: UsageEntry): number {
  if (entry.unlimited || entry.limit <= 0) return 0
  return Math.min(100, Math.round((entry.used / entry.limit) * 100))
}

interface QuotaUsageCardProps {
  usage: Record<string, UsageEntry>
  period?: { start: string; end: string } | null
}

/**
 * Real-time quota consumption for the signed-in user (F6). Purely a UX mirror:
 * the backend remains the gate, so a full bar never blocks on its own — the API
 * still answers 402/429.
 */
export function QuotaUsageCard({ usage, period }: QuotaUsageCardProps) {
  const keys = Object.keys(usage).filter((k) => LIMIT_LABELS[k])
  if (keys.length === 0) return null

  return (
    <Card className="border-2 border-foreground shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-lg font-black">
          <Gauge className="w-5 h-5 text-primary" /> Pemakaian Kuota
        </CardTitle>
        <CardDescription>
          Sisa kuota Anda pada siklus ini.
          {period ? ` Periode berakhir ${new Date(period.end).toLocaleDateString('id-ID')}.` : ''}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {keys.map((key) => {
          const entry = usage[key]
          const pct = percentOf(entry)
          const warn = !entry.unlimited && pct >= WARN_PERCENT
          return (
            <div key={key} className="space-y-1.5">
              <div className="flex items-center justify-between text-sm font-bold">
                <span className="flex items-center gap-1.5">
                  {LIMIT_LABELS[key]}
                  {warn && <AlertTriangle className="w-4 h-4 text-amber-600 dark:text-amber-400" aria-label="Kuota menipis" />}
                </span>
                <span className="font-mono text-xs">
                  {entry.unlimited
                    ? `${entry.used.toLocaleString('id-ID')} / ∞`
                    : `${entry.used.toLocaleString('id-ID')} / ${entry.limit.toLocaleString('id-ID')}`}
                </span>
              </div>
              <div className="h-2.5 w-full border-2 border-foreground bg-muted overflow-hidden" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100}>
                <div
                  className={`h-full transition-all ${entry.unlimited ? 'bg-emerald-500' : warn ? 'bg-amber-500' : 'bg-primary'}`}
                  style={{ width: entry.unlimited ? '100%' : `${pct}%` }}
                />
              </div>
              {!entry.unlimited && (
                <p className={`text-xs font-bold ${warn ? 'text-amber-700 dark:text-amber-400' : 'text-muted-foreground'}`}>
                  Sisa {entry.remaining.toLocaleString('id-ID')} ({100 - pct}%)
                </p>
              )}
            </div>
          )
        })}
      </CardContent>
    </Card>
  )
}
