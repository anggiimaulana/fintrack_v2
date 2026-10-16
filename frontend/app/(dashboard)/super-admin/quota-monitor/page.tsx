'use client'

import { Suspense } from 'react'
import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { Activity, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useSuperAdminQuotaMonitor } from '@/hooks/useSuperAdminQuotaMonitor'
import { useSuperAdminMetrics, type MetricsPeriod } from '@/hooks/useSuperAdminMetrics'
import { QuotaMonitorTable, QuotaMonitorSummary } from '@/components/super-admin/quota/QuotaMonitorTable'
import { ModalityConsumptionChart } from '@/components/super-admin/quota/ModalityConsumptionChart'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'

/** Threshold options for the "running low" filter (percent of limit used). */
const THRESHOLDS = [
  { value: '80', label: '≥ 80% (menipis)' },
  { value: '90', label: '≥ 90% (kritis)' },
  { value: '50', label: '≥ 50% (pantau)' },
] as const

function SuperAdminQuotaMonitorContent() {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()

  const threshold = parseInt(searchParams.get('threshold') || '80', 10) || 80
  const { rows, isLoading, isError, refetch } = useSuperAdminQuotaMonitor(threshold)
  const {
    timeseries,
    isTimeseriesLoading,
    period,
    setPeriod,
    refetchAll,
  } = useSuperAdminMetrics()

  const setThreshold = (value: string) => {
    const params = new URLSearchParams(searchParams.toString())
    if (value === '80') {
      params.delete('threshold')
    } else {
      params.set('threshold', value)
    }
    const qs = params.toString()
    router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false })
  }

  const handleRefresh = () => {
    refetch()
    refetchAll()
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex flex-col gap-4 border-b border-border pb-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight text-foreground">
            <Activity className="h-6 w-6 text-primary" aria-hidden="true" />
            Pemantauan Kuota
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Pengguna yang mendekati batas paket dan konsumsi token per modalitas.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Select value={String(threshold)} onValueChange={setThreshold}>
            <SelectTrigger className="w-[190px]" aria-label="Ambang pemakaian">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {THRESHOLDS.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" onClick={handleRefresh} className="gap-2">
            <RefreshCw className="h-4 w-4" aria-hidden="true" />
            Segarkan
          </Button>
        </div>
      </div>

      <SmartInsightCard
        title="Insight Pemantauan Batas Kuota AI"
        summaryText={`Terdapat ${rows.length} akun terpantau mendekati ambang ${threshold}% batas paket siklus aktif. Pantau akun berstatus kritis untuk mencegah kegagalan eksekusi permintaan AI.`}
        deltaPercent={rows.length > 0 ? 12.5 : 0}
        deltaDirection={rows.length > 0 ? 'up' : 'flat'}
        isPositiveDelta={rows.length === 0}
        metricsSummary={[
          { label: 'Akun Terpantau', value: rows.length },
          { label: 'Ambang Batas', value: `≥ ${threshold}%` },
          { label: 'Status Pool', value: rows.length === 0 ? 'Aman' : 'Perlu Atensi' },
        ]}
      />

      <div className="flex flex-wrap items-center justify-between gap-3">
        <QuotaMonitorSummary rows={rows} />
      </div>

      <QuotaMonitorTable
        rows={rows}
        isLoading={isLoading}
        isError={isError}
        threshold={threshold}
        onRetry={() => refetch()}
      />

      <ModalityConsumptionChart points={timeseries ?? []} isLoading={isTimeseriesLoading} />

      {/* Period control for the modality chart, mirroring the dashboard. */}
      <div className="flex items-center gap-2">
        <span className="text-xs font-bold text-muted-foreground">Periode grafik:</span>
        <Select value={period} onValueChange={(v) => setPeriod(v as MetricsPeriod)}>
          <SelectTrigger className="w-[120px]" aria-label="Periode grafik modalitas">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="7d">7 hari</SelectItem>
            <SelectItem value="30d">30 hari</SelectItem>
            <SelectItem value="90d">90 hari</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

// The page reads useSearchParams, so it must sit under a Suspense boundary for
// the route to stay statically prerenderable.
export default function SuperAdminQuotaMonitorPage() {
  return (
    <Suspense
      fallback={
        <div className="space-y-6 p-6">
          <Skeleton className="h-10 w-72" />
          <Skeleton className="h-64 w-full" />
        </div>
      }
    >
      <SuperAdminQuotaMonitorContent />
    </Suspense>
  )
}
