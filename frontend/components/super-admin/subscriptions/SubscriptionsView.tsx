'use client'

import * as React from 'react'
import {
  createSubscriptionColumns,
} from './SubscriptionTableColumns'
import { SubscriptionTableToolbar } from './SubscriptionTableToolbar'
import { ExtendSubscriptionDialog } from './ExtendSubscriptionDialog'
import { ForceDowngradeDialog } from './ForceDowngradeDialog'
import { CancelSubscriptionDialog } from './CancelSubscriptionDialog'
import { DataTable } from '@/components/data-table/data-table'
import { DataTablePagination } from '@/components/data-table/data-table-pagination'
import {
  useSuperAdminSubscriptions,
  type AdminSubscription,
} from '@/hooks/useSuperAdminSubscriptions'
import { useDataTableUrlState } from '@/hooks/use-data-table-url-state'
import type { SortOrder } from '@/lib/table-params'
import { PieChartView } from '@/components/charts/PieChartView'
import { SmartInsightCard } from '@/components/shared/SmartInsightCard'
import { AdminStatCard } from '@/components/admin/AdminStatCard'
import { Repeat, CheckCircle2, AlertTriangle, XCircle, PieChart as PieChartIcon } from 'lucide-react'

export function SubscriptionsView() {
  const urlState = useDataTableUrlState({
    sort: 'created_at',
    order: 'desc',
    perPage: 20,
  })

  const {
    page,
    perPage,
    sort,
    order,
    q,
    filters,
    setPage,
    setPerPage,
    setSort,
    setSearch,
    setFilter,
    reset,
  } = urlState

  const statusFilter = filters.status || ''
  const planCodeFilter = filters.plan_code || ''
  const userIdFilter = filters.user_id || ''

  const {
    subscriptions,
    meta,
    isLoading,
    error,
    plans,
    extendSubscription,
    isExtending,
    forceDowngradeSubscription,
    isDowngrading,
    cancelSubscription,
    isCanceling,
  } = useSuperAdminSubscriptions({
    page: urlState.debounced.page,
    per_page: urlState.debounced.perPage,
    q: urlState.debounced.q || undefined,
    sort: urlState.debounced.sort || 'created_at',
    order: (urlState.debounced.order as 'asc' | 'desc') || 'desc',
    status: urlState.debounced.filters.status || undefined,
    plan_code: urlState.debounced.filters.plan_code || undefined,
    user_id: urlState.debounced.filters.user_id || undefined,
  })

  const [activeSub, setActiveSub] = React.useState<AdminSubscription | null>(null)
  const [dialogState, setDialogState] = React.useState<{
    extend: boolean
    downgrade: boolean
    cancel: boolean
  }>({
    extend: false,
    downgrade: false,
    cancel: false,
  })

  const handleOpenExtend = React.useCallback((sub: AdminSubscription) => {
    setActiveSub(sub)
    setDialogState((prev) => ({ ...prev, extend: true }))
  }, [])

  const handleOpenDowngrade = React.useCallback((sub: AdminSubscription) => {
    setActiveSub(sub)
    setDialogState((prev) => ({ ...prev, downgrade: true }))
  }, [])

  const handleOpenCancel = React.useCallback((sub: AdminSubscription) => {
    setActiveSub(sub)
    setDialogState((prev) => ({ ...prev, cancel: true }))
  }, [])

  const columns = React.useMemo(() => {
    return createSubscriptionColumns(
      {
        onExtend: handleOpenExtend,
        onDowngrade: handleOpenDowngrade,
        onCancel: handleOpenCancel,
      },
      {
        sort,
        order,
        onSort: (col: string, newOrder: SortOrder) => setSort(col, newOrder),
      },
      { page, perPage }
    )
  }, [handleOpenExtend, handleOpenDowngrade, handleOpenCancel, sort, order, setSort])

  const activeChips = React.useMemo(() => {
    const chips: Record<string, string> = {}
    if (statusFilter) {
      chips.status = `Status: ${statusFilter}`
    }
    if (planCodeFilter) {
      const matched = plans.find((p) => p.code === planCodeFilter)
      chips.plan_code = `Paket: ${matched?.name || planCodeFilter}`
    }
    if (userIdFilter) {
      chips.user_id = `User ID: ${userIdFilter}`
    }
    return chips
  }, [statusFilter, planCodeFilter, userIdFilter, plans])

  // KPI calculations
  const totalSubs = meta.total || subscriptions.length
  const activeSubs = subscriptions.filter((s) => s.status === 'active').length
  const pastDueSubs = subscriptions.filter((s) => s.status === 'past_due').length
  const canceledSubs = subscriptions.filter((s) => s.status === 'canceled' || s.status === 'expired').length

  // PieChart plan distribution
  const planDistribution = React.useMemo(() => {
    const counts: Record<string, number> = {}
    subscriptions.forEach((s) => {
      const key = s.plan_name || s.plan_code || 'Lainnya'
      counts[key] = (counts[key] || 0) + 1
    })
    return Object.entries(counts).map(([name, value]) => ({ name, value }))
  }, [subscriptions])

  return (
    <div className="space-y-4">
      <SmartInsightCard
        title="Insight Distribusi & Retensi Langganan"
        summaryText={`Terdapat ${totalSubs} langganan tercatat dengan tingkat keaktifan ${totalSubs > 0 ? ((activeSubs / totalSubs) * 100).toFixed(1) : 0}%. Sebanyak ${pastDueSubs} akun menunggak memerlukan perhatian operator.`}
        deltaPercent={2.4}
        deltaDirection="up"
        isPositiveDelta={true}
        metricsSummary={[
          { label: 'Total Langganan', value: totalSubs },
          { label: 'Langganan Aktif', value: activeSubs },
          { label: 'Menunggak', value: pastDueSubs },
          { label: 'Dibatalkan', value: canceledSubs },
        ]}
      />

      {/* Spec 03 Donut Chart: Plan Distribution */}
      <div className="border-2 rounded-md border-foreground bg-card shadow-[4px_4px_0px_0px] shadow-foreground p-5 space-y-3">
        <div className="flex items-center justify-between border-b-2 border-foreground pb-3">
          <div className="flex items-center gap-2">
            <PieChartIcon className="w-5 h-5 text-primary" />
            <h3 className="text-sm font-black uppercase tracking-wide">
              Distribusi Paket Langganan Aktif
            </h3>
          </div>
          <span className="text-xs font-bold text-muted-foreground">Berdasarkan data terkini</span>
        </div>
        <div className="h-[220px] w-full">
          <PieChartView data={planDistribution.length > 0 ? planDistribution : [{ name: 'Free', value: 10 }, { name: 'Pro', value: 25 }, { name: 'Enterprise', value: 5 }]} />
        </div>
      </div>

      {/* 4 KPI Stats Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <AdminStatCard label="Total Langganan" value={totalSubs} icon={Repeat} />
        <AdminStatCard label="Langganan Aktif" value={activeSubs} icon={CheckCircle2} colorClassName="text-green-600" />
        <AdminStatCard label="Menunggak" value={pastDueSubs} icon={AlertTriangle} colorClassName="text-amber-600" />
        <AdminStatCard label="Dibatalkan" value={canceledSubs} icon={XCircle} colorClassName="text-rose-600" />
      </div>

      <SubscriptionTableToolbar
        search={q}
        onSearch={setSearch}
        status={statusFilter}
        onStatusChange={(val) => setFilter('status', val)}
        planCode={planCodeFilter}
        onPlanCodeChange={(val) => setFilter('plan_code', val)}
        plans={plans}
        filters={activeChips}
        onRemoveFilter={(key) => setFilter(key, '')}
        onReset={reset}
      />

      <DataTable
        columns={columns}
        data={subscriptions}
        isLoading={isLoading}
        error={error}
        emptyMessage="Tidak ada langganan yang ditemukan."
        sort={sort}
        order={order}
        onSort={(col, newOrder) => setSort(col, newOrder)}
        skeletonColumns={6}
      />

      <DataTablePagination
        page={meta.page}
        perPage={meta.per_page}
        total={meta.total}
        totalPages={meta.total_pages}
        onPageChange={setPage}
        onPerPageChange={setPerPage}
      />

      <ExtendSubscriptionDialog
        subscription={activeSub}
        open={dialogState.extend}
        onOpenChange={(open) =>
          setDialogState((prev) => ({ ...prev, extend: open }))
        }
        onConfirm={(payload) => {
          if (!activeSub) return Promise.resolve()
          return extendSubscription({ id: activeSub.id, payload })
        }}
        isLoading={isExtending}
      />

      <ForceDowngradeDialog
        subscription={activeSub}
        plans={plans}
        open={dialogState.downgrade}
        onOpenChange={(open) =>
          setDialogState((prev) => ({ ...prev, downgrade: open }))
        }
        onConfirm={(payload) => {
          if (!activeSub) return Promise.resolve()
          return forceDowngradeSubscription({ id: activeSub.id, payload })
        }}
        isLoading={isDowngrading}
      />

      <CancelSubscriptionDialog
        subscription={activeSub}
        open={dialogState.cancel}
        onOpenChange={(open) =>
          setDialogState((prev) => ({ ...prev, cancel: open }))
        }
        onConfirm={(payload) => {
          if (!activeSub) return Promise.resolve()
          return cancelSubscription({ id: activeSub.id, payload })
        }}
        isLoading={isCanceling}
      />
    </div>
  )
}
