'use client'

import { useMemo } from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { TimeseriesPoint } from '@/hooks/useSuperAdminMetrics'

/** The four token modalities the AI gateway meters, with display colours. */
const MODALITIES = [
  { key: 'tokens_chat', label: 'Chat', color: '#3b82f6' },
  { key: 'tokens_vision', label: 'Vision (OCR)', color: '#8b5cf6' },
  { key: 'tokens_voice', label: 'Voice', color: '#f59e0b' },
  { key: 'tokens_embedding', label: 'Embedding', color: '#10b981' },
] as const

/** Totals per modality over the whole series. */
export function modalityTotals(points: TimeseriesPoint[]): Array<{ label: string; total: number; color: string }> {
  return MODALITIES.map((m) => ({
    label: m.label,
    color: m.color,
    total: points.reduce((sum, p) => sum + (p[m.key] ?? 0), 0),
  }))
}

/**
 * Stacked bar chart of token consumption per modality over time.
 *
 * Uses the existing metrics timeseries (which already aggregates tokens by
 * modality), so no new endpoint is required. Operators use it to see which
 * modality drives cost.
 */
export function ModalityConsumptionChart({
  points,
  isLoading,
}: {
  points: TimeseriesPoint[]
  isLoading: boolean
}) {
  const totals = useMemo(() => modalityTotals(points), [points])
  const grandTotal = totals.reduce((s, t) => s + t.total, 0)

  if (isLoading) {
    return <Skeleton className="h-[300px] w-full" />
  }

  return (
    <Card className="border-2 border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground">
      <CardHeader className="p-5 pb-2">
        <CardTitle className="text-base font-black">Konsumsi Token per Modalitas</CardTitle>
        <CardDescription className="text-xs font-medium">
          Total {grandTotal.toLocaleString('id-ID')} token pada periode ini — dasar perhitungan biaya AI.
        </CardDescription>
      </CardHeader>
      <CardContent className="p-5 pt-0">
        {points.length === 0 ? (
          <div className="flex h-[260px] items-center justify-center text-xs font-bold text-muted-foreground">
            Belum ada konsumsi token pada periode ini
          </div>
        ) : (
          <div className="h-[280px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={points} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e2e8f0" />
                <XAxis
                  dataKey="date"
                  axisLine={false}
                  tickLine={false}
                  tick={{ fontSize: 11, fill: '#94a3b8' }}
                  dy={8}
                />
                <YAxis axisLine={false} tickLine={false} tick={{ fontSize: 11, fill: '#94a3b8' }} width={56} />
                <Tooltip
                  formatter={(value) => Number(value).toLocaleString('id-ID')}
                  contentStyle={{ border: '2px solid #0f172a', borderRadius: 0, fontSize: 12, fontWeight: 700 }}
                />
                <Legend wrapperStyle={{ fontSize: 11, fontWeight: 700 }} />
                {MODALITIES.map((m) => (
                  <Bar key={m.key} dataKey={m.key} name={m.label} stackId="tokens" fill={m.color} />
                ))}
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
