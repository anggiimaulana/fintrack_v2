'use client'

import { useEffect } from 'react'
import { toast } from 'sonner'
import { RotateCw, LayoutDashboard } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useSuperAdminMetrics } from '@/hooks/useSuperAdminMetrics'
import { OverviewKpiCards, OverviewKpiCardsSkeleton } from '@/components/super-admin/dashboard/OverviewKpiCards'
import { CircuitBreakerBanner } from '@/components/super-admin/dashboard/CircuitBreakerBanner'
import { TimeseriesCharts, TimeseriesChartsSkeleton } from '@/components/super-admin/dashboard/TimeseriesCharts'
import { PaymentMethodsDonut, PaymentMethodsDonutSkeleton } from '@/components/super-admin/dashboard/PaymentMethodsDonut'

export default function SuperAdminDashboardPage() {
  const {
    period,
    setPeriod,
    overview,
    isOverviewLoading,
    overviewError,
    refreshOverview,
    isRefreshing,
    timeseries,
    isTimeseriesLoading,
    timeseriesError,
    paymentMethods,
    isPaymentMethodsLoading,
    paymentMethodsError,
  } = useSuperAdminMetrics()

  // Handle toast errors for any failed queries
  useEffect(() => {
    const error = overviewError || timeseriesError || paymentMethodsError
    if (error) {
      const axiosError = error as { response?: { data?: { message?: string } }; message?: string }
      const msg = axiosError.response?.data?.message || axiosError.message || 'Gagal memuat data metrik dashboard'
      toast.error(msg)
    }
  }, [overviewError, timeseriesError, paymentMethodsError])

  const handleRefresh = async () => {
    try {
      await refreshOverview()
      toast.success('Metrik dashboard berhasil diperbarui')
    } catch (err) {
      const axiosError = err as { response?: { data?: { message?: string } }; message?: string }
      const msg = axiosError.response?.data?.message || axiosError.message || 'Gagal menyegarkan metrik'
      toast.error(msg)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b-2 border-foreground">
        <div>
          <div className="flex items-center gap-2">
            <LayoutDashboard className="h-6 w-6" />
            <h1 className="text-2xl font-black tracking-tight">Super Admin Dashboard</h1>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Ringkasan pendapatan, operasional token, dan kesehatan sistem FinTrack.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {overview?.cached_at && (
            <span className="text-xs text-muted-foreground tabular-nums hidden md:inline">
              Cache: {new Date(overview.cached_at).toLocaleTimeString('id-ID')}
            </span>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isRefreshing || isOverviewLoading}
            className="h-8 gap-2 font-bold"
          >
            <RotateCw className={`h-3.5 w-3.5 ${isRefreshing ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Warning Banner when circuit breaker is tripped */}
      {overview && (
        <CircuitBreakerBanner count={overview.circuit_breaker_open_count} />
      )}

      {/* Overview KPI Cards */}
      {isOverviewLoading ? (
        <OverviewKpiCardsSkeleton />
      ) : overview ? (
        <OverviewKpiCards overview={overview} isLoading={isOverviewLoading} />
      ) : overviewError ? (
        <div className="p-6 border-2 border-destructive bg-card text-destructive font-bold text-sm">
          Gagal memuat ringkasan KPI. Silakan coba lagi.
        </div>
      ) : null}

      {/* Grid: Timeseries & Donut */}
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        <div className="xl:col-span-2 space-y-6">
          {isTimeseriesLoading ? (
            <TimeseriesChartsSkeleton />
          ) : (
            <TimeseriesCharts
              data={timeseries}
              isLoading={isTimeseriesLoading}
              period={period}
              onPeriodChange={setPeriod}
            />
          )}
        </div>

        <div className="space-y-4">
          <h2 className="text-lg font-black tracking-tight">Metode Pembayaran</h2>
          {isPaymentMethodsLoading ? (
            <PaymentMethodsDonutSkeleton />
          ) : (
            <PaymentMethodsDonut
              data={paymentMethods}
              isLoading={isPaymentMethodsLoading}
            />
          )}
        </div>
      </div>
    </div>
  )
}
