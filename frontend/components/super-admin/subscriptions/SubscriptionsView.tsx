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
      }
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

  return (
    <div className="space-y-4">
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
