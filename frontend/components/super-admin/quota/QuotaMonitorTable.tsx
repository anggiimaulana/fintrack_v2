'use client'

import { AlertTriangle, Gauge } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from '@/components/shared/EmptyState'
import { ErrorState } from '@/components/shared/ErrorState'
import type { QuotaMonitorRow } from '@/hooks/useSuperAdminQuotaMonitor'
import { quotaSeverity, summariseQuota } from '@/lib/quota-monitor'

/** Human labels for the limit keys the monitor reports. */
const LIMIT_LABELS: Record<string, string> = {
  ai_tokens_per_cycle: 'Token AI / siklus',
  ai_requests_per_cycle: 'Permintaan AI / siklus',
  max_transactions: 'Transaksi',
  ocr_scans_per_cycle: 'Pindai struk / siklus',
}

function limitLabel(key: string): string {
  return LIMIT_LABELS[key] ?? key
}

interface QuotaMonitorTableProps {
  rows: QuotaMonitorRow[]
  isLoading: boolean
  isError: boolean
  threshold: number
  onRetry: () => void
}

/**
 * Table of users whose quota consumption has crossed the alert threshold.
 *
 * Rows are keyed by (user, limit) so one user can appear for both token and
 * request limits. The progress bar turns destructive at >=90% so the operator
 * can triage at a glance.
 */
export function QuotaMonitorTable({ rows, isLoading, isError, threshold, onRetry }: QuotaMonitorTableProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-14 w-full" />
        ))}
      </div>
    )
  }

  if (isError) {
    return <ErrorState description="Gagal memuat pemantauan kuota." onRetry={onRetry} />
  }

  if (rows.length === 0) {
    return (
      <EmptyState
        title="Semua kuota aman"
        description={`Tidak ada pengguna yang melewati ${threshold}% dari batas paketnya saat ini.`}
      />
    )
  }

  return (
    <div className="overflow-x-auto border-2 rounded-md border-foreground">
      <Table>
        <TableHeader>
          <TableRow className="border-b-2 border-foreground bg-muted">
            <TableHead className="w-12 text-center font-black uppercase tracking-wide">No</TableHead>
            <TableHead className="font-black uppercase tracking-wide">Pengguna</TableHead>
            <TableHead className="font-black uppercase tracking-wide">Paket</TableHead>
            <TableHead className="font-black uppercase tracking-wide">Batas</TableHead>
            <TableHead className="text-right font-black uppercase tracking-wide">Terpakai</TableHead>
            <TableHead className="w-[200px] font-black uppercase tracking-wide">Pemakaian</TableHead>
            <TableHead className="text-right font-black uppercase tracking-wide">Sisa</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => {
            const band = quotaSeverity(r.used_percent)
            const pct = Math.min(100, Math.round(r.used_percent))
            return (
              <TableRow key={`${r.user_id}:${r.limit_key}`} className="border-b border-foreground/20 last:border-b-0">
                <TableCell className="text-center font-mono text-xs text-muted-foreground w-12">
                  {rows.indexOf(r) + 1}
                </TableCell>
                <TableCell>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-bold">{r.full_name || '—'}</p>
                    <p className="truncate font-mono text-xs text-muted-foreground">{r.email}</p>
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className="font-mono text-xs uppercase">
                    {r.plan_code}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs font-bold">{limitLabel(r.limit_key)}</TableCell>
                <TableCell className="text-right font-mono text-xs font-bold">
                  {r.used.toLocaleString('id-ID')} / {r.limit.toLocaleString('id-ID')}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Progress
                      value={pct}
                      className="h-2"
                      indicatorClassName={band === 'critical' ? 'bg-destructive' : 'bg-amber-500'}
                      aria-label={`${limitLabel(r.limit_key)} ${pct}% terpakai`}
                    />
                    <span
                      className={`w-10 shrink-0 text-right font-mono text-xs font-black ${
                        band === 'critical' ? 'text-destructive' : 'text-amber-600 dark:text-amber-400'
                      }`}
                    >
                      {pct}%
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  <span
                    className={`inline-flex items-center gap-1 font-mono text-xs font-bold ${
                      band === 'critical' ? 'text-destructive' : ''
                    }`}
                  >
                    {band === 'critical' && <AlertTriangle className="h-3.5 w-3.5" aria-hidden="true" />}
                    {r.remaining.toLocaleString('id-ID')}
                  </span>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}

/** Compact summary of how many rows are critical vs warning. */
export function QuotaMonitorSummary({ rows }: { rows: QuotaMonitorRow[] }) {
  const { critical, warning } = summariseQuota(rows)
  return (
    <div className="flex flex-wrap items-center gap-3 text-xs font-bold">
      <span className="inline-flex items-center gap-1.5">
        <Gauge className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
        {rows.length} peringatan
      </span>
      {critical > 0 && (
        <Badge variant="destructive" className="gap-1">
          <AlertTriangle className="h-3 w-3" aria-hidden="true" />
          {critical} kritis (≥90%)
        </Badge>
      )}
      {warning > 0 && (
        <Badge variant="secondary">{warning} mendekati batas</Badge>
      )}
    </div>
  )
}
