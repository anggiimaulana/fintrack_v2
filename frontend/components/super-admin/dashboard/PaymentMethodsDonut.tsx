'use client'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PieChartView } from '@/components/charts/PieChartView'
import { Skeleton } from '@/components/ui/skeleton'
import { Money } from '@/components/shared/Money'
import type { PaymentMethodShare } from '@/hooks/useSuperAdminMetrics'

interface PaymentMethodsDonutProps {
  data?: PaymentMethodShare[]
  isLoading?: boolean
}

export function PaymentMethodsDonutSkeleton() {
  return (
    <Card className="border-2 rounded-lg border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 flex flex-col h-[400px]">
      <div className="flex justify-between items-center mb-4">
        <Skeleton className="h-5 w-40" />
        <Skeleton className="h-4 w-20" />
      </div>
      <Skeleton className="flex-1 w-full" />
    </Card>
  )
}

export function PaymentMethodsDonut({ data, isLoading }: PaymentMethodsDonutProps) {
  if (isLoading) {
    return <PaymentMethodsDonutSkeleton />
  }

  const items = data || []
  const hasRows = items.length > 0 && items.some((item) => (item.amount_idr ?? item.total_idr ?? 0) > 0 || item.settled_count > 0)

  const chartData = items.map((item) => {
    const amount = item.amount_idr ?? item.total_idr ?? 0
    return {
      name: item.gateway.toUpperCase(),
      value: amount,
    }
  })

  return (
    <Card className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground flex flex-col h-full">
      <CardHeader className="p-5 pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base font-black">Metode Pembayaran</CardTitle>
          <span className="text-xs font-bold text-muted-foreground">30 hari</span>
        </div>
      </CardHeader>
      <CardContent className="p-5 pt-0 flex-1 flex flex-col">
        {!hasRows ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center p-8 text-muted-foreground text-sm font-bold min-h-[260px]">
            <p>Tidak ada transaksi pembayaran settled dalam 30 hari terakhir</p>
          </div>
        ) : (
          <div className="flex-1 flex flex-col">
            <div className="h-[220px] w-full">
              <PieChartView data={chartData} />
            </div>

            <div className="mt-4 border-t-2 border-foreground pt-3 space-y-2">
              {items.map((item) => {
                const amount = item.amount_idr ?? item.total_idr ?? 0
                return (
                  <div key={item.gateway} className="flex items-center justify-between text-xs font-bold">
                    <span className="uppercase text-muted-foreground">{item.gateway}</span>
                    <div className="flex items-center gap-3">
                      <span className="tabular-nums font-mono text-muted-foreground">
                        {item.settled_count} transaksi
                      </span>
                      <Money value={amount} />
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
