'use client'

import { Cpu, RefreshCw, Activity } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'
import { LineChartView } from '@/components/charts/LineChartView'
import {
  KillSwitchPanel,
  EndpointsPanel,
  RoutingMatrixPanel,
} from '@/components/super-admin/gateway/GatewayPanels'
import { useSuperAdminGateway } from '@/hooks/useSuperAdminGateway'

export default function SuperAdminAiGatewayPage() {
  const {
    refetchEndpoints,
    refetchRoutes,
    refetchKillSwitch,
    isRefetchingEndpoints,
    isRefetchingRoutes,
  } = useSuperAdminGateway()

  const isRefreshing = isRefetchingEndpoints || isRefetchingRoutes

  const handleRefreshAll = () => {
    refetchEndpoints()
    refetchRoutes()
    refetchKillSwitch()
  }

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Cpu className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              AI Gateway Control Plane
            </h1>
          </div>
          <p className="text-sm text-muted-foreground">
            Kelola endpoint penyedia model AI, konfigurasi matriks rute use-case, dan sakelar pemutus darurat (kill switch).
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRefreshAll}
          disabled={isRefreshing}
          className="gap-2 self-start sm:self-auto"
        >
          <RefreshCw className={`h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          Segarkan Data
        </Button>
      </div>

      <SmartInsightCard
        title="Insight Operasional & Reliabilitas AI Gateway"
        summaryText="Status pool AI Gateway optimal tanpa insiden pemutus sirkuit (circuit breaker) dalam 24 jam terakhir. Latensi p95 stabil pada 420ms lintas seluruh rute model aktif."
        deltaPercent={-12.5}
        deltaDirection="down"
        isPositiveDelta={true}
        metricsSummary={[
          { label: 'Circuit Breaker', value: 'Semua Normal' },
          { label: 'Latensi p95', value: '420ms' },
          { label: 'Health Status', value: '100% Sehat' },
        ]}
      />

      {/* Spec 03 LineChartView for Gateway Latency p95/p99 */}
      <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 space-y-3">
        <div className="flex items-center justify-between border-b-2 border-foreground pb-3">
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-primary" />
            <h3 className="text-sm font-black uppercase tracking-wide">
              Latensi Gateway AI (ms) per Rute Use-Case
            </h3>
          </div>
          <span className="text-xs font-bold text-muted-foreground">Periode: 7 Hari Terakhir</span>
        </div>
        <div className="h-[220px] w-full">
          <LineChartView
            data={[
              { name: 'Sen', income: 420, expense: 510, savings: 380 },
              { name: 'Sel', income: 390, expense: 480, savings: 360 },
              { name: 'Rab', income: 440, expense: 530, savings: 410 },
              { name: 'Kam', income: 410, expense: 490, savings: 390 },
              { name: 'Jum', income: 430, expense: 500, savings: 400 },
              { name: 'Sab', income: 380, expense: 460, savings: 350 },
              { name: 'Min', income: 400, expense: 470, savings: 370 },
            ]}
          />
        </div>
      </div>

      {/* 1. Kill Switch Panel */}
      <KillSwitchPanel />

      {/* 2. Routing Matrix Panel */}
      <RoutingMatrixPanel />

      {/* 3. Endpoints Panel */}
      <EndpointsPanel />
    </div>
  )
}
