'use client'

import { Cpu, RefreshCw } from 'lucide-react'
import { Button } from '@/components/ui/button'
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

      {/* 1. Kill Switch Panel */}
      <KillSwitchPanel />

      {/* 2. Routing Matrix Panel */}
      <RoutingMatrixPanel />

      {/* 3. Endpoints Panel */}
      <EndpointsPanel />
    </div>
  )
}
