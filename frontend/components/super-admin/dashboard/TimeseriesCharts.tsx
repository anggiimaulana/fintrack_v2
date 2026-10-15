'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { MultiSeriesChart } from '@/components/charts/MultiSeriesChart'
import { formatRupiah } from '@/lib/format'
import type { MetricsPeriod, TimeseriesPoint } from '@/hooks/useSuperAdminMetrics'
import {
  ResponsiveContainer,
  ComposedChart,
  Area,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  CartesianGrid,
  Legend,
} from 'recharts'

interface TimeseriesChartsProps {
  data?: TimeseriesPoint[]
  isLoading?: boolean
  period: MetricsPeriod
  onPeriodChange: (period: MetricsPeriod) => void
}

const PERIODS: Array<{ label: string; value: MetricsPeriod }> = [
  { label: '7 Hari', value: '7d' },
  { label: '30 Hari', value: '30d' },
  { label: '90 Hari', value: '90d' },
]

export function TimeseriesChartsSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-8 w-48" />
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 h-[380px]">
          <Skeleton className="h-6 w-36 mb-4" />
          <Skeleton className="h-[280px] w-full" />
        </Card>
        <Card className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 h-[380px]">
          <Skeleton className="h-6 w-36 mb-4" />
          <Skeleton className="h-[280px] w-full" />
        </Card>
      </div>
    </div>
  )
}

function RevenueChartTooltip({
  active,
  payload,
  label,
}: {
  active?: boolean
  payload?: Array<{ value: number; color: string; name: string; dataKey: string }>
  label?: string
}) {
  if (!active || !payload?.length) return null

  return (
    <div className="bg-card border-2 rounded-md border-foreground shadow-[4px_4px_0px_0px] shadow-foreground p-3 text-xs font-bold space-y-1.5 z-50">
      <p className="text-muted-foreground pb-1 border-b border-border">{label}</p>
      {payload.map((p, i) => (
        <div key={i} className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: p.color }} />
            <span>{p.name}:</span>
          </div>
          <span className="font-mono tabular-nums">
            {p.dataKey === 'payments_settled'
              ? `${p.value} trx`
              : formatRupiah(p.value)}
          </span>
        </div>
      ))}
    </div>
  )
}

export function TimeseriesCharts({
  data,
  isLoading,
  period,
  onPeriodChange,
}: TimeseriesChartsProps) {
  if (isLoading) {
    return <TimeseriesChartsSkeleton />
  }

  const points = data || []
  const hasRows = points.length > 0

  // Format labels: MM-DD or YYYY-MM-DD
  const tokenLabels = points.map((p) => p.date)
  const tokenSeries = [
    { name: 'Chat', data: points.map((p) => p.tokens_chat), color: '#3b82f6' },
    { name: 'Vision', data: points.map((p) => p.tokens_vision), color: '#8b5cf6' },
    { name: 'Voice', data: points.map((p) => p.tokens_voice), color: '#f59e0b' },
    { name: 'Embedding', data: points.map((p) => p.tokens_embedding), color: '#10b981' },
  ]

  return (
    <div className="space-y-4">
      {/* Segmented control for period */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <h2 className="text-lg font-black tracking-tight">Tren Performa & Penggunaan</h2>
        <div className="inline-flex border-2 rounded-md border-foreground bg-card p-0.5 shadow-[2px_2px_0px_0px] shadow-foreground">
          {PERIODS.map((p) => {
            const isActive = period === p.value
            return (
              <Button
                key={p.value}
                size="sm"
                variant={isActive ? 'default' : 'ghost'}
                onClick={() => onPeriodChange(p.value)}
                className={`h-7 px-3 text-xs font-black border-0 shadow-none active:translate-x-0 active:translate-y-0 rounded-sm ${
                  isActive ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
                }`}
              >
                {p.label}
              </Button>
            )
          })}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Chart 1: Revenue, MRR benchmark, & Settled Payments */}
        <Card className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col">
          <CardHeader className="p-5 pb-2">
            <CardTitle className="text-base font-black">Pendapatan & Pembayaran</CardTitle>
            <p className="text-xs text-muted-foreground font-medium">
              Pendapatan harian vs target garis MRR dan transaksi sukses
            </p>
          </CardHeader>
          <CardContent className="p-5 pt-0 flex-1 min-h-[320px]">
            {!hasRows ? (
              <div className="h-full flex items-center justify-center text-xs font-bold text-muted-foreground">
                Tidak ada data timeseries untuk periode ini
              </div>
            ) : (
              <div className="h-[300px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <ComposedChart
                    data={points}
                    margin={{ top: 15, right: 15, left: 10, bottom: 0 }}
                  >
                    <defs>
                      <linearGradient id="colorRev" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
                        <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                    <XAxis
                      dataKey="date"
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 11, fill: '#94a3b8' }}
                      dy={8}
                    />
                    <YAxis
                      yAxisId="left"
                      hide
                      domain={['auto', 'auto']}
                    />
                    <YAxis
                      yAxisId="right"
                      orientation="right"
                      hide
                      domain={[0, 'auto']}
                    />
                    <Tooltip content={<RevenueChartTooltip />} />
                    <Legend wrapperStyle={{ fontSize: 11, fontWeight: 700 }} iconType="circle" />
                    <Area
                      yAxisId="left"
                      type="monotone"
                      dataKey="revenue_idr"
                      name="Pendapatan (IDR)"
                      stroke="#10b981"
                      strokeWidth={2}
                      fillOpacity={1}
                      fill="url(#colorRev)"
                    />
                    <Line
                      yAxisId="left"
                      type="stepAfter"
                      dataKey="mrr_idr"
                      name="MRR Benchmark"
                      stroke="#64748b"
                      strokeWidth={2}
                      strokeDasharray="4 4"
                      dot={false}
                    />
                    <Line
                      yAxisId="right"
                      type="monotone"
                      dataKey="payments_settled"
                      name="Trx Settled"
                      stroke="#3b82f6"
                      strokeWidth={2}
                      dot={false}
                    />
                  </ComposedChart>
                </ResponsiveContainer>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Chart 2: Modalitas Token Usage (Stacked Bar / Multi-Series) */}
        <Card className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col">
          <CardHeader className="p-5 pb-2">
            <CardTitle className="text-base font-black">Konsumsi Token per Modalitas</CardTitle>
            <p className="text-xs text-muted-foreground font-medium">
              Distribusi token AI (Chat, Vision, Voice, Embedding)
            </p>
          </CardHeader>
          <CardContent className="p-5 pt-0 flex-1 min-h-[320px]">
            {!hasRows ? (
              <div className="h-full flex items-center justify-center text-xs font-bold text-muted-foreground">
                Tidak ada riwayat penggunaan token untuk periode ini
              </div>
            ) : (
              <div className="h-[300px] w-full">
                <MultiSeriesChart
                  labels={tokenLabels}
                  series={tokenSeries}
                  type="bar"
                />
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
